package gateway

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/internal/engine/view"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// GameGateway — фасад для взаимодействия внешнего мира с движком.
type GameGateway struct {
	engine  *engine.Engine
	view    *view.SnapshotBuilder
	network NetworkCallback
}

func New(eng *engine.Engine) *GameGateway {
	return &GameGateway{
		engine: eng,
		view:   view.New(eng),
	}
}

func (g *GameGateway) HandleLogin(token string) (ecs2.ObjectGuid, error) {
	log := logger.Log.WithField("layer", "gateway")

	if token == "" {
		log.Warn("login with empty token rejected")
		return 0, errors.New("empty token")
	}

	result := make(chan ecs2.ObjectGuid, 1)

	g.engine.PushCommand(func() {
		inst := g.engine.Instance

		guid := inst.CreateObject(enums.ObjectTypePlayer)

		inst.NewEntityBuilder(guid).
			WithName(token).
			WithPosition(10, 10).
			WithStats(100, 100).
			WithRender('@', 0x00FF00).
			WithController(token).
			WithSpells(1, 2, 4)

		result <- guid
	})

	select {
	case guid := <-result:
		log.WithField("guid", guid).
			Info("login successful")
		return guid, nil

	case <-time.After(2 * time.Second):
		log.Error("login timeout waiting for engine")
		return 0, errors.New("login timeout")
	}
}

// HandleMove обрабатывает запрос на движение (DTO -> Event)
func (g *GameGateway) HandleMove(guid ecs2.ObjectGuid, payload api.MovePayload) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "gateway",
		"guid":  guid,
	})

	if !g.engine.Instance.IsValid(guid) {
		log.Warn("move for invalid object ignored")
		return
	}

	var dir enums.Direction
	switch {
	case payload.Dy < 0:
		dir = enums.DirUp
	case payload.Dy > 0:
		dir = enums.DirDown
	case payload.Dx < 0:
		dir = enums.DirLeft
	case payload.Dx > 0:
		dir = enums.DirRight
	default:
		log.Debug("zero move ignored")
		return
	}

	g.engine.PushCommand(func() {
		// Используем ECS API для добавления команды
		// Это безопасно, пока PushCommand выполняется в главном потоке перед Input фазой.

		id := ecs.EntityID(guid)

		// Проверяем, существует ли сущность (есть ли у неё позиция, например)
		if ecs.GetStorage[components.PositionComponent](g.engine.Instance.World, components.CID_Position).Get(id) == nil {
			return
		}

		ecs.GetStorage[components.CmdMove](g.engine.Instance.World, components.CID_CmdMove).Add(id, components.CmdMove{
			Direction: dir,
		})
	})
}

// HandleCast обрабатывает запрос клиента на каст.
func (g *GameGateway) HandleCast(guid ecs2.ObjectGuid, payload api.CastPayload) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":   "gateway",
		"guid":    guid,
		"spellID": payload.SpellID,
	})

	// Валидация входных данных (базовая)
	if payload.SpellID == 0 {
		return
	}

	g.engine.PushCommand(func() {
		id := ecs.EntityID(guid)

		// Проверяем, жив ли кастер
		stats := ecs.GetStorage[components.StatsComponent](g.engine.Instance.World, components.CID_Stats).Get(id)
		if stats == nil || stats.IsDead {
			log.Warn("cast ignored: entity dead or invalid")
			return
		}

		// Создаем CmdCast компонент (ScopeInput)
		ecs.GetStorage[components.CmdCast](g.engine.Instance.World, components.CID_CmdCast).Add(id, components.CmdCast{
			SpellID:  payload.SpellID,
			TargetID: payload.TargetID,
		})
	})
}

func (g *GameGateway) HandleChat(sourceGuid ecs2.ObjectGuid, p api.ChatPayload) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "gateway",
		"guid":  sourceGuid,
	})

	rawText := strings.TrimSpace(p.Message)

	if rawText == "" {
		log.Debug("empty chat ignored")
		return
	}

	g.engine.PushCommand(func() {
		// Проверка отправителя
		if !g.engine.Instance.IsValid(sourceGuid) {
			log.Warn("chat from invalid object ignored")
			return
		}

		msgType := types.ChatTypeSay
		target := ecs2.ObjectGuid(0)
		finalText := rawText

		// Парсинг команд
		if strings.HasPrefix(rawText, "/") {
			parts := strings.SplitN(rawText, " ", 3) // /w Name Msg
			cmd := strings.ToLower(parts[0])

			switch cmd {
			case "/s", "/say":
				msgType = types.ChatTypeSay
				if len(parts) > 1 {
					finalText = rawText[len(cmd)+1:]
				}

			case "/y", "/yell":
				msgType = types.ChatTypeYell
				if len(parts) > 1 {
					finalText = rawText[len(cmd)+1:]
				}

			case "/e", "/emote":
				msgType = types.ChatTypeEmote
				if len(parts) > 1 {
					finalText = rawText[len(cmd)+1:]
				}

			case "/w", "/whisper":
				msgType = types.ChatTypeWhisper
				if len(parts) < 3 {
					// TODO: Отправить системное сообщение "Usage: /w Name Message"
					return
				}
				targetName := parts[1]
				finalText = parts[2]

				// Ищем цель в ECS (мы внутри PushCommand, это безопасно)
				target = g.engine.Instance.FindObjectByName(targetName)
				if target == 0 {
					// TODO: Отправить системное сообщение "Player not found"
					return
				}
			}
		}

		g.engine.Bus.Publish(
			eventbus.EventType(enums.EventChatRequest),
			enums.ChatRequestEvent{
				Source:  sourceGuid,
				Type:    msgType,
				Target:  target,
				Message: finalText,
			},
		)
	})
}

// GetSnapshot возвращает состояние мира для клиента
func (g *GameGateway) GetSnapshot(guid ecs2.ObjectGuid) *api.ServerResponse {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "gateway",
		"guid":  guid,
	})

	if !g.engine.Instance.IsValid(guid) {
		log.Debug("snapshot requested for invalid object")
		return nil
	}

	snap := g.view.BuildSnapshot(guid)
	if snap == nil {
		log.Warn("snapshot build returned nil")
	}
	return snap
}
