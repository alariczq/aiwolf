package role

import "github.com/alaric/eino-learn/internal/config"

type villagerRole struct{}

func init() { Register(&villagerRole{}) }

func (r *villagerRole) Name() string        { return "villager" }
func (r *villagerRole) Team() config.Team   { return config.TeamVillager }
func (r *villagerRole) HasNightAction() bool { return false }

func (r *villagerRole) Description() string {
	return "你是一名普通村民。你没有特殊能力，但你的投票和推理至关重要。" +
		"白天仔细分析其他玩家的发言，找出狼人。" +
		"留意前后矛盾、转移话题和可疑的投票模式。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
