package domain

import "testing"

func TestGameWorld_AddRemoveEntity(t *testing.T) {
	world := &GameWorld{
		Width:          10,
		Height:         10,
		SpatialHash:    make(map[int][]*Entity),
		EntityRegistry: make(map[string]*Entity),
	}

	e := &Entity{
		ID:  "e1",
		Pos: Position{X: 5, Y: 5},
	}

	// Test Add
	world.AddEntity(e)
	world.RegisterEntity(e)

	if len(world.SpatialHash) == 0 {
		t.Error("SpatialHash should not be empty after adding entity")
	}

	retrieved := world.GetEntity("e1")
	if retrieved == nil {
		t.Error("GetEntity returned nil")
	}
	if retrieved != e {
		t.Errorf("GetEntity returned wrong entity: got %v want %v", retrieved, e)
	}

	// Test Remove
	world.RemoveEntity(e)
	world.UnregisterEntity("e1")

	if world.GetEntity("e1") != nil {
		t.Error("Entity should be nil after removal")
	}
}
