package eventbus

import (
	"cognitive-server/pkg/logger"
	"runtime/debug"
	"time"
)

// PanicRecovery возвращает Middleware, который перехватывает паники в обработчиках.
// Это предотвращает падение всего сервера из-за ошибки в одной игровой системе.
func PanicRecovery() Middleware {
	return func(next Handler) Handler {
		return func(event any) {
			defer func() {
				if r := recover(); r != nil {
					// Логируем стек-трейс ошибки, но не роняем приложение.
					logger.Log.Errorf("PANIC recovered in EventBus:\nError: %v\nStack: %s", r, string(debug.Stack()))
				}
			}()
			next(event)
		}
	}
}

// PerformanceMonitor возвращает Middleware, который логирует события,
// обработка которых заняла больше threshold времени.
// Полезно для отладки лагов ("фризов") в игровом цикле.
func PerformanceMonitor(threshold time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(event any) {
			start := time.Now()
			next(event)
			duration := time.Since(start)

			if duration > threshold {
				// Используем fmt.Sprintf для определения типа события, если это возможно,
				// или просто логируем факт задержки.
				logger.Log.Warnf("Slow event handler detected! Duration: %v, Event: %T", duration, event)
			}
		}
	}
}
