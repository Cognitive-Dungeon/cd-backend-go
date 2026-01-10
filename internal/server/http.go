package server

import (
	"cognitive-server/internal/engine"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	_ "net/http/pprof" // Profiling
	"strconv"
)

type Server struct {
	Engine *engine.GameService
	Port   uint16
}

func New(engine *engine.GameService, port uint16) *Server {
	return &Server{
		Engine: engine,
		Port:   port,
	}
}

// Run запускает HTTP сервер
func (s *Server) Run() error {
	mux := http.DefaultServeMux

	// Регистрируем роуты
	mux.HandleFunc("/ws", enableCORS(s.handleWS))
	mux.HandleFunc("/health", enableCORS(s.handleHealth))
	mux.HandleFunc("/version", enableCORS(s.handleVersion))

	// Debug Routes (из вашего debug.go, который теперь часть пакета server)
	debugHandler := NewDebugHandler(s.Engine)
	debugHandler.RegisterRoutes(mux)

	logger.Log.Info("🛡️  Cognitive Dungeon Server running on :" + strconv.Itoa(int(s.Port)))
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
