package geo

import "testing"

func TestLocationPacking(t *testing.T) {
	tests := []struct {
		x, y, z int
	}{
		{0, 0, 0},
		{10, 20, 5},
		{-1, -1, -1},                // Отрицательные
		{-100, 500, -10},            // Смешанные
		{33000000, -33000000, 2000}, // Граничные (почти макс)
	}

	for _, tt := range tests {
		loc := Pos(tt.x, tt.y, tt.z)
		gotX, gotY, gotZ := loc.XYZ()

		if gotX != tt.x || gotY != tt.y || gotZ != tt.z {
			t.Errorf("Pos(%d, %d, %d) -> Unpacked(%d, %d, %d)",
				tt.x, tt.y, tt.z, gotX, gotY, gotZ)
		}
	}
}

func TestLocationMove(t *testing.T) {
	start := Pos(10, 10, 0)
	moved := start.Move(DirNorth) // Y - 1

	if moved.Y() != 9 {
		t.Errorf("Expected Y=9, got %d", moved.Y())
	}
}
