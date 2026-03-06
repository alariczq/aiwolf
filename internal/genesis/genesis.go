package genesis

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/model"
)

type playerSpec struct {
	Name    string `json:"name" desc:"unique Chinese name (2-3 characters)"`
	Role    string `json:"role" desc:"one of: werewolf, seer, witch, hunter, idiot, villager, guard, knight, wolf_king, wolf_beauty"`
	Persona string `json:"persona" desc:"rich personality portrait in Chinese (4-6 sentences covering temperament, speech habits, backstory, quirks, and interpersonal stance)"`
}

type rulesSpec struct {
	WitchSelfSave  string `json:"witch_self_save" desc:"one of: never, first_night_only, always"`
	IdentityReveal string `json:"identity_reveal" desc:"one of: never (暗牌局), always (明牌局)"`
	VictoryMode    string `json:"victory_mode" desc:"one of: edge (屠边), city (屠城)"`
}

type worldSpec struct {
	Setting string       `json:"setting" desc:"village setting flavor text in Chinese (1-2 sentences)"`
	Players []playerSpec `json:"players" desc:"all players (8-12)"`
	Rules   rulesSpec    `json:"rules" desc:"game rule variants chosen by the god"`
}

const godInstruction = `你是创世之神。一场狼人杀游戏即将开始，你需要创造这个世界——决定游戏的规模与配置，并为每一名村民赋予姓名、灵魂与命运。

【游戏配置】你可以自由决定这场游戏的规模和角色搭配：
- 总人数：8-12 人
- 狼人：2-4 人，建议约总人数的 1/3（role 填 "werewolf"）
- 预言家：必须 1 人（role 填 "seer"）
- 女巫：0 或 1 人（role 填 "witch"）
- 猎人：0 或 1 人（role 填 "hunter"）
- 白痴：0 或 1 人（role 填 "idiot"）
- 守卫：0 或 1 人（role 填 "guard"）
- 骑士：0 或 1 人（role 填 "knight"）
- 白狼王：0 或 1 人（role 填 "wolf_king"，属于狼人阵营，占狼人名额）
- 狼美人：0 或 1 人（role 填 "wolf_beauty"，属于狼人阵营，占狼人名额）
- 村民：其余都是村民，至少 2 人（role 填 "villager"）

常见板子参考（你可以选其中一种，也可以自由创造）：
- "预女猎白"12人局：4狼 + 预言家 + 女巫 + 猎人 + 白痴 + 4村民
- "预女猎"9人局：3狼 + 预言家 + 女巫 + 猎人 + 3村民
- "预女白"9人局：3狼 + 预言家 + 女巫 + 白痴 + 3村民
- "双神"8人局：3狼 + 预言家 + 女巫 + 3村民
- "满配四神"10人局：3狼 + 预言家 + 女巫 + 猎人 + 白痴 + 3村民
- "预女猎守"12人局：4狼 + 预言家 + 女巫 + 猎人 + 守卫 + 4村民
- "预女猎守+白狼王"12人局：3普狼+白狼王 + 预言家 + 女巫 + 猎人 + 守卫 + 4村民
- "预女猎骑+白狼王"12人局：3普狼+白狼王 + 预言家 + 女巫 + 猎人 + 骑士 + 4村民
- "预女猎守+狼美人"12人局：3普狼+狼美人 + 预言家 + 女巫 + 猎人 + 守卫 + 4村民

【创造指南】

1. 名字
   每人一个独特的中文名（2-3 个字）。名字要有故事感——可以暗示性格、职业或命运。
   名字风格要统一（同一个世界观下的人），但各有特色，避免谐音或字形过于相似。

2. 人设（重点！每人 4-6 句中文，必须写得丰满立体）
   每个人设必须覆盖以下全部维度：

   a) 性格内核：不只是一个形容词，要写出这个人"为什么"是这个性格。
      差: "性格多疑"
      好: "从小在复杂的家族争斗中长大，习惯性地怀疑每一句好话背后的动机"

   b) 说话方式：给出具体的语言习惯，让这个人一开口就能被认出来。
      包括：口头禅、句式偏好、用词风格、语速节奏。
      例: "喜欢用'你想想看'开头引导别人思考"、"说话总是慢半拍，但每个字都像钉子一样扎人"

   c) 行为特征：一个标志性的小动作或习惯，让人物有画面感。
      例: "紧张时会反复搓手指"、"赞同别人时喜欢拍桌子"、"思考时会闭眼念念有词"

   d) 背景故事：一两句话交代来历，解释这个人是怎么变成现在这样的。
      包含：职业/身份、关键人生经历、来到这个村庄的原因。
      例: "退休的老刑警，破了一辈子案，自认为看人一看一个准"

   e) 人际姿态：这个人如何与他人互动？是主动型还是被动型？信任他人还是防备他人？
      例: "喜欢拉帮结派，总想找到'自己人'"、"独来独往，觉得少说少错"

   示例人设（仅作参考，不要照抄）：
   "镇上茶馆的老板娘，做了二十年生意练就了察言观色的本事，能从一个人端茶的姿势判断他心里有没有鬼。说话爽利不绕弯，口头禅是'别跟我打马虎眼'。急了会直接拍桌子站起来。表面上对谁都笑脸相迎，实际上心里有一本账，谁欠了人情、谁和谁不对付，记得比谁都清楚。喜欢在关键时刻抛出一个所有人都忽略的细节，享受全场震惊的感觉。"

3. 多样性要求
   - 性格谱系要广：必须包含至少 2 个外向型、2 个内向型、1 个反复无常型
   - 说话方式不能雷同：有人简短有力、有人绕弯子、有人爱用反问、有人喜欢打比方
   - 年龄感要有差异：有老练世故的、有年轻冲动的
   - 社交姿态要有对比：有人爱拉帮结派、有人独来独往、有人左右逢源

4. 戏剧性搭配（非常重要！）
   角色分配不是随机的，你要精心设计性格与角色的反差或呼应：
   - 最受大家信任的人 → 恰恰是狼人（伪装高手）
   - 性格最暴躁冲动的人 → 是需要冷静判断的预言家（内心戏剧冲突）
   - 最胆小怕事的人 → 是关键时刻要开枪的猎人（被命运推上去）
   - 最老实木讷的人 → 是掌握生死大权的女巫
   - 最自以为聪明的人 → 偏偏是白痴（讽刺效果拉满）
   这些只是示例——你应该创造属于这局游戏独特的戏剧张力。

5. 世界观
   为这个村庄写 1-2 句设定。不要泛泛的"宁静村庄"，要有具体细节：
   什么样的地方？发生了什么事让气氛紧张？村民之间有什么暗流？
   例: "雾锁山腰的采药村，三天前村长的药田被人连夜烧毁，所有人都在暗中打量彼此。"

6. 规则变体（你来决定这场游戏的规则风格）

   a) 女巫自救规则（witch_self_save）：
      - "never"：女巫不能自救（竞技标准，策略性最强）
      - "first_night_only"：仅首夜可自救（折中方案）
      - "always"：任何时候都可自救（休闲友好）
      根据你创造的世界氛围来决定：硬核的世界用 never，温情的世界可以用 first_night_only 或 always。

   b) 身份公示规则（identity_reveal）：
      - "never"：暗牌局——玩家死亡后不公开身份，只有法官知道（竞技标准，信息博弈更深）
      - "always"：明牌局——玩家死亡后翻牌公示身份（休闲/戏剧化，信息更透明）
      悬疑紧张的世界适合暗牌局，戏剧冲突强烈的世界适合明牌局。

   c) 胜利模式（victory_mode）：
      - "edge"：屠边——狼人消灭全部神职或全部村民即获胜（竞技标准）
      - "city"：屠城——狼人需消灭全部好人（神职+村民）方可获胜（休闲/经典）

请调用 create_world 工具提交你的创世方案。`

