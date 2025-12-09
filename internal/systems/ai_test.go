package systems

import (
	"cognitive-server/internal/domain"
	"testing"
)

func TestComputeNPCAction(t *testing.T) {
	// Common Setup
	world := createTestWorld(10, 10)

	basePlayer := &domain.Entity{
		ID:   "player",
		Name: "Player",
		Pos:  domain.Position{X: 5, Y: 5},
	}

	baseNPC := &domain.Entity{
		ID:   "npc",
		Name: "Goblin",
		Pos:  domain.Position{X: 1, Y: 1},
		AI: &domain.AIComponent{
			IsHostile: true,
		},
		Stats: &domain.StatsComponent{
			HP: 10,
		},
	}

	// Helper to reset state for each test
	setup := func() (*domain.Entity, *domain.Entity) {
		// Deep copy components if needed, but for now value copy of struct is enough
		// because pointers (AI, Stats) point to same memory?
		// Better to re-create structs to avoid side effects on pointers.

		pPtr := &domain.Entity{
			ID:   basePlayer.ID,
			Name: basePlayer.Name,
			Pos:  basePlayer.Pos,
		}

		nPtr := &domain.Entity{
			ID:    baseNPC.ID,
			Name:  baseNPC.Name,
			Pos:   baseNPC.Pos,
			AI:    &domain.AIComponent{IsHostile: true},
			Stats: &domain.StatsComponent{HP: 10},
		}
		return nPtr, pPtr
	}

	t.Run("Dead NPC should Wait", func(t *testing.T) {
		npc, player := setup()
		npc.Stats.IsDead = true

		act, _, _, _ := ComputeNPCAction(npc, player, world)
		if act != domain.ActionWait {
			t.Errorf("Dead NPC should WAIT, got %v", act)
		}
	})

	t.Run("Target Too Far", func(t *testing.T) {
		npc, player := setup()
		npc.Pos = domain.Position{X: 0, Y: 0}
		player.Pos = domain.Position{X: 9, Y: 9} // Dist ~12.7

		act, _, _, _ := ComputeNPCAction(npc, player, world)
		if act != domain.ActionWait {
			t.Errorf("NPC too far should WAIT, got %v", act)
		}
	})

	t.Run("Target In Melee Range", func(t *testing.T) {
		npc, player := setup()
		player.Pos = domain.Position{X: 5, Y: 5}
		npc.Pos = domain.Position{X: 5, Y: 4} // Distance 1.0

		act, target, _, _ := ComputeNPCAction(npc, player, world)
		if act != domain.ActionAttack {
			t.Errorf("NPC in melee range should ATTACK, got %v", act)
		}
		if target != player {
			t.Error("Target should be player")
		}
	})

	t.Run("Target In Pursuit Range", func(t *testing.T) {
		npc, player := setup()
		player.Pos = domain.Position{X: 5, Y: 5}
		npc.Pos = domain.Position{X: 5, Y: 3} // Distance 2.0

		act, _, dx, dy := ComputeNPCAction(npc, player, world)
		if act != domain.ActionMove {
			t.Errorf("NPC in aggro range should MOVE, got %v", act)
		}
		// Move towards (5,5) from (5,3) => dy=+1
		if dx != 0 || dy != 1 {
			t.Errorf("Expected move (0,1), got (%d,%d)", dx, dy)
		}
	})
}
