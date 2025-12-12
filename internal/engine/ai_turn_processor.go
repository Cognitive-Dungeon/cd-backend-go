package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"encoding/json"
	"log"
)

// processAITurn обрабатывает логику NPC
func (s *GameService) processAITurn(npc *domain.Entity) {
	// Если моб мертв или не агрессивен - пропускаем ход
	if npc.Stats != nil && npc.Stats.IsDead {
		return
	}
	if !npc.AI.IsHostile {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	var target *domain.Entity
	minDist := 999.0

	// 1. Поиск цели
	for _, other := range s.Entities {
		// Пропускаем:
		// - Себя
		// - Мертвых
		// - Тех, кто на другом уровне
		if other.ID == npc.ID || other.Level != npc.Level || (other.Stats != nil && other.Stats.IsDead) {
			continue
		}

		// Логика "Свой-Чужой"
		// 1. Игнорируем неодушевленные предметы и выходы
		isInanimate := other.Type == domain.EntityTypeItem || other.Type == domain.EntityTypeExit
		if isInanimate {
			continue
		}

		// 2. Игнорируем свой вид (Гоблины не бьют Гоблинов, Враги не бьют Врагов)
		isSameType := other.Type == npc.Type
		if isSameType {
			continue
		}

		// Нашли кого-то чужого и живого. Проверяем расстояние.
		dist := npc.Pos.DistanceTo(other.Pos)
		if dist < minDist {
			minDist = dist
			target = other
		}
	}

	// 2. Если целей нет, ждем
	if target == nil {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	// 3. Получаем мир
	npcWorld, ok := s.Worlds[npc.Level]
	if !ok {
		// Перевести на logger
		log.Printf("[ERROR] NPC %s is on a non-existent level %d. Waiting.", npc.ID, npc.Level)
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	// 4. Вычисляем действие через System AI
	action, _, dx, dy := systems.ComputeNPCAction(npc, target, npcWorld)

	// 5. Выполняем
	switch action {
	case domain.ActionAttack:
		payload, _ := json.Marshal(api.EntityPayload{TargetID: target.ID})
		s.executeCommand(domain.InternalCommand{
			Action:  domain.ActionAttack,
			Token:   npc.ID,
			Payload: payload,
		}, npc)

	case domain.ActionMove:
		payload, _ := json.Marshal(api.DirectionPayload{Dx: dx, Dy: dy})
		s.executeCommand(domain.InternalCommand{
			Action:  domain.ActionMove,
			Token:   npc.ID,
			Payload: payload,
		}, npc)

	default:
		npc.AI.Wait(domain.TimeCostWait)
	}
}
