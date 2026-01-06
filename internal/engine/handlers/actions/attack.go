package actions

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/eventbus"
	"cognitive-server/pkg/api"
)

func HandleAttack(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Предварительный поиск цели (дешевая операция)
	// Мы не проверяем дистанцию здесь, это дело Системы Боя.
	target := ctx.Finder.GetEntity(domain.EntityID(p.TargetID))
	if target == nil {
		return handlers.Result{Msg: "Цель не найдена.", MsgType: "ERROR"}, nil
	}

	// 2. Публикуем событие
	ctx.EventBus.Publish(eventbus.EventType(enums.EventTypeAttackRequested), domain.AttackRequested{
		Attacker: ctx.Actor,
		Target:   target,
		World:    ctx.World,
	})

	// Пустой результат, так как логи атаки придут асинхронно через систему
	return handlers.EmptyResult(), nil
}
