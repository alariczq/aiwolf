package role

import "github.com/alaric/eino-learn/internal/config"

type werewolfRole struct{}

func init() { Register(&werewolfRole{}) }

func (r *werewolfRole) Name() string        { return "werewolf" }
func (r *werewolfRole) Team() config.Team   { return config.TeamWerewolf }
func (r *werewolfRole) HasNightAction() bool { return true }

func (r *werewolfRole) Description() string {
	return "你是一名狼人。每晚你和狼队友秘密选择一名玩家击杀。" +
		"白天你必须伪装成无辜的好人，避免被怀疑。" +
		"与队友配合，逐步淘汰好人阵营的玩家。" +
		"胜利条件（屠边）：所有神职角色被消灭（屠神）或所有平民被消灭（屠民），狼人获胜。"
}
