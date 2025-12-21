package rules

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems/interaction"
	"fmt"
)

type StairsRule struct{}

func (StairsRule) Name() string { return "UseStairs" }

func (StairsRule) Match(w *domain.GameWorld, actor, target *domain.Entity) bool {
	// ПРОВЕРКА: Есть ли у цели компонент Transition в ECS?
	_, ok := domain.GetComponent[domain.TransitionComponent](w.Components, target.ID)
	return ok
}

func (StairsRule) Apply(ctx handlers.Context, target *domain.Entity) (handlers.Result, error) {
	// ПОЛУЧЕНИЕ: Достаем компонент
	trans, ok := domain.GetComponent[domain.TransitionComponent](ctx.World.Components, target.ID)
	if !ok {
		return handlers.Result{}, fmt.Errorf("missing transition component")
	}

	// ЛОГИКА
	ctx.Switcher.ChangeLevel(ctx.Actor, trans.TargetLevel, trans.TargetPosID)

	return handlers.Result{
		Msg:     fmt.Sprintf("Вы воспользовались %s.", target.Name),
		MsgType: "INFO",
	}, nil
}

func init() {
	interaction.Register(StairsRule{})
}
