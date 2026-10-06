package controller_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const presentedApiKey = "snh_presented"

var errDatabaseDown = errors.New("database down")

type protectedResult struct {
	ApiKeyID uint `json:"apiKeyId"`
}

func createRouter(t *testing.T) (*gin.Engine, *mocks.MockIApiKeyRepository) {
	gin.SetMode(gin.TestMode)
	apiKeyRepository := mocks.NewMockIApiKeyRepository(t)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)).Maybe()
	randomProxy := mocks.NewMockIRandomProxy(t)
	randomProxy.EXPECT().GenerateBytes(32).Return(make([]byte, 32)).Maybe()
	apiKeyController := controller.NewApiKeyController(application.NewApiKeyApplication(service.NewApiKeyService(apiKeyRepository, clockProxy, randomProxy)))
	router := gin.New()
	router.POST("/api-keys", apiKeyController.IssueApiKey)
	router.GET("/api-keys/me", apiKeyController.GetApiKeyStatus)
	router.DELETE("/api-keys/me", apiKeyController.RevokeApiKey)
	router.GET("/protected", apiKeyController.RequireActiveApiKey(), func(context *gin.Context) {
		context.JSON(http.StatusOK, protectedResult{ApiKeyID: context.GetUint(controller.AuthorizedApiKeyIDContextKey)})
	})
	return router, apiKeyRepository
}

func hashOf(secret string) string {
	digest := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(digest[:])
}

func send(router *gin.Engine, method string, path string, apiKey string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set(controller.ApiKeyHeader, apiKey)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) controller.ErrorDetail {
	var errorResponseBody controller.ErrorResponseBody
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &errorResponseBody))
	return errorResponseBody.Error
}

func TestIssueApiKey_ReturnsTheFullKeyOnceAsInactive(t *testing.T) {
	router, apiKeyRepository := createRouter(t)
	apiKeyRepository.EXPECT().Create(mock.Anything).Return(nil)

	recorder := send(router, http.MethodPost, "/api-keys", "", `{"name":"我的研究腳本"}`)

	require.Equal(t, http.StatusCreated, recorder.Code)
	var issuedApiKey map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &issuedApiKey))
	assert.JSONEq(t, `"我的研究腳本"`, string(issuedApiKey["name"]))
	assert.JSONEq(t, `"inactive"`, string(issuedApiKey["status"]))
	assert.JSONEq(t, `"snh_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`, string(issuedApiKey["apiKey"]))
	assert.NotContains(t, issuedApiKey, "id")
}

func TestIssueApiKey_Rejections(t *testing.T) {
	testCases := []struct {
		name            string
		body            string
		givenStorage    func(apiKeyRepository *mocks.MockIApiKeyRepository)
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "name too long", body: `{"name":"` + strings.Repeat("研", 101) + `"}`, expectedStatus: http.StatusBadRequest, expectedCode: "api_key_name_too_long", expectedMessage: "API key 名稱不可超過 100 個字"},
		{name: "empty body", body: ``, expectedStatus: http.StatusBadRequest, expectedCode: "api_key_name_required", expectedMessage: "API key 名稱為必填"},
		{name: "missing name", body: `{}`, expectedStatus: http.StatusBadRequest, expectedCode: "api_key_name_required", expectedMessage: "API key 名稱為必填"},
		{name: "control characters in name", body: `{"name":"研\u0000究"}`, expectedStatus: http.StatusBadRequest, expectedCode: "api_key_name_invalid_characters", expectedMessage: "API key 名稱不可包含控制字元"},
		{name: "blank name", body: `{"name":"   "}`, expectedStatus: http.StatusBadRequest, expectedCode: "api_key_name_required", expectedMessage: "API key 名稱為必填"},
		{name: "malformed body", body: `{"name":`, expectedStatus: http.StatusBadRequest, expectedCode: "invalid_request_body", expectedMessage: "請求內容格式錯誤"},
		{name: "storage unavailable", body: `{"name":"研究"}`, givenStorage: func(apiKeyRepository *mocks.MockIApiKeyRepository) {
			apiKeyRepository.EXPECT().Create(mock.Anything).Return(errDatabaseDown)
		}, expectedStatus: http.StatusServiceUnavailable, expectedCode: "service_unavailable", expectedMessage: "服務暫時無法使用"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router, apiKeyRepository := createRouter(t)
			if testCase.givenStorage != nil {
				testCase.givenStorage(apiKeyRepository)
			}

			recorder := send(router, http.MethodPost, "/api-keys", "", testCase.body)

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
		})
	}
}

