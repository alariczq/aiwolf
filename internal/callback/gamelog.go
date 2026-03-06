package callback

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	modelcomp "github.com/cloudwego/eino/components/model"
	toolcomp "github.com/cloudwego/eino/components/tool"
)

type ModelStats struct {
	Calls        int
	InputTokens  int64
	OutputTokens int64
}

type callTimer struct {
	start time.Time
}

type ctxKey struct{}

type GameLogger struct {
	mu    sync.Mutex
	stats map[string]*ModelStats
}

func NewGameLogger() *GameLogger {
	return &GameLogger{
		stats: make(map[string]*ModelStats),
	}
}

func (gl *GameLogger) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			ctx = context.WithValue(ctx, ctxKey{}, &callTimer{start: time.Now()})

			switch info.Component {
			case components.ComponentOfChatModel:
				slog.Info("llm call start",
					"component", "ChatModel",
					"node", info.Name,
				)
			case components.ComponentOfTool:
				args := ""
				if ti := toolcomp.ConvCallbackInput(input); ti != nil {
					args = truncateUTF8(ti.ArgumentsInJSON, 200)
				}
				slog.Info("tool call start",
					"component", "Tool",
					"node", info.Name,
					"args", args,
				)
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			elapsed := elapsedMs(ctx)

			switch info.Component {
			case components.ComponentOfChatModel:
				var inputTokens, outputTokens, totalTokens int
				if out := modelcomp.ConvCallbackOutput(output); out != nil && out.TokenUsage != nil {
					inputTokens = out.TokenUsage.PromptTokens
					outputTokens = out.TokenUsage.CompletionTokens
					totalTokens = out.TokenUsage.TotalTokens
				}
				slog.Info("llm call end",
					"component", "ChatModel",
					"node", info.Name,
					"input_tokens", inputTokens,
					"output_tokens", outputTokens,
					"total_tokens", totalTokens,
					"elapsed_ms", elapsed,
				)

				gl.mu.Lock()
				s := gl.stats[info.Name]
				if s == nil {
					s = &ModelStats{}
					gl.stats[info.Name] = s
				}
				s.Calls++
				s.InputTokens += int64(inputTokens)
				s.OutputTokens += int64(outputTokens)
				gl.mu.Unlock()

			case components.ComponentOfTool:
				resp := ""
				if to := toolcomp.ConvCallbackOutput(output); to != nil {
					resp = truncateUTF8(to.Response, 200)
				}
				slog.Info("tool call end",
					"component", "Tool",
					"node", info.Name,
					"response", resp,
					"elapsed_ms", elapsed,
				)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			elapsed := elapsedMs(ctx)
			slog.Error("callback error",
				"component", string(info.Component),
				"node", info.Name,
				"error", err,
				"elapsed_ms", elapsed,
			)
			return ctx
		}).
		Build()
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes] + "..."
}

func elapsedMs(ctx context.Context) int64 {
	if t, ok := ctx.Value(ctxKey{}).(*callTimer); ok {
		return time.Since(t.start).Milliseconds()
	}
	return -1
}

func (gl *GameLogger) PrintStats() {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if len(gl.stats) == 0 {
		slog.Info("no model stats recorded")
		return
	}

	slog.Info("model usage summary")
	for node, s := range gl.stats {
		slog.Info("model stats",
			"node", node,
			"calls", s.Calls,
			"input_tokens", s.InputTokens,
			"output_tokens", s.OutputTokens,
		)
	}
}

func (gl *GameLogger) GetStats() map[string]*ModelStats {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	out := make(map[string]*ModelStats, len(gl.stats))
	for k, v := range gl.stats {
		cp := *v
		out[k] = &cp
	}
	return out
}
