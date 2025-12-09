package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"fmt"
)

func HandleInteract(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Поиск цели взаимодействия
	target := ctx.Finder.GetEntity(p.TargetID)
	if target == nil {
		return handlers.Result{Msg: "Вы не видите, с чем взаимодействовать.", MsgType: "ERROR"}, nil
	}

	// 2. Проверка дистанции, минимум одна клетка
	if ctx.Actor.Pos.DistanceTo(target.Pos) > 1 {
		return handlers.Result{Msg: "Нужно подойти ближе.", MsgType: "ERROR"}, nil
	}

	// 3. Проверка наличия триггера
	if target.Trigger == nil || target.Trigger.OnInteract == nil {
		return handlers.Result{Msg: fmt.Sprintf("Ничего не происходит при взаимодействии с %s.", target.Name), MsgType: "INFO"}, nil
	}

	// 4. Трата времени
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostInteract)
	}

	return handlers.Result{
		Event: target.Trigger.OnInteract,
	}, nil
}
