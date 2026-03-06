package game

import (
	"github.com/alaric/eino-learn/internal/config"
)

type WinResult struct {
	GameOver   bool
	WinnerTeam config.Team
	Reason     string
}

func CheckWinAfterNight(state *GameState) WinResult {
	var aliveWolves, aliveGods, alivePlainVillagers int
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		switch {
		case p.Role.Team() == config.TeamWerewolf:
			aliveWolves++
		case godRoles[p.Role.Name()]:
			aliveGods++
		default:
			alivePlainVillagers++
		}
	}

	if aliveWolves == 0 {
		return WinResult{
			GameOver:   true,
			WinnerTeam: config.TeamVillager,
			Reason:     "所有狼人都被淘汰了！",
		}
	}

	if result, ok := checkWolfWin(state.VictoryMode, aliveGods, alivePlainVillagers); ok {
		return result
	}

	return WinResult{GameOver: false}
}

var godRoles = map[string]bool{
	"seer": true, "witch": true, "hunter": true, "idiot": true, "guard": true, "knight": true,
}

func checkWolfWin(mode config.VictoryMode, aliveGods, alivePlainVillagers int) (WinResult, bool) {
	return checkWolfWinWithPrefix(mode, aliveGods, alivePlainVillagers, "")
}

func checkWolfWinWithPrefix(mode config.VictoryMode, aliveGods, alivePlainVillagers int, prefix string) (WinResult, bool) {
	switch mode {
	case config.VictoryModeCity:
		if aliveGods == 0 && alivePlainVillagers == 0 {
			return WinResult{
				GameOver:   true,
				WinnerTeam: config.TeamWerewolf,
				Reason:     prefix + "所有好人阵营成员都被淘汰了！（屠城）",
			}, true
		}
	default:
		if aliveGods == 0 {
			return WinResult{
				GameOver:   true,
				WinnerTeam: config.TeamWerewolf,
				Reason:     prefix + "所有神职角色都被淘汰了！（屠神）",
			}, true
		}
		if alivePlainVillagers == 0 {
			return WinResult{
				GameOver:   true,
				WinnerTeam: config.TeamWerewolf,
				Reason:     prefix + "所有平民都被淘汰了！（屠民）",
			}, true
		}
	}
	return WinResult{}, false
}

// CheckWolfKillFirst implements the wolf-kill-first principle (rule 11):
// if the wolf kill alone (without witch save) achieves a wolf win condition,
// the game ends immediately and subsequent night actions are not resolved.
func CheckWolfKillFirst(state *GameState, killTarget string) WinResult {
	if killTarget == "" {
		return WinResult{GameOver: false}
	}

	guardTarget := state.NightGuardTarget
	if guardTarget == killTarget && guardTarget != state.NightSaveTarget {
		return WinResult{GameOver: false}
	}

	target := state.GetPlayer(killTarget)
	if target == nil || !target.Alive {
		return WinResult{GameOver: false}
	}

	var aliveGods, alivePlainVillagers int
	for _, p := range state.Players {
		if !p.Alive || p.Name == killTarget {
			continue
		}
		switch {
		case p.Role.Team() == config.TeamWerewolf:
		case godRoles[p.Role.Name()]:
			aliveGods++
		default:
			alivePlainVillagers++
		}
	}

	if result, ok := checkWolfWinWithPrefix(state.VictoryMode, aliveGods, alivePlainVillagers, "狼刀在先："); ok {
		return result
	}

	return WinResult{GameOver: false}
}

func CheckWinAfterVote(state *GameState) WinResult {
	var aliveWolves, aliveGods, alivePlainVillagers int
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		switch {
		case p.Role.Team() == config.TeamWerewolf:
			aliveWolves++
		case godRoles[p.Role.Name()]:
			aliveGods++
		default:
			alivePlainVillagers++
		}
	}

	if aliveWolves == 0 {
		return WinResult{
			GameOver:   true,
			WinnerTeam: config.TeamVillager,
			Reason:     "所有狼人都被淘汰了！",
		}
	}

	if result, ok := checkWolfWin(state.VictoryMode, aliveGods, alivePlainVillagers); ok {
		return result
	}

	return WinResult{GameOver: false}
}

type VoteResult struct {
	Eliminated    string
	IsTied        bool
	TiedPlayers   []string
	Tally         map[string]int
	WeightedTally map[string]float64
}

func TallyVotes(votes map[string]string) VoteResult {
	return TallyWeightedVotes(votes, "")
}

func TallyFlatVotes(votes map[string]string) VoteResult {
	return TallyWeightedVotes(votes, "")
}

func TallyWeightedVotes(votes map[string]string, sheriff string, endorsedTargets ...string) VoteResult {
	tally := make(map[string]int)
	weighted := make(map[string]float64)
	for voter, target := range votes {
		tally[target]++
		if voter == sheriff {
			weighted[target] += 1.5
		} else {
			weighted[target] += 1.0
		}
	}
	for _, t := range endorsedTargets {
		if t != "" {
			weighted[t] += 0.5
		}
	}

	var maxVotes float64
	for _, count := range weighted {
		if count > maxVotes {
			maxVotes = count
		}
	}

	var tiedPlayers []string
	for target, count := range weighted {
		if count == maxVotes {
			tiedPlayers = append(tiedPlayers, target)
		}
	}

	tied := len(tiedPlayers) > 1

	return VoteResult{
		Eliminated:    tiedPlayers[0],
		IsTied:        tied,
		TiedPlayers:   tiedPlayers,
		Tally:         tally,
		WeightedTally: weighted,
	}
}
