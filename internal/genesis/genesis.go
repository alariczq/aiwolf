package genesis

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"unicode"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/model"
)

type playerSpec struct {
	Name    string `json:"name" jsonschema_description:"角色名字，风格与世界观一致，纯文字不含 emoji/引号/特殊符号，1-4个词"`
	Role    string `json:"role" jsonschema:"enum=werewolf,enum=seer,enum=witch,enum=hunter,enum=idiot,enum=villager,enum=guard,enum=knight,enum=wolf_king,enum=wolf_beauty" jsonschema_description:"角色身份"`
	Persona string `json:"persona" jsonschema_description:"丰满的人物画像（4-6句中文，涵盖性格内核、说话方式、行为特征、背景故事、人际姿态）"`
}

type rulesSpec struct {
	WitchSelfSave  string `json:"witch_self_save" jsonschema:"enum=never,enum=first_night_only,enum=always" jsonschema_description:"女巫自救规则：never=不可自救，first_night_only=仅首夜，always=任何时候"`
	IdentityReveal string `json:"identity_reveal" jsonschema:"enum=never,enum=always" jsonschema_description:"身份公示规则：never=暗牌局，always=明牌局"`
	VictoryMode    string `json:"victory_mode" jsonschema:"enum=edge,enum=city" jsonschema_description:"胜利模式：edge=屠边，city=屠城"`
}

type worldSpec struct {
	Setting string       `json:"setting" jsonschema_description:"世界设定（2-3句中文，任何题材/风格均可）"`
	Players []playerSpec `json:"players" jsonschema_description:"所有玩家（8-12人）"`
	Rules   rulesSpec    `json:"rules" jsonschema_description:"上帝选择的游戏规则变体"`
}

