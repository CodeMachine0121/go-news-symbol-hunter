package controller_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var analysisStartedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type analysisRouterFixture struct {
	router                   *gin.Engine
	apiKeyRepository         *mocks.MockIApiKeyRepository
	cryptocurrencyProxy      *mocks.MockICryptocurrencyProxy
	analystProxy             *mocks.MockIAnalystProxy
	analysisEventRepository  *mocks.MockIAnalysisEventRepository
	analysisResultRepository *mocks.MockIAnalysisResultRepository
}

func createAnalysisRouter(t *testing.T) analysisRouterFixture {
	gin.SetMode(gin.TestMode)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(analysisStartedAt).Maybe()
	fixture := analysisRouterFixture{
		apiKeyRepository:         mocks.NewMockIApiKeyRepository(t),
		cryptocurrencyProxy:      mocks.NewMockICryptocurrencyProxy(t),
		analystProxy:             mocks.NewMockIAnalystProxy(t),
		analysisEventRepository:  mocks.NewMockIAnalysisEventRepository(t),
		analysisResultRepository: mocks.NewMockIAnalysisResultRepository(t),
	}
	fixture.analystProxy.EXPECT().ModelName().Return("claude-opus-5-5").Maybe()
	symbolResolutionService := service.NewSymbolResolutionService(mocks.NewMockIListedCompanyProxy(t), fixture.cryptocurrencyProxy)
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{})
	apiKeyController := controller.NewApiKeyController(application.NewApiKeyApplication(service.NewApiKeyService(fixture.apiKeyRepository, clockProxy, mocks.NewMockIRandomProxy(t))))
	analysisEventController := controller.NewAnalysisEventController(application.NewSymbolAnalysisApplication(service.NewSymbolAnalysisService(
		symbolResolutionService, newsSearchService, fixture.analystProxy, fixture.analysisEventRepository, fixture.analysisResultRepository, clockProxy,
	)))
	fixture.router = gin.New()
	protectedRoutes := fixture.router.Group("/", apiKeyController.RequireActiveApiKey())
	protectedRoutes.POST("/analysis-events", analysisEventController.StartSymbolAnalysis)
	protectedRoutes.GET("/analysis-events/:analysisEventId", analysisEventController.GetAnalysisEvent)
	return fixture
}

func (fixture analysisRouterFixture) givenActiveKey() {
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: true}, nil)
}

func TestStartSymbolAnalysis_AcceptsANewAnalysisForTheCallingKey(t *testing.T) {
	fixture := createAnalysisRouter(t)
	fixture.givenActiveKey()
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("Bitcoin", true, nil)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil)
	createdApiKeyID := make(chan uint, 1)
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 31
		createdApiKeyID <- analysisEvent.ApiKeyID
		return nil
	})
	analysisDone := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(31)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisDone)
		return nil, nil
	})

	recorder := send(fixture.router, http.MethodPost, "/analysis-events", presentedApiKey, `{"symbol":"btc","category":"crypto"}`)

	assert.Equal(t, http.StatusAccepted, recorder.Code)
	assert.JSONEq(t, `{"analysisEventId":31,"symbol":"BTC","category":"crypto","status":"running","startedAt":"2026-10-06T12:00:00Z"}`, recorder.Body.String())
	assert.Equal(t, uint(12), <-createdApiKeyID)
	<-analysisDone
}

