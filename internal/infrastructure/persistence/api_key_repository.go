package persistence

import (
	"errors"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type ApiKeyRepository struct {
	database *gorm.DB
}

func NewApiKeyRepository(database *gorm.DB) *ApiKeyRepository {
	return &ApiKeyRepository{database: database}
}

func (apiKeyRepository *ApiKeyRepository) Create(apiKey *entities.ApiKey) error {
	return apiKeyRepository.database.Create(apiKey).Error
}

func (apiKeyRepository *ApiKeyRepository) FindBySecretHash(secretHash string) (*entities.ApiKey, error) {
	var apiKey entities.ApiKey
	err := apiKeyRepository.database.Where(&entities.ApiKey{SecretHash: secretHash}).First(&apiKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (apiKeyRepository *ApiKeyRepository) UpdateRevokedAt(apiKeyID uint, revokedAt time.Time) error {
	return apiKeyRepository.database.Model(&entities.ApiKey{ID: apiKeyID}).Updates(entities.ApiKey{RevokedAt: &revokedAt}).Error
}
