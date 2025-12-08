package actions

import "cognitive-server/internal/engine/handlers"

func HandleInit(ctx handlers.Context) (handlers.Result, error) {
	return handlers.Result{
		Msg:     "Добро пожаловать в Cognitive Dungeon.",
		MsgType: "INFO",
	}, nil
}
