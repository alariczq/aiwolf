package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SheriffVoteRequest struct {
	Target string `json:"target" desc:"你要投票选为警长的玩家名字。必须是候选人之一。"`
}

type BadgeTransferRequest struct {
	Target  string `json:"target" desc:"你要移交警徽的存活玩家名字。撕毁警徽时留空。"`
	Destroy bool   `json:"destroy" desc:"设为 true 表示撕毁警徽。警长职位将永久取消。"`
}

func CreateSheriffVoteTool(candidates []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("投票选出警长。警长在淘汰投票中拥有 1.5 倍票权。候选人: %s", strings.Join(candidates, ", "))

	valid := make(map[string]bool, len(candidates))
	for _, n := range candidates {
		valid[n] = true
	}

	fn := func(ctx context.Context, req SheriffVoteRequest) (string, error) {
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下候选人中选择: %s", req.Target, strings.Join(candidates, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你投票选 %s 为警长。", req.Target), nil
	}

	return utils.InferTool("sheriff_vote", desc, fn)
}

func CreateBadgeTransferTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("你是警长且已出局。将警徽移交给一名存活玩家，或撕毁警徽。可选玩家: %s。撕毁请设 destroy=true。", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req BadgeTransferRequest) (string, error) {
		if req.Destroy {
			*result = ""
			return "你撕毁了警徽。本局不再有警长。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你将警徽移交给了 %s。", req.Target), nil
	}

	return utils.InferTool("transfer_badge", desc, fn)
}
