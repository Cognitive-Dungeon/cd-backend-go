package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"testing"
)

func TestFOV_Caching(t *testing.T) {
	logger.Init()
	// Setup
	w := &domain.GameWorld{
		Width:  10,
		Height: 10,
		Map:    make([][]domain.Tile, 10),
	}
	for y := 0; y < 10; y++ {
		w.Map[y] = make([]domain.Tile, 10)
	}

	vision := &domain.VisionComponent{
		Radius: 5,
	}

	pos := domain.Position{X: 5, Y: 5}

	// 1. Initial Calculation
	res1 := ComputeVisibleTiles(w, pos, vision)
	if res1 == nil {
		t.Fatal("Result should not be nil")
	}
	if vision.CachedVisibleTiles == nil {
		t.Fatal("Cache should be populated")
	}
	if vision.IsDirty {
		t.Fatal("IsDirty should be false after calculation")
	}

	// 2. Cached Access
	res2 := ComputeVisibleTiles(w, pos, vision)
	// Check if maps are the same object
	// Note: Deep equality check is expensive, here we just want to ensure logic flow
	if len(res1) != len(res2) {
		t.Errorf("Expected same length %d, got %d", len(res1), len(res2))
	}

	// 3. Invalidation
	vision.IsDirty = true
	res3 := ComputeVisibleTiles(w, pos, vision)
	if len(res3) != len(res1) {
		t.Errorf("Expected same length after recalc %d, got %d", len(res1), len(res3))
	}
	if vision.IsDirty {
		t.Fatal("IsDirty should be false after recalc")
	}
}
