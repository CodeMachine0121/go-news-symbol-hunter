package domains

import (
	"errors"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	ApiKeyStatusInactive = "inactive"
	ApiKeyStatusActive   = "active"
)

var (
	ErrApiKeyInvalid  = errors.New("API key 無效")
	ErrApiKeyInactive = errors.New("API key 尚未啟用")
)

type ApiKeyDomain struct {
	apiKey entities.ApiKey
}

func NewApiKeyDomain(apiKey entities.ApiKey) ApiKeyDomain {
	return ApiKeyDomain{apiKey: apiKey}
}

func NewIssuedApiKeyDomain(name vo.ApiKeyNameVo, secret vo.ApiKeySecretVo) ApiKeyDomain {
	return ApiKeyDomain{apiKey: entities.ApiKey{Name: name.Value, SecretHash: secret.Hash, IsActive: false}}
}

func (apiKeyDomain ApiKeyDomain) Authorize() error {
	if apiKeyDomain.isRevoked() {
		return ErrApiKeyInvalid
	}
	if !apiKeyDomain.apiKey.IsActive {
		return ErrApiKeyInactive
	}
	return nil
}

func (apiKeyDomain *ApiKeyDomain) Revoke(revokedAt time.Time) error {
	if apiKeyDomain.isRevoked() {
		return ErrApiKeyInvalid
	}
	apiKeyDomain.apiKey.RevokedAt = &revokedAt
	return nil
}

func (apiKeyDomain ApiKeyDomain) DescribeStatus() (dto.ApiKeyStatusDto, error) {
	if apiKeyDomain.isRevoked() {
		return dto.ApiKeyStatusDto{}, ErrApiKeyInvalid
	}
	return dto.ApiKeyStatusDto{Name: apiKeyDomain.apiKey.Name, Status: apiKeyDomain.status()}, nil
}

func (apiKeyDomain ApiKeyDomain) ToIssuedDto(secret vo.ApiKeySecretVo) dto.IssuedApiKeyDto {
	return dto.IssuedApiKeyDto{
		ID:     apiKeyDomain.apiKey.ID,
		Name:   apiKeyDomain.apiKey.Name,
		ApiKey: secret.Plaintext,
		Status: apiKeyDomain.status(),
	}
}

func (apiKeyDomain ApiKeyDomain) ToAuthorizedDto() dto.AuthorizedApiKeyDto {
	return dto.AuthorizedApiKeyDto{ApiKeyID: apiKeyDomain.apiKey.ID}
}

func (apiKeyDomain ApiKeyDomain) ToEntity() entities.ApiKey {
	return apiKeyDomain.apiKey
}

func (apiKeyDomain ApiKeyDomain) isRevoked() bool {
	return apiKeyDomain.apiKey.RevokedAt != nil
}

func (apiKeyDomain ApiKeyDomain) status() string {
	if apiKeyDomain.apiKey.IsActive {
		return ApiKeyStatusActive
	}
	return ApiKeyStatusInactive
}
