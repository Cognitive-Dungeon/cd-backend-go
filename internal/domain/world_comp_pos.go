package domain

import "math"

// DistanceTo возвращает точное расстояние до другой точки (float)
func (p Position) DistanceTo(other Position) float64 {
	return math.Sqrt(math.Pow(float64(p.X-other.X), 2) + math.Pow(float64(p.Y-other.Y), 2))
}

// DistanceSquaredTo возвращает квадрат расстояния (int) для сравнения без корней
func (p Position) DistanceSquaredTo(other Position) int {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return dx*dx + dy*dy
}

// IsAdjacent возвращает true, если цель в соседней клетке (включая диагональ)
func (p Position) IsAdjacent(other Position) bool {
	dx := p.X - other.X
	dy := p.Y - other.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	// Если разница по X и Y не больше 1, значит соседи
	return dx <= 1 && dy <= 1 && (dx != 0 || dy != 0)
}

// Shift Move возвращает новую позицию со смещением (не меняя текущую, т.к. Go передает структуры по значению, если не указатель)
func (p Position) Shift(dx, dy int) Position {
	return Position{X: p.X + dx, Y: p.Y + dy}
}

// Add возвращает сумму векторов (для сложения позиции и направления)
func (p Position) Add(other Position) Position {
	return Position{X: p.X + other.X, Y: p.Y + other.Y}
}

// Sub возвращает разницу векторов (для вычитания позиции и направления
func (p Position) Sub(other Position) Position {
	return Position{X: p.X - other.X, Y: p.Y - other.Y}
}

// DirectionTo возвращает нормализованный вектор направления к цели (-1, 0, 1)
func (p Position) DirectionTo(target Position) (dx, dy int) {
	if target.X > p.X {
		dx = 1
	} else if target.X < p.X {
		dx = -1
	}
	if target.Y > p.Y {
		dy = 1
	} else if target.Y < p.Y {
		dy = -1
	}
	return
}
