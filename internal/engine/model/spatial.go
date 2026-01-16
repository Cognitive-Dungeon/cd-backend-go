package model

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
)

// FindObjectsInRange возвращает список GUID объектов в радиусе.
func (inst *Instance) FindObjectsInRange(center types.TilePos, radius int32) []types.ObjectGuid {
	var result []types.ObjectGuid

	// Итерируемся только по сущностям, имеющим PositionComponent.
	// ecs.View1 бежит по плотному массиву данных, это кэш-френдли.
	for id, pos := range ecs.View1[components.PositionComponent](inst.World, components.CID_Position) {
		// Используем метод InRadius из geometry.go (быстрая проверка без sqrt)
		if pos.TilePos.InRadius(center, radius) {
			result = append(result, types.ObjectGuid(id))
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
