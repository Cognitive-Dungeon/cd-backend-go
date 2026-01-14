package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	server *Server

	// GUID сущности. Если 0 (Nil), значит клиент еще не залогинился.
	objectGuid types.ObjectGuid

	sendChan chan interface{}

	closed atomic.Bool
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Errorf("WS Upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		server:   s,
		sendChan: make(chan interface{}, 64),
	}

	// Мы НЕ спавним игрока сразу. Мы ждем команду LOGIN.
	// Запускаем только чтение.
	go client.readLoop()
}

func (c *Client) readLoop() {
	defer c.cleanup()

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
	// ЛОГ №1: Видим ли мы вообще сообщение?
	logger.Log.Debugf("WS Recv: Action=%s Payload=%s", msg.Action, string(msg.Payload))
	if c.objectGuid == 0 && msg.Action != "LOGIN" {
		logger.Log.Warn("Ignored command before LOGIN")
		return
	}

	switch msg.Action {
	case "LOGIN":
		// 1. Берем токен из корня
		token := msg.Token

		// 2. Если пусто, пробуем достать из Payload
		if token == "" && len(msg.Payload) > 0 {
			var loginPayload struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(msg.Payload, &loginPayload); err == nil {
				token = loginPayload.Token
			}
		}

		c.server.Gateway.HandleLogin(token, func(guid engine.ObjectGuid) {
			c.objectGuid = guid
			c.server.registerClient(guid, c)
			// Запускаем отправку данных
			go c.writeLoop()
		})

	case "MOVE":
		var payload api.MovePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			// Gateway сам разберется с векторами и enum-ами
			c.server.Gateway.HandleMove(c.objectGuid, payload)
		}

	case "CAST":
		var payload api.CastPayload
		// ЛОГ №2: Ошибка JSON парсинга
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			c.server.Gateway.HandleCast(c.objectGuid, payload)
		} else {
			logger.Log.Errorf("CAST Unmarshal Error: %v. Payload: %s", err, string(msg.Payload))
		}

	case "CHAT":
		var payload api.ChatPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		c.server.Gateway.HandleChat(c.objectGuid, payload)

	default:
		logger.Log.Warnf("Unknown Action: %s", msg.Action)
	}

}

func (c *Client) writeLoop() {
	ticker := time.NewTicker(50 * time.Millisecond) // Вернули как было
	defer func() {
		ticker.Stop()
		c.cleanup()
	}()

	for {
		select {
		// 🔁 Мир
		case <-ticker.C:
			// Получаем снапшот через Gateway
			snapshot := c.server.Gateway.GetSnapshot(c.objectGuid)

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(snapshot); err != nil {
				return
			}

		// 💬 Асинхронные сообщения (чат, нотификации)
		case msg := <-c.sendChan:
			c.conn.WriteJSON(msg)

		}
	}
}

func (c *Client) cleanup() {
	if c.closed.Swap(true) {
		return // уже закрыт
	}

	if c.objectGuid != types.NilObjectGuid {
		c.server.unregisterClient(c.objectGuid)
	}

	c.conn.Close()
	close(c.sendChan)
}
