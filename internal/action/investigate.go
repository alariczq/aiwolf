package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type InvestigateRequest struct {
	Target string `json:"target" desc:"要查验的玩家名字。必须是尚未查验过的存活玩家之一。"`
}

func CreateInvestigateTool(alivePlayers []string, alreadyChecked []string, result *string) (tool.InvokableTool, error) {
	checked := make(map[string]bool, len(alreadyChecked))
	for _, n := range alreadyChecked {
		checked[n] = true
	}

	valid := make(map[string]bool, len(alivePlayers))
	available := make([]string, 0, len(alivePlayers))
	for _, n := range alivePlayers {
		if !checked[n] {
			valid[n] = true
			available = append(available, n)
		}
	}

	desc := fmt.Sprintf("选择一名玩家进行查验，得知其身份是好人还是狼人。可查验的玩家: %s", strings.Join(available, ", "))

	fn := func(ctx context.Context, req InvestigateRequest) (string, error) {
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(available, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你选择查验 %s。", req.Target), nil
	}

	return utils.InferTool("investigate", desc, fn)
}
