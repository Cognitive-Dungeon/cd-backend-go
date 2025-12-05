package systems

import (
	"cognitive-server/internal/domain"
	"fmt"
)

// ApplyAttack рассчитывает урон, применяет его и возвращает лог события
func ApplyAttack(attacker, target *domain.Entity) string {
	damage := attacker.Stats.Strength

	// В будущем тут будет защита (Armor) и уклонение
	target.Stats.HP -= damage

	logMsg := fmt.Sprintf("%s наносит %d урона по %s.", attacker.Name, damage, target.Name)

	if target.Stats.HP <= 0 {
		target.IsDead = true
		target.Symbol = "%"
		target.Color = "text-gray-500"
		target.IsHostile = false // Мертвые не кусаются
		logMsg += fmt.Sprintf(" %s погибает.", target.Name)
	}

	return logMsg
}
