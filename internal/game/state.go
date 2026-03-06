package game

import (
	"fmt"
	"strings"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/player"
)

type Phase string

const (
	PhaseNight Phase = "night"
	PhaseDay   Phase = "day"
	PhaseVote  Phase = "vote"
)

type EventType string

const (
	EventKill        EventType = "kill"
	EventInvestigate EventType = "investigate"
	EventHeal        EventType = "heal"
	EventPoison      EventType = "poison"
	EventSpeech      EventType = "speech"
	EventVote        EventType = "vote"
	EventEliminate   EventType = "eliminate"
	EventDeath       EventType = "death"
	EventShoot       EventType = "shoot"
	EventNarration   EventType = "narration"
	EventHealBlock   EventType = "heal_block"
)

type GameEvent struct {
	Round   int
	Phase   Phase
	Type    EventType
	Actor   string
	Target  string
	Content string
	Public  bool
}

type Speech struct {
	Speaker string
	Content string
}

type GameState struct {
	Players []*player.Player
	Round   int
	Events  []GameEvent

	NightKillTarget      string
	NightSaveTarget      string
	NightPoisonTarget    string
	NightGuardTarget     string
	LastGuardTarget      string
	PrevNightPoisonTarget string

	SeerResults     map[string]string
	WitchHealUsed   bool
	WitchPoisonUsed bool

	HunterShotUsed           bool
	IdiotRevealed            map[string]bool
	Sheriff                  string
	WolfSelfExploded         string
	SheriffElectionCancelled bool
	VictoryMode              config.VictoryMode
	KnightDuelUsed           bool
	DuelKilled               string
	CharmTarget              string

	Speeches   map[int][]Speech
	VoteRecord map[int]map[string]string
}

func NewGameState(players []*player.Player) *GameState {
	return &GameState{
		Players:       players,
		Round:         0,
		Events:        make([]GameEvent, 0),
		SeerResults:   make(map[string]string),
		IdiotRevealed: make(map[string]bool),
		Speeches:      make(map[int][]Speech),
		VoteRecord:    make(map[int]map[string]string),
	}
}

func (s *GameState) AddEvent(e GameEvent) {
	s.Events = append(s.Events, e)
}

func (s *GameState) AlivePlayerNames() []string {
	var names []string
	for _, p := range s.Players {
		if p.Alive {
			names = append(names, p.Name)
		}
	}
	return names
}

func (s *GameState) AlivePlayersExcept(name string) []string {
	var names []string
	for _, p := range s.Players {
		if p.Alive && p.Name != name {
			names = append(names, p.Name)
		}
	}
	return names
}

func (s *GameState) AlivePlayers() []*player.Player {
	var result []*player.Player
	for _, p := range s.Players {
		if p.Alive {
			result = append(result, p)
		}
	}
	return result
}

func (s *GameState) GetPlayer(name string) *player.Player {
	for _, p := range s.Players {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (s *GameState) AliveNonWerewolfNames() []string {
	var names []string
	for _, p := range s.Players {
		if p.Alive && p.Role.Team() != config.TeamWerewolf {
			names = append(names, p.Name)
		}
	}
	return names
}

func (s *GameState) AliveWerewolves() []*player.Player {
	var result []*player.Player
	for _, p := range s.Players {
		if p.Alive && p.Role.Team() == config.TeamWerewolf {
			result = append(result, p)
		}
	}
	return result
}

func (s *GameState) WerewolfTeammates(name string) []string {
	var names []string
	for _, p := range s.Players {
		if p.Role.Team() == config.TeamWerewolf && p.Name != name {
			names = append(names, p.Name)
		}
	}
	return names
}

func (s *GameState) VisibleEvents(playerName string) []GameEvent {
	p := s.GetPlayer(playerName)
	if p == nil {
		return nil
	}

	var visible []GameEvent
	for _, e := range s.Events {
		if e.Public {
			visible = append(visible, e)
			continue
		}
		if e.Actor == playerName {
			visible = append(visible, e)
			continue
		}
		if p.Role.Team() == config.TeamWerewolf && (e.Type == EventKill || e.Type == EventHealBlock) {
			visible = append(visible, e)
			continue
		}
	}
	return visible
}

func (s *GameState) ResetNightActions() {
	s.LastGuardTarget = s.NightGuardTarget
	s.PrevNightPoisonTarget = s.NightPoisonTarget
	s.NightKillTarget = ""
	s.NightSaveTarget = ""
	s.NightPoisonTarget = ""
	s.NightGuardTarget = ""
}

func (s *GameState) FormatVisibleEvents(playerName string) string {
	events := s.VisibleEvents(playerName)
	if len(events) == 0 {
		return "No events yet."
	}

	var sb strings.Builder
	for _, e := range events {
		sb.WriteString(fmt.Sprintf("[Round %d, %s] %s", e.Round, e.Phase, e.Content))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (s *GameState) LastRoundSpeeches() []Speech {
	return s.Speeches[s.Round]
}

func (s *GameState) FormatSpeeches(round int) string {
	speeches := s.Speeches[round]
	if len(speeches) == 0 {
		return "No speeches yet."
	}

	var sb strings.Builder
	for _, sp := range speeches {
		sb.WriteString(fmt.Sprintf("%s: %s\n", sp.Speaker, sp.Content))
	}
	return sb.String()
}
