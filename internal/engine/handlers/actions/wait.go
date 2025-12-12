package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"fmt"
)

func HandleWait(ctx handlers.Context) (handlers.Result, error) {
	// Тратим время (для всех одинаково)
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostWait)
	}

	// Возвращаем результат для ВСЕХ сущностей.
	return handlers.Result{
		Msg:     fmt.Sprintf("%s пропускает ход.", ctx.Actor.Name),
		MsgType: "INFO",
	}, nil
}
