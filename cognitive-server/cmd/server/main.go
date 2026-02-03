package main

import (
	"cognitive-server/internal/app"
	"cognitive-server/pkg/logger"
	"os"
)

func main() {
	application, err := app.New()
	if err != nil {
		logger.Log.WithError(err).
			Fatal("application initialization failed")
	}

	if err := application.Run(); err != nil {
		logger.Log.WithError(err).
			Error("application stopped with error")
		os.Exit(1)
	}
}
