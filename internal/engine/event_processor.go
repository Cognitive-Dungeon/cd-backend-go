package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/logger"
	"encoding/json"
)

// processEvent вызывает соответствующий хендлер для события
func (s *GameService) processEvent(actor *domain.Entity, eventData json.RawMessage) {
	var genericEvent struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(eventData, &genericEvent); err != nil {
		logger.Log.Errorf("Error parsing event: %v", err)
		return
	}

	eventType := domain.ParseEvent(genericEvent.Event)
	if eventType == domain.EventUnknown {
		logger.Log.Warnf("Unknown event type: %s", genericEvent.Event)
		return
	}

	handler, ok := s.eventHandlers[eventType]
	if !ok {
		logger.Log.Warnf("No handler registered for event type: %s", eventType)
		return
	}

	levelID := s.EntityLocations[actor.ID]
	instance := s.Instances[levelID]

	ctx := handlers.Context{
		Finder:   instance.World,
		World:    instance.World,
		Entities: instance.Entities,
		Actor:    actor,
		Switcher: s,
	}

	// Выполняем хендлер
	result, err := handler(ctx, eventData)
	if err != nil {
		logger.Log.Errorf("Error handling event %s: %v", genericEvent.Event, err)
		return
	}

	// Логируем результат, если есть
	if result.Msg != "" {
		s.AddLog(result.Msg, result.MsgType)
	}
}
