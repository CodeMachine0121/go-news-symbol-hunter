package persistence_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiKeyRepository_CreateThenFindBySecretHash(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))
	apiKey := entities.ApiKey{Name: "研究", SecretHash: "hash-created"}

	require.NoError(t, apiKeyRepository.Create(&apiKey))
	foundApiKey, err := apiKeyRepository.FindBySecretHash("hash-created")

	require.NoError(t, err)
	require.NotNil(t, foundApiKey)
	assert.Equal(t, apiKey.ID, foundApiKey.ID)
	assert.Equal(t, "研究", foundApiKey.Name)
	assert.False(t, foundApiKey.IsActive)
	assert.Nil(t, foundApiKey.RevokedAt)
}

func TestApiKeyRepository_FindBySecretHash_UnknownHashReturnsNothing(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))

	foundApiKey, err := apiKeyRepository.FindBySecretHash("hash-unknown")

	assert.NoError(t, err)
	assert.Nil(t, foundApiKey)
}

func TestApiKeyRepository_Create_RejectsDuplicateSecretHash(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))
	require.NoError(t, apiKeyRepository.Create(&entities.ApiKey{Name: "第一把", SecretHash: "hash-duplicate"}))

	err := apiKeyRepository.Create(&entities.ApiKey{Name: "第二把", SecretHash: "hash-duplicate"})

	assert.Error(t, err)
}

func TestApiKeyRepository_MarkRevoked_LeavesActivationUntouched(t *testing.T) {
	database := openTestDatabase(t)
	apiKeyRepository := persistence.NewApiKeyRepository(database)
	apiKey := entities.ApiKey{Name: "研究", SecretHash: "hash-revoked"}
	require.NoError(t, apiKeyRepository.Create(&apiKey))
	require.NoError(t, database.Model(&entities.ApiKey{ID: apiKey.ID}).Updates(entities.ApiKey{IsActive: true}).Error)
	revokedAt := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	isMarkedRevoked, markError := apiKeyRepository.MarkRevoked(apiKey.ID, revokedAt)
	foundApiKey, err := apiKeyRepository.FindBySecretHash("hash-revoked")

	require.NoError(t, markError)
	require.NoError(t, err)
	assert.True(t, isMarkedRevoked)
	require.NotNil(t, foundApiKey.RevokedAt)
	assert.True(t, foundApiKey.RevokedAt.Equal(revokedAt))
	assert.True(t, foundApiKey.IsActive)
}

func TestApiKeyRepository_FindBySecretHash_ReportsStorageFailure(t *testing.T) {
	database := openTestDatabase(t)
	apiKeyRepository := persistence.NewApiKeyRepository(database)
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())

	foundApiKey, err := apiKeyRepository.FindBySecretHash("hash-any")

	assert.Error(t, err)
	assert.Nil(t, foundApiKey)
}

func TestApiKeyRepository_MarkRevoked_SecondRevocationChangesNothing(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))
	apiKey := entities.ApiKey{Name: "研究", SecretHash: "hash-twice"}
	require.NoError(t, apiKeyRepository.Create(&apiKey))
	firstRevokedAt := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	_, err := apiKeyRepository.MarkRevoked(apiKey.ID, firstRevokedAt)
	require.NoError(t, err)

	isMarkedRevoked, err := apiKeyRepository.MarkRevoked(apiKey.ID, firstRevokedAt.Add(time.Hour))
	foundApiKey, findError := apiKeyRepository.FindBySecretHash("hash-twice")

	require.NoError(t, err)
	require.NoError(t, findError)
	assert.False(t, isMarkedRevoked)
	assert.True(t, foundApiKey.RevokedAt.Equal(firstRevokedAt))
}

func TestApiKeyRepository_FindBySecretHash_EmptyHashMatchesNothing(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))
	require.NoError(t, apiKeyRepository.Create(&entities.ApiKey{Name: "研究", SecretHash: "hash-existing"}))

	foundApiKey, err := apiKeyRepository.FindBySecretHash("")

	assert.NoError(t, err)
	assert.Nil(t, foundApiKey)
}

func TestApiKeyRepository_MarkRevoked_ZeroIdRevokesNothing(t *testing.T) {
	apiKeyRepository := persistence.NewApiKeyRepository(openTestDatabase(t))
	require.NoError(t, apiKeyRepository.Create(&entities.ApiKey{Name: "研究", SecretHash: "hash-untouched"}))

	isMarkedRevoked, err := apiKeyRepository.MarkRevoked(0, time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	untouchedApiKey, findError := apiKeyRepository.FindBySecretHash("hash-untouched")

	require.NoError(t, err)
	require.NoError(t, findError)
	assert.False(t, isMarkedRevoked)
	assert.Nil(t, untouchedApiKey.RevokedAt)
}
