package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Log является глобальным экземпляром логгера для всего приложения.
var Log *logrus.Logger

// Init инициализирует глобальный логгер.
// Эта функция должна быть вызвана один раз при старте приложения в main.go.
func Init() {
	Log = logrus.New()

	// 1. Устанавливаем уровень логирования из переменной окружения.
	// По умолчанию - "info". Для отладки можно выставить "debug".
	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "info"
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)

	// 2. Устанавливаем форматтер.
	// "json" - для продакшена и сбора логов.
	// "text" - для удобной разработки.
	logFormat := strings.ToLower(os.Getenv("LOG_FORMAT"))
	if logFormat == "json" {
		Log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	// 3. Устанавливаем, куда писать логи (в стандартный вывод).
	Log.SetOutput(os.Stdout)
}
