package main

import (
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/logger"
	"os"
)

func init() {
	logger.Init()
}

func main() {
	logger.Log.Info("Starting Cognitive Dungeon...")
	logger.Log.Info(version.String())

	port := os.Getenv("CD_PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Инициализация ядра (Game Service)
	// Сервис создает миры, запускает циклы симуляции.
	gameService := engine.NewService()
	gameService.Start() // Запускает DispatcherLoop

	// 2. Инициализация и запуск API сервера (HTTP + WS)
	srv := server.New(gameService, port)

	if err := srv.Run(); err != nil {
		logger.Log.Fatal("Server start error:", err)
	}
}
