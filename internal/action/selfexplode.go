package action

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SelfExplodeRequest struct {
	Confirm bool `json:"confirm" desc:"设为 true 确认自爆。自爆后你立即出局，白天阶段结束，不进行投票。"`
}

func CreateSelfExplodeTool(playerName string, result *bool) (tool.InvokableTool, error) {
	fn := func(ctx context.Context, req SelfExplodeRequest) (string, error) {
		if req.Confirm {
			*result = true
			return playerName + " 选择自爆！白天阶段立即结束。", nil
		}
		*result = false
		return "你决定不自爆。", nil
	}

	return utils.InferTool("self_explode",
		"狼人可以在白天自爆，立即结束白天阶段，不进行投票。这是孤注一掷的操作：你会出局，但本轮不会投票淘汰任何人。仅在极端情况下使用。",
		fn)
}
