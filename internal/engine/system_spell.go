package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"math"
)

type SpellSystem struct {
	Instance *Instance
	Bus      *eventbus.EventBus
	Registry *SpellRegistry
}

func NewSpellSystem(inst *Instance, bus *eventbus.EventBus, reg *SpellRegistry) *SpellSystem {
	sys := &SpellSystem{
		Instance: inst,
		Bus:      bus,
		Registry: reg,
	}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventCastRequest), sys.onCastRequest)
	return sys
}

func (s *SpellSystem) onCastRequest(ev enums.CastRequestEvent) {
	logger.Log.Debugf("SpellSystem: Handling Cast SpellID=%d Caster=%s Target=%s", ev.SpellID, ev.Caster, ev.Target)

	// 1. Получаем данные (Компоненты)
	casterPos := s.Instance.GetPosition(ev.Caster)
	casterStats := s.Instance.GetStats(ev.Caster)
	targetPos := s.Instance.GetPosition(ev.Target)

	if casterPos == nil || casterStats == nil {
		logger.Log.Warn("SpellSystem: Caster Invalid (missing pos/stats)")
		return
	}
	if casterStats.IsDead {
		logger.Log.Warn("SpellSystem: Caster is Dead")
		return
	}

	// 2. Спелл
	spell, found := s.Registry.Get(ev.SpellID)
	if !found {
		logger.Log.Errorf("SpellSystem: Spell %d not found in registry", ev.SpellID)
		return
	}

	// 3. Дистанция
	if targetPos != nil {
		dist := distance(casterPos.TilePos, targetPos.TilePos)
		if dist > spell.Range {
			logger.Log.Warnf("SpellSystem: Out of range (Dist: %.1f, Range: %.1f)", dist, spell.Range)
			return
		}
	} else if !spell.Attributes.Has(types.SpellAttrTargetSelf) {
		// Если цели нет и это не селф-каст -> ошибка
		logger.Log.Warn("SpellSystem: Target required but missing")
		return
	}

	// 4. Мана
	if spell.CostType == types.SpellResourceMana {
		if casterStats.Mana < spell.CostValue {
			logger.Log.Warn("SpellSystem: Not enough mana")
			return
		}
		casterStats.Mana -= spell.CostValue
	}

	logger.Log.Debugf("SpellSystem: Cast Validated. Dispatching %d effects...", len(spell.Effects))

	// 5. Диспатч Эффектов (PUBLISH EVENTS)
	for _, effect := range spell.Effects {
		switch effect.Type {
		case types.SpellEffectSchoolDamage:
			logger.Log.Debugf(" -> Dispatching DAMAGE (Value: %d)", effect.BaseValue)
			s.Bus.Publish(eventbus.EventType(enums.EventDamageApply), enums.DamageEvent{
				Source: ev.Caster,
				Target: ev.Target,
				Amount: effect.BaseValue,
				School: types.SpellSchoolFire, // В будущем брать из SpellDef
			})

		case types.SpellEffectHeal:
			logger.Log.Debugf(" -> Dispatching HEAL (Value: %d)", effect.BaseValue)
			s.Bus.Publish(eventbus.EventType(enums.EventHealApply), enums.HealEvent{
				Source: ev.Caster,
				Target: ev.Target,
				Amount: effect.BaseValue,
			})

		default:
			logger.Log.Warnf("Unknown Effect Type: %d", effect.Type)
		}
	}
}

func distance(p1, p2 types.TilePos) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}
