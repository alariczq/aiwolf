package prompt

import (
	"fmt"
	"strings"
)

const gameRules = `【游戏规则】

游戏流程:
1. 夜晚: 狼人选择击杀目标 -> 女巫决定是否用药 -> 预言家查验一名玩家
2. 白天: 公布昨晚死亡情况 -> 所有存活玩家轮流发言讨论 -> 投票淘汰一名玩家
3. 重复以上流程直到一方获胜

%s
- 狼刀在先: 若狼人击杀导致胜利条件达成，游戏立即结束，女巫/预言家不行动

女巫用药规则:
- 解药: 可以救活当晚被狼人杀害的玩家，但不能自救
- 毒药: 可以毒杀任意存活玩家（每局仅一次）
- 同一晚只能使用一种药水
- 解药用完后，女巫不再获知每晚被刀者身份

猎人规则:
- 被狼人杀害或被投票淘汰时，可以开枪带走一名存活玩家
- 被女巫毒杀时，不能开枪
- 开枪是可选的，猎人可以选择不开枪

白痴规则:
- 被投票淘汰时，可以翻牌亮出身份免死
- 翻牌后失去投票权，但仍可发言
- 被狼人夜间杀害则正常死亡，无法使用技能

警长规则:
- 第一个白天前进行警长竞选，所有存活玩家投票选出警长
- 警长的投票权重为 1.5 票
- 警长拥有归票权：所有玩家发言后，警长可以归票给一名玩家（该玩家票数 +0.5）
- 归票权和 1.5 票权重在 PK 复投中不生效
- 警长死亡时可以将警徽转移给一名存活玩家，或选择撕毁警徽

狼人特殊行动:
- 空刀: 狼人可选择不杀人，意见不一致时视为空刀
- 自刀: 狼人可击杀自己或队友（策略性使用）
- 自爆: 白天发言时狼人可选择自爆，立即死亡并结束当天，不投票

其他规则:
- 夜晚行动秘密进行，白天只公布死亡结果（不公布死因）
- 夜间多人死亡按座位号顺序公布
- 投票平票时进入 PK 环节：平票玩家发言，其他玩家重新投票（每人1票，无警长加权）；PK 再平票则无人淘汰
- 预言家查验结果只显示"好人"或"狼人"，不显示具体角色`

type RulesVariant struct {
	WitchSelfSave  int
	IdentityReveal int
	VictoryMode    int
}

