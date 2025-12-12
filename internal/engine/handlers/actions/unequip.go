package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// HandleUnequip обрабатывает команду UNEQUIP - снятие экипировки
func HandleUnequip(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	msg, err := systems.TryUnequip(ctx.Actor, p.ItemID)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostUnequip)
	}

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
