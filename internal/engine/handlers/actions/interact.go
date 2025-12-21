package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/internal/systems/interaction"
	_ "cognitive-server/internal/systems/interaction/rules"
	"cognitive-server/pkg/api"
)

func HandleInteract(ctx handlers.Context, p api.EntityPayload) (handlers.Result, error) {
	// 1. Валидация через TargetingSystem
	// Дистанция 1.5 (можно нажать рычаг под ногами или рядом)
	// LOS = false, так как если мы стоим на лестнице, мы её "чувствуем", даже если под ногами
	res := systems.ValidateInteraction(ctx.Actor, domain.EntityID(p.TargetID), 1.5, false, ctx.Finder, ctx.World)

	if !res.Valid {
		return handlers.Result{Msg: res.Message, MsgType: "ERROR"}, nil
	}

	return interaction.Resolve(ctx, res.Target)
}
