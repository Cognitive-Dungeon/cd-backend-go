package connection

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/network/session"
	"cognitive-server/pkg/logger"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Client struct {
	conn *websocket.Conn

	session *session.Session

	sink      MessageSink
	snapshots SnapshotProvider
	events    ClientEvents

	sendChan chan interface{}
	closed   atomic.Bool
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

func (c *Client) ReadLoop() {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "conn",
		"remote": c.conn.RemoteAddr().String(),
	})

	defer c.cleanup()

	for {
		var msg api.InboundMessage
		if err := c.conn.ReadJSON(&msg); err != nil {

			// Неожиданное закрытие — логируем
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.WithError(err).Warn("unexpected websocket close")
			}

			// Обычное закрытие — молча
			return
		}

		c.sink.HandleMessage(c, msg)
	}
}

func (c *Client) WriteLoop() {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "conn",
		"remote": c.conn.RemoteAddr().String(),
	})

	ticker := time.NewTicker(50 * time.Millisecond)
	defer func() {
		ticker.Stop()
		c.cleanup()
	}()

	for {
		select {
		case <-ticker.C:
			snapshot := c.snapshots.GetSnapshotFor(c)
			if snapshot == nil {
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(snapshot); err != nil {
				log.WithError(err).Debug("snapshot write failed")
				return
			}

		case msg, ok := <-c.sendChan:
			if !ok {
				// Канал закрыт — корректно выходим
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				log.WithError(err).Debug("async message write failed")
				return
			}
		}
	}
}

func (c *Client) Send(msg interface{}) bool {
	select {
	case c.sendChan <- msg:
		return true
	default:
		return false
	}
}

func (c *Client) OnLogin(guid types.ObjectGuid) {
	c.session.Authenticate(guid)
	c.events.OnAuthenticated(c, c.session)
}

func (c *Client) cleanup() {
	if c.closed.Swap(true) {
		return
	}

	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "conn",
		"remote": c.conn.RemoteAddr().String(),
	})

	if c.session.IsAuthenticated() {
		log = log.WithField("guid", c.session.ObjectGuid())
	}

	log.Debug("client disconnected")

	c.events.OnDisconnected(c)

	_ = c.conn.Close()
	close(c.sendChan)
}
