package controller_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var newsSearchedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type newsRouterFixture struct {
	router              *gin.Engine
	apiKeyRepository    *mocks.MockIApiKeyRepository
	listedCompanyProxy  *mocks.MockIListedCompanyProxy
	cryptocurrencyProxy *mocks.MockICryptocurrencyProxy
	cnyesNewsProxy      *mocks.MockINewsProxy
	googleNewsProxy     *mocks.MockINewsProxy
}

func createNewsRouter(t *testing.T) newsRouterFixture {
	gin.SetMode(gin.TestMode)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(newsSearchedAt).Maybe()
	fixture := newsRouterFixture{
		apiKeyRepository:    mocks.NewMockIApiKeyRepository(t),
		listedCompanyProxy:  mocks.NewMockIListedCompanyProxy(t),
		cryptocurrencyProxy: mocks.NewMockICryptocurrencyProxy(t),
		cnyesNewsProxy:      mocks.NewMockINewsProxy(t),
		googleNewsProxy:     mocks.NewMockINewsProxy(t),
	}
	fixture.cnyesNewsProxy.EXPECT().ProviderName().Return("鉅亨網").Maybe()
	fixture.googleNewsProxy.EXPECT().ProviderName().Return("Google 新聞").Maybe()
	apiKeyController := controller.NewApiKeyController(application.NewApiKeyApplication(service.NewApiKeyService(fixture.apiKeyRepository, clockProxy, mocks.NewMockIRandomProxy(t))))
	newsSearchService := service.NewNewsSearchService(
		service.NewSymbolResolutionService(fixture.listedCompanyProxy, fixture.cryptocurrencyProxy),
		clockProxy,
		map[string][]dto.NewsProviderDto{vo.MarketCategoryTwStock: {{NewsProxy: fixture.cnyesNewsProxy}, {NewsProxy: fixture.googleNewsProxy}}},
	)
	newsController := controller.NewNewsController(application.NewNewsSearchApplication(newsSearchService))
	fixture.router = gin.New()
	fixture.router.GET("/news", apiKeyController.RequireActiveApiKey(), newsController.SearchSymbolNews)
	return fixture
}

func (fixture newsRouterFixture) givenApiKey(isActive bool) {
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 1, IsActive: isActive}, nil)
}

func TestSearchSymbolNews_ReturnsCuratedNewsAndFailedProviders(t *testing.T) {
	fixture := createNewsRouter(t)
	fixture.givenApiKey(true)
	fixture.listedCompanyProxy.EXPECT().FindCompanyShortName("2330").Return("台積電", true, nil)
	fixture.cnyesNewsProxy.EXPECT().FetchNews("台積電").Return([]vo.NewsVo{{Title: "台積電法說會", Link: "https://news.cnyes.com/news/id/1", PublishedAt: newsSearchedAt, ProviderName: "鉅亨網", Summary: "摘要"}}, nil)
	fixture.googleNewsProxy.EXPECT().FetchNews("台積電").Return(nil, errors.New("timeout"))

	recorder := send(fixture.router, http.MethodGet, "/news?symbol=2330&category=twStock", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"symbol":"2330","category":"twStock","news":[{"title":"台積電法說會","link":"https://news.cnyes.com/news/id/1","publishedAt":"2026-10-06T12:00:00Z","providerName":"鉅亨網","summary":"摘要"}],"failedNewsProviders":["Google 新聞"]}`, recorder.Body.String())
}

func TestSearchSymbolNews_RejectsKeysThatAreNotActive(t *testing.T) {
	fixture := createNewsRouter(t)
	fixture.givenApiKey(false)

	inactiveRecorder := send(fixture.router, http.MethodGet, "/news?symbol=2330&category=twStock", presentedApiKey, "")
	missingRecorder := send(fixture.router, http.MethodGet, "/news?symbol=2330&category=twStock", "", "")

	assert.Equal(t, http.StatusForbidden, inactiveRecorder.Code)
	assert.Equal(t, controller.ErrorDetail{Code: "api_key_inactive", Message: "API key 尚未啟用"}, decodeError(t, inactiveRecorder))
	assert.Equal(t, http.StatusUnauthorized, missingRecorder.Code)
	assert.Equal(t, controller.ErrorDetail{Code: "api_key_missing", Message: "需要提供 API key"}, decodeError(t, missingRecorder))
}

func TestSearchSymbolNews_ErrorResponses(t *testing.T) {
	testCases := []struct {
		name            string
		query           string
		givenProxies    func(fixture newsRouterFixture)
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "unsupported market", query: "symbol=0700&category=hk", expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "missing market", query: "symbol=0700", expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "missing symbol", query: "category=twStock", expectedStatus: http.StatusBadRequest, expectedCode: "symbol_required", expectedMessage: "標的為必填"},
		{name: "unknown symbol", query: "symbol=9999&category=twStock", givenProxies: func(fixture newsRouterFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName("9999").Return("", false, nil)
		}, expectedStatus: http.StatusNotFound, expectedCode: "symbol_not_found", expectedMessage: "找不到此標的"},
		{name: "every provider fails", query: "symbol=2330&category=twStock", givenProxies: func(fixture newsRouterFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName("2330").Return("台積電", true, nil)
			fixture.cnyesNewsProxy.EXPECT().FetchNews("台積電").Return(nil, errors.New("down"))
			fixture.googleNewsProxy.EXPECT().FetchNews("台積電").Return(nil, errors.New("down"))
		}, expectedStatus: http.StatusBadGateway, expectedCode: "news_providers_unavailable", expectedMessage: "新聞來源暫時無法使用"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createNewsRouter(t)
			fixture.givenApiKey(true)
			if testCase.givenProxies != nil {
				testCase.givenProxies(fixture)
			}

			recorder := send(fixture.router, http.MethodGet, "/news?"+testCase.query, presentedApiKey, "")

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
		})
	}
}
