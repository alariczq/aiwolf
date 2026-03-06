package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GuardRequest struct {
	Skip   bool   `json:"skip" desc:"设为 true 表示今晚空守（不守护任何人），false 表示选择守护目标。"`
	Target string `json:"target" desc:"今晚要守护的玩家名字。skip 为 false 时必填。必须是可守护的存活玩家之一。"`
}

func CreateGuardTool(alivePlayers []string, lastTarget string, result *string) (tool.InvokableTool, error) {
	var validTargets []string
	for _, n := range alivePlayers {
		if n != lastTarget {
			validTargets = append(validTargets, n)
		}
	}

	restriction := ""
	if lastTarget != "" {
		restriction = fmt.Sprintf(" 你昨晚守护了 %s，今晚不能守护同一人。", lastTarget)
	}
	desc := fmt.Sprintf("选择今晚的守护目标，或空守不守。可选目标: %s。%s", strings.Join(validTargets, ", "), restriction)

	valid := make(map[string]bool, len(validTargets))
	for _, n := range validTargets {
		valid[n] = true
	}

	fn := func(ctx context.Context, req GuardRequest) (string, error) {
		if req.Skip {
			*result = ""
			return "你选择今晚空守。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(validTargets, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你选择今晚守护 %s。", req.Target), nil
	}

	return utils.InferTool("guard", desc, fn)
}
