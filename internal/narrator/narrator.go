package narrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type Narrator struct {
	openingRunnable compose.Runnable[map[string]any, *schema.Message]
	deathRunnable   compose.Runnable[map[string]any, *schema.Message]
	summaryRunnable compose.Runnable[map[string]any, *schema.Message]
	endRunnable     compose.Runnable[map[string]any, *schema.Message]
}

func New(ctx context.Context, model einomodel.ToolCallingChatModel) (*Narrator, error) {
	openingRunnable, err := buildOpeningChain(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building opening chain: %w", err)
	}

	deathRunnable, err := buildDeathChain(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building death chain: %w", err)
	}

	summaryRunnable, err := buildSummaryChain(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building summary chain: %w", err)
	}

	endRunnable, err := buildEndChain(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building end chain: %w", err)
	}

	return &Narrator{
		openingRunnable: openingRunnable,
		deathRunnable:   deathRunnable,
		summaryRunnable: summaryRunnable,
		endRunnable:     endRunnable,
	}, nil
}

func buildOpeningChain(ctx context.Context, model einomodel.ToolCallingChatModel) (compose.Runnable[map[string]any, *schema.Message], error) {
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一场狼人杀游戏的旁白解说。"+
			"用中文生成引人入胜的开场白，像悬疑剧的开篇旁白。"+
			"4-6句话。营造悬疑紧张又趣味盎然的氛围。"+
			"自然地提及几位人物的性格特点，但绝不透露任何人的角色身份。"),
		schema.UserMessage("世界设定: {setting}\n\n登场人物:\n{roster}\n\n"+
			"用中文做一段精彩的开场旁白，介绍这些人物和这个村庄的故事。"),
	)

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(tpl)
	chain.AppendChatModel(model)

	return chain.Compile(ctx)
}

func buildDeathChain(ctx context.Context, model einomodel.ToolCallingChatModel) (compose.Runnable[map[string]any, *schema.Message], error) {
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一场狼人杀游戏的旁白解说。"+
			"用中文生成生动、戏剧化的死亡公告，2-3句话。"+
			"要有戏剧张力但简洁。不要透露死者的角色身份。"),
		schema.UserMessage("第 {round} 回合：村庄陷入悲痛。{names} 因 {cause} 而倒下。"+
			"用中文宣布他们的死亡。"),
	)

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(tpl)
	chain.AppendChatModel(model)

	return chain.Compile(ctx)
}

func buildSummaryChain(ctx context.Context, model einomodel.ToolCallingChatModel) (compose.Runnable[map[string]any, *schema.Message], error) {
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一场狼人杀游戏的旁白解说。"+
			"用中文生成扣人心弦的回合总结，2-4句话。"+
			"营造紧张氛围，站在村庄的视角叙述。"),
		schema.UserMessage("第 {round} 回合结束。关键事件: {events}。"+
			"用中文总结本回合的结局。"),
	)

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(tpl)
	chain.AppendChatModel(model)

	return chain.Compile(ctx)
}

func buildEndChain(ctx context.Context, model einomodel.ToolCallingChatModel) (compose.Runnable[map[string]any, *schema.Message], error) {
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一场狼人杀游戏的旁白解说。"+
			"用中文生成震撼的游戏结局叙述，3-5句话。"+
			"赞颂胜利者，也致敬所有玩家的奋斗。"),
		schema.UserMessage("游戏结束！{winner} 取得了最终胜利！{summary}。"+
			"用中文做一段令人难忘的闭幕旁白。"),
	)

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(tpl)
	chain.AppendChatModel(model)

	return chain.Compile(ctx)
}

func (n *Narrator) NarrateOpening(ctx context.Context, setting string, roster string) (string, error) {
	result, err := n.openingRunnable.Invoke(ctx, map[string]any{
		"setting": setting,
		"roster":  roster,
	})
	if err != nil {
		return "", fmt.Errorf("narrating opening: %w", err)
	}

	return result.Content, nil
}

func (n *Narrator) NarrateDeath(ctx context.Context, round int, deaths []string, cause string) (string, error) {
	names := strings.Join(deaths, ", ")
	verb := "倒下了"

	result, err := n.deathRunnable.Invoke(ctx, map[string]any{
		"round": round,
		"names": names,
		"verb":  verb,
		"cause": cause,
	})
	if err != nil {
		return "", fmt.Errorf("narrating death: %w", err)
	}

	return result.Content, nil
}

func (n *Narrator) NarrateRoundSummary(ctx context.Context, round int, events string) (string, error) {
	result, err := n.summaryRunnable.Invoke(ctx, map[string]any{
		"round":  round,
		"events": events,
	})
	if err != nil {
		return "", fmt.Errorf("narrating round summary: %w", err)
	}

	return result.Content, nil
}

func (n *Narrator) NarrateGameEnd(ctx context.Context, winner string, summary string) (string, error) {
	result, err := n.endRunnable.Invoke(ctx, map[string]any{
		"winner":  winner,
		"summary": summary,
	})
	if err != nil {
		return "", fmt.Errorf("narrating game end: %w", err)
	}

	return result.Content, nil
}
