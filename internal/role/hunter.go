package role

import "github.com/alaric/eino-learn/internal/config"

type hunterRole struct{}

func init() { Register(&hunterRole{}) }

func (r *hunterRole) Name() string        { return "hunter" }
func (r *hunterRole) Team() config.Team   { return config.TeamVillager }
func (r *hunterRole) HasNightAction() bool { return false }

func (r *hunterRole) Description() string {
	return "你是猎人。你没有主动的夜间能力，但你拥有强大的被动技能：" +
		"当你被狼人杀害或被投票淘汰时，你可以开枪带走一名玩家。" +
		"但如果你是被女巫毒杀的，则不能开枪。" +
		"请谨慎选择开枪目标，确保带走的是狼人而非好人。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
