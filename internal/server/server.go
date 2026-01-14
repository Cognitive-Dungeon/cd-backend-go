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
	_ "net/http/pprof" // Profiling

	"github.com/gorilla/websocket"
)

type Server struct {
	dispatcher *protocol.Dispatcher // обработка входящих сообщений
	gateway    *gateway.GameGateway // доступ к игровому миру (snapshot)
	upgrader   websocket.Upgrader
	registry   *connection.ClientRegistry
}

func New(gw *gateway.GameGateway, dispatcher *protocol.Dispatcher) *Server {
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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Errorf("WS upgrade error: %v", err)
		return
	}

	client := connection.NewClient(
		conn,
		s, // MessageSink
		s, // SnapshotProvider
		s, // ClientEvents
	)

	go client.ReadLoop()
}

func (s *Server) OnAuthenticated(c *connection.Client, sess *session.Session) {
	guid := sess.ObjectGuid()
	s.registry.Register(guid, c)
	go c.WriteLoop()
}

func (s *Server) OnDisconnected(c *connection.Client) {
	if c.Session().IsAuthenticated() {
		s.registry.Unregister(c.Session().ObjectGuid())
	}
}

func (s *Server) SendToAgent(guid types.ObjectGuid, msg interface{}) {
	s.registry.SendToAgent(guid, msg)
}

func (s *Server) HandleMessage(c *connection.Client, msg api.InboundMessage) {
	s.dispatcher.Handle(c, msg)
}

func (s *Server) GetSnapshotFor(c *connection.Client) *api.ServerResponse {
	if !c.Session().IsAuthenticated() {
		return nil
	}
	return s.gateway.GetSnapshot(c.Session().ObjectGuid())
}
