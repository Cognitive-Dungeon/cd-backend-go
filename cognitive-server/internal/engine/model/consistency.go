package model

import (
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/logger"
)

// VerifySpatialIndex проверяет, что каждая сущность с позицией корректно зарегистрирована в индексе.
// Это O(N) операция, тяжелая. Использовать при старте или в дебаг-команде.
func (inst *Instance) VerifySpatialIndex() {
	count := 0
	errors := 0

	// Проходим по всем сущностям с позицией
	for id, pos := range ecs.View1[components.PositionComponent](inst.World, components.CID_Position) {
		count++

		// Конвертируем в geo
		geoPos := geo.Pos(int(pos.X), int(pos.Y), 0)

		// Запрашиваем "ведро" из индекса
		candidates := inst.EntityGrid.QueryBucket(geoPos)

		found := false
		for _, candidateID := range candidates {
			if candidateID == id {
				found = true
				break
			}
		}

		if !found {
			errors++
			logger.Log.Errorf("SYNC ERROR: Entity %d at %v is missing from Spatial Grid bucket!", id, pos.TilePos)
		}
	}

	if errors == 0 {
		logger.Log.Debugf("Spatial Index Verified: %d entities sync OK.", count)
	} else {
		logger.Log.Errorf("Spatial Index Verification FAILED: %d errors found.", errors)
	}
}
