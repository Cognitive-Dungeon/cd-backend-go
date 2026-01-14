package engine

import "cognitive-server/internal/core/types"

// FindObjectsInRange возвращает список GUID объектов в радиусе.
func (inst *Instance) FindObjectsInRange(center types.TilePos, radius int32) []ObjectGuid {
	var result []ObjectGuid

	for chunkIdx, chunk := range inst.Guids {
		for slotIdx, guid := range chunk {
			if guid == 0 {
				continue
			}

			pos := inst.Positions[chunkIdx][slotIdx]
			if pos == nil {
				continue
			}

			// Быстрая проверка
			if pos.TilePos.InRadius(center, radius) {
				result = append(result, guid)
			}
		}
	}
	return result
}

// FindObjectByName ищет объект по полному совпадению имени.
// Нужно для Whisper.
func (inst *Instance) FindObjectByName(name string) ObjectGuid {
	for chunkIdx, chunk := range inst.Guids {
		for slotIdx, guid := range chunk {
			if guid == 0 {
				continue
			}

			n := inst.Names[chunkIdx][slotIdx]
			if n != nil && n.Name == name {
				return guid
			}
		}
	}
	return 0 // Nil
}
