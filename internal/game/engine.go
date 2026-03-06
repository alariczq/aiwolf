package game

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/alaric/eino-learn/internal/action"
	"github.com/alaric/eino-learn/internal/callback"
	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/model"
	"github.com/alaric/eino-learn/internal/narrator"
	"github.com/alaric/eino-learn/internal/player"
	"github.com/alaric/eino-learn/internal/prompt"
	"github.com/alaric/eino-learn/internal/role"
)

type Engine struct {
	state    *GameState
	provider *model.Provider
	narrator *narrator.Narrator
	logger   *callback.GameLogger
	emit     EventEmitter
	silent   bool
	setting  string

	gameRules      string
	lastAPICall    time.Time
	callInterval   time.Duration
	callTimeout    time.Duration
	witchSelfSave  config.WitchSelfSave
	identityReveal config.IdentityReveal
}

func NewEngine(ctx context.Context, cfg config.GameConfig, opts ...EngineOption) (*Engine, error) {
	prov := model.NewProvider(cfg.Models)

	var players []*player.Player
	for _, pc := range cfg.Players {
		r, err := role.Get(pc.Role)
		if err != nil {
			return nil, fmt.Errorf("player %s: %w", pc.Name, err)
		}
		m, err := prov.GetModel(ctx, pc.ModelID)
		if err != nil {
			return nil, fmt.Errorf("player %s: %w", pc.Name, err)
		}
		players = append(players, player.New(pc.Name, r, pc.ModelID, m, pc.Persona))
	}

	narratorModel, err := pickNarratorModel(ctx, prov)
	if err != nil {
		return nil, fmt.Errorf("narrator model: %w", err)
	}
	narr, err := narrator.New(ctx, narratorModel)
	if err != nil {
		return nil, fmt.Errorf("creating narrator: %w", err)
	}

	gl := callback.NewGameLogger()
	callbacks.AppendGlobalHandlers(gl.Handler())

	state := NewGameState(players)
	state.VictoryMode = cfg.VictoryMode

	eng := &Engine{
		state:          state,
		provider:       prov,
		narrator:       narr,
		logger:         gl,
		callInterval:   1500 * time.Millisecond,
		callTimeout:    120 * time.Second,
		setting:        cfg.Setting,
		witchSelfSave:  cfg.WitchSelfSave,
		identityReveal: cfg.IdentityReveal,
	}

	roleCounts := make(map[string]int)
	for _, p := range players {
		roleCounts[p.Role.Name()]++
	}
	eng.gameRules = prompt.BuildGameRules(roleCounts, prompt.RulesVariant{
		WitchSelfSave:  int(cfg.WitchSelfSave),
		IdentityReveal: int(cfg.IdentityReveal),
		VictoryMode:    int(cfg.VictoryMode),
	})

	for _, opt := range opts {
		opt(eng)
	}
	return eng, nil
}

func pickNarratorModel(ctx context.Context, prov *model.Provider) (einomodel.ToolCallingChatModel, error) {
	for _, id := range []string{"claude-haiku", "gemini-flash", "claude-sonnet", "gemini-pro"} {
		m, err := prov.GetModel(ctx, id)
		if err == nil {
			return m, nil
		}
	}
	return nil, fmt.Errorf("no model available for narrator")
}

func (e *Engine) Run(ctx context.Context) error {
	e.println("=== AI 狼人杀游戏开始！ ===")
	e.println()
	e.printPlayerRoster()
	if e.setting != "" {
		e.println(e.setting)
		e.println()
	}
	e.emitEvent(UIEvent{
		Type:    "game_start",
		Players: buildUIPlayers(e.state.Players, true),
		Content: e.setting,
	})

	e.openingNarration(ctx)

	for {
		e.state.Round++
		e.printf("\n========== 第 %d 回合 ==========\n\n", e.state.Round)
		e.emitEvent(UIEvent{Type: "round_start", Round: e.state.Round})

		if err := e.nightPhase(ctx); err != nil {
			return fmt.Errorf("night phase round %d: %w", e.state.Round, err)
		}

		deaths, err := e.resolveNight(ctx)
		if err != nil {
			return fmt.Errorf("resolve night round %d: %w", e.state.Round, err)
		}

		if win := CheckWinAfterNight(e.state); win.GameOver {
			return e.endGame(ctx, win)
		}

		if e.state.Round == 1 && e.state.Sheriff == "" && !e.state.SheriffElectionCancelled {
			if err := e.sheriffElection(ctx); err != nil {
				return fmt.Errorf("sheriff election: %w", err)
			}
			if e.state.WolfSelfExploded != "" {
				e.state.WolfSelfExploded = ""
				if win := CheckWinAfterVote(e.state); win.GameOver {
					return e.endGame(ctx, win)
				}
			}
		}

		if err := e.dayPhase(ctx, deaths); err != nil {
			return fmt.Errorf("day phase round %d: %w", e.state.Round, err)
		}

		if e.state.WolfSelfExploded != "" {
			e.state.WolfSelfExploded = ""
			if win := CheckWinAfterVote(e.state); win.GameOver {
				return e.endGame(ctx, win)
			}
			continue
		}

		if err := e.votePhase(ctx); err != nil {
			return fmt.Errorf("vote phase round %d: %w", e.state.Round, err)
		}

		if win := CheckWinAfterVote(e.state); win.GameOver {
			return e.endGame(ctx, win)
		}
	}
}

func (e *Engine) Logger() *callback.GameLogger {
	return e.logger
}

