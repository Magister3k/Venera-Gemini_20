package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID генерирует случайный идентификатор фиксированной длины (16 символов, hex).
func GenerateID() string {
	bytes := make([]byte, 8) // 8 байт = 16 hex символов
	if _, err := rand.Read(bytes); err != nil {
		// В крайнем случае возвращаем fallback, хотя rand.Read редко ошибается
		return "0000000000000000"
	}
	return hex.EncodeToString(bytes)
}
