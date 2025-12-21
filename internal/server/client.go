package server

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/dungeon"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/utils"
	"math/rand"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// Настройки WebSocket
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client - посредник между Websocket и GameService
type Client struct {
	Game     *engine.GameService
	Conn     *websocket.Conn
	Send     chan api.ServerResponse
	EntityID domain.EntityID
}

func NewClient(game *engine.GameService, conn *websocket.Conn) *Client {
	return &Client{
		Game: game,
		Conn: conn,
		Send: make(chan api.ServerResponse, 256),
	}
}

// readPump читает команды от клиента
func (c *Client) readPump() {
	defer func() {
		c.Game.Hub.Unregister(c.EntityID)
		if err := c.Conn.Close(); err != nil {
			logger.Log.WithError(err).Warn("failed to close websocket connection")
		}
		// Освобождаем сущность, чтобы AI мог перехватить управление (если захотим)
		// или просто чтобы пометить, что игрок оффлайн
		if ent := c.Game.GetEntity(c.EntityID); ent != nil {
			ent.ControllerID = ""
			logger.Log.WithField("entity_id", c.EntityID).Info("Client disconnected")
			// Сообщаем движку, что игрок ушел, чтобы прервать его ход немедленно
			// Используем select, чтобы не заблокировать readPump, если канал полон (маловероятно, но безопасно)
			select {
			case c.Game.DisconnectChan <- c.EntityID:
			default:
			}
		}
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logger.Log.WithError(err).Warn("failed to set read deadline")
	}
	c.Conn.SetPongHandler(func(string) error {
		if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logger.Log.WithError(err).Warn("failed to set pong read deadline")
		}
		return nil
	})

	// 1. HANDSHAKE (LOGIN)
	var loginCmd api.ClientCommand
	if err := c.Conn.ReadJSON(&loginCmd); err != nil {
		logger.Log.Warn("Handshake failed")
		return
	}

	c.EntityID = domain.EntityID(loginCmd.Token)
	if c.EntityID == "" {
		c.EntityID = utils.GenerateID()
	}

	// 2. ПОИСК ИЛИ СОЗДАНИЕ ИГРОКА
	ent := c.Game.GetEntity(c.EntityID)
	if ent == nil {
		logger.Log.Infof("Player %s not found. Spawning...", c.EntityID)
		// Сид зависит только от имени игрока.
		// Это гарантирует, что и в Live-режиме, и в Replay-режиме
		// предметы в инвентаре получат одни и те же ID.
		playerSeed := utils.StringToSeed(c.EntityID.String())
		playerRng := rand.New(rand.NewSource(playerSeed))

		newPlayer := dungeon.CreatePlayer(c.EntityID, playerRng)

		// Ищем место для спавна на уровне 0
		world := c.Game.Worlds[0]
		placed := false
		// Сканируем центр карты
		for y := 10; y < 20; y++ {
			for x := 15; x < 25; x++ {
				if !world.Map[y][x].IsWall && len(world.GetEntitiesAt(x, y)) == 0 {
					newPlayer.Pos = domain.Position{X: x, Y: y}
					placed = true
					goto Done
				}
			}
		}
	Done:
		if !placed {
			newPlayer.Pos = domain.Position{X: 1, Y: 1} // Fallback
		}

		// Отправляем в движок через канал
		c.Game.JoinChan <- newPlayer

		// Даем движку мгновение на обработку
		time.Sleep(50 * time.Millisecond)
		ent = newPlayer
	}

	ent.ControllerID = "session_" + c.EntityID.String()
	logger.Log.WithFields(logrus.Fields{
		"entity_id": c.EntityID,
		"name":      ent.Name,
	}).Info("Client logged in")

	// 3. ПОДПИСКА НА ОБНОВЛЕНИЯ
	gameUpdates := c.Game.Hub.Register(c.EntityID)

	// Запускаем пересылку обновлений из Hub в writePump
	go func() {
		for msg := range gameUpdates {
			c.Send <- msg
		}
		close(c.Send)
	}()

	// Отправляем INIT (триггер первой отрисовки)
	c.Game.ProcessCommand(api.ClientCommand{Action: "INIT", Token: c.EntityID.String()})

	// 4. ЦИКЛ ЧТЕНИЯ КОМАНД
	for {
		var cmd api.ClientCommand
		err := c.Conn.ReadJSON(&cmd)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Errorf("WS Error: %v", err)
			}
			break
		}
		cmd.Token = c.EntityID.String()
		c.Game.ProcessCommand(cmd)
	}
}

// writePump отправляет данные клиенту + Ping
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.Conn.Close(); err != nil {
			logger.Log.WithError(err).Warn("failed to close websocket connection in writePump")
		}
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Log.WithError(err).Warn("failed to set write deadline")
			}
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.Log.WithError(err).Debug("write close message failed")
				}
				return
			}
			if err := c.Conn.WriteJSON(message); err != nil {
				logger.Log.WithError(err).Debug("write json message failed")
				return
			}

		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Log.WithError(err).Warn("failed to set ping write deadline")
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log.WithError(err).Debug("ping failed")
				return
			}
		}
	}
}
