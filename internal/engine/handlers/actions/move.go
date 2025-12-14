package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

func HandleMove(ctx handlers.Context, p api.DirectionPayload) (handlers.Result, error) {
	if ctx.Actor.AI == nil {
		return handlers.EmptyResult(), nil // Или ошибка, по желанию
	}

	res := systems.CalculateMove(ctx.Actor, p.Dx, p.Dy, ctx.World)

	// Если движение заблокировано враждебным мобом, атакуем его
	if res.BlockedBy != nil {
		actorHostile := ctx.Actor.AI.IsHostile
		targetHostile := false
		if res.BlockedBy.AI != nil {
			targetHostile = res.BlockedBy.AI.IsHostile
		}

		if actorHostile != targetHostile {
			logMsg := systems.ApplyAttack(ctx.Actor, res.BlockedBy, ctx.Rng)
			ctx.Actor.AI.Wait(domain.TimeCostAttackLight)
			return handlers.Result{Msg: logMsg, MsgType: "COMBAT"}, nil
		}
	}

	if res.HasMoved {
		err := ctx.World.UpdateEntityPos(ctx.Actor, res.NewX, res.NewY)
		if err != nil {
			return handlers.Result{Msg: "Ошибка перемещения", MsgType: "ERROR"}, nil
		}

		// Invalidate FOV Cache
		if ctx.Actor.Vision != nil {
			ctx.Actor.Vision.IsDirty = true
		}

		ctx.Actor.AI.Wait(domain.TimeCostMove)
		return handlers.EmptyResult(), nil
	}

	if res.IsWall {
		if ctx.Actor.Type == domain.EntityTypePlayer {
			return handlers.Result{Msg: "Путь прегражден.", MsgType: "ERROR"}, nil
		}
		ctx.Actor.AI.Wait(domain.TimeCostWait)
	}

	return handlers.EmptyResult(), nil
}
