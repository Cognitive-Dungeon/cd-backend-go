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
