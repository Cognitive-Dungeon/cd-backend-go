package network

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/gateway"
	"cognitive-server/internal/server"
	"cognitive-server/pkg/logger"
	"context"
	"net/http"
)

type Service struct {
	cfg *config.ServerConfig
	gw  *gateway.GameGateway // <--- Вместо Engine
	srv *server.Server
}

// New принимает Engine, создает Gateway и передает его серверу
func New(cfg *config.ServerConfig, eng *engine.Engine) *Service {
	gw := gateway.New(eng)
	return &Service{
		cfg: cfg,
		gw:  gw,
	}
}

func (n *Service) Start() {
	// Передаем Gateway в сервер
	n.srv = server.New(n.gw, n.cfg.Port)

	go func() {
		logger.Log.Infof("🌐 Network: Starting listener on port %d...", n.cfg.Port)
		if err := n.srv.Run(); err != nil && err != http.ErrServerClosed {
			logger.Log.Errorf("🌐 Network Error: %v", err)
		}
	}()
}

func (n *Service) Stop(ctx context.Context) {
	logger.Log.Info("🌐 Network: Stopped")
}