func (e *Engine) sheriffElection(ctx context.Context) error {
	slog.Info("phase start", "phase", "sheriff", "round", e.state.Round, "alive", len(e.state.AlivePlayers()))
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "sheriff", Round: e.state.Round})

	alivePlayers := e.state.AlivePlayers()
	if len(alivePlayers) == 0 {
		return nil
	}

	e.println("--- 警长竞选：上警/不上警 ---")

	type decisionResult struct {
		run  *bool
		name string
	}
	var decisionAgents []adk.Agent
	decisions := make([]decisionResult, len(alivePlayers))

	for i, p := range alivePlayers {
		var run bool
		decisions[i] = decisionResult{run: &run, name: p.Name}

		decTool, err := action.CreateCampaignDecisionTool(&run)
		if err != nil {
			return fmt.Errorf("creating campaign decision tool for %s: %w", p.Name, err)
		}

		pctx := e.buildPromptContext(p)
		instruction := prompt.BuildCampaignDecision(pctx)

		agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("campaign_dec_%s", p.Name),
			Description: fmt.Sprintf("%s decides whether to run for sheriff", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{decTool},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("creating campaign decision agent for %s: %w", p.Name, err)
		}
		decisionAgents = append(decisionAgents, agent)
	}

	slog.Info("sheriff campaign decision start", "player_count", len(alivePlayers))
	decCtx, decCancel := e.withCallTimeout(ctx)
	defer decCancel()

	parAgent, err := adk.NewParallelAgent(decCtx, &adk.ParallelAgentConfig{
		Name:        "campaign_decisions",
		Description: "All players decide whether to run for sheriff",
		SubAgents:   decisionAgents,
	})
	if err != nil {
		return fmt.Errorf("creating campaign decision parallel agent: %w", err)
	}

	runner := adk.NewRunner(decCtx, adk.RunnerConfig{Agent: parAgent})
	iter := runner.Query(decCtx, "请决定你是否要上警竞选警长。用中文。")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			slog.Warn("campaign decision API error", "error", event.Err)
			continue
		}
		msg, _, merr := adk.GetMessage(event)
		if merr != nil {
			continue
		}
		if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && event.AgentName != "" {
			pName := strings.TrimPrefix(event.AgentName, "campaign_dec_")
			if p := e.state.GetPlayer(pName); p != nil {
				e.printf("  [%s 内心] %s\n", displayTag(p), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: msg.Content, Round: e.state.Round, Phase: "sheriff"})
			}
		}
	}

	slog.Info("sheriff campaign decision complete", "player_count", len(alivePlayers))

	var candidates []*player.Player
	var voters []*player.Player
	candidateNames := []string{}
	for _, d := range decisions {
		p := e.state.GetPlayer(d.name)
		if p == nil {
			continue
		}
		if *d.run {
			candidates = append(candidates, p)
			candidateNames = append(candidateNames, p.Name)
			e.printf("[上警] %s 决定上警竞选。\n", displayTag(p))
			e.emitEvent(UIEvent{Type: "campaign_run", Player: p.Name, Round: e.state.Round})
		} else {
			voters = append(voters, p)
			e.printf("[不上警] %s 决定不上警。\n", displayTag(p))
		}
	}

	if len(candidates) == 0 {
		e.println("没有人上警竞选，本局无警长。")
		e.emitEvent(UIEvent{Type: "narration", Content: "没有人上警竞选，本局无警长。", Round: e.state.Round})
		return nil
	}

	if len(candidates) == 1 {
		winner := candidates[0]
		e.state.Sheriff = winner.Name
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseDay,
			Type:    EventVote,
			Target:  winner.Name,
			Content: fmt.Sprintf("只有 %s 一人上警，自动当选警长！", winner.Name),
			Public:  true,
		})
		e.printf("\n[警长竞选] %s 自动当选警长！（唯一候选人）\n\n", displayTag(winner))
		e.emitEvent(UIEvent{Type: "sheriff_elected", Player: winner.Name, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
		return nil
	}

	e.println("--- 警长竞选发言阶段 ---")

	var campaignSpeeches []Speech
	for _, p := range candidates {
		var speechResult string
		speakTool, serr := action.CreateSpeakTool(&speechResult)
		if serr != nil {
			return fmt.Errorf("creating speak tool for %s: %w", p.Name, serr)
		}

		var selfExploded bool
		var campaignTools []tool.BaseTool
		campaignTools = append(campaignTools, speakTool)

		if p.Role.Team() == config.TeamWerewolf {
			explodeTool, eerr := action.CreateSelfExplodeTool(p.Name, &selfExploded)
			if eerr != nil {
				return fmt.Errorf("creating self-explode tool for %s: %w", p.Name, eerr)
			}
			campaignTools = append(campaignTools, explodeTool)
		}

		pctx := e.buildPromptContext(p)
		var sb strings.Builder
		for _, sp := range campaignSpeeches {
			fmt.Fprintf(&sb, "%s: %s\n", sp.Speaker, sp.Content)
		}
		pctx.SheriffSpeeches = sb.String()
		instruction := prompt.BuildSheriffCampaign(pctx)

		e.emitEvent(UIEvent{Type: "thinking_start", Player: p.Name, Round: e.state.Round, Phase: "sheriff"})

		var thoughts strings.Builder
		serr = e.retryOnTransient(p.Name, func() error {
			thoughts.Reset()
			callCtx, cancel := e.withCallTimeout(ctx)
			defer cancel()

			agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
				Name:        p.Name,
				Description: fmt.Sprintf("%s campaigns for sheriff", p.Name),
				Instruction: instruction,
				Model:       p.Model,
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: campaignTools,
					},
				},
			})
			if aerr != nil {
				return aerr
			}

			r := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
			it := r.Query(callCtx, "你已上警竞选警长。请发表你的竞选演说。用中文。")

			for {
				event, ok := it.Next()
				if !ok {
					break
				}
				if event.Err != nil {
					return event.Err
				}
				msg, _, merr := adk.GetMessage(event)
				if merr != nil || msg == nil {
					continue
				}
				if msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
					thoughts.WriteString(msg.Content)
				}
			}
			return nil
		})
		if serr != nil {
			return fmt.Errorf("sheriff campaign %s: %w", p.Name, serr)
		}

		if selfExploded {
			p.Alive = false
			e.state.WolfSelfExploded = p.Name
			e.state.SheriffElectionCancelled = true
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseDay,
				Type:    EventDeath,
				Actor:   p.Name,
				Target:  p.Name,
				Content: fmt.Sprintf("%s 在警长竞选中自爆！（身份：%s）警长竞选取消（吞警徽）。", p.Name, roleChineseName(p.Role.Name())),
				Public:  true,
			})
			e.printf("[自爆] %s 在警长竞选中自爆！竞选取消（吞警徽）。\n", displayTag(p))
			e.emitEvent(UIEvent{Type: "self_explode", Player: p.Name, Role: p.Role.Name(), Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
			return nil
		}

		if thoughts.Len() > 0 && thoughts.String() != speechResult {
			e.printf("  [%s 内心] %s\n", displayTag(p), thoughts.String())
			e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: thoughts.String(), Round: e.state.Round, Phase: "sheriff"})
		}

		if speechResult != "" {
			campaignSpeeches = append(campaignSpeeches, Speech{Speaker: p.Name, Content: speechResult})
			e.printf("[竞选发言] %s: %s\n", displayTag(p), speechResult)
			e.emitEvent(UIEvent{Type: "speech", Player: p.Name, Content: speechResult, Round: e.state.Round, Phase: "sheriff"})
		}
	}

	if len(candidates) > 1 {
		e.println("--- 退水阶段 ---")

		var speechSummaryForWithdraw strings.Builder
		for _, sp := range campaignSpeeches {
			fmt.Fprintf(&speechSummaryForWithdraw, "%s: %s\n", sp.Speaker, sp.Content)
		}

		var remainingCandidates []*player.Player
		remainingCandidateNames := []string{}

		for _, p := range candidates {
			var wd bool
			wdTool, werr := action.CreateWithdrawDecisionTool(&wd)
			if werr != nil {
				return fmt.Errorf("creating withdraw decision tool for %s: %w", p.Name, werr)
			}

			var selfExploded bool
			var wdTools []tool.BaseTool
			wdTools = append(wdTools, wdTool)

			if p.Role.Team() == config.TeamWerewolf {
				explodeTool, eerr := action.CreateSelfExplodeTool(p.Name, &selfExploded)
				if eerr != nil {
					return fmt.Errorf("creating self-explode tool for %s: %w", p.Name, eerr)
				}
				wdTools = append(wdTools, explodeTool)
			}

			pctx := e.buildPromptContext(p)
			pctx.SheriffSpeeches = speechSummaryForWithdraw.String()
			instruction := prompt.BuildWithdrawDecision(pctx)

			werr = e.retryOnTransient(p.Name, func() error {
				callCtx, cancel := e.withCallTimeout(ctx)
				defer cancel()

				agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
					Name:        fmt.Sprintf("withdraw_%s", p.Name),
					Description: fmt.Sprintf("%s decides whether to withdraw from sheriff race", p.Name),
					Instruction: instruction,
					Model:       p.Model,
					ToolsConfig: adk.ToolsConfig{
						ToolsNodeConfig: compose.ToolsNodeConfig{
							Tools: wdTools,
						},
					},
				})
				if aerr != nil {
					return aerr
				}

				r := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
				it := r.Query(callCtx, "竞选发言结束，请决定是否退水。用中文。")

				for {
					event, ok := it.Next()
					if !ok {
						break
					}
					if event.Err != nil {
						return event.Err
					}
					msg, _, merr := adk.GetMessage(event)
					if merr != nil || msg == nil {
						continue
					}
					if msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
						e.printf("  [%s 内心] %s\n", displayTag(p), msg.Content)
						e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: msg.Content, Round: e.state.Round, Phase: "sheriff"})
					}
				}
				return nil
			})
			if werr != nil {
				return fmt.Errorf("withdraw decision %s: %w", p.Name, werr)
			}

			if selfExploded {
				p.Alive = false
				e.state.WolfSelfExploded = p.Name
				e.state.SheriffElectionCancelled = true
				e.state.AddEvent(GameEvent{
					Round:   e.state.Round,
					Phase:   PhaseDay,
					Type:    EventDeath,
					Actor:   p.Name,
					Target:  p.Name,
					Content: fmt.Sprintf("%s 在退水阶段自爆！（身份：%s）警长竞选取消（吞警徽）。", p.Name, roleChineseName(p.Role.Name())),
					Public:  true,
				})
				e.printf("[自爆] %s 在退水阶段自爆！竞选取消（吞警徽）。\n", displayTag(p))
				e.emitEvent(UIEvent{Type: "self_explode", Player: p.Name, Role: p.Role.Name(), Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
				return nil
			}

			if wd {
				e.printf("[退水] %s 选择退水，放弃竞选。\n", displayTag(p))
				e.emitEvent(UIEvent{Type: "narration", Content: fmt.Sprintf("%s 退水，放弃竞选。", p.Name), Round: e.state.Round})
			} else {
				remainingCandidates = append(remainingCandidates, p)
				remainingCandidateNames = append(remainingCandidateNames, p.Name)
				e.printf("[继续竞选] %s 选择继续竞选。\n", displayTag(p))
			}
		}

		candidates = remainingCandidates
		candidateNames = remainingCandidateNames

		if len(candidates) == 0 {
			e.println("所有候选人退水，本局无警长。")
			e.emitEvent(UIEvent{Type: "narration", Content: "所有候选人退水，本局无警长。", Round: e.state.Round})
			return nil
		}

		if len(candidates) == 1 {
			winner := candidates[0]
			e.state.Sheriff = winner.Name
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseDay,
				Type:    EventVote,
				Target:  winner.Name,
				Content: fmt.Sprintf("其他候选人退水，%s 自动当选警长！", winner.Name),
				Public:  true,
			})
			e.printf("\n[警长竞选] %s 自动当选警长！（其他候选人退水）\n\n", displayTag(winner))
			e.emitEvent(UIEvent{Type: "sheriff_elected", Player: winner.Name, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
			return nil
		}
	}

	if len(voters) == 0 {
		e.println("所有人都上警，没有投票人，本局无警长。")
		e.emitEvent(UIEvent{Type: "narration", Content: "所有人都上警竞选，无人投票，本局无警长。", Round: e.state.Round})
		return nil
	}

	e.println("--- 警长投票阶段 ---")

	var subAgents []adk.Agent
	voteResults := make(map[string]*string)

	var speechSummary strings.Builder
	for _, sp := range campaignSpeeches {
		fmt.Fprintf(&speechSummary, "%s: %s\n", sp.Speaker, sp.Content)
	}

	for _, p := range voters {
		var result string
		voteResults[p.Name] = &result

		voteTool, verr := action.CreateSheriffVoteTool(candidateNames, &result)
		if verr != nil {
			return fmt.Errorf("creating sheriff vote tool for %s: %w", p.Name, verr)
		}

		pctx := e.buildPromptContext(p)
		pctx.SheriffSpeeches = speechSummary.String()
		pctx.SheriffCandidates = candidateNames
		instruction := prompt.BuildSheriffElection(pctx)

		agent, verr := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("sheriff_vote_%s", p.Name),
			Description: fmt.Sprintf("%s votes for sheriff", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{voteTool},
				},
			},
		})
		if verr != nil {
			return fmt.Errorf("creating sheriff vote agent for %s: %w", p.Name, verr)
		}
		subAgents = append(subAgents, agent)
	}

	parVoteAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "sheriff_election",
		Description: "Non-candidates vote for sheriff simultaneously",
		SubAgents:   subAgents,
	})
	if err != nil {
		return fmt.Errorf("creating sheriff parallel agent: %w", err)
	}

	slog.Info("sheriff vote start", "voter_count", len(voters), "candidates", candidateNames)
	svCtx, svCancel := e.withCallTimeout(ctx)
	defer svCancel()

	voteRunner := adk.NewRunner(svCtx, adk.RunnerConfig{Agent: parVoteAgent})
	voteIter := voteRunner.Query(svCtx, "竞选发言已结束，请投票选出你心中的警长。请用中文。")

	for {
		event, ok := voteIter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			slog.Warn("sheriff vote API error", "error", event.Err)
			continue
		}
		msg, _, merr := adk.GetMessage(event)
		if merr != nil {
			continue
		}
		if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && event.AgentName != "" {
			voterName := strings.TrimPrefix(event.AgentName, "sheriff_vote_")
			if voter := e.state.GetPlayer(voterName); voter != nil {
				e.printf("  [%s 内心] %s\n", displayTag(voter), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: voter.Name, Content: msg.Content, Round: e.state.Round, Phase: "sheriff"})
			}
		}
	}

	votes := make(map[string]string)
	for _, p := range voters {
		if r := voteResults[p.Name]; r != nil && *r != "" {
			votes[p.Name] = *r
			e.printf("[警长投票] %s 投给了: %s\n", displayTag(p), *r)
			e.emitEvent(UIEvent{Type: "sheriff_vote", Player: p.Name, Target: *r, Round: e.state.Round})
		}
	}

	if len(votes) == 0 {
		e.println("没有人投票，本轮无警长。")
		return nil
	}

	result := TallyVotes(votes)
	if result.IsTied {
		e.printf("警长选举平票！平票玩家: %s。进入 PK 环节。\n", strings.Join(result.TiedPlayers, ", "))

		if err := e.pkRound(ctx, result.TiedPlayers); err != nil {
			return fmt.Errorf("sheriff PK round: %w", err)
		}

		pkWinner, perr := e.sheriffPKVote(ctx, result.TiedPlayers, voters)
		if perr != nil {
			return fmt.Errorf("sheriff PK vote: %w", perr)
		}

		if pkWinner == "" {
			e.println("警长竞选 PK 仍然平票！警徽流失，本局无警长。")
			e.emitEvent(UIEvent{Type: "narration", Content: "警长竞选PK平票，本局无警长。", Round: e.state.Round})
			return nil
		}
		result.Eliminated = pkWinner
		result.IsTied = false
	}

	e.state.Sheriff = result.Eliminated
	e.state.AddEvent(GameEvent{
		Round:   e.state.Round,
		Phase:   PhaseDay,
		Type:    EventVote,
		Target:  result.Eliminated,
		Content: fmt.Sprintf("%s 当选为警长！", result.Eliminated),
		Public:  true,
	})

	sheriff := e.state.GetPlayer(result.Eliminated)
	e.printf("\n[警长竞选] %s 当选警长！（投票权重 1.5 票）\n\n", displayTag(sheriff))
	e.emitEvent(UIEvent{Type: "sheriff_elected", Player: result.Eliminated, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})

	return nil
}