func TestStartSymbolAnalysis_ReturnsAReusedAnalysisWithOk(t *testing.T) {
	fixture := createAnalysisRouter(t)
	fixture.givenActiveKey()
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("Bitcoin", true, nil)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 30, Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}, nil)

	recorder := send(fixture.router, http.MethodPost, "/analysis-events", presentedApiKey, `{"symbol":"BTC","category":"crypto"}`)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"analysisEventId":30,"symbol":"BTC","category":"crypto","status":"running","startedAt":"2026-10-06T12:00:00Z"}`, recorder.Body.String())
}

func TestStartSymbolAnalysis_ErrorResponses(t *testing.T) {
	testCases := []struct {
		name            string
		body            string
		given           func(fixture analysisRouterFixture)
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "unsupported market", body: `{"symbol":"0700","category":"hk"}`, expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "missing symbol", body: `{"category":"crypto"}`, expectedStatus: http.StatusBadRequest, expectedCode: "symbol_required", expectedMessage: "標的為必填"},
		{name: "empty body", body: ``, expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "malformed body", body: `{"symbol":`, expectedStatus: http.StatusBadRequest, expectedCode: "invalid_request_body", expectedMessage: "請求內容格式錯誤"},
		{name: "unknown coin", body: `{"symbol":"NOTACOIN","category":"crypto"}`, given: func(fixture analysisRouterFixture) {
			fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "NOTACOIN").Return("", false, nil)
		}, expectedStatus: http.StatusNotFound, expectedCode: "symbol_not_found", expectedMessage: "找不到此標的"},
		{name: "coin directory unavailable", body: `{"symbol":"BTC","category":"crypto"}`, given: func(fixture analysisRouterFixture) {
			fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("", false, errDatabaseDown)
		}, expectedStatus: http.StatusBadGateway, expectedCode: "news_providers_unavailable", expectedMessage: "新聞來源暫時無法使用"},
		{name: "storage unavailable", body: `{"symbol":"BTC","category":"crypto"}`, given: func(fixture analysisRouterFixture) {
			fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("Bitcoin", true, nil)
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, errDatabaseDown)
		}, expectedStatus: http.StatusServiceUnavailable, expectedCode: "service_unavailable", expectedMessage: "服務暫時無法使用"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createAnalysisRouter(t)
			fixture.givenActiveKey()
			if testCase.given != nil {
				testCase.given(fixture)
			}

			recorder := send(fixture.router, http.MethodPost, "/analysis-events", presentedApiKey, testCase.body)

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
		})
	}
}

func TestAnalysisEventRoutes_RequireAnActiveKey(t *testing.T) {
	fixture := createAnalysisRouter(t)

	startRecorder := send(fixture.router, http.MethodPost, "/analysis-events", "", `{"symbol":"BTC","category":"crypto"}`)
	getRecorder := send(fixture.router, http.MethodGet, "/analysis-events/1", "", "")

	assert.Equal(t, http.StatusUnauthorized, startRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, getRecorder.Code)
}

func TestGetAnalysisEvent_ReturnsTheResultOfASucceededAnalysis(t *testing.T) {
	fixture := createAnalysisRouter(t)
	fixture.givenActiveKey()
	finishedAt := analysisStartedAt.Add(time.Minute)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(30)).Return(&entities.AnalysisEvent{ID: 30, ApiKeyID: 99, Symbol: "BTC", Category: "crypto", Status: "succeeded", StartedAt: analysisStartedAt, FinishedAt: &finishedAt}, nil)
	fixture.analysisResultRepository.EXPECT().FindByAnalysisEventID(mock.Anything, uint(30)).Return(&entities.AnalysisResult{
		AnalysisEventID: 30, Symbol: "BTC", Category: "crypto", Grade: "bullish", Confidence: 70, TimeHorizon: "short", Reason: "理由",
		KeyEvents:   []entities.AnalysisKeyEvent{{Title: "t", Link: "https://news/1", PublishedAt: analysisStartedAt}},
		RiskFactors: []string{"風險"},
		Evidence:    []entities.AnalysisEvidence{{Title: "t", Link: "https://news/1", PublishedAt: analysisStartedAt, ProviderName: "CoinDesk"}},
		CreatedAt:   finishedAt,
	}, nil)

	recorder := send(fixture.router, http.MethodGet, "/analysis-events/30", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"analysisEventId":30,"symbol":"BTC","category":"crypto","status":"succeeded","startedAt":"2026-10-06T12:00:00Z","finishedAt":"2026-10-06T12:01:00Z",
		"result":{"analysisEventId":30,"symbol":"BTC","category":"crypto","grade":"bullish","confidence":70,"timeHorizon":"short","reason":"理由",
		"keyEvents":[{"title":"t","link":"https://news/1","publishedAt":"2026-10-06T12:00:00Z"}],"riskFactors":["風險"],
		"evidence":[{"title":"t","link":"https://news/1","publishedAt":"2026-10-06T12:00:00Z","providerName":"CoinDesk"}],"createdAt":"2026-10-06T12:01:00Z"}}`, recorder.Body.String())
}

func TestGetAnalysisEvent_ShowsAFailureReason(t *testing.T) {
	fixture := createAnalysisRouter(t)
	fixture.givenActiveKey()
	finishedAt := analysisStartedAt.Add(time.Minute)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(32)).Return(&entities.AnalysisEvent{ID: 32, Symbol: "BTC", Category: "crypto", Status: "failed", FailureReason: "AI 服務暫時無法使用", StartedAt: analysisStartedAt, FinishedAt: &finishedAt}, nil)

	recorder := send(fixture.router, http.MethodGet, "/analysis-events/32", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"analysisEventId":32,"symbol":"BTC","category":"crypto","status":"failed","failureReason":"AI 服務暫時無法使用","startedAt":"2026-10-06T12:00:00Z","finishedAt":"2026-10-06T12:01:00Z"}`, recorder.Body.String())
}

func TestGetAnalysisEvent_UnknownIdsAreNotFound(t *testing.T) {
	for _, path := range []string{"/analysis-events/999", "/analysis-events/abc", "/analysis-events/-1"} {
		t.Run(path, func(t *testing.T) {
			fixture := createAnalysisRouter(t)
			fixture.givenActiveKey()
			fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(999)).Return(nil, nil).Maybe()

			recorder := send(fixture.router, http.MethodGet, path, presentedApiKey, "")

			assert.Equal(t, http.StatusNotFound, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: "analysis_event_not_found", Message: "找不到此分析事件"}, decodeError(t, recorder))
		})
	}
}

func TestAnalysisEventRoutes_RejectAnInactiveKey(t *testing.T) {
	fixture := createAnalysisRouter(t)
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: false}, nil)

	recorder := send(fixture.router, http.MethodPost, "/analysis-events", presentedApiKey, `{"symbol":"BTC","category":"crypto"}`)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, controller.ErrorDetail{Code: "api_key_inactive", Message: "API key 尚未啟用"}, decodeError(t, recorder))
}
