package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// HandleUnequip обрабатывает команду UNEQUIP - снятие экипировки
func HandleUnequip(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	msg, err := systems.TryUnequip(ctx.Actor, domain.EntityID(p.ItemID))
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostUnequip)

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
