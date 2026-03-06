package role

import "github.com/alaric/eino-learn/internal/config"

type knightRole struct{}

func init() { Register(&knightRole{}) }

func (r *knightRole) Name() string        { return "knight" }
func (r *knightRole) Team() config.Team   { return config.TeamVillager }
func (r *knightRole) HasNightAction() bool { return false }

func (r *knightRole) Description() string {
	return "你是骑士。你可以在白天发言阶段主动翻牌，选择一名玩家发起决斗。" +
		"若决斗目标是狼人，该狼人立即出局，白天结束进入夜晚。" +
		"若决斗目标是好人，你自己立即出局，白天继续正常流程。" +
		"决斗技能仅能使用一次，不可在警长竞选、PK发言或遗言阶段使用。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
