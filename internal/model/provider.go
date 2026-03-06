package model

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"

	"github.com/alaric/eino-learn/internal/config"
)

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

func (p *Provider) GetModel(ctx context.Context, modelID string) (einomodel.ToolCallingChatModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m, ok := p.cache[modelID]; ok {
		return m, nil
	}

	var m einomodel.ToolCallingChatModel
	var err error

	switch config.ModelBackend(modelID) {
	case "claude":
		m, err = p.createClaude(ctx, modelID)
	case "gemini":
		m, err = p.createGemini(ctx, modelID)
	case "openai":
		m, err = p.createOpenAI(ctx, modelID)
	default:
		return nil, fmt.Errorf("unknown model backend for %q", modelID)
	}

	if err != nil {
		return nil, fmt.Errorf("creating model %q: %w", modelID, err)
	}

	p.cache[modelID] = m
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

func (p *Provider) createOpenAI(ctx context.Context, model string) (einomodel.ToolCallingChatModel, error) {
	if p.cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for model %q", model)
	}
	temp := float32(0.7)
	maxTokens := 1024
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      p.cfg.OpenAIAPIKey,
		Model:       model,
		MaxTokens:   &maxTokens,
		Temperature: &temp,
	})
}
