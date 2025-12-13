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
