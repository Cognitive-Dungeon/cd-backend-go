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
	// ИСПОЛЬЗУЕМ НОВЫЙ МЕТОД Shift
	// Он возвращает новую структуру Position, не меняя текущую
	targetPos := e.Pos.Shift(dx, dy)

	res := MovementResult{NewX: targetPos.X, NewY: targetPos.Y}

	// 1. Проверка границ
	if targetPos.X < 0 || targetPos.X >= w.Width || targetPos.Y < 0 || targetPos.Y >= w.Height {
		res.IsWall = true
		return res
	}

	// 2. Проверка стен
	if w.Map[targetPos.Y][targetPos.X].IsWall {
		res.IsWall = true
		return res
	}

	// 3. Проверка сущностей
	for i := range entities {
		other := &entities[i]
		if other.ID == e.ID {
			continue
		}

		// Игнорируем самого себя
		if other.ID == e.ID {
			continue
		}

		if other.Pos.X == targetPos.X && other.Pos.Y == targetPos.Y {
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
