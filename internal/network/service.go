package network

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"cognitive-server/pkg/logger"
	"context"
	"errors"
	"net/http"
)

type Service struct {
	cfg    *config.ServerConfig
	engine *engine.Engine
	srv    *server.Server
}

func New(cfg *config.ServerConfig, eng *engine.Engine) *Service {
	return &Service{
		cfg:    cfg,
		engine: eng,
	}
}

func (n *Service) Start() {
	// Создаем и запускаем реальный HTTP сервер
	n.srv = server.New(n.engine, n.cfg.Port)

	go func() {
		logger.Log.Infof("🌐 Network: Starting listener on port %d...", n.cfg.Port)
		if err := n.srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Errorf("🌐 Network Error: %v", err)
		}
	}()
}

func (n *Service) Stop(ctx context.Context) {
	logger.Log.Info("🌐 Network: Stopped")
}
