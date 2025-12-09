package systems

import (
	"cognitive-server/internal/domain"
	"log"
	"math"
)

// ComputeNPCAction решает, что делать NPC.
// Возвращает (команда, цель_атаки_если_есть, dx, dy)
func ComputeNPCAction(npc *domain.Entity, player *domain.Entity, w *domain.GameWorld) (action domain.ActionType, target *domain.Entity, dx, dy int) {
	// --- ОТЛАДКА: НАЧАЛО ПРОВЕРКИ ---
	log.Printf("[AI DEBUG | %s] Turn Start. Target: %s at (%d,%d)", npc.Name, player.Name, player.Pos.X, player.Pos.Y)

	if npc.AI == nil || npc.Stats == nil || npc.Stats.IsDead || !npc.AI.IsHostile {
		log.Printf("[AI DEBUG | %s] Invalid state (dead, not hostile, etc). Action: WAIT", npc.Name)
		return domain.ActionWait, nil, 0, 0
	}

	dist := npc.Pos.DistanceTo(player.Pos)
	log.Printf("[AI DEBUG | %s] Distance to target: %.2f", npc.Name, dist)

	// Проверка видимости
	canSee := HasLineOfSight(w, npc.Pos, player.Pos)
	log.Printf("[AI DEBUG | %s] HasLineOfSight to target: %t", npc.Name, canSee)

	// Если не видим цель, не делаем ничего.
	if !canSee {
		log.Printf("[AI DEBUG | %s] Target not visible. Action: WAIT", npc.Name)
		return domain.ActionWait, nil, 0, 0
	}

	// Если в радиусе атаки (включая диагонали)
	if dist <= 1.5 {
		log.Printf("[AI DEBUG | %s] Target in attack range. Action: ATTACK", npc.Name)
		return domain.ActionAttack, player, 0, 0
	}

	// Если видим, но цель слишком далеко (за пределами агро-радиуса)
	if dist > domain.AggroRadius {
		log.Printf("[AI DEBUG | %s] Target visible but out of aggro range (%.2f > %d). Action: WAIT", npc.Name, dist, domain.AggroRadius)
		return domain.ActionWait, nil, 0, 0
	}

	// Если мы здесь, значит, цель видима и находится в радиусе преследования.
	log.Printf("[AI DEBUG | %s] Target in pursuit range. Calculating move...", npc.Name)
	moveDx, moveDy := calculateSmartMove(npc, player, w)

	if moveDx == 0 && moveDy == 0 {
		log.Printf("[AI DEBUG | %s] Path is blocked or destination reached. Action: WAIT", npc.Name)
		return domain.ActionWait, nil, 0, 0
	}

	log.Printf("[AI DEBUG | %s] Path found. Action: MOVE (dx:%d, dy:%d)", npc.Name, moveDx, moveDy)
	return domain.ActionMove, nil, moveDx, moveDy
}

// Внутренние утилиты (приватные для пакета systems)

func calculateSmartMove(npc, target *domain.Entity, w *domain.GameWorld) (int, int) {
	dxRaw := target.Pos.X - npc.Pos.X
	dyRaw := target.Pos.Y - npc.Pos.Y

	stepX := sign(dxRaw)
	stepY := sign(dyRaw)

	// Попытка 1: Идеальный путь
	res := CalculateMove(npc, stepX, stepY, w)
	if res.HasMoved {
		return stepX, stepY
	}

	// Попытка 2: Smart Sliding (выбор приоритетной оси)
	tryXFirst := math.Abs(float64(dxRaw)) > math.Abs(float64(dyRaw))

	if tryXFirst {
		if stepX != 0 && checkMove(npc, stepX, 0, w) {
			return stepX, 0
		}
		if stepY != 0 && checkMove(npc, 0, stepY, w) {
			return 0, stepY
		}
	} else {
		if stepY != 0 && checkMove(npc, 0, stepY, w) {
			return 0, stepY
		}
		if stepX != 0 && checkMove(npc, stepX, 0, w) {
			return stepX, 0
		}
	}

	return 0, 0 // Тупик
}

func checkMove(e *domain.Entity, dx, dy int, w *domain.GameWorld) bool {
	res := CalculateMove(e, dx, dy, w)
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
