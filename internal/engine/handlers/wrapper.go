package handlers

import (
	"cognitive-server/pkg/api"
	"encoding/json"
	"fmt"
)

// TypedHandlerFunc - это "чистый" хендлер, который работает с готовой структурой T
type TypedHandlerFunc[T any] func(ctx Context, payload T) (Result, error)

// EmptyHandlerFunc - хендлер, которому НЕ нужны данные (INIT, WAIT)
type EmptyHandlerFunc func(ctx Context) (Result, error)

// WithPayload берет "чистый" хендлер и превращает его в стандартный HandlerFunc.
// Она берет на себя Unmarshal и Validate.
func WithPayload[T any](handler TypedHandlerFunc[T]) HandlerFunc {
	return func(ctx Context, raw json.RawMessage) (Result, error) {
		var payload T

		// 1. Распаковка JSON
		if err := json.Unmarshal(raw, &payload); err != nil {
			return Result{}, fmt.Errorf("invalid payload format: %w", err)
		}

		// 2. Автоматическая валидация
		// Проверяем, реализует ли структура T интерфейс Validator
		if v, ok := any(payload).(api.Validator); ok {
			if err := v.Validate(); err != nil {
				return Result{}, fmt.Errorf("validation failed: %w", err)
			}
		}

		// 3. Вызов чистой логики
		return handler(ctx, payload)
	}
}

// WithEmptyPayload - обертка для команд без данных (INIT, WAIT)
func WithEmptyPayload(handler EmptyHandlerFunc) HandlerFunc {
	return func(ctx Context, _ json.RawMessage) (Result, error) {
		// Мы просто игнорируем входящий JSON, так как он не нужен логике.
		return handler(ctx)
	}
}
