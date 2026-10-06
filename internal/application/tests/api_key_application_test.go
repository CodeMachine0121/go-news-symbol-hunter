package application_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const presentedApiKey = "snh_presented"

var errDatabaseDown = errors.New("database down")

func createApiKeyApplication(t *testing.T) (*application.ApiKeyApplication, *mocks.MockIApiKeyRepository) {
	apiKeyRepository := mocks.NewMockIApiKeyRepository(t)
	return application.NewApiKeyApplication(service.NewApiKeyService(apiKeyRepository)), apiKeyRepository
}

func hashOf(secret string) string {
	digest := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(digest[:])
}

func TestIssueApiKey_StoresInactiveKeyWithOnlyTheHashOfTheSecret(t *testing.T) {
	apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
	storedApiKey := entities.ApiKey{}
	apiKeyRepository.EXPECT().Create(mock.Anything).RunAndReturn(func(apiKey *entities.ApiKey) error {
		apiKey.ID = 42
		storedApiKey = *apiKey
		return nil
	})

	issuedApiKey, err := apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: "  我的研究腳本  "})

	require.NoError(t, err)
	assert.Equal(t, "我的研究腳本", issuedApiKey.Name)
	assert.Equal(t, "inactive", issuedApiKey.Status)
	assert.Regexp(t, `^snh_`, issuedApiKey.ApiKey)
	assert.Equal(t, entities.ApiKey{ID: 42, Name: "我的研究腳本", SecretHash: hashOf(issuedApiKey.ApiKey), IsActive: false}, storedApiKey)
}

func TestIssueApiKey_SameNameTwiceYieldsDifferentKeys(t *testing.T) {
	apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
	apiKeyRepository.EXPECT().Create(mock.Anything).Return(nil).Times(2)

	firstApiKey, firstErr := apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: "研究"})
	secondApiKey, secondErr := apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: "研究"})

	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.NotEqual(t, firstApiKey.ApiKey, secondApiKey.ApiKey)
}

func TestIssueApiKey_Rejections(t *testing.T) {
	testCases := []struct {
		name          string
		apiKeyName    string
		givenStorage  func(apiKeyRepository *mocks.MockIApiKeyRepository)
		expectedError error
	}{
		{name: "missing name", apiKeyName: "", givenStorage: func(*mocks.MockIApiKeyRepository) {}, expectedError: service.ErrApiKeyNameRequired},
		{name: "name too long", apiKeyName: strings.Repeat("研", 101), givenStorage: func(*mocks.MockIApiKeyRepository) {}, expectedError: service.ErrApiKeyNameTooLong},
		{name: "storage unavailable", apiKeyName: "研究", givenStorage: func(apiKeyRepository *mocks.MockIApiKeyRepository) {
			apiKeyRepository.EXPECT().Create(mock.Anything).Return(errDatabaseDown)
		}, expectedError: service.ErrApiKeyStorageUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
			testCase.givenStorage(apiKeyRepository)

			issuedApiKey, err := apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: testCase.apiKeyName})

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Empty(t, issuedApiKey.ApiKey)
		})
	}
}

func TestGetApiKeyStatus(t *testing.T) {
	revokedAt := time.Now()
	testCases := []struct {
		name           string
		presentedKey   string
		storedApiKey   *entities.ApiKey
		storageError   error
		expectedStatus dto.ApiKeyStatusDto
		expectedError  error
	}{
		{name: "inactive key", presentedKey: presentedApiKey, storedApiKey: &entities.ApiKey{Name: "研究"}, expectedStatus: dto.ApiKeyStatusDto{Name: "研究", Status: "inactive"}},
		{name: "activated key", presentedKey: presentedApiKey, storedApiKey: &entities.ApiKey{Name: "研究", IsActive: true}, expectedStatus: dto.ApiKeyStatusDto{Name: "研究", Status: "active"}},
		{name: "revoked key", presentedKey: presentedApiKey, storedApiKey: &entities.ApiKey{Name: "研究", IsActive: true, RevokedAt: &revokedAt}, expectedError: service.ErrApiKeyInvalid},
		{name: "unknown key", presentedKey: presentedApiKey, storedApiKey: nil, expectedError: service.ErrApiKeyInvalid},
		{name: "storage unavailable", presentedKey: presentedApiKey, storageError: errDatabaseDown, expectedError: service.ErrApiKeyStorageUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
			apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(testCase.storedApiKey, testCase.storageError)

			status, err := apiKeyApplication.GetApiKeyStatus(testCase.presentedKey)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedStatus, status)
		})
	}
}

