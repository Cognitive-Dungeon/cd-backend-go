package logger

import (
	"cognitive-server/internal/config"
	"os"

	"github.com/sirupsen/logrus"
)

// Log является глобальным экземпляром логгера для всего приложения.
var Log *logrus.Logger

// Init инициализирует глобальный логгер.
// Эта функция должна быть вызвана один раз при старте приложения в main.go.
func Init(cfg config.LogConfig) {
	Log = logrus.New()

	Log.SetLevel(cfg.Level)

	switch cfg.Format {
	case config.LogJSON:
		Log.SetFormatter(&logrus.JSONFormatter{})
	case config.LogText:
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	Log.SetOutput(os.Stdout)
}
