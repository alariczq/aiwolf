package role

import "github.com/alaric/eino-learn/internal/config"

type wolfKingRole struct{}

func init() { Register(&wolfKingRole{}) }

func (r *wolfKingRole) Name() string        { return "wolf_king" }
func (r *wolfKingRole) Team() config.Team   { return config.TeamWerewolf }
func (r *wolfKingRole) HasNightAction() bool { return true }

func (r *wolfKingRole) Description() string {
	return "你是白狼王，属于狼人阵营。夜间你与其他狼人一起行动（睁眼、指刀）。" +
		"你拥有独特的主动技能：在白天自爆时，可以同时带走一名玩家。" +
		"仅在自爆时才能发动带人技能。被投票放逐或被刀等其他方式出局时不能带人。" +
		"如果你是场上最后一只狼，自爆时不能发动带人技能。" +
		"如果你昨晚被女巫毒药标记（次日进入死亡名单），则不能发动自爆带人。" +
		"与队友配合，逐步淘汰好人阵营的玩家。"
}
