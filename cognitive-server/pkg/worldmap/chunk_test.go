package worldmap

import (
	"cognitive-server/pkg/geo"
	"testing"
)

func TestChunkCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		worldX    int
		wantChunk int // Ожидаемый X координата чанка
		wantLocal int // Ожидаемый X внутри чанка (0-15)
	}{
		{"Zero", 0, 0, 0},
		{"Positive Inner", 5, 0, 5},
		{"Positive Edge", 15, 0, 15},
		{"Positive Next", 16, 1, 0},
		{"Positive Far", 33, 2, 1}, // 32+1
		{"Negative Inner", -1, -1, 15},
		{"Negative Edge", -16, -1, 0},
		{"Negative Next", -17, -2, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := geo.Pos(tt.worldX, 0, 0)

			// Проверка ключа чанка
			chunkKey := GetChunkKey(pos)
			if chunkKey.X() != tt.wantChunk {
				t.Errorf("Chunk Key X: got %d, want %d", chunkKey.X(), tt.wantChunk)
			}

			// Проверка локальных координат
			lx, _ := GetLocalCoords(pos)
			if lx != tt.wantLocal {
				t.Errorf("Local X: got %d, want %d", lx, tt.wantLocal)
			}
		})
	}
}
