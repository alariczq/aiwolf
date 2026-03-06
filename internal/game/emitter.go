package game

import (
	"fmt"
	"time"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/player"
)

type UIEvent struct {
	Type    string         `json:"type"`
	Round   int            `json:"round,omitempty"`
	Phase   string         `json:"phase,omitempty"`
	Player  string         `json:"player,omitempty"`
	ModelID string         `json:"model_id,omitempty"`
	Content string         `json:"content,omitempty"`
	Target  string         `json:"target,omitempty"`
	Action  string         `json:"action,omitempty"`
	Result  string         `json:"result,omitempty"`
	Role    string         `json:"role,omitempty"`
	Winner  string         `json:"winner,omitempty"`
	Tied    bool           `json:"tied,omitempty"`
	Players []UIPlayer     `json:"players,omitempty"`
	Tally   map[string]int `json:"tally,omitempty"`
}

type UIPlayer struct {
	Name    string `json:"name"`
	ModelID string `json:"model_id"`
	Display string `json:"display"`
	Role    string `json:"role"`
	Alive   bool   `json:"alive"`
	Persona string `json:"persona,omitempty"`
}

type EventEmitter func(UIEvent)

type EngineOption func(*Engine)

func WithEmitter(emit EventEmitter) EngineOption {
	return func(e *Engine) {
		e.emit = emit
	}
}

func WithSilent() EngineOption {
	return func(e *Engine) {
		e.silent = true
	}
}

func WithCallInterval(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.callInterval = d
	}
}

func (e *Engine) emitEvent(event UIEvent) {
	if event.Type == "thought" || event.Type == "thinking_start" {
		return
	}
	if event.Player != "" && e.state != nil {
		if p := e.state.GetPlayer(event.Player); p != nil {
			if event.ModelID == "" {
				event.ModelID = p.ModelID
			}
			if event.Role == "" {
				event.Role = p.Role.Name()
			}
		}
	}
	if e.emit != nil {
		e.emit(event)
	}
}

func (e *Engine) printf(format string, args ...any) {
	if !e.silent {
		fmt.Printf(format, args...)
	}
}

func (e *Engine) println(args ...any) {
	if !e.silent {
		fmt.Println(args...)
	}
}

func buildUIPlayers(players []*player.Player, revealRoles bool) []UIPlayer {
	result := make([]UIPlayer, len(players))
	for i, p := range players {
		role := ""
		if revealRoles {
			role = p.Role.Name()
		}
		result[i] = UIPlayer{
			Name:    p.Name,
			ModelID: p.ModelID,
			Display: fmt.Sprintf("%s (%s)", p.Name, config.DisplayName(p.ModelID)),
			Role:    role,
			Alive:   p.Alive,
			Persona: p.Persona,
		}
	}
	return result
}
