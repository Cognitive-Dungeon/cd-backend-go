package main

import (
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"flag"
	"os"
)

func init() {
	logger.Init()
}

func main() {
	// 1. Парсинг конфигурации
	var seed int64
	// Читаем флаг -seed. По умолчанию 0 (значит сгенерировать случайно).
	flag.Int64Var(&seed, "seed", 0, "Initial world seed (0 for random)")
	flag.Parse()

	logger.Log.Info("Starting Cognitive Dungeon...")
	logger.Log.Info(version.String())

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

	// 3. Запуск сервера
	srv := server.New(gameService, port)
	if err := srv.Run(); err != nil {
		logger.Log.Fatal("Server start error:", err)
	}
}
