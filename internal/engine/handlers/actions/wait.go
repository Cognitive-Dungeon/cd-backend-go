package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"fmt"
)

func HandleWait(ctx handlers.Context) (handlers.Result, error) {
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostWait)
	}

	if ctx.Actor.Type == domain.EntityTypePlayer {
		return handlers.Result{
			Msg:     fmt.Sprintf("%s пропускает ход.", ctx.Actor.Name),
			MsgType: "INFO",
		}, nil
	}

	// Боты ждут молча
	return handlers.EmptyResult(), nil
}
