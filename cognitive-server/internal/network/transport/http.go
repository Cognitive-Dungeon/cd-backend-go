package transport

import (
	"cognitive-server/internal/server"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/version"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPServer struct {
	port   uint16
	server *server.Server
}

func NewHTTP(port uint16, srv *server.Server) *HTTPServer {
	return &HTTPServer{
		port:   port,
		server: srv,
	}
}

func (h *HTTPServer) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", enableCORS(h.server.HandleWS))
	mux.HandleFunc("/health", enableCORS(handleHealth))
	mux.HandleFunc("/version", enableCORS(handleVersion))

	addr := fmt.Sprintf(":%d", h.port)
	logger.Log.Infof("🛡️ HTTP listening on %s", addr)

	return http.ListenAndServe(addr, mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version.Get())
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		next(w, r)
	}
}
