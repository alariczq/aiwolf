package role

import "github.com/alaric/eino-learn/internal/config"

type guardRole struct{}

func init() { Register(&guardRole{}) }

func (r *guardRole) Name() string        { return "guard" }
func (r *guardRole) Team() config.Team   { return config.TeamVillager }
func (r *guardRole) HasNightAction() bool { return true }

func (r *guardRole) Description() string {
	return "你是守卫。每晚你可以选择一名存活玩家进行守护，被守护者当晚不会被狼人刀杀。" +
		"你可以守护自己，也可以选择空守。但不能连续两晚守护同一名玩家。" +
		"守护只能抵挡狼刀，不能抵挡女巫毒药。" +
		"注意同守同救规则：如果你守护的人同时被女巫解药救了，两种保护抵消，该玩家反而会死亡。" +
		"当所有狼人被淘汰时，好人阵营获胜。"
}
