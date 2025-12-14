package server

import (
	"cognitive-server/internal/engine"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	_ "net/http/pprof" // Profiling
)

type Server struct {
	Engine *engine.GameService
	Port   string
}

func New(engine *engine.GameService, port string) *Server {
	return &Server{
		Engine: engine,
		Port:   port,
	}
}

// Run запускает HTTP сервер
func (s *Server) Run() error {
	mux := http.DefaultServeMux

	// Регистрируем роуты
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)

	// Debug Routes (из вашего debug.go, который теперь часть пакета server)
	debugHandler := NewDebugHandler(s.Engine)
	debugHandler.RegisterRoutes(mux)

	logger.Log.Infof("🛡️  Cognitive Dungeon Server running on :%s", s.Port)
	return http.ListenAndServe(":"+s.Port, mux)
}

// handleWS обрабатывает подключение по WebSocket
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Error("Upgrade error:", err)
		return
	}

	client := NewClient(s.Engine, conn)

	// Запускаем пампы
	go client.writePump()
	go client.readPump()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version.Info())
}
