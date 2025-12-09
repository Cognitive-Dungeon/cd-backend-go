package systems

import (
	"cognitive-server/internal/domain"
	"log"
)

// MovementResult - результат вычисления движения
type MovementResult struct {
	NewX, NewY int
	HasMoved   bool
	BlockedBy  *domain.Entity // Если врезались в кого-то (для атаки)
	IsWall     bool           // Если врезались в стену
}

func CalculateMove(e *domain.Entity, dx, dy int, w *domain.GameWorld) MovementResult {
	targetPos := e.Pos.Shift(dx, dy)
	res := MovementResult{NewX: targetPos.X, NewY: targetPos.Y}

	// --- ОТЛАДКА ---
	log.Printf("[PHYSICS DEBUG | %s] Trying to move from (%d,%d) by (%d,%d) to (%d,%d)", e.Name, e.Pos.X, e.Pos.Y, dx, dy, targetPos.X, targetPos.Y)

	// 1. Проверка границ
	if targetPos.X < 0 || targetPos.X >= w.Width || targetPos.Y < 0 || targetPos.Y >= w.Height {
		res.IsWall = true
		log.Printf("[PHYSICS DEBUG | %s] Move blocked by map BOUNDS.", e.Name)
		return res
	}

	// 2. Проверка стен
	if w.Map[targetPos.Y][targetPos.X].IsWall {
		res.IsWall = true
		log.Printf("[PHYSICS DEBUG | %s] Move blocked by WALL.", e.Name)
		return res
	}

	// 3. Проверка сущностей
	entitiesAtTarget := w.GetEntitiesAt(targetPos.X, targetPos.Y)
	for _, other := range entitiesAtTarget {
		if other.ID == e.ID {
			continue // Игнорируем себя
		}

		if other.Stats != nil && !other.Stats.IsDead {
			isActorAI := e.ControllerID == ""
			isOtherAI := other.ControllerID == ""

			if isActorAI && isOtherAI {
				actorHostile := e.AI != nil && e.AI.IsHostile
				otherHostile := other.AI != nil && other.AI.IsHostile
				if actorHostile && otherHostile {
					log.Printf("[PHYSICS DEBUG | %s] Passing through friendly AI: %s", e.Name, other.Name)
					continue
				}
			}

			log.Printf("[PHYSICS DEBUG | %s] Move blocked by ENTITY: %s", e.Name, other.Name)
			res.BlockedBy = other
			return res
		}
	}

	log.Printf("[PHYSICS DEBUG | %s] Move is VALID.", e.Name)
	res.HasMoved = true
	return res
}
