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
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.Log.Warnf("[conn] read error: %v", err)
			}
			return
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

			if snapshot == nil {
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(snapshot); err != nil {
				logger.Log.Debugf("[conn] snapshot write failed: %v", err)
				return
			}

		// 💬 Асинхронные сообщения (чат, нотификации)
		case msg := <-c.sendChan:
			if err := c.conn.WriteJSON(msg); err != nil {
				logger.Log.Debugf("[conn] msg write failed, closing client: %v", err)
				return
			}

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
