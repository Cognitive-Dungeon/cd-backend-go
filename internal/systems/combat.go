package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

func ApplyAttack(attacker, target *domain.Entity) string {
	combatLogger := logger.Log.WithFields(logrus.Fields{
		"component":     "combat_system",
		"attacker_id":   attacker.ID,
		"attacker_name": attacker.Name,
		"target_id":     target.ID,
		"target_name":   target.Name,
	})

	// --- Проверка граничных условий ---

	if target.Stats == nil {
		combatLogger.Warn("Attack failed: target has no StatsComponent.")
		return fmt.Sprintf("Вы атакуете %s, но это бесполезно.", target.Name)
	}
	if target.Stats.IsDead {
		combatLogger.Info("Attack ineffective: target is already dead.")
		return fmt.Sprintf("Вы пинаете труп %s.", target.Name)
	}

	// --- Расчёт урона ---

	// Базовый урон = Strength атакующего
	baseDamage := 1
	if attacker.Stats != nil {
		baseDamage = attacker.Stats.Strength
	}

	// Бонус от экипированного оружия
	weaponDamage := 0
	if attacker.Equipment != nil && attacker.Equipment.Weapon != nil {
		if attacker.Equipment.Weapon.Item != nil {
			weaponDamage = attacker.Equipment.Weapon.Item.Damage
		}
	}

	totalDamage := baseDamage + weaponDamage

	// --- Расчёт защиты ---

	defense := 0
	if target.Equipment != nil && target.Equipment.Armor != nil {
		if target.Equipment.Armor.Item != nil {
			defense = target.Equipment.Armor.Item.Defense
		}
	}

	// Финальный урон (минимум 1)
	finalDamage := totalDamage - defense
	if finalDamage < 1 {
		finalDamage = 1
	}

	hpBefore := target.Stats.HP
	died := target.Stats.TakeDamage(finalDamage)
	hpAfter := target.Stats.HP

	// Логируем событие
	combatLogger.WithFields(logrus.Fields{
		"base_damage":   baseDamage,
		"weapon_damage": weaponDamage,
		"defense":       defense,
		"final_damage":  finalDamage,
		"hp_before":     hpBefore,
		"hp_after":      hpAfter,
		"target_died":   died,
	}).Info("Attack resolved.")

	// --- Формируем сообщение для клиента ---

	logMsg := fmt.Sprintf("%s наносит %d урона по %s.", attacker.Name, finalDamage, target.Name)

	// TODO: Генерация событий для живых предметов
	// Если у атакующего экипировано живое оружие (IsSentient=true),
	// генерируем событие ITEM_EVENT для микросервиса:
	// - event: "attack"
	// - itemId: weapon.ID
	// - personality: weapon.Item.Personality
	// - context: {damage: finalDamage, targetName: target.Name}
	// Микросервис получит это событие через WebSocket и вернёт диалог

	if died {
		// Визуально меняем труп
		if target.Render != nil {
			target.Render.Symbol = "%"
			target.Render.Color = "text-gray-500"
		}
		// "Успокаиваем" ИИ трупа
		if target.AI != nil {
			target.AI.IsHostile = false
		}
		logMsg += fmt.Sprintf(" %s погибает.", target.Name)
	}

	return logMsg
}

// CreateLootBag создаёт "мешок с лутом" из инвентаря мёртвой сущности
func CreateLootBag(deadEntity *domain.Entity) *domain.Entity {
	// Если у сущности нет инвентаря или он пустой, не создаём мешок
	if deadEntity.Inventory == nil || len(deadEntity.Inventory.Items) == 0 {
		return nil
	}

	return &domain.Entity{
		ID:    domain.GenerateID(),
		Type:  domain.EntityTypeItem,
		Name:  "Останки " + deadEntity.Name,
		Pos:   deadEntity.Pos,
		Level: deadEntity.Level,
		Render: &domain.RenderComponent{
			Symbol: "☠",
			Color:  "#8B4513",
		},
		Item: &domain.ItemComponent{
			Category:      domain.ItemCategoryContainer,
			IsTransparent: false,
			IsIntangible:  false,
		},
		Inventory: &domain.InventoryComponent{
			Items:    deadEntity.Inventory.Items,
			MaxSlots: 999,
		},
	}
}
