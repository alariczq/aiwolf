package action

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SpeakRequest struct {
	Content string `json:"content" desc:"你的发言内容。分享你的观察、怀疑或论点来影响投票。"`
}

func CreateSpeakTool(result *string) (tool.InvokableTool, error) {
	fn := func(ctx context.Context, req SpeakRequest) (string, error) {
		*result = req.Content
		return "你的发言已记录。", nil
	}

	return utils.InferTool("speak", "在白天讨论阶段发言。分享你的想法、怀疑或辩护来影响其他玩家。", fn)
}
