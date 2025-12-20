package engine

import "time"

// Config хранит параметры запуска движка
type Config struct {
	// Seed - мастер-зерно. От него будут зависеть все уровни.
	// Level N Seed = MasterSeed + N (или хеш от этого сочетания)
	Seed    int64
	ShardId uint8
}

// NewConfig создает конфиг по умолчанию (случайный сид)
func NewConfig() Config {
	return Config{
		Seed:    time.Now().UnixNano(),
		ShardId: 0,
	}
}
