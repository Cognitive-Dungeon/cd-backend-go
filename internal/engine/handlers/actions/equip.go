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

	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostEquip)

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
