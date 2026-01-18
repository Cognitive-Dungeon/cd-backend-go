package grid

import "testing"

func TestInDiamond(t *testing.T) {
	center := TilePos{0, 0}
	diamond1 := TilePos{1, 1}
	diamond2 := TilePos{2, 2}

	if !diamond1.InDiamond(center, 2) {
		t.Fatal("expected inside diamond")
	}

	if diamond2.InDiamond(center, 2) {
		t.Fatal("expected outside diamond")
	}
}

func TestForEachInSquareCount(t *testing.T) {
	count := 0
	ForEachInSquare(TilePos{0, 0}, 1, func(TilePos) bool {
		count++
		return true
	})

	if count != 9 {
		t.Fatalf("expected 9 tiles, got %d", count)
	}
}
