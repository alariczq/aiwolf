package genesis

import (
	"testing"

	"github.com/alaric/eino-learn/internal/config"
)

func TestParseWitchSelfSave(t *testing.T) {
	tests := []struct {
		input string
		want  config.WitchSelfSave
	}{
		{"never", config.WitchSelfSaveNever},
		{"Never", config.WitchSelfSaveNever},
		{"first_night_only", config.WitchSelfSaveFirstOnly},
		{"FIRST_NIGHT_ONLY", config.WitchSelfSaveFirstOnly},
		{"always", config.WitchSelfSaveAlways},
		{"Always", config.WitchSelfSaveAlways},
		{"", config.WitchSelfSaveNever},
		{"invalid", config.WitchSelfSaveNever},
		{" never ", config.WitchSelfSaveNever},
	}
	for _, tt := range tests {
		got := parseWitchSelfSave(tt.input)
		if got != tt.want {
			t.Errorf("parseWitchSelfSave(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseIdentityReveal(t *testing.T) {
	tests := []struct {
		input string
		want  config.IdentityReveal
	}{
		{"never", config.IdentityRevealNever},
		{"Never", config.IdentityRevealNever},
		{"always", config.IdentityRevealAlways},
		{"Always", config.IdentityRevealAlways},
		{"", config.IdentityRevealNever},
		{"invalid", config.IdentityRevealNever},
		{" always ", config.IdentityRevealAlways},
	}
	for _, tt := range tests {
		got := parseIdentityReveal(tt.input)
		if got != tt.want {
			t.Errorf("parseIdentityReveal(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestValidate_WithRules(t *testing.T) {
	w := worldSpec{
		Setting: "test setting",
		Players: []playerSpec{
			{Name: "A", Role: "werewolf", Persona: "persona"},
			{Name: "B", Role: "werewolf", Persona: "persona"},
			{Name: "C", Role: "seer", Persona: "persona"},
			{Name: "D", Role: "witch", Persona: "persona"},
			{Name: "E", Role: "villager", Persona: "persona"},
			{Name: "F", Role: "villager", Persona: "persona"},
			{Name: "G", Role: "villager", Persona: "persona"},
			{Name: "H", Role: "villager", Persona: "persona"},
		},
		Rules: rulesSpec{
			WitchSelfSave:  "first_night_only",
			IdentityReveal: "always",
		},
	}

	if err := validate(w); err != nil {
		t.Errorf("validate() returned error for valid world with rules: %v", err)
	}
}

func TestParseVictoryMode(t *testing.T) {
	tests := []struct {
		input string
		want  config.VictoryMode
	}{
		{"city", config.VictoryModeCity},
		{"City", config.VictoryModeCity},
		{"CITY", config.VictoryModeCity},
		{" city ", config.VictoryModeCity},
		{"edge", config.VictoryModeEdge},
		{"Edge", config.VictoryModeEdge},
		{"", config.VictoryModeEdge},
		{"invalid", config.VictoryModeEdge},
	}
	for _, tt := range tests {
		got := parseVictoryMode(tt.input)
		if got != tt.want {
			t.Errorf("parseVictoryMode(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestValidate_ExtendedRoles(t *testing.T) {
	base := func() worldSpec {
		return worldSpec{
			Setting: "test",
			Players: []playerSpec{
				{Name: "A", Role: "werewolf", Persona: "p"},
				{Name: "B", Role: "werewolf", Persona: "p"},
				{Name: "C", Role: "seer", Persona: "p"},
				{Name: "D", Role: "witch", Persona: "p"},
				{Name: "E", Role: "hunter", Persona: "p"},
				{Name: "F", Role: "villager", Persona: "p"},
				{Name: "G", Role: "villager", Persona: "p"},
				{Name: "H", Role: "villager", Persona: "p"},
				{Name: "I", Role: "villager", Persona: "p"},
				{Name: "J", Role: "villager", Persona: "p"},
			},
		}
	}

	t.Run("guard and knight valid", func(t *testing.T) {
		w := base()
		w.Players[5].Role = "guard"
		w.Players[6].Role = "knight"
		if err := validate(w); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("wolf_king replaces one wolf", func(t *testing.T) {
		w := base()
		w.Players[1].Role = "wolf_king"
		if err := validate(w); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("wolf_beauty needs 2+ wolves", func(t *testing.T) {
		w := base()
		w.Players[0].Role = "wolf_beauty"
		w.Players[1].Role = "villager"
		if err := validate(w); err == nil {
			t.Error("expected error for wolf_beauty with only 1 wolf")
		}
	})

	t.Run("wolf_beauty with 2 wolves valid", func(t *testing.T) {
		w := base()
		w.Players[1].Role = "wolf_beauty"
		if err := validate(w); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate guard invalid", func(t *testing.T) {
		w := base()
		w.Players[5].Role = "guard"
		w.Players[6].Role = "guard"
		if err := validate(w); err == nil {
			t.Error("expected error for 2 guards")
		}
	})

	t.Run("duplicate wolf_king invalid", func(t *testing.T) {
		w := base()
		w.Players[0].Role = "wolf_king"
		w.Players[1].Role = "wolf_king"
		if err := validate(w); err == nil {
			t.Error("expected error for 2 wolf_kings")
		}
	})
}
