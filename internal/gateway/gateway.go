package gateway

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/engine/view"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"strings"
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

// HandleLogin обрабатывает вход игрока и возвращает его GUID.
// Выполняется синхронно (блокирует до выполнения в движке), либо асинхронно (тогда возвращает channel).
// Для простоты пока сделаем channel-based callback внутри.
func (g *GameGateway) HandleLogin(token string, callback func(engine.ObjectGuid)) {
	if token == "" {
		return
	}

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
				KnownSpells: []uint32{1, 2, 4}, // Умеет бить, фаербол и блинк
				Cooldowns:   make(map[uint32]float64),
			})

		logger.Log.Infof("Gateway: Login '%s' -> %s", token, guid)

		// Возвращаем результат вызывающему
		callback(guid)
	})
}

// HandleMove обрабатывает запрос на движение (DTO -> Event)
func (g *GameGateway) HandleMove(guid engine.ObjectGuid, payload api.MovePayload) {
	// Адаптер: Вектор -> Enum (Легаси поддержка)
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
		return
	}

	g.engine.PushCommand(func() {
		if !g.engine.Instance.IsValid(guid) {
			return
		}

		g.engine.Bus.Publish(eventbus.EventType(enums.EventMoveRequest), enums.MoveRequestEvent{
			Object:    guid,
			Direction: dir,
		})
	})
}

func (g *GameGateway) HandleCast(casterGuid engine.ObjectGuid, payload api.CastPayload) {
	g.engine.PushCommand(func() {
		if !g.engine.Instance.IsValid(casterGuid) {
			return
		}

		g.engine.Bus.Publish(eventbus.EventType(enums.EventCastRequest), enums.CastRequestEvent{
			Caster:  casterGuid,
			Target:  payload.TargetID,
			SpellID: payload.SpellID,
		})
	})
}

func (g *GameGateway) HandleChat(sourceGuid engine.ObjectGuid, p api.ChatPayload) {
	text := strings.TrimSpace(p.Message)
	if text == "" {
		return
	}

	msgType := types.ChatTypeSay
	target := engine.ObjectGuid(0)

	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		cmd := strings.ToLower(parts[0])
		if len(parts) > 1 {
			text = parts[1]
		}

		switch cmd {
		case "/s", "/say":
			msgType = types.ChatTypeSay
		case "/y", "/yell":
			msgType = types.ChatTypeYell
		case "/e", "/emote":
			msgType = types.ChatTypeEmote
		}
	}

	g.engine.PushCommand(func() {
		if !g.engine.Instance.IsValid(sourceGuid) {
			return
		}

		g.engine.Bus.Publish(
			eventbus.EventType(enums.EventChatRequest),
			enums.ChatRequestEvent{
				Source:  sourceGuid,
				Type:    msgType,
				Target:  target,
				Message: text,
			},
		)
	})
}

// GetSnapshot возвращает состояние мира для клиента
func (g *GameGateway) GetSnapshot(guid engine.ObjectGuid) *api.ServerResponse {
	// Этот метод читает данные ECS. В идеале он должен быть thread-safe.
	// Paged ECS позволяет чтение без локов, если структура чанков не меняется.
	return g.view.BuildSnapshot(guid)
}
