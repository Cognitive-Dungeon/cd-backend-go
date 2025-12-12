package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// HandleDrop обрабатывает команду DROP - выброс предмета из инвентаря
func HandleDrop(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	// Для Drop не нужен TargetingSystem, так как цель - предмет ВНУТРИ инвентаря, а не на карте.

	msg, err := systems.TryDrop(ctx.Actor, p.ItemID, p.Count, ctx.World)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostDrop)
	}

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
