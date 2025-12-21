package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"fmt"
)

func HandleInteract(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Валидация через TargetingSystem
	// Дистанция 1.5 (можно нажать рычаг под ногами или рядом)
	// LOS = false, так как если мы стоим на лестнице, мы её "чувствуем", даже если под ногами
	res := systems.ValidateInteraction(ctx.Actor, domain.EntityID(p.TargetID), 1.5, false, ctx.Finder, ctx.World)

	if !res.Valid {
		return handlers.Result{Msg: res.Message, MsgType: "ERROR"}, nil
	}

	target := res.Target

	// 2. Проверка наличия триггера (специфика Interact)
	if target.Trigger == nil || target.Trigger.OnInteract == nil {
		return handlers.Result{
			Msg:     fmt.Sprintf("Ничего не происходит при взаимодействии с %s.", target.Name),
			MsgType: "INFO",
		}, nil
	}

	// 3. Трата времени
	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostInteract)

	return handlers.Result{
		Event: target.Trigger.OnInteract,
	}, nil
}
