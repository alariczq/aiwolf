package player

import (
	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/alaric/eino-learn/internal/role"
)

type Player struct {
	Name    string
	Role    role.Role
	ModelID string
	Model   einomodel.ToolCallingChatModel
	Alive   bool
	Persona string
}

func New(name string, r role.Role, modelID string, m einomodel.ToolCallingChatModel, persona string) *Player {
	return &Player{
		Name:    name,
		Role:    r,
		ModelID: modelID,
		Model:   m,
		Alive:   true,
		Persona: persona,
	}
}
