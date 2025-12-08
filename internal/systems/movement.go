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
func CalculateMove(e *domain.Entity, dx, dy int, w *domain.GameWorld) MovementResult {
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
	entitiesAtTarget := w.GetEntitiesAt(targetPos.X, targetPos.Y)

	for _, other := range entitiesAtTarget {
		if other.ID == e.ID {
			continue
		} // Игнор себя

		// Логика коллизии (блокируем только живых и твердых)
		if other.Stats != nil && !other.Stats.IsDead {
			res.BlockedBy = other
			return res
		}
	}

	res.HasMoved = true
	return res
}
