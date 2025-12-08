package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"fmt"
)

func HandleTalk(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// Если ID пустой — просто бормотание
	if p.TargetID == "" {
		return handlers.Result{
			Msg:     "Вы бормочете в пустоту.",
			MsgType: "SPEECH",
		}, nil
	}

	// 1. Поиск собеседника
	target := ctx.FindEntity(p.TargetID)

	if target == nil {
		return handlers.Result{
			Msg:     "Вас никто не слышит.",
			MsgType: "INFO",
		}, nil
	}

	// 2. Трата времени (разговоры в бою тоже стоят времени)
	// В мирном режиме это не критично, но для порядка добавим
	if ctx.Actor.AI != nil {
		ctx.Actor.AI.Wait(domain.TimeCostInteract)
	}

	// TODO: Здесь будет вызов Gemini API
	// response := ai.GenerateResponse(...)

	return handlers.Result{
		Msg:     fmt.Sprintf("Вы говорите с %s (ИИ пока спит).", target.Name),
		MsgType: "SPEECH",
	}, nil
}
