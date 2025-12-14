package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"github.com/sirupsen/logrus"
)

// HasLineOfSight проверяет прямую видимость между двумя точками.
// Использует оптимизированный алгоритм Брезенхэма (только целочисленная арифметика).
func HasLineOfSight(w *domain.GameWorld, p1, p2 domain.Position) bool {
	losLogger := logger.Log.WithFields(logrus.Fields{
		"component": "physics_system",
		"function":  "HasLineOfSight",
		"start_pos": p1,
		"end_pos":   p2,
	})

	losLogger.Debug("--- Line of Sight Check Start ---")

	if p1.X == p2.X && p1.Y == p2.Y {
		losLogger.Debug("Check finished: Points are identical. Result: true")
		return true
	}

	x0, y0 := p1.X, p1.Y
	x1, y1 := p2.X, p2.Y

	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx, sy := p1.DirectionTo(p2)

	err := dx - dy

	for {
		// Проверяем препятствия, ИСКЛЮЧАЯ стартовую и конечную точки.
		isStartPoint := x0 == p1.X && y0 == p1.Y
		isEndPoint := x0 == p2.X && y0 == p2.Y

		if !isStartPoint && !isEndPoint {
			// 1. Проверка границ карты
			if x0 < 0 || x0 >= w.Width || y0 < 0 || y0 >= w.Height {
				losLogger.WithField("blocking_point", map[string]int{"x": x0, "y": y0}).
					Debug("Check finished: Line is blocked by map BOUNDS. Result: false")
				return false
			}
			// 2. Проверка стены
			if w.Map[y0][x0].IsWall {
				losLogger.WithField("blocking_point", map[string]int{"x": x0, "y": y0}).
					Debug("Check finished: Line is blocked by WALL. Result: false")
				return false
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}

	losLogger.Debug("Check finished: No obstructions found. Result: true")
	return true
}
