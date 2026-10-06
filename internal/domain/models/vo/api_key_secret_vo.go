package vo

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	ApiKeySecretRandomByteLength = 32
	apiKeySecretPrefix           = "snh_"
)

var ErrApiKeyMissing = errors.New("需要提供 API key")

type ApiKeySecretVo struct {
	Plaintext string
	Hash      string
}

func NewIssuedApiKeySecretVo(randomBytes []byte) ApiKeySecretVo {
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
