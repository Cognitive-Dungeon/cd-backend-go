package interaction

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"fmt"
)

var registry []Rule

// Register добавляет правило в глобальный список
func Register(r Rule) {
	registry = append(registry, r)
}

// Resolve находит первое подходящее правило и выполняет его.
func Resolve(ctx handlers.Context, target *domain.Entity) (handlers.Result, error) {
	for _, rule := range registry {
		if rule.Match(ctx.World, ctx.Actor, target) {
			return rule.Apply(ctx, target)
		}
	}

	return handlers.Result{
		Msg:     fmt.Sprintf("С %s невозможно взаимодействовать.", target.Name),
		MsgType: "INFO",
	}, nil
}
