package systems

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine/data"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/grid"
)

// InputMoveSystem обрабатывает сырые команды (Cmd) и превращает их в намерения (Intent).
// Здесь можно проверить станы, руты, страх и т.д.
func InputMoveSystem(ctx ecs2.InputContext) {

	// Итерируемся по всем, у кого есть запрос на движение
	// View1 принимает (World, ComponentID)
	for id, cmd := range ecs.View1[components.CmdMove](ctx.World, components.CID_CmdMove) {
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
		default:
			break
		}

		if dx != 0 || dy != 0 {
			// Добавляем намерение.
			// Используем ecs.Add через CommandBuffer
			ecs.Add(ctx.Commands, components.CID_IntentMove, id, components.IntentMove{Dx: int32(dx), Dy: int32(dy)})
		}
	}
}

// LogicMoveSystem применяет намерения к физическому миру.
// Проверяет коллизии и обновляет координаты.
func LogicMoveSystem(ctx ecs2.LogicContext) {
	// Итерируемся по сущностям, у которых есть IntentMove И Position.
	// Используем View2 с ID компонентов для скорости.
	for id, join := range ecs.View2[components.IntentMove, components.PositionComponent](ctx.World, components.CID_IntentMove, components.CID_Position) {

		intent := join.First
		pos := join.Second

		targetX := pos.X + data.TileCoord(intent.Dx)
		targetY := pos.Y + data.TileCoord(intent.Dy)
		targetPos := grid.TilePos{X: targetX, Y: targetY}
		// TODO: Выкинуть старый пакет grid
		// Конвертируем grid.TilePos (int32) -> geo.Location (uint64)
		geoPos := geo.Pos(int(targetX), int(targetY), 0)

		// Проверка коллизий
		if !ctx.WorldMap.IsSolidFast(geoPos) {
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
