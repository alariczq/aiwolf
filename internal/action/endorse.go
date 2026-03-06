package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type EndorseRequest struct {
	Target string `json:"target" desc:"你要归票的玩家名字（该玩家 +0.5 票）。不归票时留空。"`
	Skip   bool   `json:"skip" desc:"设为 true 表示不归票（不给任何人 +0.5）。"`
}

func CreateEndorseTool(candidates []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("作为警长，你可以归票给一名玩家（+0.5 票权重）。可选目标: %s。设 skip=true 表示不归票。", strings.Join(candidates, ", "))

	valid := make(map[string]bool, len(candidates))
	for _, n := range candidates {
		valid[n] = true
	}

	fn := func(ctx context.Context, req EndorseRequest) (string, error) {
		if req.Skip || req.Target == "" {
			*result = ""
			return "你选择不归票。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(candidates, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你归票给了 %s（+0.5 票）。", req.Target), nil
	}

	return utils.InferTool("endorse", desc, fn)
}
