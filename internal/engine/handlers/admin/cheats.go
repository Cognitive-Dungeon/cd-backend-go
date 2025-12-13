package admin

import (
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/dungeon"
	"fmt"
)

// AdminTeleportPayload: { "x": 10, "y": 10, "level": 1 }
type TeleportPayload struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Level int `json:"level"` // Опционально
}

func HandleTeleport(ctx handlers.Context, p TeleportPayload) (handlers.Result, error) {
	// 1. Смена уровня, если нужно
	if p.Level != 0 && p.Level != ctx.Actor.Level {
		ctx.Switcher.ChangeLevel(ctx.Actor, p.Level, "") // "" targetPosID -> force coords later
		// Координаты обновятся в следующем цикле или тут же, если Switcher синхронный.
		// Но в нашей архитектуре ChangeLevel перемещает в дефолтную точку.
		// Для точного телепорта лучше реализовать метод ForcePosition в GameService.
	}

	// 2. Перемещение внутри уровня
	err := ctx.World.UpdateEntityPos(ctx.Actor, p.X, p.Y)
	if err != nil {
		return handlers.Result{Msg: fmt.Sprintf("Teleport failed: %v", err), MsgType: "ERROR"}, nil
	}

	// Сброс кэша видимости
	if ctx.Actor.Vision != nil {
		ctx.Actor.Vision.IsDirty = true
		ctx.Actor.Vision.CachedVisibleTiles = nil
	}

	return handlers.Result{Msg: "⚡ Teleported via Admin Magic", MsgType: "INFO"}, nil
}

// AdminSpawnPayload: { "template": "orc" }
type SpawnPayload struct {
	Template string `json:"template"`
}

func HandleSpawn(ctx handlers.Context, p SpawnPayload) (handlers.Result, error) {
	// Ищем врага
	if tmpl, ok := dungeon.EnemyTemplates[p.Template]; ok {
		// Спавним рядом с игроком
		pos := ctx.Actor.Pos.Shift(1, 0)
		if ctx.World.Map[pos.Y][pos.X].IsWall {
			pos = ctx.Actor.Pos // Fallback под ноги
		}

		enemy := tmpl.SpawnEntity(pos, ctx.Actor.Level)
		ctx.AddGlobalEntity(&enemy)
		return handlers.Result{Msg: fmt.Sprintf("Spawned %s", p.Template), MsgType: "INFO"}, nil
	}

	// Ищем предмет
	if tmpl, ok := dungeon.ItemTemplates[p.Template]; ok {
		pos := ctx.Actor.Pos // Под ноги
		item := tmpl.SpawnItem(pos, ctx.Actor.Level)
		ctx.AddGlobalEntity(item)
		return handlers.Result{Msg: fmt.Sprintf("Spawned item %s", p.Template), MsgType: "INFO"}, nil
	}

	return handlers.Result{Msg: "Unknown template", MsgType: "ERROR"}, nil
}

func HandleHeal(ctx handlers.Context) (handlers.Result, error) {
	if ctx.Actor.Stats != nil {
		ctx.Actor.Stats.HP = ctx.Actor.Stats.MaxHP
		ctx.Actor.Stats.Stamina = ctx.Actor.Stats.MaxStamina
		ctx.Actor.Stats.IsDead = false
	}
	return handlers.Result{Msg: "❤️ Fully Healed", MsgType: "INFO"}, nil
}

type KillPayload struct {
	TargetID string `json:"targetId"`
}

func HandleKill(ctx handlers.Context, p KillPayload) (handlers.Result, error) {
	target := ctx.Finder.GetEntity(p.TargetID)
	if target == nil {
		return handlers.Result{Msg: "Target not found", MsgType: "ERROR"}, nil
	}
	if target.Stats != nil {
		target.Stats.TakeDamage(9999)
	}
	return handlers.Result{Msg: fmt.Sprintf("💀 Smited %s", target.Name), MsgType: "COMBAT"}, nil
}

func HandleToggleOmni(ctx handlers.Context) (handlers.Result, error) {
	if ctx.Actor.Vision == nil {
		return handlers.Result{Msg: "No vision component", MsgType: "ERROR"}, nil
	}

	// Переключаем флаг
	ctx.Actor.Vision.Omniscient = !ctx.Actor.Vision.Omniscient

	// Сбрасываем кэш, чтобы пересчитать видимость немедленно
	ctx.Actor.Vision.IsDirty = true
	ctx.Actor.Vision.CachedVisibleTiles = nil

	status := "OFF"
	if ctx.Actor.Vision.Omniscient {
		status = "ON"
	}

	return handlers.Result{
		Msg:     fmt.Sprintf("👁️ God Vision toggled %s", status),
		MsgType: "INFO",
	}, nil
}
