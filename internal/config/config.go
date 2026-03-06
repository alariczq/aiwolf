package config

import (
	"math/rand/v2"
	"os"
)

type Team int

const (
	TeamVillager Team = iota
	TeamWerewolf
)

func (t Team) String() string {
	switch t {
	case TeamVillager:
		return "好人阵营"
	case TeamWerewolf:
		return "狼人阵营"
	default:
		return "未知"
	}
}

type PlayerConfig struct {
	Name    string
	Role    string
	ModelID string
	Persona string
}

type ModelConfig struct {
	ClaudeAPIKey string
	GeminiAPIKey string
}

type WitchSelfSave int

const (
	WitchSelfSaveNever     WitchSelfSave = iota
	WitchSelfSaveFirstOnly
	WitchSelfSaveAlways
)

type IdentityReveal int

const (
	IdentityRevealNever  IdentityReveal = iota
	IdentityRevealAlways
)

type VictoryMode int

const (
	VictoryModeEdge VictoryMode = iota // default: wolves win if all gods OR all villagers dead
	VictoryModeCity                    // wolves win only if all gods AND all villagers dead
)

type GameConfig struct {
	Players        []PlayerConfig
	Models         ModelConfig
	Setting        string
	WitchSelfSave  WitchSelfSave
	IdentityReveal IdentityReveal
	VictoryMode    VictoryMode
}

func ModelConfigFromEnv() ModelConfig {
	return ModelConfig{
		ClaudeAPIKey: os.Getenv("CLAUDE_API_KEY"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
	}
}

var defaultModels = []string{
	"claude-opus",
	"gemini-pro-preview",
	"claude-sonnet",
	"claude-sonnet",
	"gemini-pro",
	"gemini-pro",
	"claude-haiku",
	"claude-haiku",
	"claude-haiku",
	"gemini-flash",
	"gemini-flash",
	"gemini-flash",
}

var defaultNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank",
	"Grace", "Henry", "Ivy", "Jack", "Kate", "Leo",
}

var defaultRoles = []string{
	"werewolf",
	"werewolf",
	"werewolf",
	"werewolf",
	"seer",
	"witch",
	"hunter",
	"idiot",
	"villager",
	"villager",
	"villager",
	"villager",
}

var modelAliasDisplay = map[string]string{
	"claude-haiku":      "Claude-Haiku-4.5",
	"claude-sonnet":     "Claude-Sonnet-4.6",
	"claude-opus":       "Claude-Opus-4.6",
	"gemini-flash":      "Gemini-2.5-Flash",
	"gemini-pro":        "Gemini-2.5-Pro",
	"gemini-pro-preview": "Gemini-3.1-Pro",
}

func DisplayName(modelID string) string {
	if name, ok := modelAliasDisplay[modelID]; ok {
		return name
	}
	return modelID
}

var personalities = []string{
	"冲动易怒，容易和人争吵",
	"冷静理性，喜欢用逻辑分析",
	"热情开朗，喜欢活跃气氛",
	"多疑敏感，总觉得别人在针对自己",
	"胆小谨慎，害怕成为目标",
	"自信张扬，喜欢主导讨论",
	"沉默寡言，只在关键时刻发言",
	"圆滑世故，擅长左右逢源",
	"正义感爆棚，容不得任何可疑行为",
	"悲观消极，总往坏处想",
	"幽默搞怪，喜欢用段子表达观点",
	"温柔体贴，总想保护弱势玩家",
	"强势霸道，说一不二",
	"心思细腻，善于观察细节",
	"大大咧咧，不太在意细节",
	"老谋深算，说话总留三分",
}

var speakingStyles = []string{
	"说话简短有力，不废话",
	"喜欢用反问句",
	"经常用比喻和类比",
	"说话带点讽刺味",
	"语气温和但立场坚定",
	"喜欢引经据典",
	"说话直来直去，不绕弯子",
	"喜欢用排比句加强语气",
	"语速快，信息量大",
	"慢条斯理，一字一句很有分量",
}

var hobbies = []string{
	"棋牌爱好者，喜欢用博弈论思考",
	"推理小说迷，总想当侦探",
	"历史迷，喜欢用历史故事类比",
	"美食家，紧张时会聊吃的缓解气氛",
	"哲学爱好者，偶尔会思考人性",
	"体育迷，喜欢用球赛类比策略",
	"诗词爱好者，说话偶尔带点文艺范",
	"电影迷，会引用经典台词",
	"音乐迷，心情好坏会影响节奏感",
	"科技迷，喜欢用数据和概率说话",
}

var emotionalTraits = []string{
	"容易激动，情绪波动大",
	"喜怒不形于色，很难被看穿",
	"遇到压力会紧张说错话",
	"越危险越兴奋",
	"容易共情，对死亡事件反应强烈",
	"情绪稳定，不受他人影响",
	"容易被煽动，立场不够坚定",
	"表面镇定内心慌张",
}

func generatePersona() string {
	p := personalities[rand.IntN(len(personalities))]
	s := speakingStyles[rand.IntN(len(speakingStyles))]
	h := hobbies[rand.IntN(len(hobbies))]
	e := emotionalTraits[rand.IntN(len(emotionalTraits))]
	return "性格: " + p + "。说话风格: " + s + "。兴趣: " + h + "。情绪特点: " + e + "。"
}

func DefaultModelPool() []string {
	pool := make([]string, len(defaultModels))
	copy(pool, defaultModels)
	return pool
}

func DefaultGameConfig() GameConfig {
	roles := make([]string, len(defaultRoles))
	copy(roles, defaultRoles)
	rand.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	models := make([]string, len(defaultModels))
	copy(models, defaultModels)
	rand.Shuffle(len(models), func(i, j int) {
		models[i], models[j] = models[j], models[i]
	})

	players := make([]PlayerConfig, len(models))
	for i, modelID := range models {
		players[i] = PlayerConfig{
			Name:    defaultNames[i],
			Role:    roles[i],
			ModelID: modelID,
			Persona: generatePersona(),
		}
	}

	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	return GameConfig{
		Players: players,
		Models:  ModelConfigFromEnv(),
	}
}