func TestGetApiKeyStatus_MissingKeyNeverReachesStorage(t *testing.T) {
	apiKeyApplication, _ := createApiKeyApplication(t)

	_, err := apiKeyApplication.GetApiKeyStatus("")

	assert.ErrorIs(t, err, service.ErrApiKeyMissing)
}

func TestRevokeApiKey_PersistsRevocationForActiveAndInactiveKeys(t *testing.T) {
	for _, isActive := range []bool{true, false} {
		apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
		apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 9, IsActive: isActive}, nil)
		apiKeyRepository.EXPECT().MarkRevoked(uint(9), mock.AnythingOfType("time.Time")).Return(true, nil)

		err := apiKeyApplication.RevokeApiKey(presentedApiKey)

		assert.NoError(t, err)
	}
}

func TestRevokeApiKey_Rejections(t *testing.T) {
	revokedAt := time.Now()
	testCases := []struct {
		name          string
		storedApiKey  *entities.ApiKey
		findError     error
		updateError   error
		isNotMarked   bool
		expectedError error
	}{
		{name: "already revoked", storedApiKey: &entities.ApiKey{ID: 9, RevokedAt: &revokedAt}, expectedError: service.ErrApiKeyInvalid},
		{name: "unknown key", storedApiKey: nil, expectedError: service.ErrApiKeyInvalid},
		{name: "storage unavailable on lookup", findError: errDatabaseDown, expectedError: service.ErrApiKeyStorageUnavailable},
		{name: "storage unavailable on update", storedApiKey: &entities.ApiKey{ID: 9}, updateError: errDatabaseDown, expectedError: service.ErrApiKeyStorageUnavailable},
		{name: "revoked concurrently by another request", storedApiKey: &entities.ApiKey{ID: 9}, isNotMarked: true, expectedError: service.ErrApiKeyInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
			apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(testCase.storedApiKey, testCase.findError)
			if testCase.updateError != nil || testCase.isNotMarked {
				apiKeyRepository.EXPECT().MarkRevoked(uint(9), mock.AnythingOfType("time.Time")).Return(false, testCase.updateError)
			}

			err := apiKeyApplication.RevokeApiKey(presentedApiKey)

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestRevokeApiKey_MissingKey(t *testing.T) {
	apiKeyApplication, _ := createApiKeyApplication(t)

	err := apiKeyApplication.RevokeApiKey("")

	assert.ErrorIs(t, err, service.ErrApiKeyMissing)
}

func TestAuthorizeApiKey(t *testing.T) {
	revokedAt := time.Now()
	testCases := []struct {
		name               string
		storedApiKey       *entities.ApiKey
		storageError       error
		expectedAuthorized dto.AuthorizedApiKeyDto
		expectedError      error
	}{
		{name: "active and not revoked", storedApiKey: &entities.ApiKey{ID: 5, IsActive: true}, expectedAuthorized: dto.AuthorizedApiKeyDto{ApiKeyID: 5}},
		{name: "inactive", storedApiKey: &entities.ApiKey{ID: 5}, expectedError: service.ErrApiKeyInactive},
		{name: "revoked then reactivated", storedApiKey: &entities.ApiKey{ID: 5, IsActive: true, RevokedAt: &revokedAt}, expectedError: service.ErrApiKeyInvalid},
		{name: "unknown key", storedApiKey: nil, expectedError: service.ErrApiKeyInvalid},
		{name: "storage unavailable", storageError: errDatabaseDown, expectedError: service.ErrApiKeyStorageUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiKeyApplication, apiKeyRepository := createApiKeyApplication(t)
			apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(testCase.storedApiKey, testCase.storageError)

			authorizedApiKey, err := apiKeyApplication.AuthorizeApiKey(presentedApiKey)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedAuthorized, authorizedApiKey)
		})
	}
}

func TestAuthorizeApiKey_MissingKey(t *testing.T) {
	apiKeyApplication, _ := createApiKeyApplication(t)

	_, err := apiKeyApplication.AuthorizeApiKey("")

	assert.ErrorIs(t, err, service.ErrApiKeyMissing)
}
