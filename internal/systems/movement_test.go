package systems

import (
	"cognitive-server/internal/domain"
	"testing"
)

func TestCalculateMove(t *testing.T) {
	// Setup world
	world := &domain.GameWorld{
		Width:  10,
		Height: 10,
		Map:    make([][]domain.Tile, 10),
	}
	for y := 0; y < 10; y++ {
		world.Map[y] = make([]domain.Tile, 10)
		for x := 0; x < 10; x++ {
			world.Map[y][x] = domain.Tile{X: x, Y: y, IsWall: false}
		}
	}
	// Add a wall
	world.Map[5][5].IsWall = true

	// Setup actor
	actor := &domain.Entity{
		Pos: domain.Position{X: 4, Y: 5},
	}

	// Test 1: Move into empty space
	res := CalculateMove(actor, 0, -1, world) // Move Up (from 4,5 to 4,4)
	if !res.HasMoved {
		t.Error("Expected move to succeed")
	}
	if res.NewX != 4 || res.NewY != 4 {
		t.Errorf("Expected pos (4,4), got (%d,%d)", res.NewX, res.NewY)
	}

	// Test 2: Move into wall
	res = CalculateMove(actor, 1, 0, world) // Move Right (from 4,5 to 5,5 - WALL)
	if res.HasMoved {
		t.Error("Expected move to fail (wall)")
	}
	if !res.IsWall {
		t.Error("Expected IsWall=true")
	}

	// Test 3: Move OOB
	actor.Pos = domain.Position{X: 0, Y: 0}
	res = CalculateMove(actor, -1, 0, world)
	if res.HasMoved {
		t.Error("Expected move to fail (OOB)")
	}
}
