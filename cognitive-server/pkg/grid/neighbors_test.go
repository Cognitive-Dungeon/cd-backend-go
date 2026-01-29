package grid

import "testing"

func TestForEachNeighbor4(t *testing.T) {
	center := TilePos{0, 0}
	seen := map[TilePos]bool{}

	ForEachNeighbor4(center, func(p TilePos) {
		seen[p] = true
	})

	if len(seen) != 4 {
		t.Fatalf("expected 4 neighbors, got %d", len(seen))
	}

	expected := []TilePos{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	}
	for _, e := range expected {
		if !seen[e] {
			t.Fatalf("missing neighbor %v", e)
		}
	}
}

func TestForEachNeighbor8(t *testing.T) {
	center := TilePos{0, 0}
	seen := map[TilePos]bool{}

	ForEachNeighbor8(center, func(p TilePos) {
		seen[p] = true
	})

	if len(seen) != 8 {
		t.Fatalf("expected 8 neighbors, got %d", len(seen))
	}

	expected := []TilePos{
		{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1},
	}

	for _, e := range expected {
		if !seen[e] {
			t.Fatalf("missing neighbor %v", e)
		}
	}
}

func TestForEachNeighbor4WithCost(t *testing.T) {
	center := TilePos{0, 0}
	total := 0

	ForEachNeighbor4WithCost(center, func(p TilePos, cost int32) {
		total += int(cost)
	})

	if total != (4 * 10) {
		t.Fatalf("unexpected cost sum")
	}
}

func TestForEachNeighbor8WithCost(t *testing.T) {
	center := TilePos{0, 0}
	total := 0

	ForEachNeighbor8WithCost(center, func(p TilePos, cost int32) {
		total += int(cost)
	})

	if total != (4*10 + 4*14) {
		t.Fatalf("unexpected cost sum")
	}
}
