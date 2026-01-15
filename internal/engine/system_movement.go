package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
)

// SystemInput обрабатывает сырые команды (Cmd) и превращает их в намерения (Intent).
// Здесь можно проверить станы, руты, страх и т.д.
func SystemInput(w *ecs.World) {
	// Создаем буфер команд для безопасного добавления IntentMove
	cb := ecs.NewCommandBuffer(w)

	// Итерируемся по всем, у кого есть запрос на движение
	// View1 принимает (World, ComponentID)
	for id, cmd := range ecs.View1[CmdMove](w, CID_CmdMove) {
		dx, dy := 0, 0
		switch cmd.Direction {
		case enums.DirUp:
			dy = -1
		case enums.DirDown:
			dy = 1
		case enums.DirLeft:
			dx = -1
		case enums.DirRight:
			dx = 1
		}

		if dx != 0 || dy != 0 {
			// Добавляем намерение.
			// Используем ecs.Add через CommandBuffer
			ecs.Add(cb, CID_IntentMove, id, IntentMove{Dx: int32(dx), Dy: int32(dy)})
		}
	}

	// Применяем созданные интенты
	cb.Execute()
}

// SystemMovement применяет намерения к физическому миру.
// Проверяет коллизии и обновляет координаты.
func SystemMovement(w *ecs.World, grid *Grid, bus *eventbus.EventBus) {
	// Итерируемся по сущностям, у которых есть IntentMove И Position.
	// Используем View2 с ID компонентов для скорости.
	for id, join := range ecs.View2[IntentMove, PositionComponent](w, CID_IntentMove, CID_Position) {

		intent := join.First
		pos := join.Second

		targetX := pos.X + TileCoord(intent.Dx)
		targetY := pos.Y + TileCoord(intent.Dy)
		targetPos := types.TilePos{X: types.TileCoord(targetX), Y: types.TileCoord(targetY)}

		// Проверка коллизий
		if grid.IsWalkable(targetPos) {
			oldPos := pos.TilePos

			// Мутация состояния (in-place update)
			pos.TilePos = targetPos

			// Отправляем событие для других систем (например, триггеров или сети)
			// События пока оставляем на EventBus для совместимости с Gateway notification,
			// но в будущем их тоже можно перевести на ECS (ScopeFrame).
			bus.Publish(eventbus.EventType(enums.EventObjectMoved), enums.ObjectMovedEvent{
				Object: types.ObjectGuid(id), // Обратная конвертация ID -> Guid
				From:   oldPos,
				To:     targetPos,
			})
		}
	}
}
