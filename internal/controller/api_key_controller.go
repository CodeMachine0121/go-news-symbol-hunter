package controller

import (
	"errors"
	"io"
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

var apiKeyErrorResponseTable = ErrorResponseTable{
	{Err: service.ErrApiKeyNameRequired, Status: http.StatusBadRequest, Code: "api_key_name_required"},
	{Err: service.ErrApiKeyNameTooLong, Status: http.StatusBadRequest, Code: "api_key_name_too_long"},
	{Err: service.ErrApiKeyNameInvalidCharacters, Status: http.StatusBadRequest, Code: "api_key_name_invalid_characters"},
	{Err: service.ErrApiKeyMissing, Status: http.StatusUnauthorized, Code: "api_key_missing"},
	{Err: service.ErrApiKeyInvalid, Status: http.StatusUnauthorized, Code: "api_key_invalid"},
	{Err: service.ErrApiKeyInactive, Status: http.StatusForbidden, Code: "api_key_inactive"},
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
	if err := context.ShouldBindJSON(&issueApiKeyRequest); err != nil && !errors.Is(err, io.EOF) {
		context.JSON(http.StatusBadRequest, ErrorResponseBody{Error: ErrorDetail{Code: "invalid_request_body", Message: "請求內容格式錯誤"}})
		return
	}
	issuedApiKey, err := apiKeyController.apiKeyApplication.IssueApiKey(dto.IssueApiKeyDto{Name: issueApiKeyRequest.Name})
	if err != nil {
		apiKeyErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusCreated, issuedApiKey)
}

func (apiKeyController *ApiKeyController) GetApiKeyStatus(context *gin.Context) {
	apiKeyStatus, err := apiKeyController.apiKeyApplication.GetApiKeyStatus(context.GetHeader(ApiKeyHeader))
	if err != nil {
		apiKeyErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusOK, apiKeyStatus)
}

func (apiKeyController *ApiKeyController) RevokeApiKey(context *gin.Context) {
	if err := apiKeyController.apiKeyApplication.RevokeApiKey(context.GetHeader(ApiKeyHeader)); err != nil {
		apiKeyErrorResponseTable.Respond(context, err)
		return
	}
	context.Status(http.StatusNoContent)
}

func (apiKeyController *ApiKeyController) RequireActiveApiKey() gin.HandlerFunc {
	return func(context *gin.Context) {
		authorizedApiKey, err := apiKeyController.apiKeyApplication.AuthorizeApiKey(context.GetHeader(ApiKeyHeader))
		if err != nil {
			apiKeyErrorResponseTable.Respond(context, err)
			context.Abort()
			return
		}
		context.Set(AuthorizedApiKeyIDContextKey, authorizedApiKey.ApiKeyID)
		context.Next()
	}
}
