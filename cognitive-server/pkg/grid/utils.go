package grid

// Clamp ограничивает значение v диапазоном [min, max].
//
// Если v меньше min, возвращается min.
// Если v больше max, возвращается max.
func Clamp(v, min, max int32) int32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Rect описывает ось-ориентированный прямоугольник в тайловых координатах.
type Rect struct {
	Min, Max TilePos
}

// Contains проверяет, находится ли позиция p внутри прямоугольника r
// (включая границы).
func (r Rect) Contains(p TilePos) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X &&
		p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

// abs32 возвращает абсолютное значение знакового 32-битного числа
// без использования ветвлений.
//
// Реализация предназначена для горячих участков кода
// (pathfinding, LOS, симуляция).
func abs32(x int32) int32 {
	mask := x >> 31
	return (x + mask) ^ mask
}
