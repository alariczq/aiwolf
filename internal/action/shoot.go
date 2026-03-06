package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type ShootRequest struct {
	Shoot  bool   `json:"shoot" desc:"是否开枪带走一名玩家。true 表示开枪，false 表示不开枪。"`
	Target string `json:"target" desc:"要射杀的玩家名字。shoot 为 true 时必填。必须是存活玩家之一。"`
}

func CreateShootTool(alivePlayers []string, result *string) (tool.InvokableTool, error) {
	desc := fmt.Sprintf("你是猎人，你已出局！可以开枪带走一名存活玩家。可选目标: %s。也可以选择不开枪。", strings.Join(alivePlayers, ", "))

	valid := make(map[string]bool, len(alivePlayers))
	for _, n := range alivePlayers {
		valid[n] = true
	}

	fn := func(ctx context.Context, req ShootRequest) (string, error) {
		if req.Shoot {
			if !valid[req.Target] {
				return fmt.Sprintf("无效目标 %q，请从以下玩家中选择: %s", req.Target, strings.Join(alivePlayers, ", ")), nil
			}
			*result = req.Target
			return fmt.Sprintf("你开枪带走了 %s！", req.Target), nil
		}
		*result = ""
		return "你选择不开枪。", nil
	}

	return utils.InferTool("shoot", desc, fn)
}
