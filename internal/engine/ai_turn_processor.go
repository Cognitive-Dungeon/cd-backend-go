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
	if !npc.AI.IsHostile {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	var target *domain.Entity
	minDist := 999.0

	// 1. Ищем ближайшую цель на ТОМ ЖЕ УРОВНЕ
	for _, other := range s.Entities {
		if other.Level != npc.Level {
			continue // Игнорируем сущностей на других уровнях
		}
		if other.ID == npc.ID || (other.Stats != nil && other.Stats.IsDead) {
			continue // Игнорируем себя и мертвых
		}

		// Агрессия на Игроков
		if other.Type == domain.EntityTypePlayer {
			dist := npc.Pos.DistanceTo(other.Pos)
			if dist < minDist {
				minDist = dist
				target = other
			}
		}
	}

	// 2. Если целей на этом уровне нет, ждем
	if target == nil {
		npc.AI.Wait(domain.TimeCostWait)
		return // ВАЖНО: Выходим, если нет цели. Не вызываем AI.
	}

	// 3. Получаем мир, в котором находится NPC
	npcWorld, ok := s.Worlds[npc.Level]
	if !ok {
		log.Printf("[ERROR] NPC %s is on a non-existent level %d. Waiting.", npc.ID, npc.Level)
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	// 4. Вычисляем действие
	action, _, dx, dy := systems.ComputeNPCAction(npc, target, npcWorld)

	// 5. Конвертируем решение AI во внутреннюю команду
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
		// Wait
		npc.AI.Wait(domain.TimeCostWait)
	}
}
