package events

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/dungeon"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

func HandleLevelTransition(ctx handlers.Context, eventData json.RawMessage) (handlers.Result, error) {
	var transitionEvent struct {
		TargetLevel int    `json:"targetLevel"`
		TargetPosId string `json:"targetPosId"`
	}
	if err := json.Unmarshal(eventData, &transitionEvent); err != nil {
		logger.Log.Errorf("Error parsing LEVEL_TRANSITION event: %v", err)
		return handlers.EmptyResult(), nil
	}

	actor := ctx.Actor
	oldLevel := actor.Level
	newLevel := transitionEvent.TargetLevel

	// 1. Находим старый и новый миры
	// Используем ctx.Worlds вместо s.Worlds
	oldWorld, okOld := ctx.Worlds[oldLevel]
	newWorld, okNew := ctx.Worlds[newLevel]

	if !okOld {
		logger.Log.Warnf("Actor %s tried to transition from a non-existent level %d", actor.ID, oldLevel)
		return handlers.EmptyResult(), nil
	}

	// Если нового мира нет - генерируем его на лету
	if !okNew {
		logger.Log.Debugf("[EVENTS DEBUG | LevelTransition] Generating new level on the fly: %d", newLevel)
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		generatedWorld, newEntities, _ := dungeon.Generate(newLevel, rng)

		ctx.Worlds[newLevel] = generatedWorld
		newWorld = generatedWorld

		// Регистрируем новых сущностей через коллбэк
		if ctx.AddGlobalEntity != nil {
			for i := range newEntities {
				e := &newEntities[i]
				ctx.AddGlobalEntity(e)
				newWorld.AddEntity(e)
				newWorld.RegisterEntity(e)
			}
		}
	}

	// 2. Находим целевую позицию в новом мире
	var targetPos domain.Position
	// Используем ctx.Finder.GetEntity (так как TargetPosId может быть в другом мире, но GetEntity ищет везде)
	// Однако, если мы только что сгенерировали мир, сущности там уже есть, но Finder может их ещё не видеть,
	// если они не добавлены в глобальный список (мы это сделали выше).
	targetEntity := ctx.Finder.GetEntity(transitionEvent.TargetPosId)

	if targetEntity != nil {
		targetPos = targetEntity.Pos
	} else {
		logger.Log.Debugf("[EVENTS DEBUG | LevelTransition] Could not find target position entity %s on level %d. Placing at default.", transitionEvent.TargetPosId, newLevel)
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
	var logMsg string
	if newLevel > oldLevel {
		logMsg = fmt.Sprintf("%s спускается глубже...", actor.Name)
	} else {
		logMsg = fmt.Sprintf("%s поднимается наверх...", actor.Name)
	}

	return handlers.Result{
		Msg:     logMsg,
		MsgType: "INFO",
	}, nil
}
