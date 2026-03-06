package action

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type HealRequest struct {
	UsePotion bool `json:"use_potion" desc:"是否使用解药救人。true 表示救，false 表示不救。"`
}

type PoisonRequest struct {
	UsePotion bool   `json:"use_potion" desc:"是否使用毒药。true 表示毒人，false 表示不用。"`
	Target    string `json:"target" desc:"要毒杀的玩家名字。use_potion 为 true 时必填。必须是存活玩家之一。"`
}

func CreateHealTool(victimName string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("今晚被刀的人是 %s。决定是否使用解药救人（解药全局仅一瓶）。", victimName)

	fn := func(ctx context.Context, req HealRequest) (string, error) {
		*result = strconv.FormatBool(req.UsePotion)
		if req.UsePotion {
			return fmt.Sprintf("你使用了解药救了 %s。", victimName), nil
		}
		return fmt.Sprintf("你选择不救 %s。", victimName), nil
	}

	return utils.InferTool("heal", desc, fn)
}

func CreatePoisonTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("决定是否使用毒药毒杀一名玩家（毒药全局仅一瓶）。可选目标: %s", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req PoisonRequest) (string, error) {
		if req.UsePotion {
			if !valid[req.Target] {
				return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
			}
			*result = req.Target
			return fmt.Sprintf("你毒杀了 %s。", req.Target), nil
		}
		*result = ""
		return "你选择不使用毒药。", nil
	}

	return utils.InferTool("poison", desc, fn)
}
