package engine

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
)

type DamageSystem struct {
	Instance *Instance
	Bus      *eventbus.EventBus
}

func NewDamageSystem(inst *Instance, bus *eventbus.EventBus) *DamageSystem {
	sys := &DamageSystem{Instance: inst, Bus: bus}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventDamageApply), sys.onDamage)
	return sys
}

func (s *DamageSystem) onDamage(ev enums.DamageEvent) {
	targetStats := s.Instance.GetStats(ev.Target)
	if targetStats == nil || targetStats.IsDead {
		return
	}

	// 1. Расчет митигации (Броня, Резисты)
	// В будущем тут будет: damage = damage * (1 - armor/100)
	finalDamage := ev.Amount

	// 2. Применение
	targetStats.Health -= finalDamage

	logger.Log.Infof("DMG: %s took %d damage from %s", ev.Target, finalDamage, ev.Source)

	// 3. Проверка на смерть
	if targetStats.Health <= 0 {
		targetStats.Health = 0
		targetStats.IsDead = true

		// Публикуем событие смерти.
		// DamageSystem НЕ должна менять визуал или удалять объект.
		s.Bus.Publish(eventbus.EventType(enums.EventObjectDied), enums.ObjectDiedEvent{
			Object: ev.Target,
			Killer: ev.Source,
		})
	}
}
