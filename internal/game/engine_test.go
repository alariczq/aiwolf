package game

import (
	"testing"

	"github.com/alaric/eino-learn/internal/config"
)

func TestDeathContent_IdentityRevealAlways(t *testing.T) {
	e := &Engine{identityReveal: config.IdentityRevealAlways}

	got := e.deathContent("Alice 在昨夜死亡。", "werewolf")
	want := "Alice 在昨夜死亡。（身份揭晓：狼人）"
	if got != want {
		t.Errorf("deathContent() = %q, want %q", got, want)
	}
}

func TestDeathContent_IdentityRevealNever(t *testing.T) {
	e := &Engine{identityReveal: config.IdentityRevealNever}

	got := e.deathContent("Alice 在昨夜死亡。", "werewolf")
	want := "Alice 在昨夜死亡。"
	if got != want {
		t.Errorf("deathContent() = %q, want %q", got, want)
	}
}

func TestDeathContent_HunterShoot(t *testing.T) {
	e := &Engine{identityReveal: config.IdentityRevealAlways}

	got := e.deathContent("猎人 Bob 开枪带走了 Charlie！", "seer")
	want := "猎人 Bob 开枪带走了 Charlie！（身份揭晓：预言家）"
	if got != want {
		t.Errorf("deathContent() = %q, want %q", got, want)
	}
}

func TestDeathContent_VoteElimination(t *testing.T) {
	e := &Engine{identityReveal: config.IdentityRevealAlways}

	got := e.deathContent("Diana 被投票淘汰了。", "villager")
	want := "Diana 被投票淘汰了。（身份揭晓：村民）"
	if got != want {
		t.Errorf("deathContent() = %q, want %q", got, want)
	}
}

func TestRoleChineseName(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"werewolf", "狼人"},
		{"seer", "预言家"},
		{"witch", "女巫"},
		{"hunter", "猎人"},
		{"idiot", "白痴"},
		{"villager", "村民"},
		{"guard", "守卫"},
		{"knight", "骑士"},
		{"wolf_king", "白狼王"},
		{"wolf_beauty", "狼美人"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := roleChineseName(tt.role)
		if got != tt.want {
			t.Errorf("roleChineseName(%q) = %q, want %q", tt.role, got, tt.want)
		}
	}
}