func BuildGameRules(roleCounts map[string]int, variant RulesVariant) string {
	total := 0
	for _, c := range roleCounts {
		total += c
	}

	type ri struct{ key, label, desc string }
	godRoles := []ri{
		{"seer", "预言家", "每晚查验一名玩家是好人还是狼人"},
		{"witch", "女巫", "拥有解药和毒药各一瓶"},
		{"hunter", "猎人", "死亡时可开枪带走一名玩家"},
		{"idiot", "白痴", "被投票淘汰时可翻牌免死但失去投票权"},
		{"guard", "守卫", "每晚守护一名玩家免受狼刀"},
		{"knight", "骑士", "白天可发起决斗，决斗狼人则狼人出局"},
	}

	var configParts []string
	configParts = append(configParts, fmt.Sprintf("%d狼人", roleCounts["werewolf"]))
	var godDescs []string
	godCount := 0
	for _, r := range godRoles {
		if roleCounts[r.key] > 0 {
			configParts = append(configParts, r.label)
			godDescs = append(godDescs, r.label+" - "+r.desc)
			godCount++
		}
	}
	configParts = append(configParts, fmt.Sprintf("%d村民", roleCounts["villager"]))

	var sb strings.Builder
	fmt.Fprintf(&sb, "【本局配置】%d人局 — %s\n\n", total, strings.Join(configParts, " + "))

	sb.WriteString("阵营与角色:\n")
	fmt.Fprintf(&sb, "- 狼人阵营（%d人）: 狼人 - 每晚选择击杀目标（允许空刀和自刀）\n", roleCounts["werewolf"])
	if godCount > 0 {
		fmt.Fprintf(&sb, "- 神职角色（%d人）: %s\n", godCount, strings.Join(godDescs, "；"))
	}
	fmt.Fprintf(&sb, "- 平民（%d人）: 村民 - 无特殊能力\n\n", roleCounts["villager"])

	victoryText := "胜利条件（屠边规则）:\n" +
		"- 好人阵营: 所有狼人被淘汰\n" +
		"- 狼人阵营: 所有神职死亡（屠神）或 所有平民死亡（屠民），两个条件满足其一即获胜"
	if variant.VictoryMode == 1 {
		victoryText = "胜利条件（屠城规则）:\n" +
			"- 好人阵营: 所有狼人被淘汰\n" +
			"- 狼人阵营: 所有好人阵营成员（神职+村民）全部死亡才获胜"
	}
	fmt.Fprintf(&sb, gameRules, victoryText)

	sb.WriteString("\n\n【本局规则变体】\n")

	switch variant.WitchSelfSave {
	case 1:
		sb.WriteString("- 女巫自救: 仅首夜可自救\n")
	case 2:
		sb.WriteString("- 女巫自救: 任何时候都可自救\n")
	default:
		sb.WriteString("- 女巫自救: 不能自救\n")
	}

	switch variant.IdentityReveal {
	case 1:
		sb.WriteString("- 身份公示: 明牌局——玩家死亡后翻牌公示身份\n")
	default:
		sb.WriteString("- 身份公示: 暗牌局——玩家死亡后不公开身份\n")
	}

	switch variant.VictoryMode {
	case 1:
		sb.WriteString("- 胜利模式: 屠城——狼人需消灭全部好人方可获胜\n")
	default:
		sb.WriteString("- 胜利模式: 屠边——狼人消灭全部神职或全部村民即获胜\n")
	}

	return sb.String()
}

