package interfaces

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type IApiKeyRepository interface {
	Create(apiKey *entities.ApiKey) error
	FindBySecretHash(secretHash string) (*entities.ApiKey, error)
	MarkRevoked(apiKeyID uint, revokedAt time.Time) (bool, error)
}
