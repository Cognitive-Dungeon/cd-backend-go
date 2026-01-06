package systems

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/eventbus"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// MovementResult - результат вычисления движения
type MovementResult struct {
	NewX, NewY int
	HasMoved   bool
	BlockedBy  *domain.Entity // Если врезались в кого-то (для атаки)
	IsWall     bool           // Если врезались в стену
}

type MovementSystem struct {
	BaseSystem
}

func (s *MovementSystem) Name() string {
	return "MovementSystem"
}

func (s *MovementSystem) Init(bus *eventbus.EventBus) {
	moveBus := s.initBus(bus)
	bind(moveBus, enums.EventTypeMoveRequested, s.onMoveRequested)
}

// onMoveRequested — реактивная логика.
// Хендлер говорит "Хочу пойти", Система решает "Пойдешь, врежешься или атакуешь".
func (s *MovementSystem) onMoveRequested(ev domain.MoveRequested) {
	actor := ev.Actor
	world := ev.World
	// ev.Position — это целевая координата (Target X, Y)
	targetX, targetY := ev.X, ev.Y

	moveLogger := logger.Log.WithFields(logrus.Fields{
		"component": "movement_system",
		"actor_id":  actor.ID,
		"target":    fmt.Sprintf("[%d, %d]", targetX, targetY),
	})

	// 1. Проверка границ карты
	if targetX < 0 || targetX >= world.Width || targetY < 0 || targetY >= world.Height {
		s.publishLog(world, "Нельзя выйти за границы мира.", "INFO")
		// Тратим немного времени на осознание тупика? (Опционально)
		if actor.AI != nil {
			actor.AI.Wait(domain.TimeCostWait)
		}
		return
	}

	// 2. Проверка стен
	if world.Map[targetY][targetX].IsWall {
		if actor.Type == domain.EntityTypePlayer {
			s.publishLog(world, "Путь прегражден стеной.", "INFO")
		}
		if actor.AI != nil {
			actor.AI.Wait(domain.TimeCostWait)
		}
		return
	}

	// 3. Проверка на живые препятствия (Коллизии)
	entitiesAtTarget := world.GetEntitiesAt(targetX, targetY)
	for _, other := range entitiesAtTarget {
		if other.ID == actor.ID {
			continue
		}

		// Если врезались в кого-то с HP
		if other.Stats != nil && !other.Stats.IsDead {

			shouldAttack := false

			// 1. Игрок всегда атакует Врагов
			if actor.Type == domain.EntityTypePlayer && other.Type == domain.EntityTypeEnemy {
				shouldAttack = true
			}
			// 2. Враги всегда атакуют Игрока
			if actor.Type == domain.EntityTypeEnemy && other.Type == domain.EntityTypePlayer {
				shouldAttack = true
			}
			// 3. Враги атакуют Врагов (если включено Friendly Fire, пока выключим)
			// if actor.Type == domain.EntityTypeEnemy && other.Type == domain.EntityTypeEnemy { shouldAttack = false }

			// 4. Fallback: если у кого-то есть явный AI с флагом Hostile
			if !shouldAttack {
				actorHostile := actor.AI != nil && actor.AI.IsHostile
				targetHostile := other.AI != nil && other.AI.IsHostile
				// Атакуем, если мы злые, а цель добрая (или наоборот)
				if actorHostile != targetHostile {
					shouldAttack = true
				}
			}
			// -----------------------------

			if shouldAttack {
				moveLogger.Infof("Bump collision -> Triggering Attack on %s", other.ID)

				s.emit(enums.EventTypeAttackRequested, domain.AttackRequested{
					Attacker: actor,
					Target:   other,
					World:    world,
				})

				return // Выходим, движение заменяется атакой
			}

			// Если не враги (NPC или другой игрок в мирной зоне), просто блокируем
			s.publishLog(world, fmt.Sprintf("%s мешает пройти.", other.Name), "INFO")
			if actor.AI != nil {
				actor.AI.Wait(domain.TimeCostWait)
			}
			return
		}
	}

	// 4. Движение разрешено — Применяем изменения (State Mutation)
	oldPos := actor.Pos

	// Обновляем SpatialHash и координаты
	err := world.UpdateEntityPos(actor, targetX, targetY)
	if err != nil {
		moveLogger.Error("Failed to update entity pos in SpatialHash")
		return
	}

	// 5. Побочные эффекты успешного движения

	// Сброс кэша зрения (FOV)
	if actor.Vision != nil {
		actor.Vision.IsDirty = true
	}

	// Трата времени
	if actor.AI != nil {
		actor.AI.Wait(domain.TimeCostMove)
	}

	// Публикуем событие "Сущность переместилась"
	// На это могут подписаться: Ловушки, Триггеры сюжета, UI звуки шагов
	s.emit(enums.EventTypeEntityMoved, domain.EntityMoved{
		Actor:        actor,
		FromPosition: oldPos,
		ToPosition:   actor.Pos,
		World:        world,
	})

	moveLogger.Debug("Move success")
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