func (e *Engine) sheriffPKVote(ctx context.Context, tiedPlayers []string, allPlayers []*player.Player) (string, error) {
	e.println("--- 警长竞选 PK 投票 ---")

	tiedSet := make(map[string]bool, len(tiedPlayers))
	for _, n := range tiedPlayers {
		tiedSet[n] = true
	}

	var subAgents []adk.Agent
	voteResults := make(map[string]*string)
	var voters []*player.Player

	for _, p := range allPlayers {
		if tiedSet[p.Name] {
			continue
		}

		voters = append(voters, p)
		var result string
		voteResults[p.Name] = &result

		voteTool, err := action.CreateSheriffVoteTool(tiedPlayers, &result)
		if err != nil {
			return "", fmt.Errorf("creating sheriff PK vote tool for %s: %w", p.Name, err)
		}

		pctx := e.buildPromptContext(p)
		pctx.PreviousSpeeches = e.state.FormatSpeeches(e.state.Round)
		instruction := prompt.BuildSheriffElection(pctx)

		agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("sheriff_pk_%s", p.Name),
			Description: fmt.Sprintf("%s casts sheriff PK vote", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{voteTool},
				},
			},
		})
		if err != nil {
			return "", fmt.Errorf("creating sheriff PK vote agent for %s: %w", p.Name, err)
		}
		subAgents = append(subAgents, agent)
	}

	if len(subAgents) == 0 {
		return "", nil
	}

	parAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "sheriff_pk_vote",
		Description: "Sheriff PK re-vote among tied candidates",
		SubAgents:   subAgents,
	})
	if err != nil {
		return "", fmt.Errorf("creating sheriff PK parallel agent: %w", err)
	}

	slog.Info("sheriff PK vote start", "voter_count", len(voters), "tied", tiedPlayers)
	pkCtx, pkCancel := e.withCallTimeout(ctx)
	defer pkCancel()

	runner := adk.NewRunner(pkCtx, adk.RunnerConfig{Agent: parAgent})
	query := fmt.Sprintf("警长竞选 PK 投票。请从平票玩家中选择: %s。请用中文。", strings.Join(tiedPlayers, ", "))
	iter := runner.Query(pkCtx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			slog.Warn("sheriff PK vote API error", "error", event.Err)
			continue
		}
		msg, _, merr := adk.GetMessage(event)
		if merr != nil {
			continue
		}
		if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && event.AgentName != "" {
			voterName := strings.TrimPrefix(event.AgentName, "sheriff_pk_")
			if voter := e.state.GetPlayer(voterName); voter != nil {
				e.printf("  [%s 内心] %s\n", displayTag(voter), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: voter.Name, Content: msg.Content, Round: e.state.Round, Phase: "sheriff"})
			}
		}
	}

	votes := make(map[string]string)
	for _, p := range voters {
		if r := voteResults[p.Name]; r != nil && *r != "" {
			votes[p.Name] = *r
			e.printf("[警长PK投票] %s 投给了: %s\n", displayTag(p), *r)
			e.emitEvent(UIEvent{Type: "sheriff_vote", Player: p.Name, Target: *r, Round: e.state.Round})
		}
	}

	if len(votes) == 0 {
		return "", nil
	}

	pkResult := TallyFlatVotes(votes)

	e.println("\n警长 PK 投票统计:")
	for target, count := range pkResult.Tally {
		e.printf("  %s: %d 票\n", target, count)
	}

	if pkResult.IsTied {
		return "", nil
	}

	return pkResult.Eliminated, nil
}

func (e *Engine) badgeTransfer(ctx context.Context, sheriffName string) error {
	sheriff := e.state.GetPlayer(sheriffName)
	if sheriff == nil {
		return nil
	}

	targets := e.state.AlivePlayersExcept(sheriffName)
	if len(targets) == 0 {
		e.state.Sheriff = ""
		e.printf("[警徽] 无存活玩家可接收警徽，警徽作废。\n")
		return nil
	}

	var result string
	transferTool, err := action.CreateBadgeTransferTool(targets, &result)
	if err != nil {
		return fmt.Errorf("creating badge transfer tool: %w", err)
	}

	pctx := prompt.PromptContext{
		PlayerName:   sheriffName,
		RoleName:     sheriff.Role.Name(),
		AlivePlayers: targets,
		Round:        e.state.Round,
		KnownInfo:    e.state.FormatVisibleEvents(sheriffName),
	}
	if sheriff.Role.Team() == config.TeamWerewolf {
		pctx.Teammates = e.state.WerewolfTeammates(sheriffName)
	}
	instruction := prompt.BuildBadgeTransfer(pctx)

	err = e.runAgentWithTool(ctx, sheriff, instruction, []tool.InvokableTool{transferTool})
	if err != nil {
		return fmt.Errorf("running badge transfer agent: %w", err)
	}

	if result != "" {
		e.state.Sheriff = result
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseDay,
			Type:    EventVote,
			Target:  result,
			Content: fmt.Sprintf("警徽从 %s 转移到了 %s。", sheriffName, result),
			Public:  true,
		})
		newSheriff := e.state.GetPlayer(result)
		e.printf("[警徽转移] %s 将警徽转移给了 %s\n", displayTag(sheriff), displayTag(newSheriff))
		e.emitEvent(UIEvent{Type: "badge_transfer", Player: sheriffName, Target: result, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
	} else {
		e.state.Sheriff = ""
		e.printf("[警徽] %s 选择撕毁警徽，本局不再有警长。\n", displayTag(sheriff))
		e.emitEvent(UIEvent{Type: "badge_destroy", Player: sheriffName, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
	}

	return nil
}

func (e *Engine) nightPhase(ctx context.Context) error {
	slog.Info("phase start", "phase", "night", "round", e.state.Round, "alive", len(e.state.AlivePlayers()))
	e.println("--- 天黑请闭眼... ---")
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "night", Round: e.state.Round})
	e.state.ResetNightActions()

	if err := e.guardAction(ctx); err != nil {
		return fmt.Errorf("guard action: %w", err)
	}

	if err := e.werewolfAction(ctx); err != nil {
		return fmt.Errorf("werewolf action: %w", err)
	}

	if win := CheckWolfKillFirst(e.state, e.state.NightKillTarget); win.GameOver {
		e.println("[狼刀在先] 狼人的击杀已达成胜利条件，后续夜间行动不结算。")
		return nil
	}

	if err := e.wolfBeautyCharmAction(ctx); err != nil {
		return fmt.Errorf("wolf beauty charm: %w", err)
	}

	if err := e.witchAction(ctx); err != nil {
		return fmt.Errorf("witch action: %w", err)
	}

	if err := e.seerAction(ctx); err != nil {
		return fmt.Errorf("seer action: %w", err)
	}

	return nil
}

func (e *Engine) guardAction(ctx context.Context) error {
	var guard *player.Player
	for _, p := range e.state.AlivePlayers() {
		if p.Role.Name() == "guard" {
			guard = p
			break
		}
	}
	if guard == nil {
		return nil
	}

	var result string
	targets := e.state.AlivePlayerNames()
	guardTool, err := action.CreateGuardTool(targets, e.state.LastGuardTarget, &result)
	if err != nil {
		return fmt.Errorf("creating guard tool: %w", err)
	}

	pctx := e.buildPromptContext(guard)
	instruction := prompt.BuildGuardNight(pctx, e.state.LastGuardTarget)

	err = e.runAgentWithTool(ctx, guard, instruction, []tool.InvokableTool{guardTool})
	if err != nil {
		return fmt.Errorf("running guard agent: %w", err)
	}

	if result != "" {
		e.state.NightGuardTarget = result
		e.printf("[夜晚] %s [守卫] 守护了 %s\n", displayTag(guard), result)
		e.emitEvent(UIEvent{Type: "night_action", Action: "guard", Player: guard.Name, Target: result, Round: e.state.Round})
	} else {
		e.printf("[夜晚] %s [守卫] 选择空守\n", displayTag(guard))
		e.emitEvent(UIEvent{Type: "night_action", Action: "guard", Player: guard.Name, Round: e.state.Round})
	}

	return nil
}

func (e *Engine) wolfBeautyCharmAction(ctx context.Context) error {
	var wb *player.Player
	for _, p := range e.state.AlivePlayers() {
		if p.Role.Name() == "wolf_beauty" {
			wb = p
			break
		}
	}
	if wb == nil {
		return nil
	}

	targets := e.state.AlivePlayerNames()
	var result string
	charmTool, err := action.CreateCharmTool(targets, &result)
	if err != nil {
		return fmt.Errorf("creating charm tool: %w", err)
	}

	pctx := e.buildPromptContext(wb)
	instruction := prompt.BuildWolfBeautyCharm(pctx)

	err = e.runAgentWithTool(ctx, wb, instruction, []tool.InvokableTool{charmTool})
	if err != nil {
		return fmt.Errorf("running wolf beauty charm agent: %w", err)
	}

	if result != "" {
		e.state.CharmTarget = result
		e.printf("[夜晚] %s [狼美人] 魅惑了 %s\n", displayTag(wb), result)
		e.emitEvent(UIEvent{Type: "night_action", Action: "charm", Player: wb.Name, Target: result, Round: e.state.Round})
	}

	return nil
}

func (e *Engine) resolveWolfBeautyDeath(ctx context.Context, wbName string) (string, error) {
	if e.state.DuelKilled == wbName {
		return "", nil
	}

	charmTarget := e.state.CharmTarget
	if charmTarget == "" {
		return "", nil
	}

	target := e.state.GetPlayer(charmTarget)
	if target == nil || !target.Alive {
		return "", nil
	}

	target.Alive = false
	e.state.AddEvent(GameEvent{
		Round:   e.state.Round,
		Phase:   PhaseDay,
		Type:    EventDeath,
		Actor:   wbName,
		Target:  target.Name,
		Content: e.deathContent(fmt.Sprintf("%s 因狼美人殉情死亡。", target.Name), target.Role.Name()),
		Public:  true,
	})
	e.printf("[殉情] %s 因狼美人 %s 死亡而殉情出局。\n", displayTag(target), wbName)
	e.emitEvent(UIEvent{Type: "death", Player: target.Name, Role: target.Role.Name(), Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})

	if target.Name == e.state.Sheriff {
		if err := e.badgeTransfer(ctx, target.Name); err != nil {
			return "", fmt.Errorf("badge transfer after charm death: %w", err)
		}
	}

	return target.Name, nil
}

