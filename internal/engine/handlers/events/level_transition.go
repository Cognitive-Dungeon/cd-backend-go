package events

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/logger"
	"encoding/json"
)

func HandleLevelTransition(ctx handlers.Context, eventData json.RawMessage) (handlers.Result, error) {
	var transitionEvent struct {
		TargetLevel int             `json:"targetLevel"`
		TargetPosId domain.EntityID `json:"targetPosId"`
	}
	if err := json.Unmarshal(eventData, &transitionEvent); err != nil {
		logger.Log.Errorf("Error parsing LEVEL_TRANSITION event: %v", err)
		return handlers.EmptyResult(), nil
	}

	// Хендлер делегирует сложную логику создания миров и каналов через интерфейс.
	// Сам файл остается чистым и лежит в правильной папке.
	if ctx.Switcher != nil {
		ctx.Switcher.ChangeLevel(ctx.Actor, transitionEvent.TargetLevel, transitionEvent.TargetPosId)
	} else {
		logger.Log.Error("LevelTransition failed: Switcher is nil in context")
	}

	// Сообщение вернет сам движок через AddLog внутри ChangeLevel,
	// либо можно вернуть его здесь.
	return handlers.EmptyResult(), nil
}
