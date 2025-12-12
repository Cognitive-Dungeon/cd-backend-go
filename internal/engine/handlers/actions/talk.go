package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"fmt"
)

func HandleTalk(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// Если ID пустой — просто бормотание (специфичный кейс, оставляем тут)
	if p.TargetID == "" {
		return handlers.Result{
			Msg:     "Вы бормочете в пустоту.",
			MsgType: "SPEECH",
		}, nil
	}

	// 1. Валидация через TargetingSystem
	// Дистанция 1.5 (разговор лицом к лицу), Нужен LOS (нельзя говорить сквозь стену)
	res := systems.ValidateInteraction(ctx.Actor, p.TargetID, 1.5, true, ctx.Finder, ctx.World)

	if !res.Valid {
		// Для разговора можно смягчить ошибку до INFO
		return handlers.Result{Msg: res.Message, MsgType: "INFO"}, nil
	}

	target := res.Target

	// 2. Трата времени
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostInteract)
	}

	return handlers.Result{
		Msg:     fmt.Sprintf("Вы говорите с %s (ИИ пока спит).", target.Name),
		MsgType: "SPEECH",
	}, nil
}
