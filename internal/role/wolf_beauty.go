package role

import "github.com/alaric/eino-learn/internal/config"

type wolfBeautyRole struct{}

func init() { Register(&wolfBeautyRole{}) }

func (r *wolfBeautyRole) Name() string        { return "wolf_beauty" }
func (r *wolfBeautyRole) Team() config.Team   { return config.TeamWerewolf }
func (r *wolfBeautyRole) HasNightAction() bool { return true }

func (r *wolfBeautyRole) Description() string {
	return "你是狼美人，属于狼人阵营。夜间你与其他狼人一起行动后，单独睁眼选择一名玩家进行魅惑。" +
		"魅惑效果：当你死亡时，被魅惑的玩家也随之殉情出局。" +
		"你不能被狼队自刀（狼人指刀时不能选你），你也不能自爆。" +
		"殉情的猎人不能开枪，殉情的白痴不能翻牌。" +
		"若你被骑士决斗出局，魅惑效果不触发。" +
		"与队友配合，利用魅惑能力制造更多伤亡。"
}
