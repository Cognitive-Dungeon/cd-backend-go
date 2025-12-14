package main

import (
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"flag"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	logger.Init()
}

func main() {
	// 1. Парсинг конфигурации
	var seed int64
	var replayPath string
	// Читаем флаг -seed. По умолчанию 0 (значит сгенерировать случайно).
	flag.Int64Var(&seed, "seed", 0, "Initial world seed (0 for random)")
	flag.StringVar(&replayPath, "replay", "", "Path to .cdrp replay file to simulate")
	flag.Parse()

	logger.Log.Info("Starting Cognitive Dungeon...")
	logger.Log.Info(version.String())

	// РЕЖИМ РЕПЛЕЯ
	if replayPath != "" {
		logger.Log.Info("💿 Mode: Replay Simulation")

		// Создаем пустой сервис
		cfg := engine.NewConfig()
		gameService := engine.NewService(cfg) // NewService создает дефолтные миры, но мы их перезапишем или добавим свой

		// Загружаем реплей
		if err := gameService.LoadReplay(replayPath); err != nil {
			logger.Log.Fatal("Failed to load replay:", err)
		}

		// Запускаем симуляцию (предполагаем, что LevelID берется из файла, в LoadReplay мы создали инстанс)
		// Нам нужно узнать какой уровень запускать. LoadReplay создал инстанс в s.Instances.
		// Пробегаем по всем инстансам, но запускаем только те, где есть флаг IsPlayback
		simulatedCount := 0
		for id, inst := range gameService.Instances {
			if inst.IsPlayback {
				gameService.StartPlayback(id)
				simulatedCount++
			}
		}

		if simulatedCount == 0 {
			logger.Log.Warn("No instances ready for playback found.")
		}

		return // Выходим после симуляции
	}

	// Формируем конфиг
	cfg := engine.NewConfig()
	if seed != 0 {
		cfg.Seed = seed
		logger.Log.Infof("🎲 Using explicit Master Seed: %d", seed)
	} else {
		logger.Log.Infof("🎲 Using random Master Seed: %d", cfg.Seed)
	}

	port := os.Getenv("CD_PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Инициализация ядра с конфигом
	gameService := engine.NewService(cfg)
	gameService.Start()

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 3. Запуск сервера
	srv := server.New(gameService, port)

	go func() {
		if err := srv.Run(); err != nil {
			logger.Log.Fatal("Server start error:", err)
		}
	}()

	<-stop
	logger.Log.Info("Shutting down...")

	// Сохраняем все активные миры
	for _, inst := range gameService.Instances {
		inst.SaveReplay()
	}

	logger.Log.Info("Done.")
}
