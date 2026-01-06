package systems

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/eventbus"
)

// System — это контракт для любой игровой механики (Бой, Движение, Инвентарь, AI).
type System interface {
	// Init регистрирует обработчики событий этой системы в шине.
	Init(bus *eventbus.EventBus)

	// Name возвращает имя системы для логов/отладки (опционально, но полезно).
	Name() string
}

type SystemHandler[T any] func(ev T)

type systemBus struct {
	eventBus *eventbus.EventBus
}

// BaseSystem — это миксин (встраиваемая структура), который дает доступ к шине.
type BaseSystem struct {
	bus systemBus
}

// initBus сохраняет шину. Вызывать в начале Init() каждой системы.
func (b *BaseSystem) initBus(bus *eventbus.EventBus) systemBus {
	if bus == nil {
		panic("attempt to initialize BaseSystem with nil bus")
	}
	currentBus := systemBus{eventBus: bus}
	b.bus = currentBus
	return currentBus
}

// bind привязывает событие шины к системе
func bind[T any](systemBus systemBus, et enums.EventType, handler SystemHandler[T]) {
	if systemBus.eventBus == nil {
		panic("bind called with empty SysBus. You must use the return value of initBus()!")
	}
	eventbus.Subscribe(systemBus.eventBus, eventbus.EventType(et), handler)
}

// emit обертка над Publish, чтобы не кастовать enum каждый раз.
func (b *BaseSystem) emit(et enums.EventType, event any) {
	b.bus.eventBus.Publish(eventbus.EventType(et), event)
}

// publishLog - вспомогательный метод для отправки логов
func (b *BaseSystem) publishLog(w *domain.GameWorld, text, msgType string) {
	b.emit(enums.EventTypeLogMessage, domain.LogMessage{
		World: w,
		Text:  text,
		Type:  msgType,
	})
}
