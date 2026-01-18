package grid

import (
	"reflect"
	"testing"
)

func TestLineHorizontal(t *testing.T) {
	var points []TilePos

	Line(
		TilePos{0, 0},
		TilePos{3, 0},
		func(p TilePos) bool {
			points = append(points, p)
			return true
		},
	)

	expected := []TilePos{
		{0, 0}, {1, 0}, {2, 0}, {3, 0},
	}

	if !reflect.DeepEqual(points, expected) {
		t.Fatalf("line mismatch: %v", points)
	}
}

func TestLineInterrupt(t *testing.T) {
	count := 0

	Line(TilePos{0, 0}, TilePos{10, 0}, func(p TilePos) bool {
		count++
		return count < 3
	})

	if count != 3 {
		t.Fatalf("expected interrupt at 3, got %d", count)
	}
}
