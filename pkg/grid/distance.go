package grid

import "math"

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

// EuclideanDistance возвращает евклидово расстояние между двумя тайловыми позициями.
//
// Использует извлечение квадратного корня и возвращает значение с плавающей точкой.
//
// Предназначена для:
//   - UI
//   - визуальных эффектов
//   - отладки
//
// НЕ рекомендуется для использования в горячих участках симуляции.
func (a TilePos) EuclideanDistance(b TilePos) float32 {
	dx := float32(a.X - b.X)
	dy := float32(a.Y - b.Y)
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
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

// OctileDistance возвращает октильное расстояние между двумя позициями.
//
// Это эвристика, используемая в A* для 8-направленного движения.
// Диагональный шаг считается дороже ортогонального.
//
// Стоимости шагов:
//   - ортогональный шаг: 10
//   - диагональный шаг: 14 (≈ sqrt(2) * 10)
//
// Возвращаемое значение является целочисленной оценкой стоимости пути.
func (a TilePos) OctileDistance(b TilePos) int32 {
	dx := abs32(int32(a.X - b.X))
	dy := abs32(int32(a.Y - b.Y))

	if dx < dy {
		return 14*dx + 10*(dy-dx)
	}
	return 14*dy + 10*(dx-dy)
}

// L1 возвращает расстояние L1 (манхэттенскую метрику) между двумя позициями.
//
// Является алиасом для ManhattanDistance.
// Используется для повышения читаемости математического кода.
func (a TilePos) L1(b TilePos) int32 {
	return a.ManhattanDistance(b)
}

// L2Squared возвращает квадрат расстояния L2 (евклидовой метрики).
//
// Является алиасом для DistanceSquared.
// Удобно использовать в математически ориентированном коде и AI.
func (a TilePos) L2Squared(b TilePos) int64 {
	return a.DistanceSquared(b)
}

// LInf возвращает расстояние L∞ (метрика Чебышёва).
//
// Является алиасом для ChebyshevDistance.
func (a TilePos) LInf(b TilePos) int32 {
	return a.ChebyshevDistance(b)
}

// InManhattanRange проверяет, находится ли позиция a
// в манхэттенском радиусе r от позиции center.
func (a TilePos) InManhattanRange(center TilePos, r int32) bool {
	return a.ManhattanDistance(center) <= r
}

// InChebyshevRange проверяет, находится ли позиция a
// в радиусе Чебышёва r от позиции center.
func (a TilePos) InChebyshevRange(center TilePos, r int32) bool {
	return a.ChebyshevDistance(center) <= r
}
