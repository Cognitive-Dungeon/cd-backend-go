package utils

import (
	"cognitive-server/internal/domain"
	"encoding/hex"
	"hash/fnv"
	"math/rand"
)

// GenerateID создает простой уникальный ID (замена UUID для снижения зависимостей)
func GenerateID() domain.EntityID {
	b := make([]byte, 8) // 16 символов hex
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random ID: " + err.Error())
	}
	return domain.EntityID(hex.EncodeToString(b))
}

// GenerateDeterministicID генерирует ID на основе переданного RNG.
// Это гарантирует, что последовательность ID будет одинаковой при одинаковом Seed.
func GenerateDeterministicID(rng *rand.Rand, prefix string) domain.EntityID {
	b := make([]byte, 8)
	rng.Read(b) // rand.Read заполняет байты псевдослучайно на основе сида
	return domain.EntityID(prefix + hex.EncodeToString(b))
}

// StringToSeed создает детерминированный сид из строки.
func StringToSeed(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}
