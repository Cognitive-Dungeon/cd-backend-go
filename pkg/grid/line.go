package grid

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
