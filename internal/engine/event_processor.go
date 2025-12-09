package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// processEvent - является точкой входа для обработки событий, возвращенных хендлерами.
func (s *GameService) processEvent(actor *domain.Entity, eventData json.RawMessage) {
	var genericEvent struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(eventData, &genericEvent); err != nil {
		log.Printf("Error parsing event: %v", err)
		return
	}

	switch genericEvent.Event {
	case "LEVEL_TRANSITION":
		s.handleLevelTransition(actor, eventData)
	// Здесь в будущем могут быть другие события: "SPAWN_MONSTER", "OPEN_DOOR", etc.
	default:
		log.Printf("Unknown event type: %s", genericEvent.Event)
	}
}

// handleLevelTransition - обрабатывает логику перемещения сущности между уровнями.
func (s *GameService) handleLevelTransition(actor *domain.Entity, eventData json.RawMessage) {
	var transitionEvent struct {
		TargetLevel int    `json:"targetLevel"`
		TargetPosId string `json:"targetPosId"`
	}
	if err := json.Unmarshal(eventData, &transitionEvent); err != nil {
		log.Printf("Error parsing LEVEL_TRANSITION event: %v", err)
		return
	}

	oldLevel := actor.Level
	newLevel := transitionEvent.TargetLevel

	// 1. Находим старый и новый миры
	oldWorld, okOld := s.Worlds[oldLevel]
	newWorld, okNew := s.Worlds[newLevel]

	if !okOld {
		log.Printf("Actor %s tried to transition from a non-existent level %d", actor.ID, oldLevel)
		return
	}

	// Если нового мира нет - генерируем его на лету
	if !okNew {
		log.Printf("[EVENTS DEBUG | LevelTransition] Generating new level on the fly: %d", newLevel)
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		generatedWorld, newEntities, _ := dungeon.Generate(newLevel, rng)

		s.Worlds[newLevel] = generatedWorld
		newWorld = generatedWorld
		// Регистрируем новых сущностей
		for i := range newEntities {
			e := &newEntities[i]
			s.Entities = append(s.Entities, e)
			newWorld.AddEntity(e)
			newWorld.RegisterEntity(e)
		}
	}

	// 2. Находим целевую позицию в новом мире
	var targetPos domain.Position
	targetEntity := s.GetEntity(transitionEvent.TargetPosId)
	if targetEntity != nil {
		targetPos = targetEntity.Pos
	} else {
		log.Printf("[EVENTS DEBUG | LevelTransition] Could not find target position entity %s on level %d. Placing at default.", transitionEvent.TargetPosId, newLevel)
		targetPos = domain.Position{X: newWorld.Width / 2, Y: newWorld.Height / 2} // Запасной вариант
	}

	// 3. Перемещаем актора
	// "Выписываемся" из старого мира
	oldWorld.RemoveEntity(actor)        // Удаляем из SpatialHash
	oldWorld.UnregisterEntity(actor.ID) // Удаляем из Реестра

	// Обновляем состояние самого актора
	actor.Level = newLevel
	actor.Pos = targetPos

	// "Прописываемся" в новом мире
	newWorld.AddEntity(actor)      // Добавляем в SpatialHash
	newWorld.RegisterEntity(actor) // Добавляем в Реестр

	// 4. Логирование
	if newLevel > oldLevel {
		s.AddLog(fmt.Sprintf("%s спускается глубже...", actor.Name), "INFO")
	} else {
		s.AddLog(fmt.Sprintf("%s поднимается наверх...", actor.Name), "INFO")
	}
}
