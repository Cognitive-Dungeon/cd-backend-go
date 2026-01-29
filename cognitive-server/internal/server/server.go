package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/gateway"
	"cognitive-server/internal/network/connection"
	"cognitive-server/internal/network/protocol"
	"cognitive-server/internal/network/session"
	"cognitive-server/pkg/logger"
	"net/http"
	_ "net/http/pprof"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Server struct {
	dispatcher protocol.Dispatcher  // обработка входящих сообщений
	gateway    *gateway.GameGateway // доступ к игровому миру (snapshot)
	upgrader   websocket.Upgrader
	registry   *connection.ClientRegistry
}

func New(gw *gateway.GameGateway, dispatcher protocol.Dispatcher) *Server {
	logger.Log.WithFields(logrus.Fields{
		"layer": "server",
	}).Info("server created")

	return &Server{
		gateway:    gw,
		dispatcher: dispatcher,
		registry:   connection.NewClientRegistry(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "server",
		"remote": r.RemoteAddr,
	})

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Warn("websocket upgrade failed")
		return
	}

	log.Debug("websocket connection accepted")

	client := connection.NewClient(
		conn,
		s, // MessageSink
		s, // SnapshotProvider
		s, // ClientEvents
	)

	go client.ReadLoop()
}

func (s *Server) OnAuthenticated(c *connection.Client, sess *session.Session) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "server",
		"guid":  sess.ObjectGuid(),
	})

	log.Info("client authenticated")

	s.registry.Register(sess.ObjectGuid(), c)
	go c.WriteLoop()
}

func (s *Server) OnDisconnected(c *connection.Client) {
	log := logger.Log.WithField("layer", "server")

	if c.Session().IsAuthenticated() {
		guid := c.Session().ObjectGuid()
		log = log.WithField("guid", guid)

		log.Info("authenticated client disconnected")
		s.registry.Unregister(guid)
		return
	}

	log.Debug("unauthenticated client disconnected")
}

func (s *Server) SendToAgent(guid types.ObjectGuid, msg interface{}) bool {
	ok := s.registry.SendToAgent(guid, msg)

	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"layer": "server",
			"guid":  guid,
		}).Debug("send to agent failed")
	}

	return ok
}

func (s *Server) HandleMessage(c *connection.Client, msg api.InboundMessage) {
	s.dispatcher.Handle(c, msg)
}

func (s *Server) GetSnapshotFor(c *connection.Client) *api.ServerResponse {
	if !c.Session().IsAuthenticated() {
		return nil
	}

	guid := c.Session().ObjectGuid()

	snapshot := s.gateway.GetSnapshot(guid)
	if snapshot == nil {
		logger.Log.WithFields(logrus.Fields{
			"layer": "server",
			"guid":  guid,
		}).Warn("nil snapshot returned from gateway")
	}

	return snapshot
}
