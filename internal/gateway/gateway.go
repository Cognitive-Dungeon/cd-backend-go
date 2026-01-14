package gateway

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/engine/view"
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

func (g *GameGateway) HandleLogin(token string) (engine.ObjectGuid, error) {
	log := logger.Log.WithField("layer", "gateway")

	if token == "" {
		log.Warn("login with empty token rejected")
		return 0, errors.New("empty token")
	}

	result := make(chan engine.ObjectGuid, 1)

	g.engine.PushCommand(func() {
		inst := g.engine.Instance

		guid := inst.CreateObject(enums.ObjectTypePlayer)

		inst.NewEntityBuilder(guid).
			WithName(engine.NameComponent{Name: token}).
			WithPosition(engine.PositionComponent{TilePos: engine.TilePos{X: 10, Y: 10}}).
			WithStats(engine.StatsComponent{Health: 100, MaxHealth: 100}).
			WithRender(types.MakeGlyph(0x00FF00, '@')).
			WithController(engine.ControllerComponent{AgentID: token}).
			WithSpells(engine.SpellbookComponent{
				KnownSpells: []uint32{1, 2, 4},
				Cooldowns:   make(map[uint32]float64),
			})

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
func (g *GameGateway) HandleMove(guid engine.ObjectGuid, payload api.MovePayload) {
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
		g.engine.Bus.Publish(
			eventbus.EventType(enums.EventMoveRequest),
			enums.MoveRequestEvent{
				Object:    guid,
				Direction: dir,
			},
		)
	})
}

func (g *GameGateway) HandleCast(guid engine.ObjectGuid, payload api.CastPayload) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":   "gateway",
		"guid":    guid,
		"spellID": payload.SpellID,
	})

	if !g.engine.Instance.IsValid(guid) {
		log.Warn("cast for invalid object ignored")
		return
	}

	g.engine.PushCommand(func() {
		g.engine.Bus.Publish(
			eventbus.EventType(enums.EventCastRequest),
			enums.CastRequestEvent{
				Caster:  guid,
				Target:  payload.TargetID,
				SpellID: payload.SpellID,
			},
		)
	})
}

func (g *GameGateway) HandleChat(sourceGuid engine.ObjectGuid, p api.ChatPayload) {
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
		target := engine.ObjectGuid(0)
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
func (g *GameGateway) GetSnapshot(guid engine.ObjectGuid) *api.ServerResponse {
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
