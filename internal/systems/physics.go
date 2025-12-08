package systems

import (
	"cognitive-server/internal/domain"
)

// HasLineOfSight проверяет прямую видимость между двумя точками.
// Использует оптимизированный алгоритм Брезенхэма (только целочисленная арифметика).
func HasLineOfSight(w *domain.GameWorld, p1, p2 domain.Position) bool {
	// Если точки совпадают - видно
	if p1.X == p2.X && p1.Y == p2.Y {
		return true
	}

	x0, y0 := p1.X, p1.Y
	x1, y1 := p2.X, p2.Y

	// Оптимизация: abs без float64
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	for {
		// Проверяем препятствия, ИСКЛЮЧАЯ стартовую и конечную точки.
		// Мы видим врага (p2), даже если он стоит в "стене" (или является ей),
		// главное, чтобы между нами (p1) и им (p2) было пусто.
		if !(x0 == p1.X && y0 == p1.Y) && !(x0 == p2.X && y0 == p2.Y) {
			// 1. Проверка границ
			if x0 < 0 || x0 >= w.Width || y0 < 0 || y0 >= w.Height {
				return false
			}
			// 2. Проверка стены
			if w.Map[y0][x0].IsWall {
				return false
			}
		}

		// Если дошли до конца
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

	return true
}
