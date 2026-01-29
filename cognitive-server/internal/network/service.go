package network

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/gateway"
	"cognitive-server/internal/network/protocol"
	"cognitive-server/internal/network/transport"
	"cognitive-server/internal/server"
	"cognitive-server/pkg/logger"
	"context"
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
	dispatcher := protocol.NewDispatcher(
		protocol.NewLoginHandler(n.gw),
		protocol.NewMoveHandler(n.gw),
		protocol.NewCastHandler(n.gw),
		protocol.NewChatHandler(n.gw),
	)
	n.srv = server.New(n.gw, dispatcher)
	httpSrv := transport.NewHTTP(n.cfg.Port, n.srv)
	n.gw.SetNetworkCallback(n.srv)

	go func() {
		logger.Log.Infof("🌐 Network: Starting on port %d...", n.cfg.Port)
		if err := httpSrv.Run(); err != nil {
			logger.Log.Errorf("🌐 Network Error: %v", err)
		}
	}()
}

func (n *Service) Stop(ctx context.Context) {
	logger.Log.Info("🌐 Network: Stopped")
}
