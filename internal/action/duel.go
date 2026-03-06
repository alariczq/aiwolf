package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type DuelRequest struct {
	UseDuel bool   `json:"use_duel" desc:"设为 true 表示发起决斗，false 表示不使用决斗。"`
	Target  string `json:"target" desc:"决斗目标的玩家名字。use_duel 为 true 时必填。"`
}

func CreateDuelTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("骑士决斗：翻牌选择一名玩家决斗。决斗狼人则狼人出局；决斗好人则你出局。技能仅一次。可选目标: %s", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req DuelRequest) (string, error) {
		if !req.UseDuel {
			*result = ""
			return "你选择不使用决斗技能。", nil
		}
		if !valid[req.Target] {
			return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
		}
		*result = req.Target
		return fmt.Sprintf("你决定对 %s 发起决斗！", req.Target), nil
	}

	return utils.InferTool("duel", desc, fn)
}
