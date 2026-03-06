package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type WolfKingExplodeRequest struct {
	Confirm bool   `json:"confirm" desc:"设为 true 确认自爆并带人。自爆后你立即出局，同时带走目标玩家，白天阶段结束。"`
	Target  string `json:"target" desc:"自爆时要带走的玩家名字。confirm 为 true 时必填。"`
}

func CreateWolfKingSelfExplodeTool(playerName string, targets []string, exploded *bool, takeTarget *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("白狼王自爆带人：自爆后立即出局，同时带走一名玩家。白天阶段结束。可选带走目标: %s", strings.Join(targets, ", "))

	valid := make(map[string]bool, len(targets))
	for _, n := range targets {
		valid[n] = true
	}

	fn := func(ctx context.Context, req WolfKingExplodeRequest) (string, error) {
		if !req.Confirm {
			*exploded = false
			return "你决定不自爆。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(targets, ", ")), nil
		}
		*exploded = true
		*takeTarget = req.Target
		return fmt.Sprintf("%s 自爆并带走了 %s！", playerName, req.Target), nil
	}

	return utils.InferTool("wolf_king_explode", desc, fn)
}
