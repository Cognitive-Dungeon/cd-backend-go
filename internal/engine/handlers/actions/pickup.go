package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

func HandlePickup(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	// 1. Валидация цели (TargetingSystem)
	res := systems.ValidateInteraction(ctx.Actor, p.ItemID, 1.5, true, ctx.Finder, ctx.World)
	if !res.Valid {
		return handlers.Result{Msg: res.Message, MsgType: "ERROR"}, nil
	}

	// 2. Логика инвентаря (InventorySystem)
	msg, err := systems.TryPickup(ctx.Actor, res.Target, ctx.World)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	// 3. Время
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostPickup)
	}

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
