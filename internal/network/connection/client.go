package connection

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/network/session"
	"cognitive-server/pkg/logger"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	sink      MessageSink
	snapshots SnapshotProvider
	events    ClientEvents

	session *session.Session

	sendChan chan interface{}

	closed atomic.Bool
}

func NewClient(
	conn *websocket.Conn,
	sink MessageSink,
	snapshots SnapshotProvider,
	events ClientEvents,
) *Client {
	return &Client{
		conn:      conn,
		sink:      sink,
		snapshots: snapshots,
		events:    events,
		session:   session.NewSession(),
		sendChan:  make(chan interface{}, 64),
	}
}

func (c *Client) Session() *session.Session {
	return c.session
}

func (c *Client) OnLogin(guid types.ObjectGuid) {
	c.session.Authenticate(guid)
	c.events.OnAuthenticated(c, c.session)
}

func (c *Client) ReadLoop() {
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

		c.sink.HandleMessage(c, msg)
	}
}

func (c *Client) WriteLoop() {
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
			snapshot := c.snapshots.GetSnapshotFor(c)

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
		return
	}

	c.events.OnDisconnected(c)

	c.conn.Close()
	close(c.sendChan)
}
