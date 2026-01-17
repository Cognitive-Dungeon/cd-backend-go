package worldmap

import (
	"cognitive-server/pkg/geo"
	"testing"
)

func TestWorld_SetGet(t *testing.T) {
	w := NewWorld()

	// Тестовые данные
	pos1 := geo.Pos(10, 10, 0) // Чанк (0,0,0)
	pos2 := geo.Pos(20, 10, 0) // Чанк (1,0,0) - соседний по X
	pos3 := geo.Pos(10, 10, 5) // Чанк (0,0,5) - другой Z уровень

	tileWall := Tile{Material: 1, Flags: FlagSolid}
	tileWater := Tile{Material: 2, Flags: FlagLiquid}
	tileRoof := Tile{Material: 3, Flags: FlagNone}

	// 1. Запись
	w.SetTile(pos1, tileWall)
	w.SetTile(pos2, tileWater)
	w.SetTile(pos3, tileRoof)

	// 2. Чтение и проверка
	if got := w.GetTile(pos1); got != tileWall {
		t.Errorf("Pos1: got %v, want %v", got, tileWall)
	}
	if got := w.GetTile(pos2); got != tileWater {
		t.Errorf("Pos2: got %v, want %v", got, tileWater)
	}
	if got := w.GetTile(pos3); got != tileRoof {
		t.Errorf("Pos3: got %v, want %v", got, tileRoof)
	}

	// 3. Проверка пустоты (несуществующий тайл)
	emptyPos := geo.Pos(500, 500, 0)
	if got := w.GetTile(emptyPos); !got.IsEmpty() {
		t.Errorf("EmptyPos: got %v, want Empty", got)
	}
}

func TestWorld_ChunksCreatedLazily(t *testing.T) {
	w := NewWorld()

	// Изначально пусто
	if len(w.chunks) != 0 {
		t.Error("New world should be empty")
	}

	// Пишем один тайл
	w.SetTile(geo.Pos(0, 0, 0), Tile{Material: 1})

	// Должен создаться 1 чанк
	w.mu.RLock()
	count := len(w.chunks)
	w.mu.RUnlock()

	if count != 1 {
		t.Errorf("Expected 1 chunk, got %d", count)
	}
}
