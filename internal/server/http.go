package server

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/gateway"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/version"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Profiling
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	Gateway  *gateway.GameGateway // <--- Заменили Engine на Gateway
	Port     uint16
	upgrader websocket.Upgrader
	clients  map[types.ObjectGuid]*Client
	mu       sync.RWMutex
}

func New(gw *gateway.GameGateway, port uint16) *Server {
	return &Server{
		Gateway: gw,
		Port:    port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients: make(map[types.ObjectGuid]*Client),
	}
}

// Run запускает HTTP сервер
func (s *Server) Run() error {
	mux := http.DefaultServeMux

	// Регистрируем роуты
	mux.HandleFunc("/ws", enableCORS(s.handleWS))
	mux.HandleFunc("/health", enableCORS(s.handleHealth))
	mux.HandleFunc("/version", enableCORS(s.handleVersion))

	addr := fmt.Sprintf(":%d", s.Port)
	logger.Log.Infof("🛡️  Cognitive Dungeon Server running on %s", addr)
	return http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), mux)
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем запросы с фронтенда
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Разрешаем заголовки, если фронт шлет что-то нестандартное
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version.Info())
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
