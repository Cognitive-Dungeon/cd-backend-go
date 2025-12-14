package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"math"
	"math/rand"

	"github.com/sirupsen/logrus"
)

// ComputeNPCAction решает, что делать NPC.
// Возвращает (команда, цель_атаки_если_есть, dx, dy)
func ComputeNPCAction(npc *domain.Entity, player *domain.Entity, w *domain.GameWorld, rng *rand.Rand) (action domain.ActionType, target *domain.Entity, dx, dy int) {
	aiLogger := logger.Log.WithFields(logrus.Fields{
		"component":  "ai_system",
		"npc_id":     npc.ID,
		"npc_name":   npc.Name,
		"npc_pos":    npc.Pos,
		"target_id":  player.ID,
		"target_pos": player.Pos,
	})

	aiLogger.Debug("--- AI Turn Start ---")

	// --- ШАГ 2: Проверка базовых условий ---
	if npc.AI == nil || npc.Stats == nil || npc.Stats.IsDead || !npc.AI.IsHostile {
		aiLogger.Debug("Pre-computation check failed (dead, not hostile, etc). Action: WAIT")
		return domain.ActionWait, nil, 0, 0
	}

	distSq := npc.Pos.DistanceSquaredTo(player.Pos)
	canSee := HasLineOfSight(w, npc.Pos, player.Pos)

	aiLogger.WithFields(logrus.Fields{
		"distance_sq_to_target": distSq,
		"has_line_of_sight":     canSee,
	}).Debug("Perception check complete")

	// --- ШАГ 3: Логика принятия решений ---

	// 3.1. Не видим цель -> Ждать
	if !canSee {
		aiLogger.Debug("Decision: Target not visible. Action: WAIT")
		return domain.ActionWait, nil, 0, 0
	}

	// 3.2. В радиусе атаки -> Атаковать
	// 1.5 * 1.5 = 2.25. (1,1) -> distSq = 2. (1,0) -> distSq = 1.
	if distSq <= 2 {
		aiLogger.Debug("Decision: Target in melee range. Action: ATTACK")
		return domain.ActionAttack, player, 0, 0
	}

	// 3.3. Видим, но слишком далеко -> Ждать
	if distSq > domain.AggroRadius*domain.AggroRadius {
		aiLogger.WithField("aggro_radius", domain.AggroRadius).Debug("Decision: Target is outside aggro radius. Action: WAIT")
		return domain.ActionWait, nil, 0, 0
	}

	// 3.4. Если мы здесь, значит цель в зоне преследования -> Двигаться
	aiLogger.Debug("Decision: Target in pursuit range. Calculating move path...")
	moveDx, moveDy := calculateSmartMove(npc, player, w)

	if moveDx == 0 && moveDy == 0 {
		aiLogger.Debug("Path calculation result: Path is blocked or destination reached. Action: WAIT")
		return domain.ActionWait, nil, 0, 0
	}

	aiLogger.WithFields(logrus.Fields{
		"move_dx": moveDx,
		"move_dy": moveDy,
	}).Debug("Path calculation result: Path found. Action: MOVE")

	return domain.ActionMove, nil, moveDx, moveDy
}

// Внутренние утилиты (приватные для пакета systems)

func calculateSmartMove(npc, target *domain.Entity, w *domain.GameWorld) (int, int) {
	// 1. Получаем направление шага (-1, 0, 1)
	stepX, stepY := npc.Pos.DirectionTo(target.Pos)

	// 2. Получаем "сырой" вектор разницы для определения приоритета осей
	diff := target.Pos.Sub(npc.Pos)

	// Попытка 1: Идеальный путь (по диагонали)
	res := CalculateMove(npc, stepX, stepY, w)
	if res.HasMoved {
		return stepX, stepY
	}

	// Попытка 2: Smart Sliding (выбор приоритетной оси)
	// Если по X мы дальше от цели, чем по Y, то сначала пробуем шагать по X
	tryXFirst := math.Abs(float64(diff.X)) > math.Abs(float64(diff.Y))

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
