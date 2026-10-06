package service

import (
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type ApiKeyService struct {
	apiKeyRepository interfaces.IApiKeyRepository
	clockProxy       interfaces.IClockProxy
	randomProxy      interfaces.IRandomProxy
}

func NewApiKeyService(apiKeyRepository interfaces.IApiKeyRepository, clockProxy interfaces.IClockProxy, randomProxy interfaces.IRandomProxy) *ApiKeyService {
	return &ApiKeyService{apiKeyRepository: apiKeyRepository, clockProxy: clockProxy, randomProxy: randomProxy}
}

func (apiKeyService *ApiKeyService) IssueApiKey(issueApiKeyDto dto.IssueApiKeyDto) (dto.IssuedApiKeyDto, error) {
	name, err := vo.NewApiKeyNameVo(issueApiKeyDto.Name)
	if err != nil {
		return dto.IssuedApiKeyDto{}, err
	}
	secret := vo.NewIssuedApiKeySecretVo(apiKeyService.randomProxy.GenerateBytes(vo.ApiKeySecretRandomByteLength))
	apiKey := domains.NewIssuedApiKeyDomain(name, secret).ToEntity()
	if err := apiKeyService.apiKeyRepository.Create(&apiKey); err != nil {
		return dto.IssuedApiKeyDto{}, fmt.Errorf("%w: %v", ErrApiKeyStorageUnavailable, err)
	}
	return domains.NewApiKeyDomain(apiKey).ToIssuedDto(secret), nil
}

func (apiKeyService *ApiKeyService) GetApiKeyStatus(presentedApiKey string) (dto.ApiKeyStatusDto, error) {
	apiKeyDomain, err := apiKeyService.findApiKeyDomain(presentedApiKey)
	if err != nil {
		return dto.ApiKeyStatusDto{}, err
	}
	return apiKeyDomain.DescribeStatus()
}

func (apiKeyService *ApiKeyService) RevokeApiKey(presentedApiKey string) error {
	apiKeyDomain, err := apiKeyService.findApiKeyDomain(presentedApiKey)
	if err != nil {
		return err
	}
	if err := apiKeyDomain.Revoke(apiKeyService.clockProxy.Now()); err != nil {
		return err
	}
	revokedApiKey := apiKeyDomain.ToEntity()
	isMarkedRevoked, err := apiKeyService.apiKeyRepository.MarkRevoked(revokedApiKey.ID, *revokedApiKey.RevokedAt)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrApiKeyStorageUnavailable, err)
	}
	if !isMarkedRevoked {
		return ErrApiKeyInvalid
	}
	return nil
}

func (apiKeyService *ApiKeyService) AuthorizeApiKey(presentedApiKey string) (dto.AuthorizedApiKeyDto, error) {
	apiKeyDomain, err := apiKeyService.findApiKeyDomain(presentedApiKey)
	if err != nil {
		return dto.AuthorizedApiKeyDto{}, err
	}
	if err := apiKeyDomain.Authorize(); err != nil {
		return dto.AuthorizedApiKeyDto{}, err
	}
	return apiKeyDomain.ToAuthorizedDto(), nil
}

func (apiKeyService *ApiKeyService) findApiKeyDomain(presentedApiKey string) (domains.ApiKeyDomain, error) {
	secret, err := vo.NewApiKeySecretVo(presentedApiKey)
	if err != nil {
		return domains.ApiKeyDomain{}, err
	}
	apiKey, err := apiKeyService.apiKeyRepository.FindBySecretHash(secret.Hash)
	if err != nil {
		return domains.ApiKeyDomain{}, fmt.Errorf("%w: %v", ErrApiKeyStorageUnavailable, err)
	}
	if apiKey == nil {
		return domains.ApiKeyDomain{}, ErrApiKeyInvalid
	}
	return domains.NewApiKeyDomain(*apiKey), nil
}
