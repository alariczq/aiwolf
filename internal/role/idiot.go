package role

import "github.com/alaric/eino-learn/internal/config"

type idiotRole struct{}

func init() { Register(&idiotRole{}) }

func (r *idiotRole) Name() string        { return "idiot" }
func (r *idiotRole) Team() config.Team   { return config.TeamVillager }
func (r *idiotRole) HasNightAction() bool { return false }

func (r *idiotRole) Description() string {
	return "你是白痴。你没有夜间能力，但你有独特的被动技能：" +
		"当你被投票淘汰时，你可以翻牌亮出身份免死，但之后你失去投票权。" +
		"你仍然可以参与白天讨论发言，只是不能投票。" +
		"如果你被狼人在夜间杀害，你正常死亡，无法使用技能。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
