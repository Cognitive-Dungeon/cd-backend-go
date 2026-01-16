package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
)

// InputMoveSystem обрабатывает сырые команды (Cmd) и превращает их в намерения (Intent).
// Здесь можно проверить станы, руты, страх и т.д.
func InputMoveSystem(ctx InputContext) {

	// Итерируемся по всем, у кого есть запрос на движение
	// View1 принимает (World, ComponentID)
	for id, cmd := range ecs.View1[CmdMove](ctx.World, CID_CmdMove) {
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
			ecs.Add(ctx.Commands, CID_IntentMove, id, IntentMove{Dx: int32(dx), Dy: int32(dy)})
		}
	}
}

// LogicMoveSystem применяет намерения к физическому миру.
// Проверяет коллизии и обновляет координаты.
func LogicMoveSystem(ctx LogicContext) {
	// Итерируемся по сущностям, у которых есть IntentMove И Position.
	// Используем View2 с ID компонентов для скорости.
	for id, join := range ecs.View2[IntentMove, PositionComponent](ctx.World, CID_IntentMove, CID_Position) {

		intent := join.First
		pos := join.Second

		targetX := pos.X + TileCoord(intent.Dx)
		targetY := pos.Y + TileCoord(intent.Dy)
		targetPos := types.TilePos{X: TileCoord(targetX), Y: TileCoord(targetY)}

		// Проверка коллизий
		if ctx.Grid.IsWalkable(targetPos) {
			oldPos := pos.TilePos

			// Мутация состояния (in-place update)
			pos.TilePos = targetPos

			// Отправляем событие для других систем (например, триггеров или сети)
			// События пока оставляем на EventBus для совместимости с Gateway notification,
			// но в будущем их тоже можно перевести на ECS (ScopeFrame).
			ctx.Bus.Publish(eventbus.EventType(enums.EventObjectMoved), enums.ObjectMovedEvent{
				Object: types.ObjectGuid(id), // Обратная конвертация ID -> Guid
				From:   oldPos,
				To:     targetPos,
			})
		}
	}
}
