package app

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/network"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/version"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App отвечает за жизненный цикл всего приложения.
// Он связывает слои: Config -> Gateway (Server) -> Engine (GameService).
type App struct {
	Engine  *engine.Engine
	Network *network.Service
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger.Init(cfg.Log)

	log := logger.Log.WithField("layer", "app")

	log.Info("🚀 Initializing Cognitive Dungeon...")
	log.WithField("version", version.String()).
		Info("build info")

	eng := engine.New(&cfg.Sim)
	net := network.New(&cfg.Server, eng)

	return &App{
		Engine:  eng,
		Network: net,
	}, nil
}

// Run запускает приложение и блокирует выполнение до сигнала остановки.
func (a *App) Run() error {
	log := logger.Log.WithField("layer", "app")

	log.Info("starting application")

	// Контекст жизни всего приложения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск компонентов
	a.Network.Start()
	go a.Engine.Run()

	// Ожидание сигнала
	sig := waitSignal()
	log.WithField("signal", sig.String()).
		Info("shutdown signal received")

	// Graceful shutdown
	return a.shutdown(ctx)
}

func waitSignal() os.Signal {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	return <-stop
}

func (a *App) shutdown(parent context.Context) error {
	log := logger.Log.WithField("layer", "app")

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	log.Info("stopping engine")
	a.Engine.Stop(ctx)

	log.Info("stopping network")
	a.Network.Stop(ctx)

	log.Info("shutdown complete")
	return nil
}
