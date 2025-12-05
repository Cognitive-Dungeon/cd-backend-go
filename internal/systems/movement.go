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

	// 1. Границы и Стены
	if newX < 0 || newX >= w.Width || newY < 0 || newY >= w.Height {
		res.IsWall = true
		return res
	}
	if w.Map[newY][newX].IsWall {
		res.IsWall = true
		return res
	}

	// 2. Сущности
	for i := range entities {
		other := &entities[i]

		// Игнорируем самого себя
		if other.ID == e.ID {
			continue
		}

		if other.Pos.X == newX && other.Pos.Y == newY {
			// Логика коллизии:
			// Блокируем, только если у сущности есть Stats (тело) и она жива.
			// Предметы и выходы (Stats == nil) проходимы.
			if other.Stats != nil && !other.Stats.IsDead {
				res.BlockedBy = other
				return res
			}
		}
	}

	res.HasMoved = true
	return res
}
