package systems

import (
	"cognitive-server/internal/core/types/enums"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/types/glyph"
)

type DeathSystem struct {
	Instance *ecs2.Instance
}

func NewDeathSystem(inst *ecs2.Instance, bus *eventbus.EventBus) *DeathSystem {
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
	renderStore := ecs.GetStorage[components.RenderComponent](world, components.CID_Render)
	if render := renderStore.Get(id); render != nil {
		render.Glyph = glyph.MakeGlyph(0x888888, '%') // Серый %
	}

	// 2. Имя (Добавляем пометку)
	nameStore := ecs.GetStorage[components.NameComponent](world, components.CID_Name)
	if name := nameStore.Get(id); name != nil {
		name.Name += " (Dead)"
	}

	// 3. Забираем управление
	// Удаляем ControllerComponent, чтобы игрок или AI больше не могли управлять сущностью.
	ecs.GetStorage[components.ControllerComponent](world, components.CID_Controller).Delete(id)

	// 4. Очистка намерений (вдруг он хотел скастовать или пойти в этом кадре)
	ecs.GetStorage[components.IntentMove](world, components.CID_IntentMove).Delete(id)
	ecs.GetStorage[components.IntentCast](world, components.CID_IntentCast).Delete(id)
}