func (e *Engine) werewolfAction(ctx context.Context) error {
	wolves := e.state.AliveWerewolves()
	if len(wolves) == 0 {
		return nil
	}

	targets := e.state.AlivePlayerNames()
	var wolfBeautyName string
	for _, w := range wolves {
		if w.Role.Name() == "wolf_beauty" {
			wolfBeautyName = w.Name
			break
		}
	}
	if wolfBeautyName != "" {
		var filtered []string
		for _, t := range targets {
			if t != wolfBeautyName {
				filtered = append(filtered, t)
			}
		}
		targets = filtered
	}

	const maxWolfRounds = 3
	var wolfChats []string

	for round := 1; round <= maxWolfRounds; round++ {
		votes := make(map[string]string)

		for _, wolf := range wolves {
			var result string
			killTool, err := action.CreateKillTool(targets, &result)
			if err != nil {
				return fmt.Errorf("creating kill tool for %s: %w", wolf.Name, err)
			}

			pctx := e.buildPromptContext(wolf)
			if len(wolfChats) > 0 {
				pctx.WolfDiscussion = strings.Join(wolfChats, "\n")
			}
			instruction := prompt.BuildWerewolfNight(pctx)

			e.emitEvent(UIEvent{Type: "thinking_start", Player: wolf.Name, Round: e.state.Round, Phase: "night"})

			thoughts, err := e.runAgentCapture(ctx, wolf, instruction, []tool.InvokableTool{killTool})
			if err != nil {
				return fmt.Errorf("running wolf agent %s: %w", wolf.Name, err)
			}

			if thoughts != "" {
				e.emitEvent(UIEvent{Type: "night_action", Action: "wolf_chat", Player: wolf.Name, Content: thoughts, Round: e.state.Round})
			}

			entry := fmt.Sprintf("%s: %s", wolf.Name, thoughts)
			if result != "" {
				entry += fmt.Sprintf(" [选择击杀: %s]", result)
				votes[wolf.Name] = result
				e.printf("[夜晚] %s [狼人] 投票击杀: %s (第%d轮)\n", displayTag(wolf), result, round)
				e.emitEvent(UIEvent{Type: "night_action", Action: "kill_vote", Player: wolf.Name, Target: result, Round: e.state.Round})
			} else {
				e.printf("[夜晚] %s [狼人] 选择空刀 (第%d轮)\n", displayTag(wolf), round)
			}
			wolfChats = append(wolfChats, entry)
		}

		if len(votes) == 0 {
			e.printf("[夜晚] 狼人集体选择空刀。\n")
			e.emitEvent(UIEvent{Type: "night_action", Action: "kill_decided", Round: e.state.Round})
			return nil
		}

		vr := TallyVotes(votes)
		if !vr.IsTied {
			e.state.NightKillTarget = vr.Eliminated
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseNight,
				Type:    EventKill,
				Actor:   "werewolves",
				Target:  vr.Eliminated,
				Content: fmt.Sprintf("狼人们决定击杀 %s。", vr.Eliminated),
				Public:  false,
			})
			e.printf("[夜晚] 狼人们最终决定击杀: %s (第%d轮达成一致)\n", vr.Eliminated, round)
			e.emitEvent(UIEvent{Type: "night_action", Action: "kill_decided", Target: vr.Eliminated, Round: e.state.Round})
			return nil
		}

		if round < maxWolfRounds {
			tallySummary := formatWolfTally(vr.Tally)
			tiedNames := strings.Join(vr.TiedPlayers, "、")
			retryNote := fmt.Sprintf("\n--- 第%d轮投票结果 ---\n票数: %s\n平票: %s\n意见不统一，请重新讨论并投票。剩余%d轮机会，若仍无法统一则视为空刀。",
				round, tallySummary, tiedNames, maxWolfRounds-round)
			wolfChats = append(wolfChats, retryNote)
			e.printf("[夜晚] 狼人第%d轮投票平票 (%s)，进入下一轮讨论。\n", round, tallySummary)
			e.emitEvent(UIEvent{Type: "night_action", Action: "wolf_chat", Content: fmt.Sprintf("投票平票 (%s)，狼人继续讨论...", tallySummary), Round: e.state.Round})
		} else {
			e.printf("[夜晚] 狼人%d轮投票均无法统一，本轮空刀。\n", maxWolfRounds)
			e.emitEvent(UIEvent{Type: "night_action", Action: "kill_decided", Round: e.state.Round})
		}
	}

	return nil
}

func formatWolfTally(tally map[string]int) string {
	var parts []string
	for target, count := range tally {
		parts = append(parts, fmt.Sprintf("%s:%d票", target, count))
	}
	return strings.Join(parts, " ")
}

func (e *Engine) seerAction(ctx context.Context) error {
	var seer *player.Player
	for _, p := range e.state.AlivePlayers() {
		if p.Role.Name() == "seer" {
			seer = p
			break
		}
	}
	if seer == nil {
		return nil
	}

	var result string
	targets := e.state.AlivePlayersExcept(seer.Name)
	var alreadyChecked []string
	for name := range e.state.SeerResults {
		alreadyChecked = append(alreadyChecked, name)
	}
	investigateTool, err := action.CreateInvestigateTool(targets, alreadyChecked, &result)
	if err != nil {
		return fmt.Errorf("creating investigate tool: %w", err)
	}

	pctx := e.buildPromptContext(seer)
	instruction := prompt.BuildSeerNight(pctx)

	err = e.runAgentWithTool(ctx, seer, instruction, []tool.InvokableTool{investigateTool})
	if err != nil {
		return fmt.Errorf("running seer agent: %w", err)
	}

	if result != "" {
		target := e.state.GetPlayer(result)
		if target != nil {
			teamStr := "villager"
			if target.Role.Team() == config.TeamWerewolf {
				teamStr = "werewolf"
			}
			e.state.SeerResults[result] = teamStr
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseNight,
				Type:    EventInvestigate,
				Actor:   seer.Name,
				Target:  result,
				Content: fmt.Sprintf("%s 查验了 %s，发现对方是 %s。", seer.Name, result, teamStr),
				Public:  false,
			})
			e.printf("[夜晚] %s [预言家] 查验 %s -> %s\n", displayTag(seer), result, teamStr)
			e.emitEvent(UIEvent{Type: "night_action", Action: "investigate", Player: seer.Name, Target: result, Result: teamStr, Round: e.state.Round})
		}
	}

	return nil
}

func (e *Engine) witchAction(ctx context.Context) error {
	var witch *player.Player
	for _, p := range e.state.AlivePlayers() {
		if p.Role.Name() == "witch" {
			witch = p
			break
		}
	}
	if witch == nil {
		return nil
	}

	canSelfSave := false
	switch e.witchSelfSave {
	case config.WitchSelfSaveNever:
		canSelfSave = false
	case config.WitchSelfSaveFirstOnly:
		canSelfSave = e.state.Round == 1
	case config.WitchSelfSaveAlways:
		canSelfSave = true
	}

	pctx := e.buildPromptContext(witch)
	if e.state.WitchHealUsed {
		pctx.VictimName = ""
	}
	pctx.WitchCanSelfSave = canSelfSave
	instruction := prompt.BuildWitchNight(pctx)

	var tools []tool.InvokableTool

	var healResult string
	canHeal := !e.state.WitchHealUsed && e.state.NightKillTarget != ""
	if e.state.NightKillTarget == witch.Name && !canSelfSave {
		canHeal = false
	}
	if canHeal {
		healTool, err := action.CreateHealTool(e.state.NightKillTarget, &healResult)
		if err != nil {
			return fmt.Errorf("creating heal tool: %w", err)
		}
		tools = append(tools, healTool)
	}

	var poisonResult string
	if !e.state.WitchPoisonUsed {
		targets := e.state.AlivePlayerNames()
		poisonTool, err := action.CreatePoisonTool(targets, &poisonResult)
		if err != nil {
			return fmt.Errorf("creating poison tool: %w", err)
		}
		tools = append(tools, poisonTool)
	}

	if len(tools) == 0 {
		e.printf("[夜晚] %s [女巫] 已无药水可用。\n", displayTag(witch))
		return nil
	}

	err := e.runAgentWithTool(ctx, witch, instruction, tools)
	if err != nil {
		return fmt.Errorf("running witch agent: %w", err)
	}

	if healResult == "true" {
		e.state.NightSaveTarget = e.state.NightKillTarget
		e.state.WitchHealUsed = true
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseNight,
			Type:    EventHeal,
			Actor:   witch.Name,
			Target:  e.state.NightKillTarget,
			Content: fmt.Sprintf("%s 使用解药救活了 %s。", witch.Name, e.state.NightKillTarget),
			Public:  false,
		})
		e.printf("[夜晚] %s [女巫] 使用解药救了 %s\n", displayTag(witch), e.state.NightKillTarget)
		e.emitEvent(UIEvent{Type: "night_action", Action: "heal", Player: witch.Name, Target: e.state.NightKillTarget, Round: e.state.Round})
	}

	if poisonResult != "" && healResult != "true" {
		e.state.NightPoisonTarget = poisonResult
		e.state.WitchPoisonUsed = true
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseNight,
			Type:    EventPoison,
			Actor:   witch.Name,
			Target:  poisonResult,
			Content: fmt.Sprintf("%s 对 %s 使用了毒药。", witch.Name, poisonResult),
			Public:  false,
		})
		e.printf("[夜晚] %s [女巫] 毒杀了 %s\n", displayTag(witch), poisonResult)
		e.emitEvent(UIEvent{Type: "night_action", Action: "poison", Player: witch.Name, Target: poisonResult, Round: e.state.Round})
	}

	return nil
}

