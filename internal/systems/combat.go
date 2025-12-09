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
		// Серверный лог (уровень Warn, т.к. это нештатная ситуация)
		combatLogger.Warn("Attack failed: target has no StatsComponent.")
		// Клиентский лог
		return fmt.Sprintf("Вы атакуете %s, но это бесполезно.", target.Name)
	}
	if target.Stats.IsDead {
		// Серверный лог (уровень Info, т.к. это штатная, хоть и бесполезная, ситуация)
		combatLogger.Info("Attack ineffective: target is already dead.")
		// Клиентский лог
		return fmt.Sprintf("Вы пинаете труп %s.", target.Name)
	}

	// --- Основная логика боя ---

	damage := 1
	if attacker.Stats != nil {
		damage = attacker.Stats.Strength
	}

	hpBefore := target.Stats.HP
	died := target.Stats.TakeDamage(damage)
	hpAfter := target.Stats.HP

	// Записываем основное событие на сервер с максимумом деталей.
	combatLogger.WithFields(logrus.Fields{
		"damage_dealt": damage,
		"hp_before":    hpBefore,
		"hp_after":     hpAfter,
		"target_died":  died,
	}).Info("Attack resolved.")

	// --- Формируем сообщение для клиента ---

	logMsg := fmt.Sprintf("%s наносит %d урона по %s.", attacker.Name, damage, target.Name)

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
