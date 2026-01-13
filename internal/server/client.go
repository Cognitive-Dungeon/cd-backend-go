package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	engine *engine.Engine

	// GUID сущности. Если 0 (Nil), значит клиент еще не залогинился.
	objectGuid engine.ObjectGuid
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Errorf("WS Upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		engine: s.Engine,
	}

	// Мы НЕ спавним игрока сразу. Мы ждем команду LOGIN.
	// Запускаем только чтение.
	go client.readLoop()
}

func (c *Client) readLoop() {
	defer func() {
		c.conn.Close()
		// TODO: Обработка дисконнекта (удаление сущности или пометка offline)
	}()

	for {
		var msg api.InboundMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			// Обычный разрыв соединения
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Warnf("WS Error: %v", err)
			}
			break
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg api.InboundMessage) {
	// Если мы еще не залогинены, принимаем только LOGIN
	if c.objectGuid == 0 && msg.Action != "LOGIN" {
		logger.Log.Warn("Ignored command before LOGIN")
		return
	}

	switch msg.Action {
	case "LOGIN":
		// Токен приходит в поле Token, а не в Payload (легаси клиента)
		c.handleLogin(msg.Token)

	case "MOVE":
		var payload api.MovePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			logger.Log.Warn("Invalid MOVE payload")
			return
		}
		c.handleMove(payload)
	}
}

func (c *Client) handleLogin(token string) {
	if token == "" {
		token = "Unnamed"
	}

	// Отправляем задачу в движок (Main Thread)
	c.engine.PushCommand(func() {
		// 1. Создаем сущность
		guid := c.engine.Instance.CreateObject(enums.ObjectTypePlayer)

		// 2. Наполняем компонентами
		c.engine.Instance.NewEntityBuilder(guid).
			WithName(engine.NameComponent{Name: token}). // Используем токен как имя
			WithPosition(engine.PositionComponent{TilePos: engine.TilePos{X: 10, Y: 10}}).
			WithStats(engine.StatsComponent{Health: 100, MaxHealth: 100}).
			// Визуал: Зеленая @
			WithRender(types.MakeGlyph(0x00FF00, '@')).
			WithController(engine.ControllerComponent{AgentID: token})

		// 3. Привязываем к клиенту
		c.objectGuid = guid

		logger.Log.Infof("Client logged in as '%s' -> GUID %s", token, guid)

		// 4. ТЕПЕРЬ запускаем отправку обновлений (Snapshot Loop)
		// Запускаем в отдельной горутине
		go c.writeLoop()
	})
}

func (c *Client) handleMove(p api.MovePayload) {
	var dir enums.Direction

	// АДАПТЕР: Вектор -> Enum
	// Это изолирует легаси протокол от чистой внутренней логики
	switch {
	case p.Dy < 0:
		dir = enums.DirUp
	case p.Dy > 0:
		dir = enums.DirDown
	case p.Dx < 0:
		dir = enums.DirLeft
	case p.Dx > 0:
		dir = enums.DirRight
	default:
		// Если dx=0, dy=0 или какая-то диагональ (если мы её не поддерживаем),
		// просто игнорируем
		return
	}

	// Отправляем в движок (Движок получает чистый Enum)
	c.engine.PushCommand(func() {
		if !c.engine.Instance.IsValid(c.objectGuid) {
			return
		}

		c.engine.Bus.Publish(eventbus.EventType(enums.EventMoveRequest), enums.MoveRequestEvent{
			Object:    c.objectGuid,
			Direction: dir,
		})
	})
}

func (c *Client) writeLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		// Проверка: если клиент отключился или объект удален - выходим
		// (в простой реализации достаточно проверки conn write error)

		select {
		case <-ticker.C:
			// Генерируем снапшот (thread-safe, т.к. GetWorldSnapshot читает ECS)
			// В идеале GetWorldSnapshot должен вызываться внутри PushCommand и отдавать результат в канал,
			// но для чтения Paged Slice это допустимо, если мы не ресайзим чанки каждую миллисекунду.
			snapshot := c.engine.GetWorldSnapshot(c.objectGuid)

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(snapshot); err != nil {
				return // Ошибка записи = клиент отвалился
			}
		}
	}
}
