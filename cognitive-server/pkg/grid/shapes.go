package grid

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
	d := a.Sub(center)
	return abs32(int32(d.X)) <= r &&
		abs32(int32(d.Y)) <= r
}

// InDiamond проверяет, находится ли позиция a внутри ромбовидной области
// (манхэттенский круг) с центром center и радиусом r.
//
// Область определяется условием:
//
//	|dx| + |dy| <= r
func (a TilePos) InDiamond(center TilePos, r int32) bool {
	d := a.Sub(center)
	return abs32(int32(d.X))+
		abs32(int32(d.Y)) <= r
}

// ForEachInRadius вызывает visit для каждой тайловой позиции
// внутри круга радиуса r с центром center.
//
// Обход выполняется по минимальному ограничивающему квадрату.
// Если visit возвращает false, обход прекращается.
func ForEachInRadius(center TilePos, r int32, visit func(TilePos) bool) {
	rr := int64(r) * int64(r)

	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			offset := TilePos{
				X: TileCoord(dx),
				Y: TileCoord(dy),
			}

			if int64(dx*dx+dy*dy) <= rr {
				if !visit(center.Add(offset)) {
					return
				}
			}
		}
	}
}

// ForEachInSquare вызывает visit для каждой тайловой позиции
// внутри квадратной области с центром center и полуразмером r.
//
// Если visit возвращает false, обход прекращается.
func ForEachInSquare(center TilePos, r int32, visit func(TilePos) bool) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			offset := TilePos{
				X: TileCoord(dx),
				Y: TileCoord(dy),
			}
			if !visit(center.Add(offset)) {
				return
			}
		}
	}
}

// ForEachInDiamond вызывает visit для каждой тайловой позиции
// внутри ромбовидной области (манхэттенский круг) радиуса r
// с центром center.
//
// Если visit возвращает false, обход прекращается.
func ForEachInDiamond(center TilePos, r int32, visit func(TilePos) bool) {
	for dy := -r; dy <= r; dy++ {
		limit := r - abs32(dy)
		for dx := -limit; dx <= limit; dx++ {
			offset := TilePos{
				X: TileCoord(dx),
				Y: TileCoord(dy),
			}
			if !visit(center.Add(offset)) {
				return
			}
		}
	}
}

// FindNearest ищет ближайшую точку в слайсе.
// Это быстрее, чем for range + DistanceSquared
func (a TilePos) FindNearest(targets []TilePos) (int, int64) {
	if len(targets) == 0 {
		return -1, -1
	}

	minDist := int64(1<<63 - 1)
	idx := -1
	startX, startY := int64(a.X), int64(a.Y)

	i := 0
	l := len(targets)

	// Unrolling 4x
	for ; i <= l-4; i += 4 {
		_ = targets[i+3] // Bounds Check Elimination hint

		// Прямая математика (потому что здесь мы хотим максимальную плотность инструкций)

		p0 := targets[i]
		dx0, dy0 := startX-int64(p0.X), startY-int64(p0.Y)
		d0 := dx0*dx0 + dy0*dy0

		p1 := targets[i+1]
		dx1, dy1 := startX-int64(p1.X), startY-int64(p1.Y)
		d1 := dx1*dx1 + dy1*dy1

		p2 := targets[i+2]
		dx2, dy2 := startX-int64(p2.X), startY-int64(p2.Y)
		d2 := dx2*dx2 + dy2*dy2

		p3 := targets[i+3]
		dx3, dy3 := startX-int64(p3.X), startY-int64(p3.Y)
		d3 := dx3*dx3 + dy3*dy3

		if d0 < minDist {
			minDist = d0
			idx = i
		}
		if d1 < minDist {
			minDist = d1
			idx = i + 1
		}
		if d2 < minDist {
			minDist = d2
			idx = i + 2
		}
		if d3 < minDist {
			minDist = d3
			idx = i + 3
		}
	}

	// Хвост
	for ; i < l; i++ {
		d := a.DistanceSquared(targets[i])
		if d < minDist {
			minDist = d
			idx = i
		}
	}

	return idx, minDist
}
