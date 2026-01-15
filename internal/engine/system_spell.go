package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"math"
	"time"
)

// SystemSpellInput обрабатывает запросы на каст (CmdCast).
// Проверяет: наличие спелла в книге, кулдауны, GCD.
// Генерирует: IntentCast.
func SystemSpellInput(w *ecs.World, registry *SpellRegistry) {
	cb := ecs.NewCommandBuffer(w)
	now := float64(time.Now().UnixMilli())

	// Итерируемся по тем, кто хочет кастовать и имеет книгу заклинаний
	for id, join := range ecs.View2[CmdCast, SpellbookComponent](w, CID_CmdCast, CID_Spellbook) {
		cmd := join.First
		spellbook := join.Second

		// 1. Проверка существования спелла в базе
		_, found := registry.Get(cmd.SpellID)
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
		ecs.Add(cb, CID_IntentCast, id, IntentCast{
			SpellID:  cmd.SpellID,
			TargetID: cmd.TargetID,
		})
	}

	cb.Execute()
}

// SystemSpellLogic применяет валидированные намерения.
// Проверяет: дистанцию, наличие цели, ресурсы (мана).
// Генерирует: Урон/Хил (события), запускает КД.
func SystemSpellLogic(w *ecs.World, registry *SpellRegistry, bus *eventbus.EventBus) {
	now := float64(time.Now().UnixMilli())

	// Итерируемся: IntentCast + Position + Stats + Spellbook
	// Нам нужно много компонентов, поэтому используем View2 и добираем остальное через Get
	for id, join := range ecs.View2[IntentCast, PositionComponent](w, CID_IntentCast, CID_Position) {
		intent := join.First
		pos := join.Second

		// Получаем определение спелла (оно точно есть, проверено в Input)
		spellDef, _ := registry.Get(intent.SpellID)

		// Получаем недостающие компоненты
		stats := ecs.GetStorage[StatsComponent](w, CID_Stats).Get(id)
		spellbook := ecs.GetStorage[SpellbookComponent](w, CID_Spellbook).Get(id)

		if stats == nil || spellbook == nil {
			continue
		}
		if stats.IsDead {
			continue
		}

		// --- Валидация Цели и Дистанции ---
		targetECS := ecs.EntityID(intent.TargetID)
		targetPos := ecs.GetStorage[PositionComponent](w, CID_Position).Get(targetECS)

		// Если цель нужна, но ее нет или она далеко
		if targetPos != nil {
			dist := distance(pos.TilePos, targetPos.TilePos)
			if dist > spellDef.Range {
				continue // Out of range
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
				bus.Publish(eventbus.EventType(enums.EventDamageApply), enums.DamageEvent{
					Source: types.ObjectGuid(id),
					Target: effectTarget,
					Amount: effect.BaseValue,
					School: types.SpellSchoolFire,
				})

			case types.SpellEffectHeal:
				bus.Publish(eventbus.EventType(enums.EventHealApply), enums.HealEvent{
					Source: types.ObjectGuid(id),
					Target: effectTarget,
					Amount: effect.BaseValue,
				})
			}
		}

		logger.Log.Infof("Entity %d cast %s on %d", id, spellDef.Name, intent.TargetID)
	}
}

func distance(p1, p2 types.TilePos) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}
