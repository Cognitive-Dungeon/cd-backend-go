package systems

import (
	"cognitive-server/internal/domain"
)

// MovementResult - результат вычисления движения
type MovementResult struct {
	NewX, NewY int
	HasMoved   bool
	BlockedBy  *domain.Entity // Если врезались в кого-то (для атаки)
	IsWall     bool           // Если врезались в стену
}

// CalculateMove вычисляет новую позицию. Не меняет состояние мира!
func CalculateMove(e *domain.Entity, dx, dy int, w *domain.GameWorld, entities []domain.Entity) MovementResult {
	newX := e.Pos.X + dx
	newY := e.Pos.Y + dy
	res := MovementResult{NewX: newX, NewY: newY}

	// 1. Проверка границ карты
	if newX < 0 || newX >= w.Width || newY < 0 || newY >= w.Height {
		res.IsWall = true
		return res
	}

	// 2. Проверка стен
	if w.Map[newY][newX].IsWall {
		res.IsWall = true
		return res
	}

	// 3. Проверка сущностей (Коллизия)
	for i := range entities {
		other := &entities[i]
		if !other.IsDead && other.Pos.X == newX && other.Pos.Y == newY {
			// Мы не можем наступить на другую сущность (пока что)
			res.BlockedBy = other
			return res
		}
	}

	// Если дошли сюда - путь свободен
	res.HasMoved = true
	return res
}
