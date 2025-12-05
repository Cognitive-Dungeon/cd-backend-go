package systems

import (
	"cognitive-server/internal/domain"
	"fmt"
)

// ApplyAttack рассчитывает урон
func ApplyAttack(attacker, target *domain.Entity) string {
	// 1. Проверка: Есть ли у цели здоровье? (Лестницы бессмертны)
	if target.Stats == nil {
		return fmt.Sprintf("Вы атакуете %s, но это бесполезно.", target.Name)
	}
	if target.Stats.IsDead {
		return fmt.Sprintf("Вы атакуете труп %s, это бесполезно.", target.Name)
	}

	// 2. Получаем силу атакующего
	damage := 1 // Базовый урон, если нет статов
	if attacker.Stats != nil {
		damage = attacker.Stats.Strength
	}

	// 3. Наносим урон
	target.Stats.HP -= damage
	logMsg := fmt.Sprintf("%s наносит %d урона по %s.", attacker.Name, damage, target.Name)

	// 4. Проверка смерти
	if target.Stats.HP <= 0 {
		target.Stats.IsDead = true // Теперь поле внутри Stats

		// Визуальное изменение (если есть рендер)
		if target.Render != nil {
			target.Render.Symbol = "%"
			target.Render.Color = "text-gray-500"
		}

		// Отключаем ИИ
		if target.AI != nil {
			target.AI.IsHostile = false
		}

		logMsg += fmt.Sprintf(" %s погибает.", target.Name)
	}

	return logMsg
}
