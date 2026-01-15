package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
)

type DeathSystem struct {
	Instance *Instance
}

func NewDeathSystem(inst *Instance, bus *eventbus.EventBus) *DeathSystem {
	sys := &DeathSystem{Instance: inst}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventObjectDied), sys.onUnitDied)
	return sys
}

func (s *DeathSystem) onUnitDied(ev enums.ObjectDiedEvent) {
	logger.Log.Infof("💀 UNIT DIED: %s (Killer: %s)", ev.Object, ev.Killer)

	world := s.Instance.World
	id := ecs.EntityID(ev.Object)

	// 1. Визуал (Превращаем в труп)
	// Получаем доступ к компоненту Render напрямую через ECS
	renderStore := ecs.GetStorage[RenderComponent](world, CID_Render)
	if render := renderStore.Get(id); render != nil {
		render.Glyph = types.MakeGlyph(0x888888, '%') // Серый %
	}

	// 2. Имя (Добавляем пометку)
	nameStore := ecs.GetStorage[NameComponent](world, CID_Name)
	if name := nameStore.Get(id); name != nil {
		name.Name += " (Dead)"
	}

	// 3. Забираем управление
	// Удаляем ControllerComponent, чтобы игрок или AI больше не могли управлять сущностью.
	// В новой архитектуре это делается мгновенно и безопасно
}