type PromptContext struct {
	GameRules         string
	PlayerName        string
	RoleName          string
	RoleDescription   string
	Persona           string
	Teammates         []string
	AlivePlayers      []string
	Round             int
	KnownInfo         string
	DeathsLastNight   []string
	PreviousSpeeches  string
	VictimName        string
	HealAvailable     bool
	PoisonAvailable   bool
	SeerResults       map[string]string
	WolfDiscussion    string
	CanVote           bool
	IdiotRevealed     bool
	SheriffName       string
	IsSheriff         bool
	SheriffSpeeches   string
	SheriffCandidates []string
	WitchCanSelfSave  bool
	IdentityReveal    bool
	KnightDuelUsed    bool
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func formatSeerResults(results map[string]string) string {
	if len(results) == 0 {
		return "(none yet)"
	}
	parts := make([]string, 0, len(results))
	for name, result := range results {
		parts = append(parts, fmt.Sprintf("%s -> %s", name, result))
	}
	return strings.Join(parts, "\n  ")
}

func buildPersona(ctx PromptContext) string {
	if ctx.Persona == "" {
		return ""
	}
	return fmt.Sprintf("\n\n【你的人设】\n%s\n请在发言和思考中体现你的性格特点，让你的表现更有个人色彩。", ctx.Persona)
}

func buildRoleInfo(ctx PromptContext) string {
	switch strings.ToLower(ctx.RoleName) {
	case "werewolf":
		return fmt.Sprintf("你的狼人队友: %s\n你可以使用 self_explode 工具自爆（立即死亡，白天结束不投票）。", formatList(ctx.Teammates))
	case "seer":
		return fmt.Sprintf("你的查验结果:\n  %s", formatSeerResults(ctx.SeerResults))
	case "witch":
		healStatus := "已使用"
		if ctx.HealAvailable {
			healStatus = "可用"
		}
		poisonStatus := "已使用"
		if ctx.PoisonAvailable {
			poisonStatus = "可用"
		}
		return fmt.Sprintf("你的药水状态: 解药 %s, 毒药 %s", healStatus, poisonStatus)
	case "idiot":
		if ctx.IdiotRevealed {
			return "你已翻牌亮出白痴身份。你不能投票，但可以发言。"
		}
		return ""
	case "guard":
		return "你是守卫。每晚守护一名玩家免受狼刀，不能连续两晚守同一人。"
	case "knight":
		if ctx.KnightDuelUsed {
			return "你是骑士，决斗技能已使用。"
		}
		return "你是骑士。你可以在白天发言阶段发起决斗（使用 duel 工具），选择一名玩家决斗。决斗狼人则狼人出局，决斗好人则你出局。技能仅一次。"
	case "wolf_king":
		return fmt.Sprintf("你是白狼王（狼人阵营）。你的狼人队友: %s\n白天自爆时可同时带走一名玩家。", formatList(ctx.Teammates))
	case "wolf_beauty":
		return fmt.Sprintf("你是狼美人（狼人阵营）。你的狼人队友: %s\n每晚魅惑一名玩家，你死亡时被魅惑者殉情。你不能自爆。", formatList(ctx.Teammates))
	default:
		return ""
	}
}

func buildWolfInfo(ctx PromptContext) string {
	if strings.ToLower(ctx.RoleName) == "werewolf" {
		return fmt.Sprintf("\n你的狼人队友: %s", formatList(ctx.Teammates))
	}
	return ""
}

func buildStateBlock(ctx PromptContext) string {
	var parts []string

	if len(ctx.DeathsLastNight) > 0 {
		parts = append(parts, fmt.Sprintf("昨晚被淘汰的玩家: %s", formatList(ctx.DeathsLastNight)))
	} else if ctx.Round > 1 {
		parts = append(parts, "昨晚无人被淘汰。")
	}

	if ctx.KnownInfo != "" {
		parts = append(parts, fmt.Sprintf("已知信息:\n%s", ctx.KnownInfo))
	}

	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n")
}

func BuildWerewolfNight(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	wolfChat := ""
	if ctx.WolfDiscussion != "" {
		wolfChat = fmt.Sprintf("\n\n队友的讨论和选择:\n%s", ctx.WolfDiscussion)
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任狼人。

角色: %s
%s

当前回合: %d%s

你的狼人队友: %s
存活玩家: %s%s

现在是夜晚，狼人团队秘密行动。
你的文字回复 = 狼队内部讨论（只有队友能看到，好人看不到）。
使用 kill 工具提交你的击杀投票（可以空刀或自刀）。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		formatList(ctx.Teammates),
		formatList(ctx.AlivePlayers),
		wolfChat,
		buildPersona(ctx),
	)
}

func BuildSeerNight(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任预言家。

角色: %s
%s

当前回合: %d%s

历史查验结果:
  %s

存活玩家: %s

现在是夜晚。选择一名玩家查验，了解其是"好人"还是"狼人"。
你不能查验自己，也不能重复查验已查过的玩家。

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 investigate 工具提交你要查验的玩家名字。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		formatSeerResults(ctx.SeerResults),
		formatList(ctx.AlivePlayers),
		buildPersona(ctx),
	)
}

func BuildWitchNight(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	healStatus := "已使用（不可用）"
	if ctx.HealAvailable {
		healStatus = "可用"
	}
	poisonStatus := "已使用（不可用）"
	if ctx.PoisonAvailable {
		poisonStatus = "可用"
	}

	var victimInfo string
	if !ctx.HealAvailable && ctx.VictimName == "" {
		victimInfo = "（解药已用完，法官不再向你展示被刀者信息）"
	} else if ctx.VictimName == "" {
		victimInfo = "（今晚狼人选择了空刀，无人被杀）"
	} else if ctx.VictimName == ctx.PlayerName {
		if ctx.WitchCanSelfSave {
			victimInfo = ctx.VictimName + "（就是你自己，你可以选择自救）"
		} else {
			victimInfo = ctx.VictimName + "（就是你自己，但你不能自救）"
		}
	} else {
		victimInfo = ctx.VictimName
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任女巫。

角色: %s
%s

当前回合: %d%s

今晚狼人击杀目标: %s

你的药水状态:
  解药: %s
  毒药: %s

存活玩家: %s

现在是夜晚。解药和毒药整局各限一次，同一晚只能用一种。

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用解药请调用 heal 工具。使用毒药请调用 poison 工具并指定目标。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		victimInfo,
		healStatus,
		poisonStatus,
		formatList(ctx.AlivePlayers),
		buildPersona(ctx),
	)
}

