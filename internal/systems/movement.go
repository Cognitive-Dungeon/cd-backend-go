package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"github.com/sirupsen/logrus"
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

	moveLogger := logger.Log.WithFields(logrus.Fields{
		"component":   "movement_system",
		"entity_id":   e.ID,
		"entity_name": e.Name,
		"start_pos":   e.Pos,
		"move_vector": map[string]int{"dx": dx, "dy": dy},
		"target_pos":  targetPos,
	})

	moveLogger.Debug("--- Move Calculation Start ---")

	// 1. Проверка границ
	if targetPos.X < 0 || targetPos.X >= w.Width || targetPos.Y < 0 || targetPos.Y >= w.Height {
		res.IsWall = true
		moveLogger.Debug("Move blocked by map BOUNDS.")
		return res
	}

	// 2. Проверка стен
	if w.Map[targetPos.Y][targetPos.X].IsWall {
		res.IsWall = true
		moveLogger.Debug("Move blocked by WALL.")
		return res
	}

	// 3. Проверка сущностей
	entitiesAtTarget := w.GetEntitiesAt(targetPos.X, targetPos.Y)
	for _, other := range entitiesAtTarget {
		if other.ID == e.ID {
			continue // Игнорируем себя
		}

		// Если на клетке есть что-то живое, проверяем коллизию
		if other.Stats != nil && !other.Stats.IsDead {
			// Проверяем, могут ли два NPC пройти друг сквозь друга
			isActorAI := e.ControllerID == ""
			isOtherAI := other.ControllerID == ""

			if isActorAI && isOtherAI {
				actorHostile := e.AI != nil && e.AI.IsHostile
				otherHostile := other.AI != nil && other.AI.IsHostile

				// Если оба враждебны (т.е. союзники), они могут проходить
				if actorHostile && otherHostile {
					moveLogger.WithField("passing_through", other.Name).Debug("Ignoring collision with friendly AI.")
					continue // Игнорируем этого "союзника" и продолжаем проверку.
				}
			}

			// Если это игрок или враждебный NPC, блокируем путь.
			moveLogger.WithFields(logrus.Fields{
				"blocker_id":   other.ID,
				"blocker_name": other.Name,
			}).Debug("Move blocked by ENTITY.")

			res.BlockedBy = other
			return res
		}
	}

	moveLogger.Debug("Move is VALID.")
	res.HasMoved = true
	return res
}
