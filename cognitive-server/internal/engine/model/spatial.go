package model

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/entityindex"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/grid"
)

// FindObjectsInRange возвращает список GUID объектов в радиусе.
// Использует Spatial Hash (EntityGrid) для Broad Phase и математику для Narrow Phase.
func (inst *Instance) FindObjectsInRange(center geo.Location, radius int32) []types.ObjectGuid {
	var result []types.ObjectGuid

	// Конвертируем центр в координаты тайлов для математики
	centerGrid := grid.TilePos{X: grid.TileCoord(center.X()), Y: grid.TileCoord(center.Y())}

	bucketRadius := int((radius + entityindex.BucketSize - 1) / entityindex.BucketSize)

	posStorage := ecs.GetStorage[components.PositionComponent](inst.World, components.CID_Position)

	for by := -bucketRadius; by < bucketRadius; by++ {
		for bx := -bucketRadius; bx < bucketRadius; bx++ {
			// Вычисляем пробную точку в соседнем бакете
			probePos := center.Shift(bx*entityindex.BucketSize, by*entityindex.BucketSize, 0)

			// Получаем всех из бакета
			// QueryBucket сам вычислит ключ бакета
			candidates := inst.EntityGrid.QueryBucket(probePos)

			for _, id := range candidates {
				// 2. Narrow Phase: Точная проверка
				posComp := posStorage.Get(id)
				if posComp == nil {
					continue
				}

				// Точная математика (int32 distance squared)
				if posComp.TilePos.InRadius(centerGrid, radius) {
					result = append(result, types.ObjectGuid(id))
				}
			}
		}
	}

	return result
}

// FindObjectByName ищет объект по полному совпадению имени.
// Нужно для Whisper.
func (inst *Instance) FindObjectByName(name string) types.ObjectGuid {
	// Итерируемся только по сущностям, имеющим NameComponent.
	for id, n := range ecs.View1[components.NameComponent](inst.World, components.CID_Name) {
		if n.Name == name {
			return types.ObjectGuid(id)
		}
	}
	return types.NilObjectGuid
}

// CheckLineOfSight проверяет, видит ли `from` точку `to`.
func (inst *Instance) CheckLineOfSight(from, to grid.TilePos) bool {

	blocked := false

	// Raycast (Grid Library)
	grid.LineExclusive(from, to, func(p grid.TilePos) bool {
		// Конвертация в geo для запроса к карте
		gPos := geo.Pos(int(p.X), int(p.Y), 0) // Z игнорируем пока

		// Проверяем прозрачность (стены блокируют обзор)
		if inst.WorldMap.IsOpaqueFast(gPos) {
			blocked = true
			return false // Stop ray
		}
		return true // Continue ray
	})

	return !blocked
}
