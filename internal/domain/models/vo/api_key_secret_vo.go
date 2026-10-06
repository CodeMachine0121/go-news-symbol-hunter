package vo

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	apiKeySecretPrefix      = "snh_"
	apiKeySecretRandomBytes = 32
)

var ErrApiKeyMissing = errors.New("需要提供 API key")

type ApiKeySecretVo struct {
	Plaintext string
	Hash      string
}

func GenerateApiKeySecretVo() ApiKeySecretVo {
	randomBytes := make([]byte, apiKeySecretRandomBytes)
	// crypto/rand.Read never returns an error since Go 1.24; it crashes the process instead
	_, _ = rand.Read(randomBytes)
	secret, _ := NewApiKeySecretVo(apiKeySecretPrefix + base64.RawURLEncoding.EncodeToString(randomBytes))
	return secret
}

func NewApiKeySecretVo(presentedSecret string) (ApiKeySecretVo, error) {
	trimmedSecret := strings.TrimSpace(presentedSecret)
	if trimmedSecret == "" {
		return ApiKeySecretVo{}, ErrApiKeyMissing
	}
	secretDigest := sha256.Sum256([]byte(trimmedSecret))
	return ApiKeySecretVo{Plaintext: trimmedSecret, Hash: hex.EncodeToString(secretDigest[:])}, nil
}
