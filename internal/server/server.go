package server

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/gateway"
	"net/http"
	_ "net/http/pprof" // Profiling

	"github.com/gorilla/websocket"
)

type Server struct {
	Gateway  *gateway.GameGateway
	upgrader websocket.Upgrader
	handler  MessageHandler
	registry *ClientRegistry
}

func New(gw *gateway.GameGateway, handler MessageHandler) *Server {
	return &Server{
		Gateway:  gw,
		handler:  handler,
		registry: NewClientRegistry(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) onClientAuthenticated(c *Client) {
	guid := c.Session().ObjectGuid()
	s.registry.Register(guid, c)
	go c.writeLoop()
}

func (s *Server) onClientDisconnected(c *Client) {
	if c.Session().IsAuthenticated() {
		s.registry.Unregister(c.Session().ObjectGuid())
	}
}

func (s *Server) SendToAgent(guid types.ObjectGuid, msg interface{}) {
	s.registry.SendToAgent(guid, msg)
}
