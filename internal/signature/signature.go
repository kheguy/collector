package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const Header = "HashSHA256"

func Calculate(data []byte, key string) string {
	return hex.EncodeToString(calculate(data, key))
}

func calculate(data []byte, key string) []byte {
	hash := hmac.New(sha256.New, []byte(key))
	_, _ = hash.Write(data)
	return hash.Sum(nil)
}

func Valid(data []byte, key string, encodedSignature string) bool {
	received, err := hex.DecodeString(encodedSignature)
	if err != nil {
		return false
	}

	return hmac.Equal(calculate(data, key), received)
}
