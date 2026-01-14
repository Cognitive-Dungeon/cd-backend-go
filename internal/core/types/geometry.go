package types

// DistanceSquared возвращает квадратичное евклидово расстояние между двумя тайловыми позициями.
//
// Значение равно (dx*dx + dy*dy) и НЕ содержит извлечения квадратного корня.
// Это намеренно сделано для повышения производительности.
//
// Используется в горячих участках движка:
//   - проверка радиусов (AOE, агро, звук, запах)
//   - spatial queries
//   - симуляция мира
//
// Для сравнения с радиусом необходимо сравнивать с квадратом радиуса.
func (a TilePos) DistanceSquared(b TilePos) int64 {
	dx := int64(a.X - b.X)
	dy := int64(a.Y - b.Y)
	return dx*dx + dy*dy
}

// ManhattanDistance возвращает манхэттенское расстояние между двумя позициями.
//
// Расстояние определяется как сумма абсолютных значений разностей координат:
//
//	|dx| + |dy|
//
// Используется в основном для:
//   - pathfinding (A*, Dijkstra)
//   - flood fill
//   - оценки стоимости перемещения по сетке
//
// Работает быстрее евклидового расстояния и не использует операции с плавающей точкой.
func (a TilePos) ManhattanDistance(b TilePos) int32 {
	dx := abs32(int32(a.X - b.X))
	dy := abs32(int32(a.Y - b.Y))
	return dx + dy
}

// ChebyshevDistance возвращает расстояние Чебышёва между двумя позициями.
//
// Расстояние определяется как максимум из |dx| и |dy|.
//
// Используется для:
//   - квадратных зон воздействия (AOE)
//   - расчёта количества шагов при 8-направленном движении
//   - симуляций, где диагональное перемещение эквивалентно прямому
func (a TilePos) ChebyshevDistance(b TilePos) int32 {
	dx := abs32(int32(a.X - b.X))
	dy := abs32(int32(a.Y - b.Y))
	if dx > dy {
		return dx
	}
	return dy
}

// InRadius проверяет, находится ли позиция a внутри круга радиуса r с центром center.
//
// Проверка выполняется через квадратичное расстояние без использования sqrt,
// что делает функцию пригодной для использования в горячих циклах симуляции.
func (a TilePos) InRadius(center TilePos, r int32) bool {
	rr := int64(r) * int64(r)
	return a.DistanceSquared(center) <= rr
}

// InSquare проверяет, находится ли позиция a внутри квадратной области
// с центром center и полуразмером r.
//
// Квадрат ориентирован по осям координат.
func (a TilePos) InSquare(center TilePos, r int32) bool {
	return abs32(int32(a.X-center.X)) <= r &&
		abs32(int32(a.Y-center.Y)) <= r
}

// InDiamond проверяет, находится ли позиция a внутри ромбовидной области
// (манхэттенский круг) с центром center и радиусом r.
//
// Область определяется условием:
//
//	|dx| + |dy| <= r
func (a TilePos) InDiamond(center TilePos, r int32) bool {
	return abs32(int32(a.X-center.X))+
		abs32(int32(a.Y-center.Y)) <= r
}

// Neighbors4 содержит смещения к четырём ортогональным соседним тайлам
// (вверх, вниз, влево, вправо).
//
// Используется для 4-направленного движения и flood fill.
var Neighbors4 = [4]TilePos{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
}

// Neighbors8 содержит смещения к восьми соседним тайлам,
// включая диагональные направления.
//
// Используется для 8-направленного движения, FOV и симуляций с диагоналями.
var Neighbors8 = [8]TilePos{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

// Line строит дискретную линию между двумя тайловыми позициями
// с использованием алгоритма Брезенхэма.
//
// Для каждого тайла на линии вызывается функция visit.
// Если visit возвращает false, обход линии немедленно прекращается.
//
// Используется для:
//   - line of sight (LOS)
//   - стрельбы и трассировки
//   - проверки видимости
func Line(from, to TilePos, visit func(TilePos) bool) {
	x0, y0 := int32(from.X), int32(from.Y)
	x1, y1 := int32(to.X), int32(to.Y)

	dx := abs32(x1 - x0)
	dy := -abs32(y1 - y0)

	sx := int32(1)
	if x0 >= x1 {
		sx = -1
	}
	sy := int32(1)
	if y0 >= y1 {
		sy = -1
	}

	err := dx + dy

	for {
		if !visit(TilePos{TileCoord(x0), TileCoord(y0)}) {
			return
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := err << 1
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

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
