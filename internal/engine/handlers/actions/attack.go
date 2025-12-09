package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

func HandleAttack(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Поиск цели
	target := ctx.Finder.GetEntity(p.TargetID)

	if target == nil {
		return handlers.Result{Msg: "Цель не найдена.", MsgType: "ERROR"}, nil
	}

	// 2. Проверка дистанции (Melee range = 1.5)
	// В будущем WeaponRange можно брать из инвентаря
	const WeaponRange = 1.5
	dist := ctx.Actor.Pos.DistanceTo(target.Pos)

	if dist > WeaponRange {
		return handlers.Result{
			Msg:     "Цель слишком далеко.",
			MsgType: "ERROR",
		}, nil
	}

	// 3. Проверка видимости (Сквозь стены бить нельзя)
	if !systems.HasLineOfSight(ctx.World, ctx.Actor.Pos, target.Pos) {
		return handlers.Result{
			Msg:     "Вы не видите цель.",
			MsgType: "ERROR",
		}, nil
	}

	// 4. Вызов Системы Боя
	logMsg := systems.ApplyAttack(ctx.Actor, target)

	// 5. Трата времени
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostAttackLight)
	}

	return handlers.Result{
		Msg:     logMsg,
		MsgType: "COMBAT",
	}, nil
}
