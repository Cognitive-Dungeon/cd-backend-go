package server

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/gateway"
	"cognitive-server/pkg/logger"
	"net/http"
	_ "net/http/pprof" // Profiling
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	Gateway  *gateway.GameGateway
	upgrader websocket.Upgrader
	handler  MessageHandler
	clients  map[types.ObjectGuid]*Client
	mu       sync.RWMutex
}

func New(gw *gateway.GameGateway, handler MessageHandler) *Server {
	return &Server{
		Gateway: gw,
		handler: handler,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients: make(map[types.ObjectGuid]*Client),
	}
}

func (s *Server) SendToAgent(objectGuid types.ObjectGuid, msg interface{}) {
	s.mu.RLock()
	client := s.clients[objectGuid]
	s.mu.RUnlock()

	if client == nil {
		logger.Log.Warnf("[Network] Client not found in map for AgentID: %s", objectGuid)
		return
	}

	select {
	case client.sendChan <- msg:
	default:
		// Канал забит или никто не читает — ДРОП!
		// Сюда мы попадаем, если writeLoop завис или канал не буферизирован
		logger.Log.Warnf("Outgoing queue full for %s. Message dropped!", objectGuid)
	}
}

func (s *Server) registerClient(guid types.ObjectGuid, c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[guid] = c
}

func (s *Server) unregisterClient(guid types.ObjectGuid) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, guid)
}

func (s *Server) onClientAuthenticated(c *Client) {
	guid := c.objectGuid

	s.registerClient(guid, c)
	go c.writeLoop()
}
