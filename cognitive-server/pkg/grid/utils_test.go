package grid

import "testing"

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 3) != 3 {
		t.Fatal("Clamp upper bound failed")
	}
	if Clamp(-1, 0, 3) != 0 {
		t.Fatal("Clamp lower bound failed")
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{
		Min: TilePos{0, 0},
		Max: TilePos{2, 2},
	}

	if !r.Contains(TilePos{1, 1}) {
		t.Fatal("expected inside rect")
	}
	if r.Contains(TilePos{3, 3}) {
		t.Fatal("expected outside rect")
	}
}