func TestGetApiKeyStatus_ShowsNameAndStatusWithoutTheKey(t *testing.T) {
	testCases := []struct {
		name         string
		storedApiKey entities.ApiKey
		expectedBody string
	}{
		{name: "inactive", storedApiKey: entities.ApiKey{Name: "研究"}, expectedBody: `{"name":"研究","status":"inactive"}`},
		{name: "active", storedApiKey: entities.ApiKey{Name: "研究", IsActive: true}, expectedBody: `{"name":"研究","status":"active"}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router, apiKeyRepository := createRouter(t)
			apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&testCase.storedApiKey, nil)

			recorder := send(router, http.MethodGet, "/api-keys/me", presentedApiKey, "")

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.JSONEq(t, testCase.expectedBody, recorder.Body.String())
		})
	}
}

func TestRevokeApiKey_Succeeds(t *testing.T) {
	router, apiKeyRepository := createRouter(t)
	apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 3, IsActive: true}, nil)
	apiKeyRepository.EXPECT().MarkRevoked(uint(3), mock.AnythingOfType("time.Time")).Return(true, nil)

	recorder := send(router, http.MethodDelete, "/api-keys/me", presentedApiKey, "")

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestProtectedRoute_PassesAnActiveKeyThrough(t *testing.T) {
	router, apiKeyRepository := createRouter(t)
	apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 11, IsActive: true}, nil)

	recorder := send(router, http.MethodGet, "/protected", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"apiKeyId":11}`, recorder.Body.String())
}

func TestApiKeyRejections_AcrossStatusRevokeAndProtectedRoutes(t *testing.T) {
	revokedAt := time.Now()
	routes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api-keys/me"},
		{method: http.MethodDelete, path: "/api-keys/me"},
		{method: http.MethodGet, path: "/protected"},
	}
	testCases := []struct {
		name            string
		presentedKey    string
		storedApiKey    *entities.ApiKey
		storageError    error
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "missing key", presentedKey: "", expectedStatus: http.StatusUnauthorized, expectedCode: "api_key_missing", expectedMessage: "需要提供 API key"},
		{name: "unknown key", presentedKey: presentedApiKey, storedApiKey: nil, expectedStatus: http.StatusUnauthorized, expectedCode: "api_key_invalid", expectedMessage: "API key 無效"},
		{name: "revoked key reactivated by administrator", presentedKey: presentedApiKey, storedApiKey: &entities.ApiKey{IsActive: true, RevokedAt: &revokedAt}, expectedStatus: http.StatusUnauthorized, expectedCode: "api_key_invalid", expectedMessage: "API key 無效"},
		{name: "storage unavailable", presentedKey: presentedApiKey, storageError: errDatabaseDown, expectedStatus: http.StatusServiceUnavailable, expectedCode: "service_unavailable", expectedMessage: "服務暫時無法使用"},
	}
	for _, route := range routes {
		for _, testCase := range testCases {
			t.Run(route.method+" "+route.path+" "+testCase.name, func(t *testing.T) {
				router, apiKeyRepository := createRouter(t)
				if testCase.presentedKey != "" {
					apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(testCase.storedApiKey, testCase.storageError)
				}

				recorder := send(router, route.method, route.path, testCase.presentedKey, "")

				assert.Equal(t, testCase.expectedStatus, recorder.Code)
				assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
			})
		}
	}
}

func TestProtectedRoute_RejectsAnInactiveKey(t *testing.T) {
	router, apiKeyRepository := createRouter(t)
	apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 11, IsActive: false}, nil)

	recorder := send(router, http.MethodGet, "/protected", presentedApiKey, "")

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, controller.ErrorDetail{Code: "api_key_inactive", Message: "API key 尚未啟用"}, decodeError(t, recorder))
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", controller.NewHealthController().GetHealth)

	recorder := send(router, http.MethodGet, "/health", "", "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
}