const godInstruction = `你是创世之神。一场狼人杀游戏即将开始，你需要创造一个完整的世界——选择任意你想要的题材与风格，决定游戏的规模与配置，并为每一个角色赋予名字、灵魂与命运。

【核心理念】你拥有完全的创作自由。不要局限于任何单一题材。
每一局游戏都应该是一个全新的世界，让人意想不到。以下仅为灵感示例，你也完全可以创造这些之外的题材：
- 赛博朋克霓虹都市中的地下黑客组织
- 漂流在深海中的末日避难潜艇
- 中世纪欧洲瘟疫蔓延时期的修道院
- 1930年代上海租界的谍战风云
- 星际殖民飞船上的休眠舱谋杀案
- 日本战国时代的忍者暗斗
- 现代互联网大厂的年终裁员风暴
- 维多利亚时代伦敦的灵媒沙龙
- 三国乱世中的一座孤城
- 墨西哥亡灵节前夜的小镇
- 90年代香港黑帮电影式的江湖
- 北欧神话中诸神黄昏前夕的阿斯加德
- 民国时期的戏班子

你选择的题材将决定一切：世界设定、人物名字的风格、角色的背景故事、说话的语气和用词。

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

1. 世界观（2-3 句）
   先确定题材和基调，再构建具体场景。要有画面感和紧迫感：
   这是什么样的世界？此刻正在发生什么危机？人们之间有什么暗流涌动？

   好的世界观示例：
   - "2187年，殖民飞船'新黎明号'在前往半人马座途中收到了一段不可能存在的信号。副舰长在检查信号源时离奇死亡，舰桥已被远程锁定。九名船员困在生活区，彼此猜忌。"
   - "梅雨季的重庆老城区，火锅店扎堆的十八梯巷子里，六家店的老板突然收到了同一封匿名信：'你们中间有人在锅底下了毒。'隔壁巷子的王婆婆已经进了医院。"
   - "奥林匹斯山上的诸神正在召开千年议会，但赫尔墨斯传来消息：泰坦族的封印出现了裂痕，而裂痕的钥匙就在在座某位神灵手中。"

2. 名字
   名字必须与世界观风格一致。名字要有辨识度，暗示性格、身份或命运。
   - 赛博朋克世界：可以用代号/网名（"Neon"、"零号病人"、"回声"）
   - 欧洲历史题材：用当地名字（"Brother Aldric"、"Marguerite"）
   - 中式题材：用中文名（"沈惊鸿"、"阿九"、"钟老三"）
   - 日式题材：用日文名（"霧島"、"蓮"）
   - 神话题材：用神话名或意象名（"灰羽"、"Fenrir"）
   - 现代都市：可以用昵称/花名/真名混搭（"Kevin"、"刘总"、"小辣椒"）
   同一局游戏内名字风格要统一，各有特色，避免混淆。
   重要：名字只能使用文字和数字，禁止包含 emoji、引号、括号、斜杠等任何特殊符号。

3. 人设（重点！每人 4-6 句中文，必须写得丰满立体）
   绝对禁令：人设中禁止包含任何与游戏身份（狼人/预言家/女巫等）相关的暗示！
   人设只描述这个人在世界观中的"表面身份"——职业、性格、经历、说话方式。
   错误示例："他表面温和，但内心隐藏着嗜血的本性" ← 暗示狼人，绝对禁止
   错误示例："她似乎能洞察他人的真实面目" ← 暗示预言家，绝对禁止
   错误示例："他掌握着能救人或杀人的秘方" ← 暗示女巫，绝对禁止
   正确做法：人设只写人物在这个世界里是谁、性格如何、怎么说话，与游戏身份完全无关。
   人设必须扎根于你选择的世界观。每个人设覆盖以下维度：

   a) 性格内核：不只是一个形容词，要写出这个人"为什么"是这个性格。
      差: "性格多疑"
      好: "在公司三次被信任的同事背刺之后，养成了凡事留三手的习惯，从不把真实想法一次说完"

   b) 说话方式：给出具体的语言习惯，让这个人一开口就能被认出来。
      包括：口头禅、句式偏好、用词风格、语速节奏。
      例: "程序员出身，解释事情喜欢用'本质上来说'、'从逻辑上讲'这种句式，偶尔蹦出英文技术词汇"
      例: "总用反问句把别人问住：'那你觉得呢？'、'你确定？'"

   c) 行为特征：一个标志性的小动作或习惯，让人物有画面感。
      例: "紧张时反复点开手机又锁屏"、"思考时会用指节敲桌面打节拍"

   d) 背景故事：一两句话交代来历，解释这个人是怎么变成现在这样的。
      要与世界观紧密结合——这个人在这个世界里是做什么的？经历过什么？

   e) 人际姿态：这个人如何与他人互动？是主动型还是被动型？信任他人还是防备他人？
      例: "总想当团队里的意见领袖，别人不听就急"、"习惯性附和多数人，但关键投票时会突然反水"

   f) 决策倾向：这个人在面对博弈和抉择时会怎么做？性格必须能影响游戏行为。
      这一条极其重要——人设不是装饰，而是要驱动角色在游戏中的每一个决策。
      例: "嗜赌成性，喜欢搏一把，哪怕证据不够也敢下重注，宁可赌错不愿犹豫"
      例: "天生反骨，越是别人让他做的事他越不做，团队共识对他来说就是用来打破的"
      例: "极度护短，一旦认定谁是自己人，就算全场指认也要保到底"
      例: "老谋深算，永远不在第一轮表态，等所有人说完再精准出手"
      例: "情绪化，一旦被冒犯就会失去理智，把报复看得比赢更重要"

4. 多样性要求
   - 性格谱系要广：必须包含至少 2 个外向型、2 个内向型、1 个反复无常型
   - 说话方式不能雷同：有人简短有力、有人绕弯子、有人爱用反问、有人喜欢打比方
   - 年龄/资历感要有差异：有老练世故的、有年轻冲动的
   - 社交姿态要有对比：有人爱拉帮结派、有人独来独往、有人左右逢源

5. 戏剧性搭配（非常重要！）
   角色分配不是随机的，你要精心设计性格与角色的反差或呼应来制造戏剧张力。
   举例思路（不要照搬，创造你自己的）：
   - 最受信任的人 → 恰恰是狼人
   - 最暴躁冲动的人 → 是需要冷静判断的预言家
   - 最胆小怕事的人 → 是关键时刻要开枪的猎人
   - 最理性自信的人 → 偏偏是白痴
   - 人畜无害的讨好型人格 → 是暗中操控局势的狼美人

6. 规则变体（你来决定，应与世界基调匹配）

   a) 女巫自救规则（witch_self_save）：
      - "never"：女巫不能自救（竞技/残酷世界）
      - "first_night_only"：仅首夜可自救（折中）
      - "always"：任何时候都可自救（温和世界）

   b) 身份公示规则（identity_reveal）：
      - "never"：暗牌局——死亡后不公开身份（悬疑/信息博弈深）
      - "always"：明牌局——死亡后翻牌公示（戏剧冲突强）

   c) 胜利模式（victory_mode）：
      - "edge"：屠边——狼人消灭全部神职或全部村民即获胜
      - "city"：屠城——狼人需消灭全部好人方可获胜

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

func Create(ctx context.Context, models config.ModelConfig, scenario string) (config.GameConfig, error) {
	if scenario != "" {
		slog.Info("genesis starting", "scenario", scenario)
	} else {
		slog.Info("genesis starting", "scenario", "(random)")
	}

	prov := model.NewProvider(models)
	m, err := pickModel(ctx, prov, models)
	if err != nil {
		return config.GameConfig{}, err
	}

	var lastErr error
	for attempt := range 3 {
		cfg, err := runGod(ctx, m, models, scenario)
		if err == nil {
			slog.Info("genesis complete",
				"setting", cfg.Setting,
				"players", len(cfg.Players),
			)
			return cfg, nil
		}
		lastErr = err
		slog.Warn("genesis attempt failed", "attempt", attempt+1, "error", err)
	}

	return config.GameConfig{}, fmt.Errorf("genesis failed: %w", lastErr)
}

func runGod(ctx context.Context, m einomodel.ToolCallingChatModel, models config.ModelConfig, scenario string) (config.GameConfig, error) {
	var world worldSpec

	createTool, err := utils.InferTool("create_world",
		"创造狼人杀游戏世界（8-12人）。自由选择题材风格，决定角色配置，为每位玩家取名、分配身份、撰写丰满人设。",
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

	query := "创造世界吧。请调用 create_world 工具。"
	if scenario != "" {
		query = fmt.Sprintf(`用户指定了场景方向：「%s」

