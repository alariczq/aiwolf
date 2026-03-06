package action

import (
	"context"
	"testing"
)

func TestCreatePoisonTool_WitchCanTargetSelf(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob", "Witch"}
	var result string

	tool, err := CreatePoisonTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreatePoisonTool: %v", err)
	}

	out, err := tool.InvokableRun(context.Background(), `{"use_potion":true,"target":"Witch"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "Witch" {
		t.Errorf("expected result=%q, got %q", "Witch", result)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestCreatePoisonTool_InvalidTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob"}
	var result string

	tool, err := CreatePoisonTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreatePoisonTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"use_potion":true,"target":"Charlie"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "" {
		t.Errorf("invalid target should not set result, got %q", result)
	}
}

func TestCreateGuardTool_BlocksLastTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob", "Charlie"}
	var result string

	tool, err := CreateGuardTool(alivePlayers, "Alice", &result)
	if err != nil {
		t.Fatalf("CreateGuardTool: %v", err)
	}

	out, _ := tool.InvokableRun(context.Background(), `{"skip":false,"target":"Alice"}`)
	if result == "Alice" {
		t.Errorf("guard should not be able to guard last night's target; out=%s", out)
	}
}

func TestCreateGuardTool_ValidTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob", "Charlie"}
	var result string

	tool, err := CreateGuardTool(alivePlayers, "Alice", &result)
	if err != nil {
		t.Fatalf("CreateGuardTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"skip":false,"target":"Bob"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "Bob" {
		t.Errorf("expected result=%q, got %q", "Bob", result)
	}
}

func TestCreateGuardTool_Skip(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob"}
	var result string

	tool, err := CreateGuardTool(alivePlayers, "", &result)
	if err != nil {
		t.Fatalf("CreateGuardTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"skip":true,"target":""}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "" {
		t.Errorf("skip should set empty result, got %q", result)
	}
}

func TestCreateDuelTool_ValidTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob"}
	var result string

	tool, err := CreateDuelTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreateDuelTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"use_duel":true,"target":"Alice"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected result=%q, got %q", "Alice", result)
	}
}

func TestCreateDuelTool_NoDuel(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob"}
	var result string

	tool, err := CreateDuelTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreateDuelTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"use_duel":false,"target":""}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "" {
		t.Errorf("no duel should set empty result, got %q", result)
	}
}

func TestCreateWolfKingSelfExplodeTool_ConfirmWithTarget(t *testing.T) {
	targets := []string{"Alice", "Bob"}
	var exploded bool
	var takeTarget string

	tool, err := CreateWolfKingSelfExplodeTool("WolfKing", targets, &exploded, &takeTarget)
	if err != nil {
		t.Fatalf("CreateWolfKingSelfExplodeTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"confirm":true,"target":"Alice"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !exploded {
		t.Error("expected exploded=true")
	}
	if takeTarget != "Alice" {
		t.Errorf("expected takeTarget=%q, got %q", "Alice", takeTarget)
	}
}

func TestCreateWolfKingSelfExplodeTool_Decline(t *testing.T) {
	targets := []string{"Alice"}
	var exploded bool
	var takeTarget string

	tool, err := CreateWolfKingSelfExplodeTool("WolfKing", targets, &exploded, &takeTarget)
	if err != nil {
		t.Fatalf("CreateWolfKingSelfExplodeTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"confirm":false,"target":""}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if exploded {
		t.Error("expected exploded=false")
	}
}

func TestCreateCharmTool_ValidTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob", "Charlie"}
	var result string

	tool, err := CreateCharmTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreateCharmTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"target":"Bob"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "Bob" {
		t.Errorf("expected result=%q, got %q", "Bob", result)
	}
}

func TestCreateCharmTool_InvalidTarget(t *testing.T) {
	alivePlayers := []string{"Alice", "Bob"}
	var result string

	tool, err := CreateCharmTool(alivePlayers, &result)
	if err != nil {
		t.Fatalf("CreateCharmTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"target":"Nobody"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if result != "" {
		t.Errorf("invalid target should not set result, got %q", result)
	}
}
