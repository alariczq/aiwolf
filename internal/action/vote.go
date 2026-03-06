package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type VoteRequest struct {
	Target string `json:"target" desc:"你投票淘汰的玩家名字。必须是候选人之一。"`
}

func CreateVoteTool(candidates []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("投票淘汰一名玩家。候选人: %s", strings.Join(candidates, ", "))

	valid := make(map[string]bool, len(candidates))
	for _, n := range candidates {
		valid[n] = true
	}

	fn := func(ctx context.Context, req VoteRequest) (string, error) {
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下候选人中选择: %s", req.Target, strings.Join(candidates, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你投票淘汰了 %s。", req.Target), nil
	}

	return utils.InferTool("vote", desc, fn)
}
