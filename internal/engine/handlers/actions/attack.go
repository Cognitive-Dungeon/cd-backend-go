package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems" // Импортируем системы
	"cognitive-server/pkg/api"
)

func HandleAttack(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Валидация через TargetingSystem
	// Дистанция 1.5 (ближний бой), Нужен LOS (сквозь стены бить нельзя)
	res := systems.ValidateInteraction(ctx.Actor, p.TargetID, 1.5, true, ctx.Finder, ctx.World)

	if !res.Valid {
		return handlers.Result{Msg: res.Message, MsgType: "ERROR"}, nil
	}

	target := res.Target

	// 2. Вызов Системы Боя
	logMsg := systems.ApplyAttack(ctx.Actor, target)

	// 3. Трата времени
	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostAttackLight)

	return handlers.Result{
		Msg:     logMsg,
		MsgType: "COMBAT",
	}, nil
}
