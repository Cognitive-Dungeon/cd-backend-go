package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	server *Server

	// GUID сущности. Если 0 (Nil), значит клиент еще не залогинился.
	objectGuid types.ObjectGuid
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Errorf("WS Upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		server: s,
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
	// ЛОГ №1: Видим ли мы вообще сообщение?
	logger.Log.Debugf("WS Recv: Action=%s Payload=%s", msg.Action, string(msg.Payload))
	if c.objectGuid == 0 && msg.Action != "LOGIN" {
		logger.Log.Warn("Ignored command before LOGIN")
		return
	}

	switch msg.Action {
	case "LOGIN":
		// Вся логика создания игрока теперь в Gateway
		// Мы передаем callback, который выполнится, когда игрок будет создан
		c.server.Gateway.HandleLogin(msg.Token, func(guid engine.ObjectGuid) {
			c.objectGuid = guid
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

	default:
		logger.Log.Warnf("Unknown Action: %s", msg.Action)
	}

}

func (c *Client) writeLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ticker.C:
			// Получаем снапшот через Gateway
			snapshot := c.server.Gateway.GetSnapshot(c.objectGuid)

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(snapshot); err != nil {
				return
			}
		}
	}
}
