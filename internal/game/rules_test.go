package game

import (
	"testing"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/player"
	"github.com/alaric/eino-learn/internal/role"
)

func newTestPlayer(name, roleName string, alive bool) *player.Player {
	r, err := role.Get(roleName)
	if err != nil {
		panic(err)
	}
	return &player.Player{
		Name:  name,
		Role:  r,
		Alive: alive,
	}
}

// Fix 3: VictoryModeCity - gods dead + villagers alive = no wolf win
func TestCheckWinAfterVote_CityMode_GodsDeadVillagersAlive(t *testing.T) {
	state := &GameState{
		VictoryMode: config.VictoryModeCity,
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("W2", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("Witch", "witch", false),
			newTestPlayer("V1", "villager", true),
			newTestPlayer("V2", "villager", true),
		},
	}

	result := CheckWinAfterVote(state)
	if result.GameOver {
		t.Errorf("city mode: gods dead but villagers alive should NOT end game, got GameOver=true reason=%q", result.Reason)
	}
}

// Fix 3: VictoryModeCity - gods dead + villagers dead = wolf win
func TestCheckWinAfterVote_CityMode_AllGoodDead(t *testing.T) {
	state := &GameState{
		VictoryMode: config.VictoryModeCity,
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("V1", "villager", false),
		},
	}

	result := CheckWinAfterVote(state)
	if !result.GameOver {
		t.Fatal("city mode: all good dead should end game")
	}
	if result.WinnerTeam != config.TeamWerewolf {
		t.Errorf("expected werewolf win, got %v", result.WinnerTeam)
	}
}

// Fix 3: VictoryModeEdge - gods dead = wolf win (even if villagers alive)
func TestCheckWinAfterVote_EdgeMode_GodsDeadVillagersAlive(t *testing.T) {
	state := &GameState{
		VictoryMode: config.VictoryModeEdge,
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("Witch", "witch", false),
			newTestPlayer("V1", "villager", true),
		},
	}

	result := CheckWinAfterVote(state)
	if !result.GameOver {
		t.Fatal("edge mode: gods dead should end game")
	}
	if result.WinnerTeam != config.TeamWerewolf {
		t.Errorf("expected werewolf win, got %v", result.WinnerTeam)
	}
}

// Fix 3: VictoryModeCity in CheckWinAfterNight
func TestCheckWinAfterNight_CityMode_VillagersDeadGodsAlive(t *testing.T) {
	state := &GameState{
		VictoryMode: config.VictoryModeCity,
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", true),
			newTestPlayer("V1", "villager", false),
			newTestPlayer("V2", "villager", false),
		},
	}

	result := CheckWinAfterNight(state)
	if result.GameOver {
		t.Errorf("city mode: villagers dead but gods alive should NOT end game, got reason=%q", result.Reason)
	}
}

// Fix 6a: guard protection blocks wolf kill in CheckWolfKillFirst
func TestCheckWolfKillFirst_GuardBlocks(t *testing.T) {
	state := &GameState{
		VictoryMode:      config.VictoryModeEdge,
		NightGuardTarget: "V1",
		NightSaveTarget:  "",
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("V1", "villager", true),
		},
	}

	result := CheckWolfKillFirst(state, "V1")
	if result.GameOver {
		t.Error("guard protecting kill target should block wolf-kill-first, got GameOver=true")
	}
}

// Fix 6a: same-guard-same-save does NOT block wolf kill in CheckWolfKillFirst
func TestCheckWolfKillFirst_SameGuardSameSave(t *testing.T) {
	state := &GameState{
		VictoryMode:      config.VictoryModeEdge,
		NightGuardTarget: "V1",
		NightSaveTarget:  "V1",
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("V1", "villager", true),
		},
	}

	// same-guard-same-save: guard cancels heal, kill proceeds.
	// V1 is last villager; kill goes through => wolf win.
	result := CheckWolfKillFirst(state, "V1")
	if !result.GameOver {
		t.Error("same-guard-same-save should allow kill to proceed, expected GameOver=true")
	}
	if result.WinnerTeam != config.TeamWerewolf {
		t.Errorf("expected werewolf win, got %v", result.WinnerTeam)
	}
}

// Guard protection without guard state: normal kill proceeds
func TestCheckWolfKillFirst_NoGuard(t *testing.T) {
	state := &GameState{
		VictoryMode:      config.VictoryModeEdge,
		NightGuardTarget: "",
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", true),
			newTestPlayer("Seer", "seer", false),
			newTestPlayer("V1", "villager", true),
		},
	}

	result := CheckWolfKillFirst(state, "V1")
	if !result.GameOver {
		t.Error("no guard: kill last villager should trigger wolf win")
	}
}

// All wolves dead => villager win
func TestCheckWinAfterVote_AllWolvesDead(t *testing.T) {
	state := &GameState{
		VictoryMode: config.VictoryModeEdge,
		Players: []*player.Player{
			newTestPlayer("W1", "werewolf", false),
			newTestPlayer("Seer", "seer", true),
			newTestPlayer("V1", "villager", true),
		},
	}

	result := CheckWinAfterVote(state)
	if !result.GameOver {
		t.Fatal("all wolves dead should end game")
	}
	if result.WinnerTeam != config.TeamVillager {
		t.Errorf("expected villager win, got %v", result.WinnerTeam)
	}
}
