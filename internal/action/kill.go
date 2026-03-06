package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type KillRequest struct {
	Skip   bool   `json:"skip" desc:"设为 true 表示今晚空刀（不杀人），false 表示选择目标。"`
	Target string `json:"target" desc:"今晚要击杀的玩家名字。skip 为 false 时必填。必须是存活玩家之一。"`
}

func CreateKillTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("选择今晚的击杀目标，或空刀不杀。可选目标: %s", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req KillRequest) (string, error) {
		if req.Skip {
			*result = ""
			return "你选择了今晚空刀。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你选择今晚击杀 %s。", req.Target), nil
	}

	return utils.InferTool("kill", desc, fn)
}