func (e *Engine) resolveNight(ctx context.Context) ([]string, error) {
	var deaths []string

	killTarget := e.state.NightKillTarget
	saveTarget := e.state.NightSaveTarget
	guardTarget := e.state.NightGuardTarget

	if guardTarget != "" && guardTarget == killTarget {
		if guardTarget == saveTarget {
			saveTarget = ""
			e.state.NightSaveTarget = ""
		} else {
			killTarget = ""
		}
	}

	if killTarget != "" && killTarget != saveTarget {
		p := e.state.GetPlayer(killTarget)
		if p != nil && p.Alive {
			p.Alive = false
			deaths = append(deaths, p.Name)
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseNight,
				Type:    EventDeath,
				Target:  p.Name,
				Content: e.deathContent(fmt.Sprintf("%s 在昨夜死亡。", p.Name), p.Role.Name()),
				Public:  true,
			})
		}
	} else if killTarget != "" && killTarget == saveTarget {
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseNight,
			Type:    EventHealBlock,
			Actor:   "witch",
			Target:  killTarget,
			Content: fmt.Sprintf("你的击杀目标 %s 被女巫救活了。", killTarget),
			Public:  false,
		})
	}

	if e.state.NightPoisonTarget != "" {
		p := e.state.GetPlayer(e.state.NightPoisonTarget)
		if p != nil && p.Alive {
			p.Alive = false
			deaths = append(deaths, p.Name)
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseNight,
				Type:    EventDeath,
				Target:  p.Name,
				Content: e.deathContent(fmt.Sprintf("%s 在昨夜死亡。", p.Name), p.Role.Name()),
				Public:  true,
			})
		}
	}

	for _, name := range deaths {
		p := e.state.GetPlayer(name)
		if p != nil && p.Role.Name() == "wolf_beauty" {
			charmDeath, err := e.resolveWolfBeautyDeath(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("wolf beauty charm resolution: %w", err)
			}
			if charmDeath != "" {
				deaths = append(deaths, charmDeath)
			}
		}
	}

	e.sortBySeat(deaths)

	for _, name := range deaths {
		p := e.state.GetPlayer(name)
		roleName := ""
		if p != nil {
			roleName = p.Role.Name()
		}
		e.emitEvent(UIEvent{Type: "death", Player: name, Role: roleName, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
	}

	if len(deaths) > 0 {
		narration, err := e.narrator.NarrateDeath(ctx, e.state.Round, deaths, "夜晚的事件")
		if err != nil {
			e.printf("[旁白出错: %v]\n", err)
			e.printf("\n--- 天亮了。%s 没能活过这个夜晚。 ---\n\n", strings.Join(deaths, ", "))
		} else {
			e.printf("\n[旁白] %s\n\n", narration)
			e.emitEvent(UIEvent{Type: "narration", Content: narration, Round: e.state.Round})
		}
	} else {
		e.printf("\n--- 天亮了。昨晚是个平安夜！ ---\n\n")
		e.emitEvent(UIEvent{Type: "narration", Content: "昨晚是个平安夜，无人伤亡。", Round: e.state.Round})
	}

	if e.state.Round == 1 && len(deaths) > 0 {
		for _, name := range deaths {
			p := e.state.GetPlayer(name)
			if p != nil {
				if err := e.lastWords(ctx, p); err != nil {
					e.printf("  [error] last words for %s: %v\n", name, err)
				}
			}
		}
	}

	for _, name := range deaths {
		p := e.state.GetPlayer(name)
		if p == nil {
			continue
		}

		if p.Role.Name() == "hunter" {
			if name == e.state.NightPoisonTarget {
				e.printf("[猎人] %s 被毒杀，无法开枪。\n", displayTag(p))
				e.emitEvent(UIEvent{Type: "narration", Content: fmt.Sprintf("%s 被毒杀，无法发动猎人技能。", name), Round: e.state.Round})
			} else {
				shotTarget, err := e.hunterShootTrigger(ctx, name)
				if err != nil {
					return nil, fmt.Errorf("hunter shoot: %w", err)
				}
				if shotTarget != "" {
					deaths = append(deaths, shotTarget)
					shotP := e.state.GetPlayer(shotTarget)
					if shotP != nil && shotP.Role.Name() == "wolf_beauty" {
						charmDeath, cerr := e.resolveWolfBeautyDeath(ctx, shotTarget)
						if cerr != nil {
							return nil, fmt.Errorf("wolf beauty charm after hunter shot: %w", cerr)
						}
						if charmDeath != "" {
							deaths = append(deaths, charmDeath)
						}
					}
					if shotTarget == e.state.Sheriff {
						if err := e.badgeTransfer(ctx, shotTarget); err != nil {
							return nil, fmt.Errorf("badge transfer after hunter shot: %w", err)
						}
					}
				}
			}
		}

		if name == e.state.Sheriff {
			if err := e.badgeTransfer(ctx, name); err != nil {
				return nil, fmt.Errorf("badge transfer after night death: %w", err)
			}
		}
	}

	return deaths, nil
}

func (e *Engine) hunterShootTrigger(ctx context.Context, hunterName string) (string, error) {
	hunter := e.state.GetPlayer(hunterName)
	if hunter == nil || e.state.HunterShotUsed {
		return "", nil
	}
	e.state.HunterShotUsed = true

	targets := e.state.AlivePlayersExcept(hunterName)
	if len(targets) == 0 {
		return "", nil
	}

	var result string
	shootTool, err := action.CreateShootTool(targets, &result)
	if err != nil {
		return "", fmt.Errorf("creating shoot tool: %w", err)
	}

	pctx := prompt.PromptContext{
		PlayerName:   hunterName,
		RoleName:     "hunter",
		AlivePlayers: targets,
		Round:        e.state.Round,
		KnownInfo:    e.state.FormatVisibleEvents(hunterName),
	}
	instruction := prompt.BuildHunterShoot(pctx)

	err = e.runAgentWithTool(ctx, hunter, instruction, []tool.InvokableTool{shootTool})
	if err != nil {
		return "", fmt.Errorf("running hunter shoot agent: %w", err)
	}

	if result != "" {
		target := e.state.GetPlayer(result)
		if target != nil && target.Alive {
			target.Alive = false
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseDay,
				Type:    EventShoot,
				Actor:   hunterName,
				Target:  result,
				Content: e.deathContent(fmt.Sprintf("猎人 %s 开枪带走了 %s！", hunterName, result), target.Role.Name()),
				Public:  true,
			})
			e.printf("[猎人开枪] %s 带走了 %s！\n", displayTag(hunter), result)
			e.emitEvent(UIEvent{Type: "hunter_shoot", Player: hunterName, Target: result, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
			return result, nil
		}
	}
	e.printf("[猎人] %s 选择不开枪。\n", displayTag(hunter))
	return "", nil
}

func (e *Engine) dayPhase(ctx context.Context, deaths []string) error {
	slog.Info("phase start", "phase", "day", "round", e.state.Round, "alive", len(e.state.AlivePlayers()), "deaths", deaths)
	e.println("--- 白天讨论阶段 ---")
	e.println()
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "day", Round: e.state.Round})

	alivePlayers := e.state.AlivePlayers()
	if len(alivePlayers) == 0 {
		return nil
	}

	rand.Shuffle(len(alivePlayers), func(i, j int) {
		alivePlayers[i], alivePlayers[j] = alivePlayers[j], alivePlayers[i]
	})

	e.state.Speeches[e.state.Round] = nil
	e.state.DuelKilled = ""

	for _, p := range alivePlayers {
		var speechResult string
		speakTool, err := action.CreateSpeakTool(&speechResult)
		if err != nil {
			return fmt.Errorf("creating speak tool for %s: %w", p.Name, err)
		}

		var selfExploded bool
		var duelResult string
		var wolfKingTakeTarget string
		var dayTools []tool.BaseTool
		dayTools = append(dayTools, speakTool)

		if p.Role.Team() == config.TeamWerewolf && p.Role.Name() != "wolf_beauty" {
			if p.Role.Name() == "wolf_king" {
				aliveWolves := e.state.AliveWerewolves()
				isLastWolf := len(aliveWolves) == 1
				wasPoisoned := e.state.PrevNightPoisonTarget == p.Name
				if !isLastWolf && !wasPoisoned {
					takeTargets := e.state.AliveNonWerewolfNames()
					wkTool, eerr := action.CreateWolfKingSelfExplodeTool(p.Name, takeTargets, &selfExploded, &wolfKingTakeTarget)
					if eerr != nil {
						return fmt.Errorf("creating wolf king explode tool for %s: %w", p.Name, eerr)
					}
					dayTools = append(dayTools, wkTool)
				}
			} else {
				explodeTool, eerr := action.CreateSelfExplodeTool(p.Name, &selfExploded)
				if eerr != nil {
					return fmt.Errorf("creating self-explode tool for %s: %w", p.Name, eerr)
				}
				dayTools = append(dayTools, explodeTool)
			}
		}

		if p.Role.Name() == "knight" && !e.state.KnightDuelUsed {
			duelTargets := e.state.AlivePlayersExcept(p.Name)
			duelTool, derr := action.CreateDuelTool(duelTargets, &duelResult)
			if derr != nil {
				return fmt.Errorf("creating duel tool for %s: %w", p.Name, derr)
			}
			dayTools = append(dayTools, duelTool)
		}

		pctx := e.buildPromptContext(p)
		pctx.DeathsLastNight = deaths
		pctx.PreviousSpeeches = e.state.FormatSpeeches(e.state.Round)
		instruction := prompt.BuildDayDiscussion(pctx)

		e.emitEvent(UIEvent{Type: "thinking_start", Player: p.Name, Round: e.state.Round, Phase: "day"})

		var thoughts strings.Builder
		err = e.retryOnTransient(p.Name, func() error {
			thoughts.Reset()
			callCtx, cancel := e.withCallTimeout(ctx)
			defer cancel()

			agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
				Name:        p.Name,
				Description: fmt.Sprintf("%s speaks during day discussion", p.Name),
				Instruction: instruction,
				Model:       p.Model,
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: dayTools,
					},
				},
			})
			if aerr != nil {
				return aerr
			}

			runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
			iter := runner.Query(callCtx, "轮到你发言了。用中文说出你的看法和怀疑。")

			for {
				event, ok := iter.Next()
				if !ok {
					break
				}
				if event.Err != nil {
					return event.Err
				}
				msg, _, merr := adk.GetMessage(event)
				if merr != nil {
					continue
				}
				if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
					thoughts.WriteString(msg.Content)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("day discussion error for %s: %w", p.Name, err)
		}

		if selfExploded {
			p.Alive = false
			e.state.WolfSelfExploded = p.Name

			if p.Role.Name() == "wolf_king" && wolfKingTakeTarget != "" {
				e.state.AddEvent(GameEvent{
					Round:   e.state.Round,
					Phase:   PhaseDay,
					Type:    EventDeath,
					Actor:   p.Name,
					Target:  p.Name,
					Content: fmt.Sprintf("%s（白狼王）自爆并带走了 %s！白天阶段立即结束。", p.Name, wolfKingTakeTarget),
					Public:  true,
				})
				e.printf("[白狼王自爆] %s 自爆并带走了 %s！\n", displayTag(p), wolfKingTakeTarget)
				e.emitEvent(UIEvent{Type: "self_explode", Player: p.Name, Role: p.Role.Name(), Target: wolfKingTakeTarget, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})

				takeP := e.state.GetPlayer(wolfKingTakeTarget)
				if takeP != nil && takeP.Alive {
					takeP.Alive = false
					e.state.AddEvent(GameEvent{
						Round:   e.state.Round,
						Phase:   PhaseDay,
						Type:    EventDeath,
						Actor:   p.Name,
						Target:  takeP.Name,
						Content: fmt.Sprintf("%s 被白狼王带走出局。", takeP.Name),
						Public:  true,
					})
					e.emitEvent(UIEvent{Type: "death", Player: takeP.Name, Role: takeP.Role.Name(), Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})

					if takeP.Name == e.state.Sheriff {
						if berr := e.badgeTransfer(ctx, takeP.Name); berr != nil {
							return fmt.Errorf("badge transfer after wolf king takedown: %w", berr)
						}
					}
				}
			} else {
				e.state.AddEvent(GameEvent{
					Round:   e.state.Round,
					Phase:   PhaseDay,
					Type:    EventDeath,
					Actor:   p.Name,
					Target:  p.Name,
					Content: fmt.Sprintf("%s 选择自爆！（身份：%s）白天阶段立即结束，不进行投票。", p.Name, roleChineseName(p.Role.Name())),
					Public:  true,
				})
				e.printf("[自爆] %s 选择自爆！白天阶段立即结束。\n", displayTag(p))
				e.emitEvent(UIEvent{Type: "self_explode", Player: p.Name, Role: p.Role.Name(), Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
			}

			if p.Name == e.state.Sheriff {
				if berr := e.badgeTransfer(ctx, p.Name); berr != nil {
					return fmt.Errorf("badge transfer after self-explode: %w", berr)
				}
			}
			return nil
		}

		if duelResult != "" {
			e.state.KnightDuelUsed = true
			target := e.state.GetPlayer(duelResult)
			if target != nil && target.Alive {
				if target.Role.Team() == config.TeamWerewolf {
					target.Alive = false
					e.state.DuelKilled = target.Name
					e.state.AddEvent(GameEvent{
						Round:   e.state.Round,
						Phase:   PhaseDay,
						Type:    EventDeath,
						Actor:   p.Name,
						Target:  target.Name,
						Content: fmt.Sprintf("骑士 %s 决斗 %s 成功！%s 是狼人，立即出局！白天结束。", p.Name, target.Name, target.Name),
						Public:  true,
					})
					e.printf("[决斗] %s 决斗 %s 成功！对方是狼人，出局！\n", displayTag(p), displayTag(target))
					e.emitEvent(UIEvent{Type: "duel", Player: p.Name, Target: target.Name, Result: "wolf", Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
					if target.Name == e.state.Sheriff {
						if berr := e.badgeTransfer(ctx, target.Name); berr != nil {
							return fmt.Errorf("badge transfer after duel: %w", berr)
						}
					}
					return nil
				}
				p.Alive = false
				e.state.AddEvent(GameEvent{
					Round:   e.state.Round,
					Phase:   PhaseDay,
					Type:    EventDeath,
					Actor:   p.Name,
					Target:  p.Name,
					Content: fmt.Sprintf("骑士 %s 决斗 %s 失败！%s 是好人，骑士出局。白天继续。", p.Name, target.Name, target.Name),
					Public:  true,
				})
				e.printf("[决斗] %s 决斗 %s 失败！对方是好人，骑士出局。\n", displayTag(p), displayTag(target))
				e.emitEvent(UIEvent{Type: "duel", Player: p.Name, Target: target.Name, Result: "good", Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
				if p.Name == e.state.Sheriff {
					if berr := e.badgeTransfer(ctx, p.Name); berr != nil {
						return fmt.Errorf("badge transfer after failed duel: %w", berr)
					}
				}
				continue
			}
		}

		speech := speechResult
		if speech == "" && thoughts.Len() > 0 {
			speech = extractSpeakFromText(thoughts.String())
		}

		if thoughts.Len() > 0 && thoughts.String() != speech {
			e.printf("  [%s 内心] %s\n", displayTag(p), thoughts.String())
			e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: thoughts.String(), Round: e.state.Round, Phase: "day"})
		}

		if speech != "" {
			e.printf("[%s 发言] %s\n\n", displayTag(p), speech)
			e.emitEvent(UIEvent{Type: "speech", Player: p.Name, Content: speech, Round: e.state.Round})
			e.state.Speeches[e.state.Round] = append(e.state.Speeches[e.state.Round], Speech{
				Speaker: p.Name,
				Content: speech,
			})
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseDay,
				Type:    EventSpeech,
				Actor:   p.Name,
				Content: fmt.Sprintf("%s 说: %s", p.Name, speech),
				Public:  true,
			})
		}
	}

	return nil
}

