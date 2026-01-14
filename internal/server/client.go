package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/pkg/logger"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	server *Server

	session *Session

	sendChan chan interface{}

	closed atomic.Bool
}

func (c *Client) Session() *Session {
	return c.session
}

func (c *Client) onAuthenticated() {
	c.server.onClientAuthenticated(c)
}

func (c *Client) OnLogin(guid types.ObjectGuid) {
	c.session.Authenticate(guid)
	c.onAuthenticated()
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
		session:  NewSession(),
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

		c.server.handler.Handle(c, msg)
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
			snapshot := c.server.Gateway.GetSnapshot(c.session.objectGuid)

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

	if c.session.ObjectGuid() != types.NilObjectGuid {
		c.server.unregisterClient(c.session.ObjectGuid())
	}

	c.conn.Close()
	close(c.sendChan)
}
