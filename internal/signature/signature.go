package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header содержит имя HTTP заголовка с подписью тела запроса или ответа
const Header = "HashSHA256"

// Calculate вычисляет HMAC SHA256 и возвращает подпись в шестнадцатеричном виде
func Calculate(data []byte, key string) string {
	return hex.EncodeToString(calculate(data, key))
}

func calculate(data []byte, key string) []byte {
	hash := hmac.New(sha256.New, []byte(key))
	_, _ = hash.Write(data)
	return hash.Sum(nil)
}

// Valid проверяет шестнадцатеричную подпись HMAC SHA256 для data и key
func Valid(data []byte, key string, encodedSignature string) bool {
	received, err := hex.DecodeString(encodedSignature)
	if err != nil {
		return false
	}

	return hmac.Equal(calculate(data, key), received)
}
