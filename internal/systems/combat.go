package systems

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/eventbus"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// CombatSystem реализует интерфейс System.
// Она разделяет монолитную логику боя на реактивные шаги.
type CombatSystem struct {
	BaseSystem
}

func (s *CombatSystem) Name() string {
	return "CombatSystem"
}

// Init регистрирует методы системы в шине событий.
func (s *CombatSystem) Init(bus *eventbus.EventBus) {
	combatBus := s.initBus(bus)

	bind(combatBus, enums.EventTypeAttackRequested, s.onAttackRequested)
	bind(combatBus, enums.EventTypeDamageInflicted, s.onDamageInflicted)
	bind(combatBus, enums.EventTypeEntityDied, s.onEntityDied)
}

// --- 1. ФАЗА: ВАЛИДАЦИЯ И РАСЧЕТ (Чистая логика) ---

func (s *CombatSystem) onAttackRequested(ev domain.AttackRequested) {
	attacker := ev.Attacker
	target := ev.Target
	world := ev.World

	// Логгер
	combatLog := logger.Log.WithFields(logrus.Fields{
		"component":   "combat_system",
		"attacker_id": attacker.ID,
		"target_id":   target.ID,
	})

	// 1.1. ВАЛИДАЦИЯ СОСТОЯНИЯ
	if target.Stats == nil {
		s.publishLog(ev.World, fmt.Sprintf("Вы атакуете %s, но это бесполезно.", target.Name), "ERROR")
		return
	}

	if target.Stats.IsDead {
		s.publishLog(ev.World, fmt.Sprintf("Вы пинаете труп %s.", target.Name), "INFO")
		// Даже пинание трупа требует времени!
		if attacker.AI != nil {
			attacker.AI.Wait(domain.TimeCostWait)
		}
		return
	}

	// 1.2. ВАЛИДАЦИЯ ДИСТАНЦИИ И ВИДИМОСТИ
	// (Раньше это было в handler, теперь тут)

	// Хардкод дистанции атаки (1.5 клетки - это соседние, включая диагональ)
	const MeleeRange = 1.5

	dist := attacker.Pos.DistanceTo(target.Pos)
	if dist > MeleeRange {
		combatLog.Debug("Attack failed: target out of range")
		// Если это был игрок, пишем ему ошибку
		if attacker.Type == domain.EntityTypePlayer {
			s.publishLog(world, "Цель слишком далеко.", "ERROR")
		}
		return
	}

	if !HasLineOfSight(world, attacker.Pos, target.Pos) {
		combatLog.Debug("Attack failed: no LOS")
		if attacker.Type == domain.EntityTypePlayer {
			s.publishLog(world, "Вы не видите цель.", "ERROR")
		}
		return
	}

	// 1.3. СПИСАНИЕ ВРЕМЕНИ (!!! Самое важное !!!)
	// Мы списываем время ДО расчета урона.
	cost := domain.TimeCostAttackLight

	if attacker.AI != nil {
		attacker.AI.Wait(cost)

		combatLog.WithFields(logrus.Fields{
			"attacker": attacker.Name,
			"cost":     cost,
			"new_tick": attacker.AI.NextActionTick,
		}).Debug("Time charged for attack")
	}

	// 1.4. РАСЧЁТ УРОНА
	baseDamage := 1
	if attacker.Stats != nil {
		baseDamage = attacker.Stats.Strength
	}

	weaponDamage := 0
	if attacker.Equipment != nil && attacker.Equipment.Weapon != nil {
		if attacker.Equipment.Weapon.Item != nil {
			weaponDamage = attacker.Equipment.Weapon.Item.Damage
		}
	}
	totalDamage := baseDamage + weaponDamage

	// 1.5. РАСЧЁТ ЗАЩИТЫ
	defense := 0
	if target.Equipment != nil && target.Equipment.Armor != nil {
		if target.Equipment.Armor.Item != nil {
			defense = target.Equipment.Armor.Item.Defense
		}
	}

	finalDamage := totalDamage - defense
	if finalDamage < 0 {
		finalDamage = 0
	}

	// 1.6. ПУБЛИКАЦИЯ
	s.emit(enums.EventTypeDamageInflicted, domain.DamageInflicted{
		Attacker: attacker,
		Target:   target,
		Amount:   finalDamage,
		World:    ev.World,
	})
}

