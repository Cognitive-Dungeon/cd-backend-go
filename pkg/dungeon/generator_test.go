package dungeon

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	level := 1
	world, entities, startPos := Generate(level)

	// 1. Проверка размеров мира
	if world.Width != MapWidth || world.Height != MapHeight {
		t.Errorf("Expected map size %dx%d, got %dx%d", MapWidth, MapHeight, world.Width, world.Height)
	}

	// 2. Проверка, что карта не пустая
	if len(world.Map) == 0 {
		t.Fatal("Map is empty")
	}

	// 3. Проверка стартовой позиции
	// Игрок не должен появиться в стене
	startX, startY := startPos.X, startPos.Y
	if world.Map[startY][startX].IsWall {
		t.Errorf("Start position [%d,%d] is inside a wall!", startX, startY)
	}

	// 4. Проверка сущностей
	// Должны быть как минимум выходы (Exit) и, вероятно, враги
	if len(entities) == 0 {
		t.Error("No entities generated (expected exits and enemies)")
	}

	hasExitDown := false
	for _, e := range entities {
		if e.Type == "EXIT" && e.Label == ">" {
			hasExitDown = true
			break
		}
	}

	if !hasExitDown {
		t.Error("Level exit (>) not found among entities")
	}
}

// Тест вспомогательной функции пересечения комнат
func TestRect_Intersects(t *testing.T) {
	r1 := Rect{0, 0, 10, 10}
	r2 := Rect{5, 5, 10, 10} // Пересекается
	r3 := Rect{20, 20, 5, 5} // Не пересекается

	if !r1.Intersects(r2) {
		t.Error("Rects should intersect")
	}

	if r1.Intersects(r3) {
		t.Error("Rects should NOT intersect")
	}
}
