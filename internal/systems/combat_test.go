package systems

import (
	"cognitive-server/internal/domain"
	"testing"
)

func TestApplyAttack(t *testing.T) {
	attacker := &domain.Entity{
		Name: "Hero",
		Stats: &domain.StatsComponent{
			Strength: 5,
		},
	}

	target := &domain.Entity{
		Name: "Ork",
		Stats: &domain.StatsComponent{
			HP:    20,
			MaxHP: 20,
		},
	}

	// Attack logic: damage = max(1, attacker.Str)
	msg := ApplyAttack(attacker, target)

	if target.Stats.HP != 15 {
		t.Errorf("Expected target HP to be 15, got %d", target.Stats.HP)
	}

	if msg == "" {
		t.Error("Expected attack log message, got empty string")
	}

	// Kill shot
	attacker.Stats.Strength = 100
	ApplyAttack(attacker, target)

	if target.Stats.HP > 0 {
		t.Errorf("Expected target to be dead (HP <= 0), got %d", target.Stats.HP)
	}
	if !target.Stats.IsDead {
		t.Error("Expected IsDead flag to be true")
	}
}
