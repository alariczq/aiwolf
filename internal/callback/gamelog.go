package callback

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	modelcomp "github.com/cloudwego/eino/components/model"
)

type ModelStats struct {
	Calls        int
	InputTokens  int64
	OutputTokens int64
}

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
			if info.Component != components.ComponentOfChatModel {
				return ctx
			}
			fmt.Printf("[game-log] ChatModel start  node=%q\n", info.Name)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info.Component != components.ComponentOfChatModel {
				return ctx
			}

			var inputTokens, outputTokens int64
			if out := modelcomp.ConvCallbackOutput(output); out != nil && out.TokenUsage != nil {
				inputTokens = int64(out.TokenUsage.PromptTokens)
				outputTokens = int64(out.TokenUsage.CompletionTokens)
				fmt.Printf("[game-log] ChatModel end    node=%q input_tokens=%d output_tokens=%d total=%d\n",
					info.Name, inputTokens, outputTokens, out.TokenUsage.TotalTokens)
			} else {
				fmt.Printf("[game-log] ChatModel end    node=%q (no token usage available)\n", info.Name)
			}

			gl.mu.Lock()
			s := gl.stats[info.Name]
			if s == nil {
				s = &ModelStats{}
				gl.stats[info.Name] = s
			}
			s.Calls++
			s.InputTokens += inputTokens
			s.OutputTokens += outputTokens
			gl.mu.Unlock()

			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if info.Component != components.ComponentOfChatModel {
				return ctx
			}
			fmt.Printf("[game-log] ChatModel error  node=%q error=%v\n", info.Name, err)
			return ctx
		}).
		Build()
}

func (gl *GameLogger) PrintStats() {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if len(gl.stats) == 0 {
		fmt.Println("[game-log] No model stats recorded.")
		return
	}

	fmt.Println("[game-log] === Model Usage Summary ===")
	for node, s := range gl.stats {
		fmt.Printf("[game-log]   node=%-20q  calls=%d  input_tokens=%d  output_tokens=%d\n",
			node, s.Calls, s.InputTokens, s.OutputTokens)
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
