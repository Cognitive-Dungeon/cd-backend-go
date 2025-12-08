package systems

import (
	"cognitive-server/internal/domain"
	"math"
)

// ComputeNPCAction решает, что делать NPC.
// Возвращает (команда, цель_атаки_если_есть, dx, dy)
func ComputeNPCAction(npc *domain.Entity, player *domain.Entity, w *domain.GameWorld, ents []domain.Entity) (action domain.ActionType, target *domain.Entity, dx, dy int) {
	// 1. Проверка наличия компонентов
	// Если нет мозгов (AI) или тела (Stats) - ничего не делаем
	if npc.AI == nil || npc.Stats == nil {
		return domain.ActionWait, nil, 0, 0
	}

	// 2. Если мертв или не враждебен
	if npc.Stats.IsDead || !npc.AI.IsHostile {
		return domain.ActionWait, nil, 0, 0
	}

	dist := npc.Pos.DistanceTo(player.Pos)

	// 3. Логика дистанции
	if dist > domain.AggroRadius {
		return domain.ActionWait, nil, 0, 0
	}

	if dist <= 1.5 {
		return domain.ActionAttack, player, 0, 0
	}

	moveDx, moveDy := calculateSmartMove(npc, player, w, ents)
	if moveDx == 0 && moveDy == 0 {
		return domain.ActionWait, nil, 0, 0
	}

	return domain.ActionMove, nil, moveDx, moveDy
}

// Внутренние утилиты (приватные для пакета systems)

func calculateSmartMove(npc, target *domain.Entity, w *domain.GameWorld, ents []domain.Entity) (int, int) {
	dxRaw := target.Pos.X - npc.Pos.X
	dyRaw := target.Pos.Y - npc.Pos.Y

	stepX := sign(dxRaw)
	stepY := sign(dyRaw)

	// Попытка 1: Идеальный путь
	res := CalculateMove(npc, stepX, stepY, w, ents)
	if res.HasMoved {
		return stepX, stepY
	}

	// Попытка 2: Smart Sliding (выбор приоритетной оси)
	tryXFirst := math.Abs(float64(dxRaw)) > math.Abs(float64(dyRaw))

	if tryXFirst {
		if stepX != 0 && checkMove(npc, stepX, 0, w, ents) {
			return stepX, 0
		}
		if stepY != 0 && checkMove(npc, 0, stepY, w, ents) {
			return 0, stepY
		}
	} else {
		if stepY != 0 && checkMove(npc, 0, stepY, w, ents) {
			return 0, stepY
		}
		if stepX != 0 && checkMove(npc, stepX, 0, w, ents) {
			return stepX, 0
		}
	}

	return 0, 0 // Тупик
}

func checkMove(e *domain.Entity, dx, dy int, w *domain.GameWorld, ents []domain.Entity) bool {
	res := CalculateMove(e, dx, dy, w, ents)
	return res.HasMoved
}

func sign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