func (e *Engine) sheriffEndorse(ctx context.Context) string {
	if e.state.Sheriff == "" {
		return ""
	}
	sheriff := e.state.GetPlayer(e.state.Sheriff)
	if sheriff == nil || !sheriff.Alive {
		return ""
	}
	if e.state.IdiotRevealed[sheriff.Name] {
		return ""
	}

	e.println("--- 警长归票 ---")

	candidates := e.state.AlivePlayersExcept(sheriff.Name)
	if len(candidates) == 0 {
		return ""
	}

	var result string
	endorseTool, err := action.CreateEndorseTool(candidates, &result)
	if err != nil {
		e.printf("  [error] creating endorse tool: %v\n", err)
		return ""
	}

	pctx := e.buildPromptContext(sheriff)
	pctx.PreviousSpeeches = e.state.FormatSpeeches(e.state.Round)
	instruction := prompt.BuildSheriffEndorse(pctx, candidates)

	err = e.runAgentWithTool(ctx, sheriff, instruction, []tool.InvokableTool{endorseTool})
	if err != nil {
		e.printf("  [error] sheriff endorse: %v\n", err)
		return ""
	}

	if result != "" {
		e.printf("[归票] 警长 %s 归票给 %s（+0.5 票）\n", displayTag(sheriff), result)
		e.emitEvent(UIEvent{Type: "endorse", Player: sheriff.Name, Target: result, Round: e.state.Round})
	} else {
		e.printf("[归票] 警长 %s 选择不归票。\n", displayTag(sheriff))
	}

	return result
}

func (e *Engine) votePhase(ctx context.Context) error {
	slog.Info("phase start", "phase", "vote", "round", e.state.Round, "alive", len(e.state.AlivePlayers()))
	e.println("--- 投票阶段 ---")
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "vote", Round: e.state.Round})

	endorsedTarget := e.sheriffEndorse(ctx)

	alivePlayers := e.state.AlivePlayers()
	if len(alivePlayers) == 0 {
		return nil
	}

	candidates := e.state.AlivePlayerNames()

	var subAgents []adk.Agent
	voteResults := make(map[string]*string)
	var voters []*player.Player

	for _, p := range alivePlayers {
		if e.state.IdiotRevealed[p.Name] {
			e.printf("[投票] %s [白痴/已翻牌] 没有投票权。\n", displayTag(p))
			continue
		}

		voters = append(voters, p)
		var result string
		voteResults[p.Name] = &result

		voteTool, err := action.CreateVoteTool(candidates, &result)
		if err != nil {
			return fmt.Errorf("creating vote tool for %s: %w", p.Name, err)
		}

		pctx := e.buildPromptContext(p)
		instruction := prompt.BuildVote(pctx)

		agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("vote_%s", p.Name),
			Description: fmt.Sprintf("%s casts a vote", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{voteTool},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("creating vote agent for %s: %w", p.Name, err)
		}
		subAgents = append(subAgents, agent)
	}

	if len(subAgents) == 0 {
		e.println("没有玩家可以投票。本轮无人被淘汰。")
		return nil
	}

	for _, v := range voters {
		e.emitEvent(UIEvent{Type: "thinking_start", Player: v.Name, Round: e.state.Round, Phase: "vote"})
	}

	parAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "vote_phase",
		Description: "Parallel voting where all players vote simultaneously",
		SubAgents:   subAgents,
	})
	if err != nil {
		return fmt.Errorf("creating parallel agent: %w", err)
	}

	slog.Info("vote phase parallel start", "voter_count", len(voters), "round", e.state.Round)
	vtCtx, vtCancel := e.withCallTimeout(ctx)
	defer vtCancel()

	runner := adk.NewRunner(vtCtx, adk.RunnerConfig{
		Agent: parAgent,
	})

	query := fmt.Sprintf("第 %d 回合投票。每位玩家必须投票选择一名玩家淘汰。请用中文。", e.state.Round)
	iter := runner.Query(vtCtx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			slog.Warn("vote phase API error", "error", event.Err)
			continue
		}
		msg, _, err := adk.GetMessage(event)
		if err != nil {
			continue
		}
		if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && event.AgentName != "" {
			voterName := strings.TrimPrefix(event.AgentName, "vote_")
			if voter := e.state.GetPlayer(voterName); voter != nil {
				e.printf("  [%s 内心] %s\n", displayTag(voter), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: voter.Name, Content: msg.Content, Round: e.state.Round, Phase: "vote"})
			}
		}
	}

	votes := make(map[string]string)
	for _, p := range voters {
		if r := voteResults[p.Name]; r != nil && *r != "" {
			votes[p.Name] = *r
			e.printf("[投票] %s 投给了: %s\n", displayTag(p), *r)
			e.emitEvent(UIEvent{Type: "vote_cast", Player: p.Name, Target: *r, Round: e.state.Round})
		}
	}

	e.state.VoteRecord[e.state.Round] = votes

	if len(votes) == 0 {
		e.println("没有人投票。本轮无人被淘汰。")
		return nil
	}

	result := TallyWeightedVotes(votes, e.state.Sheriff, endorsedTarget)

	e.println("\n投票统计:")
	for target, count := range result.Tally {
		weight := result.WeightedTally[target]
		if weight != float64(count) {
			e.printf("  %s: %d 票 (加权 %.1f)\n", target, count, weight)
		} else {
			e.printf("  %s: %d 票\n", target, count)
		}
	}
	e.emitEvent(UIEvent{Type: "vote_tally", Tally: result.Tally, Tied: result.IsTied, Round: e.state.Round})

	if result.IsTied {
		e.printf("票数相同！平票玩家: %s。进入 PK 环节。\n", strings.Join(result.TiedPlayers, ", "))

		if err := e.pkRound(ctx, result.TiedPlayers); err != nil {
			return fmt.Errorf("PK round: %w", err)
		}

		pkEliminated, err := e.pkVote(ctx, result.TiedPlayers)
		if err != nil {
			return fmt.Errorf("PK vote: %w", err)
		}

		if pkEliminated == "" {
			e.println("PK 投票仍然平局！本轮无人被淘汰。")
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseVote,
				Type:    EventVote,
				Content: "PK 投票平局，无人被淘汰。",
				Public:  true,
			})
			return nil
		}

		result.Eliminated = pkEliminated
		result.IsTied = false
	}

	eliminated := e.state.GetPlayer(result.Eliminated)
	if eliminated == nil {
		return nil
	}

	if eliminated.Role.Name() == "idiot" && !e.state.IdiotRevealed[eliminated.Name] {
		e.state.IdiotRevealed[eliminated.Name] = true
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseVote,
			Type:    EventEliminate,
			Target:  eliminated.Name,
			Content: fmt.Sprintf("%s 翻牌亮出白痴身份，免于淘汰！但失去投票权。", eliminated.Name),
			Public:  true,
		})
		e.printf("\n%s 翻牌亮出白痴身份！免于淘汰，但失去投票权。\n\n", displayTag(eliminated))
		e.emitEvent(UIEvent{Type: "idiot_reveal", Player: eliminated.Name, Role: "idiot", Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
		return nil
	}

	eliminated.Alive = false
	e.state.AddEvent(GameEvent{
		Round:   e.state.Round,
		Phase:   PhaseVote,
		Type:    EventEliminate,
		Target:  eliminated.Name,
		Content: e.deathContent(fmt.Sprintf("%s 被投票淘汰了。", eliminated.Name), eliminated.Role.Name()),
		Public:  true,
	})

	e.emitEvent(UIEvent{Type: "elimination", Player: eliminated.Name, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
	e.printf("[淘汰] %s 被投票淘汰。\n", displayTag(eliminated))

	narration, nerr := e.narrator.NarrateDeath(ctx, e.state.Round, []string{eliminated.Name}, "全村投票")
	if nerr != nil {
		e.printf("\n%s 被淘汰了！\n\n", displayTag(eliminated))
	} else {
		e.printf("\n[旁白] %s\n\n", narration)
		e.emitEvent(UIEvent{Type: "narration", Content: narration, Round: e.state.Round})
	}

	if err := e.lastWords(ctx, eliminated); err != nil {
		e.printf("  [error] last words for %s: %v\n", eliminated.Name, err)
	}

	if eliminated.Role.Name() == "hunter" {
		shotTarget, serr := e.hunterShootTrigger(ctx, eliminated.Name)
		if serr != nil {
			return fmt.Errorf("hunter shoot after vote: %w", serr)
		}
		if shotTarget != "" {
			shotP := e.state.GetPlayer(shotTarget)
			shotRole := ""
			if shotP != nil {
				shotRole = shotP.Role.Name()
			}
			e.emitEvent(UIEvent{Type: "death", Player: shotTarget, Role: shotRole, Round: e.state.Round, Players: buildUIPlayers(e.state.Players, false)})
			if shotP != nil && shotP.Role.Name() == "wolf_beauty" {
				if _, cerr := e.resolveWolfBeautyDeath(ctx, shotTarget); cerr != nil {
					return fmt.Errorf("wolf beauty charm after hunter shot in vote: %w", cerr)
				}
			}
			if shotTarget == e.state.Sheriff {
				if err := e.badgeTransfer(ctx, shotTarget); err != nil {
					return fmt.Errorf("badge transfer after hunter shot: %w", err)
				}
			}
		}
	}

	if eliminated.Name == e.state.Sheriff {
		if err := e.badgeTransfer(ctx, eliminated.Name); err != nil {
			return fmt.Errorf("badge transfer after vote elimination: %w", err)
		}
	}

	if eliminated.Role.Name() == "wolf_beauty" {
		if _, err := e.resolveWolfBeautyDeath(ctx, eliminated.Name); err != nil {
			return fmt.Errorf("wolf beauty charm after vote: %w", err)
		}
	}

	return nil
}

func (e *Engine) lastWords(ctx context.Context, p *player.Player) error {
	e.printf("[遗言] %s 正在发表遗言...\n", displayTag(p))

	var speechResult string
	speakTool, err := action.CreateSpeakTool(&speechResult)
	if err != nil {
		return fmt.Errorf("creating speak tool: %w", err)
	}

	pctx := e.buildPromptContext(p)
	instruction := prompt.BuildLastWords(pctx)

	e.emitEvent(UIEvent{Type: "thinking_start", Player: p.Name, Round: e.state.Round, Phase: "last_words"})

	var thoughts strings.Builder
	err = e.retryOnTransient(p.Name, func() error {
		thoughts.Reset()
		callCtx, cancel := e.withCallTimeout(ctx)
		defer cancel()

		agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("lastwords_%s", p.Name),
			Description: fmt.Sprintf("%s delivers last words", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{speakTool},
				},
			},
		})
		if aerr != nil {
			return aerr
		}

		runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
		iter := runner.Query(callCtx, "你被淘汰了。请发表你的遗言。用中文。")

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				return event.Err
			}
			msg, _, merr := adk.GetMessage(event)
			if merr != nil {
				continue
			}
			if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
				thoughts.WriteString(msg.Content)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	speech := speechResult
	if speech == "" && thoughts.Len() > 0 {
		speech = extractSpeakFromText(thoughts.String())
	}

	if thoughts.Len() > 0 && thoughts.String() != speech {
		e.printf("  [%s 内心] %s\n", displayTag(p), thoughts.String())
		e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: thoughts.String(), Round: e.state.Round, Phase: "last_words"})
	}
	if speech != "" {
		e.printf("[遗言] %s: %s\n\n", displayTag(p), speech)
		e.emitEvent(UIEvent{Type: "last_words", Player: p.Name, Content: speech, Round: e.state.Round})
		e.state.AddEvent(GameEvent{
			Round:   e.state.Round,
			Phase:   PhaseDay,
			Type:    EventSpeech,
			Actor:   p.Name,
			Content: fmt.Sprintf("%s 的遗言: %s", p.Name, speech),
			Public:  true,
		})
	}

	return nil
}

