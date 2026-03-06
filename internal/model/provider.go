package model

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	einomodel "github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"

	"github.com/alaric/eino-learn/internal/config"
)

var modelAliases = map[string]string{
	"claude-haiku":       "claude-haiku-4-5-20251001",
	"claude-sonnet":      "claude-sonnet-4-6",
	"claude-opus":        "claude-opus-4-6",
	"gemini-flash":       "gemini-2.5-flash",
	"gemini-pro":         "gemini-2.5-pro",
	"gemini-pro-preview": "gemini-3.1-pro-preview",
}

type Provider struct {
	cfg   config.ModelConfig
	mu    sync.Mutex
	cache map[string]einomodel.ToolCallingChatModel

	geminiClient *genai.Client
}

func NewProvider(cfg config.ModelConfig) *Provider {
	return &Provider{
		cfg:   cfg,
		cache: make(map[string]einomodel.ToolCallingChatModel),
	}
}

func (p *Provider) ResolveAlias(modelID string) string {
	if resolved, ok := modelAliases[modelID]; ok {
		return resolved
	}
	return modelID
}

func (p *Provider) GetModel(ctx context.Context, modelID string) (einomodel.ToolCallingChatModel, error) {
	resolved := p.ResolveAlias(modelID)

	p.mu.Lock()
	defer p.mu.Unlock()

	if m, ok := p.cache[resolved]; ok {
		return m, nil
	}

	var m einomodel.ToolCallingChatModel
	var err error

	switch {
	case strings.HasPrefix(resolved, "claude-"):
		m, err = p.createClaude(ctx, resolved)
	case strings.HasPrefix(resolved, "gemini-"):
		m, err = p.createGemini(ctx, resolved)
	default:
		return nil, fmt.Errorf("unknown model backend for %q (resolved from %q)", resolved, modelID)
	}

	if err != nil {
		return nil, fmt.Errorf("creating model %q: %w", resolved, err)
	}

	p.cache[resolved] = m
	return m, nil
}

func (p *Provider) createClaude(ctx context.Context, model string) (einomodel.ToolCallingChatModel, error) {
	if p.cfg.ClaudeAPIKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY is required for model %q", model)
	}
	temp := float32(0.7)
	return claude.NewChatModel(ctx, &claude.Config{
		APIKey:      p.cfg.ClaudeAPIKey,
		Model:       model,
		MaxTokens:   1024,
		Temperature: &temp,
	})
}

func (p *Provider) createGemini(ctx context.Context, model string) (einomodel.ToolCallingChatModel, error) {
	if p.cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required for model %q", model)
	}

	if p.geminiClient == nil {
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  p.cfg.GeminiAPIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("creating gemini client: %w", err)
		}
		p.geminiClient = client
	}

	temp := float32(0.7)
	return gemini.NewChatModel(ctx, &gemini.Config{
		Client:      p.geminiClient,
		Model:       model,
		Temperature: &temp,
	})
}
