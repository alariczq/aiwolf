package action

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type CampaignDecisionRequest struct {
	Run bool `json:"run" desc:"设为 true 表示上警竞选，false 表示不上警。"`
}

func CreateCampaignDecisionTool(result *bool) (tool.InvokableTool, error) {
	fn := func(ctx context.Context, req CampaignDecisionRequest) (string, error) {
		*result = req.Run
		if req.Run {
			return "你决定上警竞选。接下来你将发表竞选演说。", nil
		}
		return "你决定不上警。你将在投票阶段投票选出警长。", nil
	}

	return utils.InferTool("campaign_decision",
		"决定是否上警竞选警长。上警后你将成为候选人并发表演说；不上警则在投票阶段投票。",
		fn)
}

type WithdrawDecisionRequest struct {
	Withdraw bool `json:"withdraw" desc:"设为 true 表示退水（放弃竞选），false 表示继续竞选。退水后你不能被投票也不能投票。"`
}

func CreateWithdrawDecisionTool(result *bool) (tool.InvokableTool, error) {
	fn := func(ctx context.Context, req WithdrawDecisionRequest) (string, error) {
		*result = req.Withdraw
		if req.Withdraw {
			return "你选择退水，放弃竞选。你不能被投票也不能投票选警长。", nil
		}
		return "你选择继续竞选，等待投票结果。", nil
	}

	return utils.InferTool("withdraw_decision",
		"决定是否退水（放弃竞选警长）。退水后你既不能被投票也不能投票选警长。",
		fn)
}
