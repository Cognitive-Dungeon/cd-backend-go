package entityindex

import "cognitive-server/pkg/geo"

const (
	// BucketSize = 16. Это совпадает с чанком карты для удобства,
	// но является внутренней константой индекса.
	BucketSize  = 16
	BucketShift = 4
	BucketMask  = 0xF // Если понадобится локальная координата
)

// BucketKey — уникальный идентификатор ячейки индекса.
// Это НЕ geo.Location. Это координата бакета (x/16, y/16).
type BucketKey uint64

// ShardIndex — индекс шарда (0..N).
type ShardIndex int

// makeBucketKey преобразует мировые координаты в ключ бакета.
func makeBucketKey(pos geo.Location) BucketKey {
	x, y, z := pos.XYZ()

	// Сдвигаем координаты (деление на 16)
	bx := x >> BucketShift
	by := y >> BucketShift

	// Упаковываем обратно.
	return BucketKey(geo.Pos(bx, by, z))
}

// getShardIndex вычисляет шард для бакета.
func getShardIndex(key BucketKey) ShardIndex {
	// Распаковываем или используем raw value для хеша.
	// Просто (key ^ (key >> 16)) & Mask дает хорошее распределение.
	hash := uint64(key)
	return ShardIndex((hash ^ (hash >> 32)) & (ShardCount - 1))
}
