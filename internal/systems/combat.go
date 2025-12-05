package systems

import (
	"cognitive-server/internal/domain"
	"fmt"
)

func ApplyAttack(attacker, target *domain.Entity) string {
	if target.Stats == nil {
		return fmt.Sprintf("Вы атакуете %s, но это бесполезно.", target.Name)
	}
	if target.Stats.IsDead {
		return fmt.Sprintf("Вы пинаете труп %s.", target.Name)
	}

	damage := 1
	if attacker.Stats != nil {
		damage = attacker.Stats.Strength
	}

	died := target.Stats.TakeDamage(damage)

	logMsg := fmt.Sprintf("%s наносит %d урона по %s.", attacker.Name, damage, target.Name)

	if died {
		if target.Render != nil {
			target.Render.Symbol = "%"
			target.Render.Color = "text-gray-500"
		}
		if target.AI != nil {
			target.AI.IsHostile = false
		}
		logMsg += fmt.Sprintf(" %s погибает.", target.Name)
	}

	return logMsg
}
