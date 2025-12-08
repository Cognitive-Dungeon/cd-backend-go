package systems

import (
	"cognitive-server/internal/domain"
	"testing"
)

// Helper для создания пустой карты со стенами в нужных местах
func createTestWorld(w, h int) *domain.GameWorld {
	world := &domain.GameWorld{
		Width: w, Height: h,
		Map: make([][]domain.Tile, h),
	}
	for y := 0; y < h; y++ {
		row := make([]domain.Tile, w)
		for x := 0; x < w; x++ {
			row[x] = domain.Tile{X: x, Y: y, IsWall: false}
		}
		world.Map[y] = row
	}
	return world
}

func TestHasLineOfSight(t *testing.T) {
	// Карта 5x5
	// . . . . .
	// . . # . .  (2,1) - стена
	// . # # # .  (1,2), (2,2), (3,2) - стена
	// . . # . .  (2,3) - стена
	// . . . . .

	w := createTestWorld(5, 5)
	w.Map[1][2].IsWall = true
	w.Map[2][1].IsWall = true
	w.Map[2][2].IsWall = true
	w.Map[2][3].IsWall = true
	w.Map[3][2].IsWall = true

	tests := []struct {
		name string
		p1   domain.Position
		p2   domain.Position
		want bool
	}{
		{"Clear horizontal", domain.Position{X: 0, Y: 0}, domain.Position{X: 4, Y: 0}, true},
		{"Blocked horizontal", domain.Position{X: 0, Y: 2}, domain.Position{X: 4, Y: 2}, false},
		{"Clear diagonal", domain.Position{X: 0, Y: 0}, domain.Position{X: 1, Y: 1}, true},
		{"Blocked diagonal", domain.Position{X: 0, Y: 0}, domain.Position{X: 4, Y: 4}, false}, // через (2,2)
		{"Adjacent wall", domain.Position{X: 2, Y: 1}, domain.Position{X: 2, Y: 2}, true},     // Стоим рядом со стеной и смотрим на неё
		{"Behind wall", domain.Position{X: 2, Y: 1}, domain.Position{X: 2, Y: 3}, false},      // Стена (2,2) мешает
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasLineOfSight(w, tt.p1, tt.p2); got != tt.want {
				t.Errorf("HasLineOfSight(%v, %v) = %v, want %v", tt.p1, tt.p2, got, tt.want)
			}
		})
	}
}
