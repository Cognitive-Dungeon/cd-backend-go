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

	logger.Log.Info("🚀 Initializing Cognitive Dungeon...")
	logger.Log.Info(version.String())

	eng := engine.New(&cfg.Sim)
	net := network.New(&cfg.Server, eng)

	return &App{
		Engine:  eng,
		Network: net,
	}, nil
}

// Run запускает приложение и блокирует выполнение до сигнала остановки.
func (a *App) Run() error {
	// Запускаем сеть
	a.Network.Start()

	// Запускаем движок (блокирует поток)
	go a.Engine.Run()

	return a.waitShutdown()
}

func (a *App) waitShutdown() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Log.Info("🛑 Shutting down...")

	// Graceful shutdown logic
	// Например, сохранение состояния миров
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a.Engine.Stop(ctx)
	a.Network.Stop(ctx)

	// Тут можно добавить a.httpServer.Shutdown(ctx) если реализовать

	logger.Log.Info("Bye.")
	return nil
}
