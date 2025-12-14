package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID создает простой уникальный ID (замена UUID для снижения зависимостей)
func GenerateID() string {
	b := make([]byte, 8) // 16 символов hex
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
