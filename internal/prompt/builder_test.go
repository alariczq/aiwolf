package prompt

import (
	"strings"
	"testing"
)

func TestBuildGameRules_VariantText(t *testing.T) {
	roleCounts := map[string]int{
		"werewolf": 3,
		"seer":     1,
		"witch":    1,
		"hunter":   1,
		"villager": 3,
	}

	tests := []struct {
		name        string
		variant     RulesVariant
		wantSave    string
		wantReveal  string
	}{
		{
			name:       "default never/never",
			variant:    RulesVariant{WitchSelfSave: 0, IdentityReveal: 0},
			wantSave:   "不能自救",
			wantReveal: "暗牌局",
		},
		{
			name:       "first night only / always reveal",
			variant:    RulesVariant{WitchSelfSave: 1, IdentityReveal: 1},
			wantSave:   "仅首夜可自救",
			wantReveal: "明牌局",
		},
		{
			name:       "always self-save / never reveal",
			variant:    RulesVariant{WitchSelfSave: 2, IdentityReveal: 0},
			wantSave:   "任何时候都可自救",
			wantReveal: "暗牌局",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildGameRules(roleCounts, tt.variant)

			if !strings.Contains(result, tt.wantSave) {
				t.Errorf("BuildGameRules() missing witch self-save text %q", tt.wantSave)
			}
			if !strings.Contains(result, tt.wantReveal) {
				t.Errorf("BuildGameRules() missing identity reveal text %q", tt.wantReveal)
			}
			if !strings.Contains(result, "本局规则变体") {
				t.Error("BuildGameRules() missing variant section header")
			}
		})
	}
}

func TestBuildGameRules_VictoryMode(t *testing.T) {
	roleCounts := map[string]int{
		"werewolf": 2,
		"seer":     1,
		"villager": 3,
	}

	t.Run("edge mode", func(t *testing.T) {
		result := BuildGameRules(roleCounts, RulesVariant{VictoryMode: 0})
		if !strings.Contains(result, "屠边") {
			t.Error("expected edge mode text")
		}
	})

	t.Run("city mode", func(t *testing.T) {
		result := BuildGameRules(roleCounts, RulesVariant{VictoryMode: 1})
		if !strings.Contains(result, "屠城") {
			t.Error("expected city mode text")
		}
	})
}

func TestBuildGameRules_ExtendedRoles(t *testing.T) {
	roleCounts := map[string]int{
		"werewolf":    2,
		"wolf_king":   1,
		"wolf_beauty": 1,
		"seer":        1,
		"witch":       1,
		"guard":       1,
		"knight":      1,
		"villager":    3,
	}

	result := BuildGameRules(roleCounts, RulesVariant{})
	for _, want := range []string{"守卫", "骑士"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected rules to contain %q", want)
		}
	}
}
