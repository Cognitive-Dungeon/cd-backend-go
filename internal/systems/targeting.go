package systems

import (
	"cognitive-server/internal/domain"
)

// EntityProvider — интерфейс для поиска сущностей (чтобы не зависеть от GameService напрямую)
type EntityProvider interface {
	GetEntity(id domain.EntityID) *domain.Entity
}

// ValidationResult — результат проверки цели
type ValidationResult struct {
	Target  *domain.Entity
	Valid   bool
	Message string // Сообщение об ошибке, если Valid == false
}

// ValidateInteraction проверяет, может ли actor взаимодействовать с targetID.
//
// Параметры:
// - rangeLimit: максимальная дистанция (1.5 для соседней клетки/диагонали).
// - needLOS: нужна ли прямая видимость (true для атаки, false для рычагов под ногами).
func ValidateInteraction(actor *domain.Entity, targetID domain.EntityID, rangeLimit float64, needLOS bool, finder EntityProvider, w *domain.GameWorld) ValidationResult {
	// 1. Поиск цели
	target := finder.GetEntity(targetID)
	if target == nil {
		return ValidationResult{Valid: false, Message: "Цель не найдена."}
	}

	// 2. Проверка уровня (сущности должны быть в одном мире)
	if target.Level != actor.Level {
		return ValidationResult{Valid: false, Message: "Цель слишком далеко."}
	}

	// 3. Проверка дистанции
	// DistanceTo возвращает float (1.0 для соседа, ~1.41 для диагонали)
	dist := actor.Pos.DistanceTo(target.Pos)
	if dist > rangeLimit {
		return ValidationResult{Valid: false, Message: "Цель слишком далеко."}
	}

	// 4. Проверка видимости (Line of Sight)
	if needLOS {
		// Если стоим на одной клетке - всегда видим
		if dist > 0 && !HasLineOfSight(w, actor.Pos, target.Pos) {
			return ValidationResult{Valid: false, Message: "Вы не видите цель."}
		}
	}

	return ValidationResult{Target: target, Valid: true}
}
