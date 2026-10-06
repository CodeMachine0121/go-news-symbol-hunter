package persistence

import (
	"errors"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	err := apiKeyRepository.database.Where(clause.Eq{Column: clause.Column{Name: "secret_hash"}, Value: secretHash}).First(&apiKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (apiKeyRepository *ApiKeyRepository) MarkRevoked(apiKeyID uint, revokedAt time.Time) (bool, error) {
	result := apiKeyRepository.database.Model(&entities.ApiKey{}).
		Where(clause.Eq{Column: clause.Column{Name: "id"}, Value: apiKeyID}).
		Where(clause.Eq{Column: clause.Column{Name: "revoked_at"}, Value: nil}).
		Updates(entities.ApiKey{RevokedAt: &revokedAt})
	return result.RowsAffected == 1, result.Error
}
