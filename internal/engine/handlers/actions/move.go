package actions

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/eventbus"
	"cognitive-server/pkg/api"
)

func HandleMove(ctx handlers.Context, p api.DirectionPayload) (handlers.Result, error) {
	// Рассчитываем целевую клетку
	targetPos := ctx.Actor.Pos.Shift(p.Dx, p.Dy)

	// Публикуем событие в шину
	// Так как EventBus в нашей реализации синхронный (вызывает хендлеры сразу),
	// результат (смена координат, лог, атака) применится немедленно, до возврата из этой функции.
	ctx.EventBus.Publish(eventbus.EventType(enums.EventTypeMoveRequested), domain.MoveRequested{
		Actor:    ctx.Actor,
		Position: targetPos, // Передаем итоговую координату
		World:    ctx.World,
	})

	// Мы возвращаем пустой результат, потому что логи (если они были)
	// уже добавлены в Instance через s.publishLog внутри системы.
	return handlers.EmptyResult(), nil
}
