package application

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type ApiKeyApplication struct {
	apiKeyService *service.ApiKeyService
}

func NewApiKeyApplication(apiKeyService *service.ApiKeyService) *ApiKeyApplication {
	return &ApiKeyApplication{apiKeyService: apiKeyService}
}

func (apiKeyApplication *ApiKeyApplication) IssueApiKey(issueApiKeyDto dto.IssueApiKeyDto) (dto.IssuedApiKeyDto, error) {
	return apiKeyApplication.apiKeyService.IssueApiKey(issueApiKeyDto)
}

func (apiKeyApplication *ApiKeyApplication) GetApiKeyStatus(presentedApiKey string) (dto.ApiKeyStatusDto, error) {
	return apiKeyApplication.apiKeyService.GetApiKeyStatus(presentedApiKey)
}

func (apiKeyApplication *ApiKeyApplication) RevokeApiKey(presentedApiKey string) error {
	return apiKeyApplication.apiKeyService.RevokeApiKey(presentedApiKey)
}

func (apiKeyApplication *ApiKeyApplication) AuthorizeApiKey(presentedApiKey string) (dto.AuthorizedApiKeyDto, error) {
	return apiKeyApplication.apiKeyService.AuthorizeApiKey(presentedApiKey)
}