func (e *Engine) pkRound(ctx context.Context, tiedPlayers []string) error {
	e.println("--- PK 发言阶段 ---")
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "pk", Round: e.state.Round})

	for _, name := range tiedPlayers {
		p := e.state.GetPlayer(name)
		if p == nil || !p.Alive {
			continue
		}

		var speechResult string
		speakTool, err := action.CreateSpeakTool(&speechResult)
		if err != nil {
			return fmt.Errorf("creating speak tool for PK %s: %w", name, err)
		}

		pctx := e.buildPromptContext(p)
		pctx.PreviousSpeeches = e.state.FormatSpeeches(e.state.Round)
		instruction := prompt.BuildPKSpeech(pctx, tiedPlayers)

		e.emitEvent(UIEvent{Type: "thinking_start", Player: p.Name, Round: e.state.Round, Phase: "pk"})

		var thoughts strings.Builder
		err = e.retryOnTransient(p.Name, func() error {
			thoughts.Reset()
			callCtx, cancel := e.withCallTimeout(ctx)
			defer cancel()

			agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
				Name:        fmt.Sprintf("pk_%s", p.Name),
				Description: fmt.Sprintf("%s delivers PK speech", p.Name),
				Instruction: instruction,
				Model:       p.Model,
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: []tool.BaseTool{speakTool},
					},
				},
			})
			if aerr != nil {
				return aerr
			}

			runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
			iter := runner.Query(callCtx, "你进入了 PK 环节。请发表你的 PK 发言，说服大家不要淘汰你。用中文。")

			for {
				event, ok := iter.Next()
				if !ok {
					break
				}
				if event.Err != nil {
					return event.Err
				}
				msg, _, merr := adk.GetMessage(event)
				if merr != nil {
					continue
				}
				if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
					thoughts.WriteString(msg.Content)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("PK speech error for %s: %w", name, err)
		}

		speech := speechResult
		if speech == "" && thoughts.Len() > 0 {
			speech = extractSpeakFromText(thoughts.String())
		}

		if thoughts.Len() > 0 && thoughts.String() != speech {
			e.printf("  [%s 内心] %s\n", displayTag(p), thoughts.String())
			e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: thoughts.String(), Round: e.state.Round, Phase: "pk"})
		}

		if speech != "" {
			e.printf("[PK发言] %s: %s\n\n", displayTag(p), speech)
			e.emitEvent(UIEvent{Type: "speech", Player: p.Name, Content: speech, Round: e.state.Round})
			e.state.Speeches[e.state.Round] = append(e.state.Speeches[e.state.Round], Speech{
				Speaker: p.Name,
				Content: speech,
			})
			e.state.AddEvent(GameEvent{
				Round:   e.state.Round,
				Phase:   PhaseVote,
				Type:    EventSpeech,
				Actor:   p.Name,
				Content: fmt.Sprintf("%s PK发言: %s", p.Name, speech),
				Public:  true,
			})
		}
	}

	return nil
}

func (e *Engine) pkVote(ctx context.Context, tiedPlayers []string) (string, error) {
	e.println("--- PK 投票阶段 ---")

	tiedSet := make(map[string]bool, len(tiedPlayers))
	for _, n := range tiedPlayers {
		tiedSet[n] = true
	}

	alivePlayers := e.state.AlivePlayers()
	var subAgents []adk.Agent
	voteResults := make(map[string]*string)
	var voters []*player.Player

	for _, p := range alivePlayers {
		if tiedSet[p.Name] {
			continue
		}
		if e.state.IdiotRevealed[p.Name] {
			continue
		}

		voters = append(voters, p)
		var result string
		voteResults[p.Name] = &result

		voteTool, err := action.CreateVoteTool(tiedPlayers, &result)
		if err != nil {
			return "", fmt.Errorf("creating PK vote tool for %s: %w", p.Name, err)
		}

		pctx := e.buildPromptContext(p)
		pctx.PreviousSpeeches = e.state.FormatSpeeches(e.state.Round)
		instruction := prompt.BuildPKVote(pctx, tiedPlayers)

		agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        fmt.Sprintf("pk_vote_%s", p.Name),
			Description: fmt.Sprintf("%s casts PK vote", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: []tool.BaseTool{voteTool},
				},
			},
		})
		if err != nil {
			return "", fmt.Errorf("creating PK vote agent for %s: %w", p.Name, err)
		}
		subAgents = append(subAgents, agent)
	}

	if len(subAgents) == 0 {
		e.println("没有非平票玩家可以投票。PK 平局，无人被淘汰。")
		return "", nil
	}

	parAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "pk_vote_phase",
		Description: "PK re-vote where non-tied players vote on tied candidates",
		SubAgents:   subAgents,
	})
	if err != nil {
		return "", fmt.Errorf("creating PK parallel agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: parAgent})
	query := fmt.Sprintf("PK 投票。请从平票玩家中选择一名淘汰: %s。请用中文。", strings.Join(tiedPlayers, ", "))
	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			e.printf("  [error] PK vote API error: %v (skipping)\n", event.Err)
			continue
		}
		msg, _, err := adk.GetMessage(event)
		if err != nil {
			continue
		}
		if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && event.AgentName != "" {
			voterName := strings.TrimPrefix(event.AgentName, "pk_vote_")
			if voter := e.state.GetPlayer(voterName); voter != nil {
				e.printf("  [%s 内心] %s\n", displayTag(voter), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: voter.Name, Content: msg.Content, Round: e.state.Round, Phase: "pk_vote"})
			}
		}
	}

	votes := make(map[string]string)
	for _, p := range voters {
		if r := voteResults[p.Name]; r != nil && *r != "" {
			votes[p.Name] = *r
			e.printf("[PK投票] %s 投给了: %s\n", displayTag(p), *r)
			e.emitEvent(UIEvent{Type: "vote_cast", Player: p.Name, Target: *r, Round: e.state.Round})
		}
	}

	if len(votes) == 0 {
		return "", nil
	}

	result := TallyFlatVotes(votes)

	e.println("\nPK 投票统计:")
	for target, count := range result.Tally {
		e.printf("  %s: %d 票\n", target, count)
	}
	e.emitEvent(UIEvent{Type: "vote_tally", Tally: result.Tally, Tied: result.IsTied, Round: e.state.Round})

	if result.IsTied {
		return "", nil
	}

	return result.Eliminated, nil
}

func (e *Engine) openingNarration(ctx context.Context) {
	var sb strings.Builder
	for _, p := range e.state.Players {
		fmt.Fprintf(&sb, "%s: %s\n", p.Name, p.Persona)
	}

	setting := e.setting
	if setting == "" {
		setting = "一个宁静的村庄，最近却暗流涌动……"
	}

	narration, err := e.narrator.NarrateOpening(ctx, setting, sb.String())
	if err != nil {
		e.printf("[开场] %s\n\n", setting)
		e.emitEvent(UIEvent{Type: "narration", Content: setting, Phase: "opening"})
		return
	}

	e.printf("[旁白] %s\n\n", narration)
	e.emitEvent(UIEvent{Type: "narration", Content: narration, Phase: "opening"})
}

func (e *Engine) endGame(ctx context.Context, win WinResult) error {
	e.println("\n========== 游戏结束 ==========")
	e.printf("获胜方: %s\n", win.WinnerTeam)
	e.printf("原因: %s\n\n", win.Reason)

	summary := e.buildGameSummary()
	narration, err := e.narrator.NarrateGameEnd(ctx, win.WinnerTeam.String(), summary)
	if err != nil {
		e.println(summary)
	} else {
		e.printf("[旁白] %s\n\n", narration)
	}

	e.printFinalScoreboard()

	e.emitEvent(UIEvent{
		Type:    "game_end",
		Winner:  win.WinnerTeam.String(),
		Content: win.Reason,
		Players: buildUIPlayers(e.state.Players, true),
		Round:   e.state.Round,
	})

	e.postGameChat(ctx, win)

	return nil
}