// --- 2. ФАЗА: ПРИМЕНЕНИЕ И ЛОГИРОВАНИЕ (Изменение стейта) ---

func (s *CombatSystem) onDamageInflicted(ev domain.DamageInflicted) {
	if ev.Target.Stats == nil {
		return
	}

	// 2.1. Изменение HP
	hpBefore := ev.Target.Stats.HP
	died := ev.Target.Stats.TakeDamage(ev.Amount)
	hpAfter := ev.Target.Stats.HP

	// Структурный лог (для админов/файла)
	logger.Log.WithFields(logrus.Fields{
		"event":     "damage",
		"target":    ev.Target.Name,
		"damage":    ev.Amount,
		"hp_before": hpBefore,
		"hp_after":  hpAfter,
		"died":      died,
	}).Info("Damage applied")

	// 2.2. Сообщение игроку (Лог чата)
	var userMsg string
	if ev.Attacker != nil {
		userMsg = fmt.Sprintf("%s наносит %d урона по %s.", ev.Attacker.Name, ev.Amount, ev.Target.Name)
	} else {
		userMsg = fmt.Sprintf("%s получает %d урона.", ev.Target.Name, ev.Amount)
	}

	s.publishLog(ev.World, userMsg, "COMBAT")

	// 2.3. TODO: Живые предметы
	// Здесь можно проверить ev.Attacker.Equipment.Weapon.Item.IsSentient
	// И опубликовать событие EventTypeItemSentientReaction

	// 2.4. Триггер смерти
	if died {
		s.emit(enums.EventTypeEntityDied, domain.EntityDied{
			Entity: ev.Target,
			Killer: ev.Attacker, // Может быть nil (если ловушка)
			World:  ev.World,
		})
	}
}

// --- 3. ФАЗА: ПОСЛЕДСТВИЯ СМЕРТИ (Cleanup & Loot) ---

func (s *CombatSystem) onEntityDied(ev domain.EntityDied) {
	victim := ev.Entity

	// 3.1. Визуал (Труп)
	if victim.Render != nil {
		victim.Render.Symbol = '%'
		victim.Render.Color = "#555555" // Серый, как text-gray-500
	}

	// 3.2. AI (Успокаиваем)
	if victim.AI != nil {
		victim.AI.IsHostile = false
		victim.AI.State = domain.AIStateIdle
	}

	// 3.3. Лог смерти
	s.publishLog(ev.World, fmt.Sprintf("%s погибает.", victim.Name), "COMBAT")

	// 3.4. Генерация лута
	if victim.Inventory != nil && len(victim.Inventory.Items) > 0 {
		lootBag := CreateLootBag(victim)
		if lootBag != nil {
			ev.World.RegisterEntity(lootBag)
			ev.World.AddEntity(lootBag)

			s.publishLog(ev.World, "Вещи падают на землю.", "INFO")
		}
		// Очищаем инвентарь трупа во избежание дюпа
		victim.Inventory.Items = nil
	}
}

// --- HELPERS ---

// CreateLootBag вынесен сюда (или импортируется из systems/loot.go, если есть)
func CreateLootBag(deadEntity *domain.Entity) *domain.Entity {
	if deadEntity.Inventory == nil || len(deadEntity.Inventory.Items) == 0 {
		return nil
	}

	// Shallow copy слайса предметов, чтобы перенести их в мешок
	// (Сами предметы как Entity остаются теми же, меняется владелец)
	items := make([]*domain.Entity, len(deadEntity.Inventory.Items))
	copy(items, deadEntity.Inventory.Items)

	return &domain.Entity{
		ID:    domain.EntityID(fmt.Sprintf("loot_%s_%d", deadEntity.ID, deadEntity.Level)), // Генерируем ID
		Type:  domain.EntityTypeItem,
		Name:  "Останки " + deadEntity.Name,
		Pos:   deadEntity.Pos,
		Level: deadEntity.Level,
		Render: &domain.RenderComponent{
			Symbol: ',',
			Color:  "#8B4513",
		},
		Item: &domain.ItemComponent{
			Category:      domain.ItemCategoryContainer,
			IsTransparent: true,
			IsIntangible:  true,
		},
		Inventory: &domain.InventoryComponent{
			Items:    items,
			MaxSlots: 999,
		},
	}
}
