package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

func HandleUse(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	msg, err := systems.TryUse(ctx.Actor, p.ItemID)
	if err != nil {
		return handlers.Result{Msg: err.Error(), MsgType: "ERROR"}, nil
	}

	handlers.SpendActionPoints(ctx.Actor, domain.TimeCostUse)

	return handlers.Result{Msg: msg, MsgType: "INFO"}, nil
}
