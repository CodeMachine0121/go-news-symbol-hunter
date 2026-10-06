package controller

import (
	"errors"
	"net/http"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
)

const (
	ApiKeyHeader                 = "X-API-Key"
	AuthorizedApiKeyIDContextKey = "authorizedApiKeyId"
)

var apiKeyErrorResponses = []struct {
	err    error
	status int
	code   string
}{
	{err: service.ErrApiKeyNameRequired, status: http.StatusBadRequest, code: "api_key_name_required"},
	{err: service.ErrApiKeyNameTooLong, status: http.StatusBadRequest, code: "api_key_name_too_long"},
	{err: service.ErrApiKeyMissing, status: http.StatusUnauthorized, code: "api_key_missing"},
	{err: service.ErrApiKeyInvalid, status: http.StatusUnauthorized, code: "api_key_invalid"},
	{err: service.ErrApiKeyInactive, status: http.StatusForbidden, code: "api_key_inactive"},
}

type IssueApiKeyRequest struct {
	Name string `json:"name"`
}

type ApiKeyController struct {
	apiKeyApplication *application.ApiKeyApplication
}

func NewApiKeyController(apiKeyApplication *application.ApiKeyApplication) *ApiKeyController {
	return &ApiKeyController{apiKeyApplication: apiKeyApplication}
}

func (apiKeyController *ApiKeyController) IssueApiKey(context *gin.Context) {
	var issueApiKeyRequest IssueApiKeyRequest
	if err := context.ShouldBindJSON(&issueApiKeyRequest); err != nil {
		context.JSON(http.StatusBadRequest, ErrorResponseBody{Error: ErrorDetail{Code: "invalid_request_body", Message: "請求內容格式錯誤"}})
		return
	}
	issuedApiKey, err := apiKeyController.apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: issueApiKeyRequest.Name})
	if err != nil {
		apiKeyController.respondError(context, err)
		return
	}
	context.JSON(http.StatusCreated, issuedApiKey)
}

func (apiKeyController *ApiKeyController) GetApiKeyStatus(context *gin.Context) {
	apiKeyStatus, err := apiKeyController.apiKeyApplication.GetApiKeyStatus(context.GetHeader(ApiKeyHeader))
	if err != nil {
		apiKeyController.respondError(context, err)
		return
	}
	context.JSON(http.StatusOK, apiKeyStatus)
}

func (apiKeyController *ApiKeyController) RevokeApiKey(context *gin.Context) {
	if err := apiKeyController.apiKeyApplication.RevokeApiKey(context.GetHeader(ApiKeyHeader)); err != nil {
		apiKeyController.respondError(context, err)
		return
	}
	context.Status(http.StatusNoContent)
}

func (apiKeyController *ApiKeyController) RequireActiveApiKey() gin.HandlerFunc {
	return func(context *gin.Context) {
		authorizedApiKey, err := apiKeyController.apiKeyApplication.AuthorizeApiKey(context.GetHeader(ApiKeyHeader))
		if err != nil {
			apiKeyController.respondError(context, err)
			context.Abort()
			return
		}
		context.Set(AuthorizedApiKeyIDContextKey, authorizedApiKey.ApiKeyID)
		context.Next()
	}
}

func (apiKeyController *ApiKeyController) respondError(context *gin.Context, err error) {
	for _, apiKeyErrorResponse := range apiKeyErrorResponses {
		if errors.Is(err, apiKeyErrorResponse.err) {
			context.JSON(apiKeyErrorResponse.status, ErrorResponseBody{Error: ErrorDetail{Code: apiKeyErrorResponse.code, Message: apiKeyErrorResponse.err.Error()}})
			return
		}
	}
	context.JSON(http.StatusServiceUnavailable, ErrorResponseBody{Error: ErrorDetail{Code: "service_unavailable", Message: "服務暫時無法使用"}})
}