func (e *Engine) postGameChat(ctx context.Context, win WinResult) {
	e.println()
	e.println("--- 赛后复盘 ---")
	e.println()
	e.emitEvent(UIEvent{Type: "phase_change", Phase: "postgame", Round: e.state.Round})

	gameSummary := e.buildGameSummary()

	var rolesBuilder strings.Builder
	for _, p := range e.state.Players {
		status := "存活"
		if !p.Alive {
			status = "已死亡"
		}
		fmt.Fprintf(&rolesBuilder, "%s — %s [%s]\n", p.Name, p.Role.Name(), status)
	}
	allRoles := rolesBuilder.String()

	var chatHistory []string

	for _, p := range e.state.Players {
		var speechResult string
		speakTool, err := action.CreateSpeakTool(&speechResult)
		if err != nil {
			continue
		}

		previousChats := strings.Join(chatHistory, "\n")
		instruction := prompt.BuildPostGameChat(
			p.Name, p.Role.Name(), p.Persona,
			win.WinnerTeam.String(), gameSummary, allRoles, previousChats,
		)

		e.throttle()
		e.emitEvent(UIEvent{Type: "thinking_start", Player: p.Name, Round: e.state.Round, Phase: "postgame"})
		err = e.retryOnTransient(p.Name, func() error {
			speechResult = ""
			callCtx, cancel := e.withCallTimeout(ctx)
			defer cancel()

			agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
				Name:        p.Name,
				Description: fmt.Sprintf("%s chats in post-game debrief", p.Name),
				Instruction: instruction,
				Model:       p.Model,
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: []tool.BaseTool{speakTool},
					},
				},
			})
			if aerr != nil {
				return aerr
			}

			runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
			iter := runner.Query(callCtx, "游戏结束了，轮到你发表赛后感言。用中文。")

			for {
				event, ok := iter.Next()
				if !ok {
					break
				}
				if event.Err != nil {
					return event.Err
				}
			}

			return nil
		})

		if err != nil {
			e.printf("[复盘] %s 发言失败: %v\n", p.Name, err)
			continue
		}

		if speechResult != "" {
			chatHistory = append(chatHistory, fmt.Sprintf("%s: %s", p.Name, speechResult))
			e.printf("[复盘] %s: %s\n", displayTag(p), speechResult)
			e.emitEvent(UIEvent{
				Type:    "postgame_chat",
				Player:  p.Name,
				Content: speechResult,
				Round:   e.state.Round,
			})
		}
	}

	e.println("\n--- 复盘结束 ---")
	e.emitEvent(UIEvent{Type: "postgame_end"})
}

const maxRetries = 3

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded {
		return true
	}
	s := err.Error()
	for _, pattern := range []string{
		"unexpected EOF", "connection reset", "connection refused",
		"i/o timeout", "TLS handshake timeout",
		"deadline exceeded", "context deadline exceeded",
		"500", "502", "503", "429",
	} {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

func (e *Engine) throttle() {
	if e.callInterval <= 0 {
		return
	}
	elapsed := time.Since(e.lastAPICall)
	if elapsed < e.callInterval {
		time.Sleep(e.callInterval - elapsed)
	}
	e.lastAPICall = time.Now()
}

func (e *Engine) retryOnTransient(name string, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * 2 * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			slog.Warn("retrying API call",
				"player", name,
				"attempt", attempt,
				"max_retries", maxRetries,
				"delay", delay.String(),
				"error", lastErr,
			)
			e.emitEvent(UIEvent{Type: "narration", Content: fmt.Sprintf("%s API 调用失败，正在重试 (%d/%d)...", name, attempt, maxRetries)})
			time.Sleep(delay)
		}
		e.throttle()
		start := time.Now()
		err := fn()
		elapsed := time.Since(start)
		if err == nil {
			if attempt > 0 {
				slog.Info("retry succeeded",
					"player", name,
					"attempt", attempt,
					"elapsed_ms", elapsed.Milliseconds(),
				)
			}
			return nil
		}
		lastErr = err
		if !isTransientError(err) {
			slog.Error("non-transient API error",
				"player", name,
				"error", err,
				"elapsed_ms", elapsed.Milliseconds(),
			)
			return err
		}
		slog.Warn("transient API error",
			"player", name,
			"attempt", attempt,
			"error", err,
			"elapsed_ms", elapsed.Milliseconds(),
		)
	}
	slog.Error("all retries exhausted",
		"player", name,
		"retries", maxRetries,
		"error", lastErr,
	)
	return fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

func (e *Engine) withCallTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if e.callTimeout > 0 {
		return context.WithTimeout(parent, e.callTimeout)
	}
	return parent, func() {}
}

func (e *Engine) runAgentWithTool(ctx context.Context, p *player.Player, instruction string, tools []tool.InvokableTool) error {
	var baseTools []tool.BaseTool
	for _, t := range tools {
		baseTools = append(baseTools, t)
	}

	return e.retryOnTransient(p.Name, func() error {
		callCtx, cancel := e.withCallTimeout(ctx)
		defer cancel()

		agent, err := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
			Name:        p.Name,
			Description: fmt.Sprintf("%s performing night action", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: baseTools,
				},
			},
		})
		if err != nil {
			return err
		}

		runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
		iter := runner.Query(callCtx, "轮到你了，执行你的夜晚行动。请用中文。")

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				return event.Err
			}
			msg, _, merr := adk.GetMessage(event)
			if merr != nil {
				continue
			}
			if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
				e.printf("  [%s 内心] %s\n", displayTag(p), msg.Content)
				e.emitEvent(UIEvent{Type: "thought", Player: p.Name, Content: msg.Content, Round: e.state.Round, Phase: "night"})
			}
		}

		return nil
	})
}

func (e *Engine) runAgentCapture(ctx context.Context, p *player.Player, instruction string, tools []tool.InvokableTool) (string, error) {
	var baseTools []tool.BaseTool
	for _, t := range tools {
		baseTools = append(baseTools, t)
	}

	var captured string
	err := e.retryOnTransient(p.Name, func() error {
		callCtx, cancel := e.withCallTimeout(ctx)
		defer cancel()

		agent, aerr := adk.NewChatModelAgent(callCtx, &adk.ChatModelAgentConfig{
			Name:        p.Name,
			Description: fmt.Sprintf("%s performing night action", p.Name),
			Instruction: instruction,
			Model:       p.Model,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: baseTools,
				},
			},
		})
		if aerr != nil {
			return aerr
		}

		runner := adk.NewRunner(callCtx, adk.RunnerConfig{Agent: agent})
		iter := runner.Query(callCtx, "轮到你了，执行你的夜晚行动。请用中文。")

		var thoughts strings.Builder
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				return event.Err
			}
			msg, _, merr := adk.GetMessage(event)
			if merr != nil {
				continue
			}
			if msg != nil && msg.Content != "" && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
				thoughts.WriteString(msg.Content)
			}
		}

		captured = thoughts.String()
		return nil
	})

	return captured, err
}

func (e *Engine) buildPromptContext(p *player.Player) prompt.PromptContext {
	pctx := prompt.PromptContext{
		GameRules:        e.gameRules,
		Setting:          e.setting,
		PlayerName:       p.Name,
		RoleName:         p.Role.Name(),
		RoleDescription:  p.Role.Description(),
		Persona:          p.Persona,
		AlivePlayers:     e.state.AlivePlayerNames(),
		Round:            e.state.Round,
		KnownInfo:        e.state.FormatVisibleEvents(p.Name),
		PreviousSpeeches: e.state.FormatSpeeches(e.state.Round),
		HealAvailable:    !e.state.WitchHealUsed,
		PoisonAvailable:  !e.state.WitchPoisonUsed,
		VictimName:       e.state.NightKillTarget,
		SheriffName:      e.state.Sheriff,
		IsSheriff:        e.state.Sheriff == p.Name,
	}

	if p.Role.Team() == config.TeamWerewolf {
		pctx.Teammates = e.state.WerewolfTeammates(p.Name)
	}

	if p.Role.Name() == "seer" {
		pctx.SeerResults = e.state.SeerResults
	}

	if p.Role.Name() == "idiot" {
		pctx.IdiotRevealed = e.state.IdiotRevealed[p.Name]
		pctx.CanVote = !e.state.IdiotRevealed[p.Name]
	} else {
		pctx.CanVote = true
	}

	return pctx
}

func (e *Engine) printPlayerRoster() {
	e.println("玩家列表:")
	for _, p := range e.state.Players {
		e.printf("  %s - 角色: %s\n", displayTag(p), p.Role.Name())
	}
	e.println()
}

func (e *Engine) printFinalScoreboard() {
	e.println("=== 最终计分板 ===")
	e.println()
	e.println("玩家结果:")
	for _, p := range e.state.Players {
		status := "存活"
		if !p.Alive {
			status = "淘汰"
		}
		e.printf("  %-30s  %-10s  %s\n", displayTag(p), p.Role.Name(), status)
	}
	e.println()

	e.println("模型使用统计:")
	e.logger.PrintStats()
}

func (e *Engine) sortBySeat(names []string) {
	seatIndex := make(map[string]int, len(e.state.Players))
	for i, p := range e.state.Players {
		seatIndex[p.Name] = i
	}
	sort.Slice(names, func(i, j int) bool {
		return seatIndex[names[i]] < seatIndex[names[j]]
	})
}

// extractSpeakFromText recovers speech content when a model writes
// speak(content='...') as plain text instead of invoking the tool.
// Uses LastIndex to handle multiple calls (takes the last one) and
// content containing quotes.
func extractSpeakFromText(text string) string {
	const marker = "speak(content="
	idx := strings.LastIndex(text, marker)
	if idx < 0 {
		return ""
	}
	rest := text[idx+len(marker):]
	if len(rest) < 3 {
		return ""
	}
	quote := rest[0]
	if quote != '\'' && quote != '"' {
		return ""
	}
	closing := string(quote) + ")"
	end := strings.LastIndex(rest, closing)
	if end <= 1 {
		return ""
	}
	return strings.TrimSpace(rest[1:end])
}

func displayTag(p *player.Player) string {
	return fmt.Sprintf("%s (%s)", p.Name, config.DisplayName(p.ModelID))
}

func (e *Engine) deathContent(baseMsg, roleName string) string {
	if e.identityReveal == config.IdentityRevealAlways {
		return fmt.Sprintf("%s（身份揭晓：%s）", baseMsg, roleChineseName(roleName))
	}
	return baseMsg
}

func roleChineseName(roleName string) string {
	switch roleName {
	case "werewolf":
		return "狼人"
	case "seer":
		return "预言家"
	case "witch":
		return "女巫"
	case "hunter":
		return "猎人"
	case "idiot":
		return "白痴"
	case "villager":
		return "村民"
	case "guard":
		return "守卫"
	case "knight":
		return "骑士"
	case "wolf_king":
		return "白狼王"
	case "wolf_beauty":
		return "狼美人"
	default:
		return roleName
	}
}

func (e *Engine) buildGameSummary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("游戏持续了 %d 回合。", e.state.Round))

	var survivors, eliminated []string
	for _, p := range e.state.Players {
		entry := fmt.Sprintf("%s [%s]", displayTag(p), p.Role.Name())
		if p.Alive {
			survivors = append(survivors, entry)
		} else {
			eliminated = append(eliminated, entry)
		}
	}

	if len(survivors) > 0 {
		sb.WriteString("存活者: " + strings.Join(survivors, ", ") + "。")
	}
	if len(eliminated) > 0 {
		sb.WriteString("淘汰者: " + strings.Join(eliminated, ", ") + "。")
	}

	return sb.String()
}