func BuildWolfBeautyCharm(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任狼美人。

角色: %s
%s

当前回合: %d%s

你的狼人队友: %s
存活玩家: %s

狼人行动已结束。现在轮到你单独行动——选择一名玩家进行魅惑。
魅惑效果：当你死亡时，被魅惑的玩家也随之殉情出局。
你可以魅惑任何存活玩家（包括狼队队友）。

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 charm 工具提交你的魅惑目标。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		formatList(ctx.Teammates),
		formatList(ctx.AlivePlayers),
		buildPersona(ctx),
	)
}

func BuildGuardNight(ctx PromptContext, lastGuardTarget string) string {
	state := buildStateBlock(ctx)

	restriction := ""
	if lastGuardTarget != "" {
		restriction = fmt.Sprintf("\n你昨晚守护了 %s，今晚不能守护同一人。", lastGuardTarget)
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任守卫。

角色: %s
%s

当前回合: %d%s

存活玩家: %s%s

现在是夜晚。选择一名玩家守护，被守护者今晚不会被狼人刀杀。
你可以守护自己，也可以空守。不能连续两晚守护同一人。
守护只能挡狼刀，不能挡毒药。注意同守同救规则：守卫和女巫同时保护被刀者，保护抵消，被刀者死亡。

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 guard 工具提交你的守护选择。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		formatList(ctx.AlivePlayers),
		restriction,
		buildPersona(ctx),
	)
}

func BuildDayDiscussion(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	previousSpeeches := ""
	if ctx.PreviousSpeeches != "" {
		previousSpeeches = fmt.Sprintf("\n本轮已有发言:\n%s", ctx.PreviousSpeeches)
	}

	roleInfo := buildRoleInfo(ctx)
	if roleInfo != "" {
		roleInfo = "\n" + roleInfo
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。
%s

当前回合: %d%s%s

存活玩家: %s%s

现在是白天讨论阶段，之后投票淘汰一名玩家。

你的行动步骤:
1. 先在回复正文中思考——分析局势、推理身份。这是你的内心独白，只有你自己看得到。
2. 想好之后，调用 speak 工具说出你的公开发言。所有玩家都能听到，简洁有力，2-4句话。

公开发言是你表演给众人看的，要符合你的身份策略，和内心分析区分开。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.RoleDescription,
		ctx.Round,
		state,
		previousSpeeches,
		formatList(ctx.AlivePlayers),
		roleInfo,
		buildPersona(ctx),
	)
}

