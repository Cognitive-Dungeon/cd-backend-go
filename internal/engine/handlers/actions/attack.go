package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

func HandleAttack(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	target := ctx.World.GetEntity(p.TargetID)

	if target == nil {
		return handlers.Result{
			Msg:     "Цель не найдена.",
			MsgType: "ERROR",
		}, nil
	}

	logMsg := systems.ApplyAttack(ctx.Actor, target)

	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostAttackLight)
	}

	return handlers.Result{
		Msg:     logMsg,
		MsgType: "COMBAT",
	}, nil
}
