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
