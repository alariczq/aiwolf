package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
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
	OpenAIAPIKey string
	Pool         []string
	Genesis      string
}

func (m ModelConfig) AvailableBackends() []string {
	var backends []string
	if m.ClaudeAPIKey != "" {
		backends = append(backends, "claude")
	}
	if m.GeminiAPIKey != "" {
		backends = append(backends, "gemini")
	}
	if m.OpenAIAPIKey != "" {
		backends = append(backends, "openai")
	}
	return backends
}

type AppConfig struct {
	Models ModelConfig
	Port   int
}

func Load() AppConfig {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")

	v.SetDefault("server.port", 8080)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("[config] error reading config.toml: %v", err)
		}
	}

	cfg := AppConfig{
		Port: v.GetInt("server.port"),
		Models: ModelConfig{
			ClaudeAPIKey: firstNonEmpty(v.GetString("keys.claude"), os.Getenv("CLAUDE_API_KEY")),
			GeminiAPIKey: firstNonEmpty(v.GetString("keys.gemini"), os.Getenv("GEMINI_API_KEY")),
			OpenAIAPIKey: firstNonEmpty(v.GetString("keys.openai"), os.Getenv("OPENAI_API_KEY")),
			Pool:         v.GetStringSlice("models.pool"),
			Genesis:      v.GetString("models.genesis"),
		},
	}

	if pool := cfg.Models.Pool; len(pool) > 0 {
		backends := make(map[string]bool)
		for _, b := range cfg.Models.AvailableBackends() {
			backends[b] = true
		}
		for _, id := range pool {
			b := ModelBackend(id)
			if b == "" {
				log.Fatalf("[config] unknown model in pool: %q", id)
			}
			if !backends[b] {
				log.Fatalf("[config] model %q requires %s key, but it is not configured", id, strings.ToUpper(b))
			}
		}
	}

	if g := cfg.Models.Genesis; g != "" {
		backends := make(map[string]bool)
		for _, b := range cfg.Models.AvailableBackends() {
			backends[b] = true
		}
		b := ModelBackend(g)
		if b == "" {
			log.Fatalf("[config] unknown genesis model: %q", g)
		}
		if !backends[b] {
			log.Fatalf("[config] genesis model %q requires %s key, but it is not configured", g, strings.ToUpper(b))
		}
	}

	return cfg
}

func (c AppConfig) Validate() error {
	m := c.Models
	if m.ClaudeAPIKey == "" && m.GeminiAPIKey == "" && m.OpenAIAPIKey == "" {
		return fmt.Errorf("at least one API key is required (keys.claude / keys.gemini / keys.openai or env vars)")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func ModelBackend(modelID string) string {
	if strings.HasPrefix(modelID, "claude-") {
		return "claude"
	}
	if strings.HasPrefix(modelID, "gemini-") {
		return "gemini"
	}
	if strings.HasPrefix(modelID, "gpt-") ||
		strings.HasPrefix(modelID, "o1") ||
		strings.HasPrefix(modelID, "o3") ||
		strings.HasPrefix(modelID, "o4") {
		return "openai"
	}
	return ""
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
	VictoryModeEdge VictoryMode = iota
	VictoryModeCity
)

type GameConfig struct {
	Players        []PlayerConfig
	Models         ModelConfig
	Setting        string
	WitchSelfSave  WitchSelfSave
	IdentityReveal IdentityReveal
	VictoryMode    VictoryMode
}

type modelTier struct {
	ID      string
	Backend string
	Weight  int
}

var allModels = []modelTier{
	{"claude-opus-4-6", "claude", 1},
	{"o3", "openai", 1},
	{"gemini-3.1-pro-preview", "gemini", 1},
	{"claude-sonnet-4-6", "claude", 2},
	{"gpt-4o", "openai", 2},
	{"gemini-2.5-pro", "gemini", 2},
	{"gemini-3-flash-preview", "gemini", 2},
	{"claude-haiku-4-5-20251001", "claude", 3},
	{"gpt-4o-mini", "openai", 3},
	{"gemini-2.5-flash", "gemini", 3},
}

var modelDisplayNames = map[string]string{
	"claude-haiku-4-5-20251001": "Claude Haiku 4.5",
	"claude-sonnet-4-6":         "Claude Sonnet 4.6",
	"claude-opus-4-6":           "Claude Opus 4.6",
	"gemini-2.5-flash":          "Gemini 2.5 Flash",
	"gemini-2.5-flash-lite":     "Gemini 2.5 Flash Lite",
	"gemini-2.5-pro":            "Gemini 2.5 Pro",
	"gemini-3-flash-preview":    "Gemini 3 Flash",
	"gemini-3.1-pro-preview":    "Gemini 3.1 Pro",
	"o3":                        "OpenAI o3",
	"gpt-4o":                    "GPT-4o",
	"gpt-4o-mini":               "GPT-4o mini",
}

func DisplayName(modelID string) string {
	if name, ok := modelDisplayNames[modelID]; ok {
		return name
	}
	return modelID
}

func ModelPool(cfg ModelConfig) []string {
	if len(cfg.Pool) > 0 {
		return cfg.Pool
	}

	backends := make(map[string]bool)
	for _, b := range cfg.AvailableBackends() {
		backends[b] = true
	}

	var available []modelTier
	for _, m := range allModels {
		if backends[m.Backend] {
			available = append(available, m)
		}
	}

	if len(available) == 0 {
		return nil
	}

	var pool []string
	for _, m := range available {
		pool = append(pool, m.ID)
		if m.Weight >= 2 {
			pool = append(pool, m.ID)
		}
		if m.Weight >= 3 {
			pool = append(pool, m.ID)
		}
	}
	return pool
}
