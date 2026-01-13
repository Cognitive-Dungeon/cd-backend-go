package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
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

	// 1. Визуал (Труп)
	render := s.Instance.GetRender(ev.Object)
	if render != nil {
		render.Glyph = types.MakeGlyph(0x888888, '%') // Серый %
	}

	// 2. Имя
	name := s.Instance.GetName(ev.Object)
	if name != nil {
		name.Name += " (Dead)"
	}

	// 3. (В будущем) Дроп лута
	// s.LootSystem.GenerateLoot(ev.Unit)

	// 4. (В будущем) Удаление AI
	// s.Instance.RemoveController(ev.Unit)
}
