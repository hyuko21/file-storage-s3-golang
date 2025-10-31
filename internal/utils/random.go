package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func MakeRandomString() (string, error) {
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(randBytes), nil
}