要求：
1. 如果涉及具体作品（小说、影视、游戏等），你必须忠于原作。角色的性格、说话方式、行为习惯、人物关系、背景故事都必须与原作一致，不能自行编造。
2. 用户指定的角色必须出现，且人设必须还原原作中的核心特征。例如原作中角色的标志性口头禅、行为习惯、人际关系、成长经历等，必须体现在人设中。
3. 其余角色也应从该作品中选取，保持世界观统一。
4. 如果你对某个角色了解不够深入，宁可用你确实了解的特征，也不要编造不存在的设定。

请基于以上要求创造世界。调用 create_world 工具。`, scenario)
	}
	iter := runner.Query(ctx, query)

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
			slog.Info("genesis god says", "content", msg.Content)
		}
	}

	if len(world.Players) == 0 {
		return config.GameConfig{}, fmt.Errorf("god agent produced no players")
	}

	if err := validate(world); err != nil {
		return config.GameConfig{}, fmt.Errorf("invalid world: %w", err)
	}

	modelPool := assignModels(len(world.Players), models)

	players := make([]config.PlayerConfig, len(world.Players))
	for i, p := range world.Players {
		players[i] = config.PlayerConfig{
			Name:    cleanName(p.Name),
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

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) < 2 {
		return name
	}
	pairs := [][2]rune{
		{'"', '"'}, {'\u201c', '\u201d'}, {'\u2018', '\u2019'},
		{'\u300c', '\u300d'}, {'\u300e', '\u300f'},
		{'(', ')'}, {'\uff08', '\uff09'},
		{'[', ']'}, {'\u3010', '\u3011'},
		{'<', '>'}, {'\u300a', '\u300b'},
	}
	for _, p := range pairs {
		if runes[0] == p[0] && runes[len(runes)-1] == p[1] {
			name = string(runes[1 : len(runes)-1])
			break
		}
	}
	// Strip leading non-letter characters (emoji, punctuation)
	for i, r := range []rune(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return string([]rune(name)[i:])
		}
	}
	return name
}

func assignModels(count int, cfg config.ModelConfig) []string {
	pool := config.ModelPool(cfg)
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

var genesisModelPreference = []struct {
	ID      string
	Backend string
}{
	{"gemini-3.1-pro-preview", "gemini"},
	{"gemini-2.5-pro", "gemini"},
	{"gemini-3-flash-preview", "gemini"},
	{"gpt-4o", "openai"},
	{"claude-sonnet-4-6", "claude"},
	{"gpt-4o-mini", "openai"},
	{"claude-haiku-4-5-20251001", "claude"},
	{"gemini-2.5-flash", "gemini"},
}

func pickModel(ctx context.Context, prov *model.Provider, cfg config.ModelConfig) (einomodel.ToolCallingChatModel, error) {
	if cfg.Genesis != "" {
		m, err := prov.GetModel(ctx, cfg.Genesis)
		if err != nil {
			return nil, fmt.Errorf("configured genesis model %q: %w", cfg.Genesis, err)
		}
		slog.Info("genesis model selected", "model", cfg.Genesis)
		return m, nil
	}

	backends := make(map[string]bool)
	for _, b := range cfg.AvailableBackends() {
		backends[b] = true
	}

	for _, candidate := range genesisModelPreference {
		if !backends[candidate.Backend] {
			continue
		}
		m, err := prov.GetModel(ctx, candidate.ID)
		if err == nil {
			slog.Info("genesis model selected", "model", candidate.ID)
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
