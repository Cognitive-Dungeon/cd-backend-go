package grid

import "testing"

func TestTilePos_AddSub(t *testing.T) {
	a := TilePos{X: 5, Y: -3}
	b := TilePos{X: -2, Y: 4}

	sum := a.Add(b)
	if sum != (TilePos{X: 3, Y: 1}) {
		t.Fatalf("Add failed: got %v", sum)
	}

	diff := a.Sub(b)
	if diff != (TilePos{X: 7, Y: -7}) {
		t.Fatalf("Sub failed: got %v", diff)
	}

	if a.Add(Zero) != a {
		t.Fatalf("Add Zero failed")
	}
}
