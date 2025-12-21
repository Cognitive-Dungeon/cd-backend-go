package interaction

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
)

// Rule описывает одно возможное взаимодействие.
type Rule interface {
	// Name возвращает имя правила для отладки
	Name() string

	// Match проверяет, применимо ли это правило.
	// Передаем World, чтобы правило могло проверить наличие компонентов в ECS.
	Match(w *domain.GameWorld, actor, target *domain.Entity) bool

	// Apply выполняет действие.
	Apply(ctx handlers.Context, target *domain.Entity) (handlers.Result, error)
}
