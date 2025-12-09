package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/logger"
	"encoding/json"
)

// processEvent - является точкой входа для обработки событий, возвращенных хендлерами.
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

	// Создаем контекст для события
	ctx := handlers.Context{
		Finder:   s,
		World:    nil,        // Будет определен внутри хендлера, если нужно (или передадим текущий мир актора)
		Entities: s.Entities, // Глобальный список
		Actor:    actor,
		Worlds:   s.Worlds, // Передаем все миры
		AddGlobalEntity: func(e *domain.Entity) {
			s.Entities = append(s.Entities, e)
		},
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
