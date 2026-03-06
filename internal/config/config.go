package config

import (
	"os"
)

type Team int

const (
	TeamVillager Team = iota
	TeamWerewolf
)

func (t Team) String() string {
	switch t {
	case TeamVillager:
		return "好人阵营"
	case TeamWerewolf:
		return "狼人阵营"
	default:
		return "未知"
	}
}

type PlayerConfig struct {
	Name    string
	Role    string
	ModelID string
	Persona string
}

type ModelConfig struct {
	ClaudeAPIKey string
	GeminiAPIKey string
}

type WitchSelfSave int

const (
	WitchSelfSaveNever     WitchSelfSave = iota
	WitchSelfSaveFirstOnly
	WitchSelfSaveAlways
)

type IdentityReveal int

const (
	IdentityRevealNever  IdentityReveal = iota
	IdentityRevealAlways
)

type VictoryMode int

const (
	VictoryModeEdge VictoryMode = iota // default: wolves win if all gods OR all villagers dead
	VictoryModeCity                    // wolves win only if all gods AND all villagers dead
)

type GameConfig struct {
	Players        []PlayerConfig
	Models         ModelConfig
	Setting        string
	WitchSelfSave  WitchSelfSave
	IdentityReveal IdentityReveal
	VictoryMode    VictoryMode
}

func ModelConfigFromEnv() ModelConfig {
	return ModelConfig{
		ClaudeAPIKey: os.Getenv("CLAUDE_API_KEY"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
	}
}

var defaultModels = []string{
	"claude-opus",
	"gemini-pro-preview",
	"claude-sonnet",
	"claude-sonnet",
	"gemini-pro",
	"gemini-pro",
	"claude-haiku",
	"claude-haiku",
	"claude-haiku",
	"gemini-flash",
	"gemini-flash",
	"gemini-flash",
}

var modelAliasDisplay = map[string]string{
	"claude-haiku":      "Claude-Haiku-4.5",
	"claude-sonnet":     "Claude-Sonnet-4.6",
	"claude-opus":       "Claude-Opus-4.6",
	"gemini-flash":      "Gemini-2.5-Flash",
	"gemini-pro":        "Gemini-2.5-Pro",
	"gemini-pro-preview": "Gemini-3.1-Pro",
}

func DisplayName(modelID string) string {
	if name, ok := modelAliasDisplay[modelID]; ok {
		return name
	}
	return modelID
}

func DefaultModelPool() []string {
	pool := make([]string, len(defaultModels))
	copy(pool, defaultModels)
	return pool
}

