package systems

import (
	"cognitive-server/internal/core/types/enums"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
)

type DamageSystem struct {
	Instance *ecs2.Instance
	Bus      *eventbus.EventBus
}

func NewDamageSystem(inst *ecs2.Instance, bus *eventbus.EventBus) *DamageSystem {
	sys := &DamageSystem{Instance: inst, Bus: bus}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventDamageApply), sys.onDamage)
	return sys
}

func (s *DamageSystem) onDamage(ev enums.DamageEvent) {
	// 1. Получаем доступ к компоненту здоровья через ECS
	targetID := ecs.EntityID(ev.Target)

	// Используем глобальный ID компонента CID_Stats для быстрого доступа
	statsStorage := ecs.GetStorage[components.StatsComponent](s.Instance.World, components.CID_Stats)
	targetStats := statsStorage.Get(targetID)

	// Проверяем, существует ли цель и жива ли она
	if targetStats == nil || targetStats.IsDead {
		return
	}

	// 2. Расчет митигации (Броня, Резисты)
	// В будущем тут будет: damage = damage * (1 - armor/100)
	finalDamage := ev.Amount

	// 3. Применение (прямое изменение данных по указателю из ECS)
	targetStats.Health -= finalDamage

	logger.Log.Infof("DMG: %s took %d damage from %s", ev.Target, finalDamage, ev.Source)

	// 4. Проверка на смерть
	if targetStats.Health <= 0 {
		targetStats.Health = 0
		targetStats.IsDead = true

		// Публикуем событие смерти.
		// DamageSystem по-прежнему работает реактивно через EventBus,
		// так как это удобно для цепочки событий (урон -> смерть -> лут).
		s.Bus.Publish(eventbus.EventType(enums.EventObjectDied), enums.ObjectDiedEvent{
			Object: ev.Target,
			Killer: ev.Source,
		})
	}
}
