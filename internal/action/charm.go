package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type CharmRequest struct {
	Target string `json:"target" desc:"今晚要魅惑的玩家名字。必须是存活玩家之一。"`
}

func CreateCharmTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("选择今晚的魅惑目标。你死亡时被魅惑者殉情。可选目标: %s", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req CharmRequest) (string, error) {
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你选择今晚魅惑 %s。", req.Target), nil
	}

	return utils.InferTool("charm", desc, fn)
}