var validRoles = map[string]bool{
	"werewolf":    true,
	"seer":        true,
	"witch":       true,
	"hunter":      true,
	"idiot":       true,
	"villager":    true,
	"guard":       true,
	"knight":      true,
	"wolf_king":   true,
	"wolf_beauty": true,
}

func Create(ctx context.Context, models config.ModelConfig) (config.GameConfig, error) {
	prov := model.NewProvider(models)
	m, err := pickModel(ctx, prov)
	if err != nil {
		return config.GameConfig{}, err
	}

	var lastErr error
	for attempt := range 2 {
		cfg, err := runGod(ctx, m, models)
		if err == nil {
			return cfg, nil
		}
		lastErr = err
		log.Printf("[genesis] attempt %d failed: %v", attempt+1, err)
	}

	return config.GameConfig{}, fmt.Errorf("genesis failed: %w", lastErr)
}

func runGod(ctx context.Context, m einomodel.ToolCallingChatModel, models config.ModelConfig) (config.GameConfig, error) {
	var world worldSpec

	createTool, err := utils.InferTool("create_world",
		"Create the werewolf game world with 8-12 players. "+
			"Decide the game variant (role composition), give each player a unique Chinese name, assign a role, and write a rich persona.",
		func(ctx context.Context, spec worldSpec) (string, error) {
			world = spec
			return "创世完成。", nil
		})
	if err != nil {
		return config.GameConfig{}, fmt.Errorf("creating genesis tool: %w", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "god",
		Description: "The God who creates the werewolf game world",
		Instruction: godInstruction,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{createTool},
			},
		},
	})
	if err != nil {
		return config.GameConfig{}, fmt.Errorf("creating god agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Query(ctx, "创造世界吧。请调用 create_world 工具。")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return config.GameConfig{}, fmt.Errorf("god agent: %w", event.Err)
		}
		msg, _, merr := adk.GetMessage(event)
		if merr == nil && msg != nil && msg.Content != "" &&
			msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
			log.Printf("[genesis] %s", msg.Content)
		}
	}

	if len(world.Players) == 0 {
		return config.GameConfig{}, fmt.Errorf("god agent produced no players")
	}

	if err := validate(world); err != nil {
		return config.GameConfig{}, fmt.Errorf("invalid world: %w", err)
	}

	modelPool := assignModels(len(world.Players))

	players := make([]config.PlayerConfig, len(world.Players))
	for i, p := range world.Players {
		players[i] = config.PlayerConfig{
			Name:    p.Name,
			Role:    p.Role,
			ModelID: modelPool[i],
			Persona: p.Persona,
		}
	}

	return config.GameConfig{
		Players:        players,
		Models:         models,
		Setting:        world.Setting,
		WitchSelfSave:  parseWitchSelfSave(world.Rules.WitchSelfSave),
		IdentityReveal: parseIdentityReveal(world.Rules.IdentityReveal),
		VictoryMode:    parseVictoryMode(world.Rules.VictoryMode),
	}, nil
}

