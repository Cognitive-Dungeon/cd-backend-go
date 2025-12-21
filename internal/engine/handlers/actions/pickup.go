package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// HandlePickup обрабатывает команду PICKUP - подбор предмета с земли
func HandlePickup(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	// 1. Валидация цели (TargetingSystem)
	res := systems.ValidateInteraction(ctx.Actor, domain.EntityID(p.ItemID), 1.5, true, ctx.Finder, ctx.World)
	if !res.Valid {
		return handlers.Result{Msg: res.Message, MsgType: "ERROR"}, nil
	}

	// 2. Логика инвентаря (InventorySystem)
	msg, err := systems.TryPickup(ctx.Actor, res.Target, ctx.World)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	// 3. Время
	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostPickup)

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
