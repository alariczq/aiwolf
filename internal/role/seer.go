package role

import "github.com/alaric/eino-learn/internal/config"

type seerRole struct{}

func init() { Register(&seerRole{}) }

func (r *seerRole) Name() string        { return "seer" }
func (r *seerRole) Team() config.Team   { return config.TeamVillager }
func (r *seerRole) HasNightAction() bool { return true }

func (r *seerRole) Description() string {
	return "你是预言家。每晚你可以查验一名玩家，得知对方是好人还是狼人。" +
		"在白天讨论中明智地运用你的查验信息。" +
		"注意不要过早暴露身份，否则狼人会优先针对你。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
