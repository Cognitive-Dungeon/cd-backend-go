package systems

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/grid"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/worldmap"
	"time"
)

// InputSpellSystem обрабатывает запросы на каст (CmdCast).
// Проверяет: наличие спелла в книге, кулдауны, GCD.
// Генерирует: IntentCast.
func InputSpellSystem(ctx ecs2.InputContext) {
	now := float64(time.Now().UnixMilli())

	// Итерируемся по тем, кто хочет кастовать и имеет книгу заклинаний
	for id, join := range ecs.View2[components.CmdCast, components.SpellbookComponent](ctx.World, components.CID_CmdCast, components.CID_Spellbook) {
		cmd := join.First
		spellbook := join.Second

		// 1. Проверка существования спелла в базе
		_, found := ctx.SpellRegistry.Get(cmd.SpellID)
		if !found {
			logger.Log.Warnf("Entity %d tried to cast unknown spell %d", id, cmd.SpellID)
			continue
		}

		// 2. Проверка: знает ли сущность этот спелл?
		known := false
		for _, s := range spellbook.KnownSpells {
			if types.SpellID(s) == cmd.SpellID {
				known = true
				break
			}
		}
		if !known {
			continue // Игрок пытается читерить или рассинхрон
		}

		// 3. Проверка кулдаунов (GCD и CD спелла)
		if now < spellbook.GCD {
			continue // ГКД еще не прошел
		}
		if cdEnd, ok := spellbook.Cooldowns[uint32(cmd.SpellID)]; ok && now < cdEnd {
			continue // Спелл на кулдауне
		}

		// Если все ок — создаем Намерение (Intent)
		// Мы пока не списываем ресурсы и не вешаем КД, это делается в фазе Logic
		ecs.Add(ctx.Commands, components.CID_IntentCast, id, components.IntentCast{
			SpellID:  cmd.SpellID,
			TargetID: cmd.TargetID,
		})
	}
}

// LogicSpellLogic применяет валидированные намерения.
// Проверяет: дистанцию, наличие цели, ресурсы (мана).
// Генерирует: Урон/Хил (события), запускает КД.
func LogicSpellLogic(ctx ecs2.LogicContext) {
	now := float64(time.Now().UnixMilli())

	// Итерируемся: IntentCast + Position + Stats + Spellbook
	// Нам нужно много компонентов, поэтому используем View2 и добираем остальное через Get
	for id, join := range ecs.View2[components.IntentCast, components.PositionComponent](ctx.World, components.CID_IntentCast, components.CID_Position) {
		intent := join.First
		pos := join.Second

		// Получаем определение спелла (оно точно есть, проверено в Input)
		spellDef, _ := ctx.SpellRegistry.Get(intent.SpellID)

		// Получаем недостающие компоненты
		stats := ecs.GetStorage[components.StatsComponent](ctx.World, components.CID_Stats).Get(id)
		spellbook := ecs.GetStorage[components.SpellbookComponent](ctx.World, components.CID_Spellbook).Get(id)

		if stats == nil || spellbook == nil {
			continue
		}
		if stats.IsDead {
			continue
		}

		// --- Валидация Цели и Дистанции ---
		targetECS := ecs.EntityID(intent.TargetID)
		targetPos := ecs.GetStorage[components.PositionComponent](ctx.World, components.CID_Position).Get(targetECS)

		// Если цель нужна, но ее нет или она далеко
		if targetPos != nil {
			distSq := pos.TilePos.DistanceSquared(targetPos.TilePos)
			rangeSq := int64(spellDef.Range * spellDef.Range)
			if distSq > rangeSq {
				continue // Out of range
			}

			// Проверка Line of Sight (Raycast по WorldMap)
			hasLoS := checkLoS(ctx.WorldMap, pos.TilePos, targetPos.TilePos)
			if !hasLoS {
				continue
			}

		} else if !spellDef.Attributes.Has(types.SpellAttrTargetSelf) {
			continue // Цель обязательна, но не найдена
		}

		// --- Проверка Ресурсов ---
		if spellDef.CostType == types.SpellResourceMana {
			if stats.Mana < spellDef.CostValue {
				continue // Not enough mana
			}
		}

		// === ПРИМЕНЕНИЕ (ACTION) ===

		// 1. Списание ресурсов
		if spellDef.CostType == types.SpellResourceMana {
			stats.Mana -= spellDef.CostValue
		}

		// 2. Установка Кулдаунов
		spellbook.GCD = now + 1500 // Глобальный КД 1.5 сек
		if spellDef.Cooldown > 0 {
			if spellbook.Cooldowns == nil {
				spellbook.Cooldowns = make(map[uint32]float64)
			}
			spellbook.Cooldowns[uint32(intent.SpellID)] = now + spellDef.Cooldown
		}

		// 3. Применение эффектов (Публикация событий)
		for _, effect := range spellDef.Effects {
			// Определяем реальную цель эффекта
			effectTarget := intent.TargetID
			if effect.Target == "TARGET_SELF" {
				effectTarget = types.ObjectGuid(id)
			}

			switch effect.Type {
			case types.SpellEffectSchoolDamage:
				ctx.Bus.Publish(eventbus.EventType(enums.EventDamageApply), enums.DamageEvent{
					Source: types.ObjectGuid(id),
					Target: effectTarget,
					Amount: effect.BaseValue,
					School: types.SpellSchoolFire,
				})

			case types.SpellEffectHeal:
				ctx.Bus.Publish(eventbus.EventType(enums.EventHealApply), enums.HealEvent{
					Source: types.ObjectGuid(id),
					Target: effectTarget,
					Amount: effect.BaseValue,
				})
			}
		}
	}
}

func checkLoS(wm *worldmap.World, from, to grid.TilePos) bool {
	blocked := false
	grid.LineExclusive(from, to, func(p grid.TilePos) bool {
		// Проверяем Opaque флаг
		// Нужно конвертировать в geo.Location
		gPos := geo.Pos(int(p.X), int(p.Y), 0)

		if wm.IsOpaqueFast(gPos) {
			blocked = true
			return false // Прерываем луч, стена найдена
		}
		return true
	})
	return !blocked
}