func BuildVote(ctx PromptContext) string {
	state := buildStateBlock(ctx)

	discussionSummary := ""
	if ctx.PreviousSpeeches != "" {
		discussionSummary = fmt.Sprintf("\n本轮讨论内容:\n%s", ctx.PreviousSpeeches)
	}

	sheriffInfo := ""
	if ctx.SheriffName != "" {
		sheriffInfo = fmt.Sprintf("\n当前警长: %s（投票权重 1.5 票）", ctx.SheriffName)
		if ctx.IsSheriff {
			sheriffInfo += "（你就是警长）"
		}
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

当前回合: %d%s%s%s

可投票淘汰的存活玩家: %s%s

讨论阶段已结束。投票选择一名玩家淘汰。
获得多数票的玩家将被淘汰（警长票权重 1.5，被归票者额外 +0.5）。

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 vote 工具提交你要投票淘汰的玩家名字。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.Round,
		state,
		discussionSummary,
		sheriffInfo,
		formatList(ctx.AlivePlayers),
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildHunterShoot(ctx PromptContext) string {
	previousSpeeches := ""
	if ctx.PreviousSpeeches != "" {
		previousSpeeches = fmt.Sprintf("\n最近的讨论内容:\n%s", ctx.PreviousSpeeches)
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中担任猎人。

你刚刚被淘汰了！作为猎人，你可以选择开枪带走一名存活的玩家，也可以选择不开枪。

存活玩家: %s

已知信息:
%s%s

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 shoot 工具提交你的选择。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		formatList(ctx.AlivePlayers),
		ctx.KnownInfo,
		previousSpeeches,
		buildPersona(ctx),
	)
}

func BuildCampaignDecision(ctx PromptContext) string {
	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

现在是警长竞选阶段的第一步：决定是否上警。
- 上警（run=true）：你将成为候选人，发表竞选演说，接受投票。
- 不上警（run=false）：你不发言，但可以在投票阶段投票选出警长。

存活玩家: %s%s

使用 campaign_decision 工具提交你的决定。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(ctx.AlivePlayers),
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildSheriffCampaign(ctx PromptContext) string {
	previousSpeeches := ""
	if ctx.SheriffSpeeches != "" {
		previousSpeeches = fmt.Sprintf("\n已有的竞选发言:\n%s", ctx.SheriffSpeeches)
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

现在是警长竞选发言阶段。
警长拥有 1.5 倍投票权重（在淘汰投票中生效），死亡时可将警徽转移。

存活玩家: %s%s%s

你的行动步骤:
1. 先在回复正文中思考——分析竞选策略。这是你的内心独白，只有你自己看得到。
2. 想好之后，调用 speak 工具发表你的竞选演说。所有玩家都能听到，简洁有力，2-3句话。

公开演说是你表演给众人看的，要符合你的身份策略，和内心分析区分开。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(ctx.AlivePlayers),
		previousSpeeches,
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildWithdrawDecision(ctx PromptContext) string {
	wolfInfo := buildWolfInfo(ctx)

	campaignSpeeches := ""
	if ctx.SheriffSpeeches != "" {
		campaignSpeeches = fmt.Sprintf("\n竞选发言记录:\n%s", ctx.SheriffSpeeches)
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

竞选发言阶段已结束。现在是退水阶段。
你是候选人之一，你可以选择"退水"（放弃竞选）。
- 退水后：你不能被投票，也不能投票选警长。
- 不退水：继续竞选，等待投票结果。
- 狼人可以在此阶段自爆（使用 self_explode 工具），立即结束警长竞选（吞警徽）。

存活玩家: %s%s%s

使用 withdraw_decision 工具提交你的决定。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(ctx.AlivePlayers),
		campaignSpeeches,
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildSheriffElection(ctx PromptContext) string {
	campaignSpeeches := ""
	if ctx.SheriffSpeeches != "" {
		campaignSpeeches = fmt.Sprintf("\n竞选发言记录:\n%s", ctx.SheriffSpeeches)
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

现在是警长投票阶段。你没有上警，所以由你来投票选出警长。

候选人: %s%s%s

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 sheriff_vote 工具投给你选择的候选人。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(ctx.SheriffCandidates),
		campaignSpeeches,
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildBadgeTransfer(ctx PromptContext) string {
	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s（%s），你是当前的警长。

你刚刚被淘汰了！作为警长，你可以将警徽转移给一名存活的玩家，或选择撕毁警徽（本局不再有警长）。
被转移警徽的玩家将继承 1.5 倍投票权重和归票权。

存活玩家: %s

已知信息:
%s%s

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 transfer_badge 工具提交你的选择。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(ctx.AlivePlayers),
		ctx.KnownInfo,
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildLastWords(ctx PromptContext) string {
	knownInfo := ""
	if ctx.KnownInfo != "" {
		knownInfo = fmt.Sprintf("\n已知信息:\n%s", ctx.KnownInfo)
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

你刚刚被淘汰了。现在是你的遗言时间，这是你最后一次发言的机会。

当前回合: %d%s

存活玩家: %s%s

你的行动步骤:
1. 先在回复正文中思考——整理你想说的最后的话。这是你的内心独白，只有你自己看得到。
2. 想好之后，调用 speak 工具说出你的遗言。所有玩家都能听到，简洁有力，1-3句话。

遗言和内心思考要区分开。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.Round,
		knownInfo,
		formatList(ctx.AlivePlayers),
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildPKSpeech(ctx PromptContext, tiedPlayers []string) string {
	state := buildStateBlock(ctx)

	previousSpeeches := ""
	if ctx.PreviousSpeeches != "" {
		previousSpeeches = fmt.Sprintf("\n本轮已有发言:\n%s", ctx.PreviousSpeeches)
	}

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

投票出现平票！你是被平票的玩家之一。
平票玩家: %s

只有平票玩家可以发言，之后非平票玩家重新投票。

当前回合: %d%s%s

存活玩家: %s

你的行动步骤:
1. 先在回复正文中思考——分析如何为自己辩护。这是你的内心独白，只有你自己看得到。
2. 想好之后，调用 speak 工具说出你的 PK 发言。所有玩家都能听到，简洁有力，2-3句话。

公开发言是你表演给众人看的，要符合你的身份策略，和内心分析区分开。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(tiedPlayers),
		ctx.Round,
		state,
		previousSpeeches,
		formatList(ctx.AlivePlayers),
		buildPersona(ctx),
	)
}

func BuildPKVote(ctx PromptContext, tiedPlayers []string) string {
	state := buildStateBlock(ctx)

	discussionSummary := ""
	if ctx.PreviousSpeeches != "" {
		discussionSummary = fmt.Sprintf("\n本轮讨论及PK发言:\n%s", ctx.PreviousSpeeches)
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，在这场狼人杀游戏中扮演 %s。

当前回合: %d%s%s

投票出现平票，现在进入 PK 复投。
平票玩家: %s

PK 复投规则: 只能从平票玩家中选一人投票，每人 1 票，警长没有额外权重。再次平票则无人被淘汰。
%s

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 vote 工具提交你要投票淘汰的玩家名字。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		ctx.Round,
		state,
		discussionSummary,
		formatList(tiedPlayers),
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildSheriffEndorse(ctx PromptContext, candidates []string) string {
	discussionSummary := ""
	if ctx.PreviousSpeeches != "" {
		discussionSummary = fmt.Sprintf("\n本轮讨论内容:\n%s", ctx.PreviousSpeeches)
	}

	wolfInfo := buildWolfInfo(ctx)

	return fmt.Sprintf(`%s

你是 %s，警长，在这场狼人杀游戏中扮演 %s。

所有玩家发言结束。作为警长，你现在可以行使归票权。
归票 = 你选择一名玩家，该玩家的票数 +0.5。你也可以选择不归票。

存活玩家: %s%s%s

你的文字回复 = 你的内心推理（私密的，其他玩家看不到）。
使用 endorse 工具归票给你选择的玩家，或选择不归票。
你必须用中文。%s`,
		ctx.GameRules,
		ctx.PlayerName,
		ctx.RoleName,
		formatList(candidates),
		discussionSummary,
		wolfInfo,
		buildPersona(ctx),
	)
}

func BuildPostGameChat(playerName, roleName, persona, winner, gameSummary, allRoles, previousChats string) string {
	prev := "(还没人说话)"
	if previousChats != "" {
		prev = previousChats
	}

	personaLine := ""
	if persona != "" {
		personaLine = "\n你的人设: " + persona
	}

	return fmt.Sprintf(`游戏结束了！%s 获胜！

你是 %s，你在游戏中的角色是 %s。%s

所有人的真实身份已揭晓:
%s
游戏回顾:
%s

其他人的赛后发言:
%s

现在是赛后复盘时间，大家像朋友一样轻松聊天。
调用 speak 工具说出你的赛后感想（1-3句话）。可以:
- 吐槽或赞赏其他玩家的神操作或迷惑行为
- 分享你当时的心路历程和关键决策背后的思考
- 开玩笑、表达遗憾或得意
- 回应别人刚才说的话

语气要轻松有趣，像朋友间打完一局游戏后的闲聊。保持你的人设性格。
你必须用中文。`,
		winner,
		playerName, roleName, personaLine,
		allRoles,
		gameSummary,
		prev,
	)
}
