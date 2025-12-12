package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// HandleEquip обрабатывает команду EQUIP - экипировка оружия/брони
func HandleEquip(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	msg, err := systems.TryEquip(ctx.Actor, p.ItemID)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostEquip)
	}

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
