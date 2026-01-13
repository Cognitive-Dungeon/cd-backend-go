package engine

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
)

type MovementSystem struct {
	Instance *Instance
	Bus      *eventbus.EventBus
}

func NewMovementSystem(inst *Instance, bus *eventbus.EventBus) *MovementSystem {
	sys := &MovementSystem{
		Instance: inst,
		Bus:      bus,
	}

	eventbus.Subscribe(bus, eventbus.EventType(enums.EventMoveRequest), sys.onMoveRequest)

	return sys
}

// onMoveRequest — это реакция на входящий пакет или решение AI
func (s *MovementSystem) onMoveRequest(ev enums.MoveRequestEvent) {
	// 1. Проверка валидности сущности
	if !s.Instance.IsValid(ev.Object) {
		return
	}

	// 2. Получаем PositionComponent
	pos := s.Instance.GetPosition(ev.Object)
	if pos == nil {
		return
	}

	from := pos.TilePos

	// 3. Вычисляем целевую позицию
	to := from
	switch ev.Direction {
	case enums.DirUp:
		to.Y--
	case enums.DirDown:
		to.Y++
	case enums.DirLeft:
		to.X--
	case enums.DirRight:
		to.X++
	default:
		return
	}

	// 4. Проверка проходимости
	if !s.Instance.Grid.IsWalkable(to) {
		logger.Log.Debugf("Movement blocked for %s at [%d, %d]", ev.Object, to.X, to.Y)
		// Тут можно кинуть EventMovementFailed, если нужно проиграть звук удара
		return
	}

	// 3. Мутация состояния (Apply)
	pos.TilePos = to

	logger.Log.Debugf("Object %s moved: [%d,%d] -> [%d,%d]", ev.Object, from.X, from.Y, to.X, to.Y)

	// 4. Уведомление (Reaction)
	// Кидаем событие, что движение состоялось.
	// На него подпишется:
	// - NetworkSystem (чтобы отправить пакет клиентам)
	// - AggroSystem (чтобы сагрить мобов)
	// - TriggerSystem (наступил на ловушку)
	s.Bus.Publish(eventbus.EventType(enums.EventObjectMoved), enums.ObjectMovedEvent{
		Object: ev.Object,
		From:   from,
		To:     to,
	})
}
