package app

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App отвечает за жизненный цикл всего приложения.
// Он связывает слои: Config -> Gateway (Server) -> Engine (GameService).
type App struct {
	cfg *config.Config

	// Пока используем старые структуры, позже заменим на интерфейсы
	gameService *engine.GameService
	httpServer  *server.Server
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger.Init(cfg.Log)

	logger.Log.Info("🚀 Initializing Cognitive Dungeon...")
	logger.Log.Info(version.String())

	return &App{
		cfg: cfg,
	}, nil
}

// Run запускает приложение и блокирует выполнение до сигнала остановки.
func (a *App) Run() error {
	// 3. Инициализация движка (Engine Layer)
	// Адаптируем конфиг под старый формат движка пока не отрефакторим Engine
	engineCfg := engine.Config{
		Seed:    a.cfg.Sim.MasterSeed,
		ShardId: a.cfg.Server.ShardID,
	}

	// ВАЖНО: Тут мы решаем, запускать Реплей или Живую игру.
	// Это убирает if/else из main.go
	a.gameService = engine.NewService(engineCfg)

	if a.cfg.Replay.Path != "" {
		return a.runReplayMode()
	}

	return a.runLiveMode()
}

func (a *App) runLiveMode() error {
	logger.Log.Info("🎮 Mode: Live Server")

	// Запуск игрового цикла
	a.gameService.Start() // Сейчас это запускает DispatcherLoop

	// Инициализация сети (Gateway Layer)
	a.httpServer = server.New(a.gameService, a.cfg.Server.Port)

	// Запуск HTTP сервера в горутине
	go func() {
		if err := a.httpServer.Run(); err != nil {
			logger.Log.Fatal("Server startup failed:", err)
		}
	}()

	// Graceful Shutdown
	return a.waitShutdown()
}

func (a *App) runReplayMode() error {
	logger.Log.Info("💿 Mode: Replay Simulation")

	if err := a.gameService.LoadReplay(a.cfg.Replay.Path); err != nil {
		return err
	}

	// Запускаем симуляцию
	// Логика взята из старого main.go, но теперь она изолирована
	simulatedCount := 0
	for id, inst := range a.gameService.Instances {
		if inst.IsPlayback {
			a.gameService.StartPlayback(id)
			simulatedCount++
		}
	}

	if simulatedCount == 0 {
		logger.Log.Warn("No instances ready for playback found.")
	}

	logger.Log.Info("Replay finished. Exiting.")
	return nil
}

func (a *App) waitShutdown() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Log.Info("🛑 Shutting down...")

	// Graceful shutdown logic
	// Например, сохранение состояния миров
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Сохраняем реплеи
	for _, inst := range a.gameService.Instances {
		inst.SaveReplay()
	}

	// Тут можно добавить a.httpServer.Shutdown(ctx) если реализовать

	logger.Log.Info("Bye.")
	return nil
}