func validate(w worldSpec) error {
	n := len(w.Players)
	if n < 8 || n > 12 {
		return fmt.Errorf("player count %d out of range [8,12]", n)
	}

	counts := make(map[string]int)
	names := make(map[string]bool)
	for _, p := range w.Players {
		if !validRoles[p.Role] {
			return fmt.Errorf("invalid role %q for %s", p.Role, p.Name)
		}
		counts[p.Role]++
		if names[p.Name] {
			return fmt.Errorf("duplicate name: %s", p.Name)
		}
		names[p.Name] = true
		if p.Persona == "" {
			return fmt.Errorf("empty persona for %s", p.Name)
		}
	}

	totalWolves := counts["werewolf"] + counts["wolf_king"] + counts["wolf_beauty"]
	if totalWolves < 2 || totalWolves > 4 {
		return fmt.Errorf("total wolf count %d out of range [2,4]", totalWolves)
	}
	if counts["seer"] != 1 {
		return fmt.Errorf("must have exactly 1 seer, got %d", counts["seer"])
	}
	for _, role := range []string{"witch", "hunter", "idiot", "guard", "knight", "wolf_king", "wolf_beauty"} {
		if counts[role] > 1 {
			return fmt.Errorf("at most 1 %s, got %d", role, counts[role])
		}
	}
	if counts["wolf_beauty"] > 0 && totalWolves < 2 {
		return fmt.Errorf("wolf_beauty requires at least 2 total wolves, got %d", totalWolves)
	}
	if counts["villager"] < 2 {
		return fmt.Errorf("need at least 2 villagers, got %d", counts["villager"])
	}

	return nil
}

func assignModels(count int) []string {
	pool := config.DefaultModelPool()
	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if count <= len(pool) {
		return pool[:count]
	}
	result := make([]string, count)
	for i := range count {
		result[i] = pool[i%len(pool)]
	}
	return result
}

func pickModel(ctx context.Context, prov *model.Provider) (einomodel.ToolCallingChatModel, error) {
	for _, id := range []string{"gemini-pro-preview", "gemini-pro", "claude-sonnet", "claude-haiku", "gemini-flash"} {
		m, err := prov.GetModel(ctx, id)
		if err == nil {
			log.Printf("[genesis] using model: %s", id)
			return m, nil
		}
	}
	return nil, fmt.Errorf("no model available for genesis")
}

func parseWitchSelfSave(s string) config.WitchSelfSave {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "first_night_only":
		return config.WitchSelfSaveFirstOnly
	case "always":
		return config.WitchSelfSaveAlways
	default:
		return config.WitchSelfSaveNever
	}
}

func parseIdentityReveal(s string) config.IdentityReveal {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "always":
		return config.IdentityRevealAlways
	default:
		return config.IdentityRevealNever
	}
}

func parseVictoryMode(s string) config.VictoryMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "city":
		return config.VictoryModeCity
	default:
		return config.VictoryModeEdge
	}
}
