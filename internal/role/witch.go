package role

import "github.com/alaric/eino-learn/internal/config"

type witchRole struct{}

func init() { Register(&witchRole{}) }

func (r *witchRole) Name() string        { return "witch" }
func (r *witchRole) Team() config.Team   { return config.TeamVillager }
func (r *witchRole) HasNightAction() bool { return true }

func (r *witchRole) Description() string {
	return "你是女巫。你拥有两瓶一次性药水：解药可以救活当晚被狼人杀害的玩家，" +
		"毒药可以毒杀任意一名玩家。每种药水整局游戏只能使用一次。" +
		"请谨慎使用，在关键时刻发挥最大价值。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
