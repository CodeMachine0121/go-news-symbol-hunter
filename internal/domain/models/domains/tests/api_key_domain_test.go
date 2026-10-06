package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var revokedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func TestApiKeyDomain_Authorize(t *testing.T) {
	testCases := []struct {
		name          string
		apiKey        entities.ApiKey
		expectedError error
	}{
		{name: "active and not revoked passes", apiKey: entities.ApiKey{IsActive: true}},
		{name: "inactive is rejected as not yet activated", apiKey: entities.ApiKey{IsActive: false}, expectedError: domains.ErrApiKeyInactive},
		{name: "revoked is invalid even when reactivated", apiKey: entities.ApiKey{IsActive: true, RevokedAt: &revokedAt}, expectedError: domains.ErrApiKeyInvalid},
		{name: "revoked and inactive is invalid", apiKey: entities.ApiKey{IsActive: false, RevokedAt: &revokedAt}, expectedError: domains.ErrApiKeyInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := domains.NewApiKeyDomain(testCase.apiKey).Authorize()

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestApiKeyDomain_Revoke(t *testing.T) {
	testCases := []struct {
		name              string
		apiKey            entities.ApiKey
		expectedRevokedAt *time.Time
		expectedError     error
	}{
		{name: "revokes an active key", apiKey: entities.ApiKey{IsActive: true}, expectedRevokedAt: &revokedAt},
		{name: "revokes an inactive key", apiKey: entities.ApiKey{IsActive: false}, expectedRevokedAt: &revokedAt},
		{name: "rejects revoking an already revoked key", apiKey: entities.ApiKey{RevokedAt: &revokedAt}, expectedRevokedAt: &revokedAt, expectedError: domains.ErrApiKeyInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiKeyDomain := domains.NewApiKeyDomain(testCase.apiKey)

			err := apiKeyDomain.Revoke(revokedAt)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedRevokedAt, apiKeyDomain.ToEntity().RevokedAt)
		})
	}
}

func TestApiKeyDomain_DescribeStatus(t *testing.T) {
	testCases := []struct {
		name           string
		apiKey         entities.ApiKey
		expectedStatus dto.ApiKeyStatusDto
		expectedError  error
	}{
		{name: "inactive key reads as inactive", apiKey: entities.ApiKey{Name: "研究", IsActive: false}, expectedStatus: dto.ApiKeyStatusDto{Name: "研究", Status: "inactive"}},
		{name: "activated key reads as active", apiKey: entities.ApiKey{Name: "研究", IsActive: true}, expectedStatus: dto.ApiKeyStatusDto{Name: "研究", Status: "active"}},
		{name: "revoked key is invalid", apiKey: entities.ApiKey{Name: "研究", IsActive: true, RevokedAt: &revokedAt}, expectedError: domains.ErrApiKeyInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := domains.NewApiKeyDomain(testCase.apiKey).DescribeStatus()

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedStatus, status)
		})
	}
}

func TestNewIssuedApiKeyDomain_StartsInactiveAndExposesSecretOnce(t *testing.T) {
	name, _ := vo.NewApiKeyNameVo("研究")
	secret, _ := vo.NewApiKeySecretVo("snh_secret")

	issuedApiKeyDomain := domains.NewIssuedApiKeyDomain(name, secret)

	assert.Equal(t, entities.ApiKey{Name: "研究", SecretHash: secret.Hash, IsActive: false}, issuedApiKeyDomain.ToEntity())
	assert.Equal(t, dto.IssuedApiKeyDto{Name: "研究", ApiKey: "snh_secret", Status: "inactive"}, issuedApiKeyDomain.ToIssuedDto(secret))
}

func TestApiKeyDomain_ToAuthorizedDto(t *testing.T) {
	authorizedApiKey := domains.NewApiKeyDomain(entities.ApiKey{ID: 7, IsActive: true}).ToAuthorizedDto()

	assert.Equal(t, dto.AuthorizedApiKeyDto{ApiKeyID: 7}, authorizedApiKey)
}
