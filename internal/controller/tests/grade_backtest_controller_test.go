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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type gradeBacktestRouterFixture struct {
	router                   *gin.Engine
	apiKeyRepository         *mocks.MockIApiKeyRepository
	analysisResultRepository *mocks.MockIAnalysisResultRepository
	priceHistoryProxy        *mocks.MockIPriceHistoryProxy
}

func createGradeBacktestRouter(t *testing.T) gradeBacktestRouterFixture {
	gin.SetMode(gin.TestMode)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(analysisStartedAt).Maybe()
	fixture := gradeBacktestRouterFixture{
		apiKeyRepository:         mocks.NewMockIApiKeyRepository(t),
		analysisResultRepository: mocks.NewMockIAnalysisResultRepository(t),
		priceHistoryProxy:        mocks.NewMockIPriceHistoryProxy(t),
	}
	apiKeyController := controller.NewApiKeyController(application.NewApiKeyApplication(service.NewApiKeyService(fixture.apiKeyRepository, clockProxy, mocks.NewMockIRandomProxy(t))))
	gradeBacktestController := controller.NewGradeBacktestController(application.NewGradeBacktestApplication(service.NewGradeBacktestService(
		fixture.analysisResultRepository,
		dto.PriceHistoryCatalogDto{TwStock: fixture.priceHistoryProxy, Crypto: fixture.priceHistoryProxy},
	)))
	fixture.router = gin.New()
	fixture.router.Group("/", apiKeyController.RequireActiveApiKey()).GET("/grade-backtest", gradeBacktestController.GetGradeBacktest)
	return fixture
}

func TestGetGradeBacktest_ReturnsTheSummaryAndEveryAnalysisAsJson(t *testing.T) {
	fixture := createGradeBacktestRouter(t)
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: true}, nil)
	// Wednesday 2026-10-07 10:15 in Taipei
	completedAt := time.Date(2026, 10, 7, 2, 15, 0, 0, time.UTC)
	fixture.analysisResultRepository.EXPECT().FindForBacktest(mock.Anything, mock.MatchedBy(func(gradeBacktestFilter vo.GradeBacktestFilterVo) bool {
		return gradeBacktestFilter.Symbol == "2330" && gradeBacktestFilter.Category == "twStock" && !gradeBacktestFilter.CompletedPeriod.From.IsZero() && !gradeBacktestFilter.CompletedPeriod.Before.IsZero()
	}), 501).Return([]entities.AnalysisResult{{AnalysisEventID: 9, Symbol: "2330", Category: "twStock", Grade: "bullish", Confidence: 75, TimeHorizon: "short", CreatedAt: completedAt}}, nil)
	fixture.priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool { return priceHistoryQuery.Interval == "1m" })).
		Return([]vo.PriceCandleVo{{OpenTime: completedAt, Close: decimal.RequireFromString("100")}}, nil)
	dailyCandles := []vo.PriceCandleVo{}
	for day := range 10 {
		dailyCandles = append(dailyCandles, vo.PriceCandleVo{OpenTime: time.Date(2026, 10, 8+day, 0, 0, 0, 0, time.UTC), Close: decimal.RequireFromString("105")})
	}
	fixture.priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool { return priceHistoryQuery.Interval == "1d" })).Return(dailyCandles, nil)

	recorder := send(fixture.router, http.MethodGet, "/grade-backtest?symbol=2330&category=twStock&completedFrom=2026-10-01&completedTo=2026-10-31", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"randomBaseline":0.5,
		"overall":{"hitCount":1,"missCount":0,"hitRate":1},
		"byCategory":{"twStock":{"hitCount":1,"missCount":0,"hitRate":1},"usStock":{"hitCount":0,"missCount":0,"hitRate":null},"crypto":{"hitCount":0,"missCount":0,"hitRate":null}},
		"byGrade":{"strongBullish":{"hitCount":0,"missCount":0,"hitRate":null},"bullish":{"hitCount":1,"missCount":0,"hitRate":1},"bearish":{"hitCount":0,"missCount":0,"hitRate":null},"strongBearish":{"hitCount":0,"missCount":0,"hitRate":null}},
		"byConfidenceBand":{"low":{"hitCount":0,"missCount":0,"hitRate":null},"medium":{"hitCount":0,"missCount":0,"hitRate":null},"high":{"hitCount":1,"missCount":0,"hitRate":1}},
		"neutralAverageChangeRate":null,
		"outcomeCounts":{"hit":1,"miss":0,"neutral":0,"pending":0,"unevaluable":0},
		"analyses":[{"analysisEventId":9,"symbol":"2330","category":"twStock","grade":"bullish","confidence":75,"confidenceBand":"high","timeHorizon":"short",
			"completedAt":"2026-10-07T02:15:00Z","entryPrice":"100","checkpointDate":"2026-10-17","checkpointPrice":"105","priceChangeRate":"0.05","outcome":"hit"}]
	}`, recorder.Body.String())
}

func TestGetGradeBacktest_ErrorResponses(t *testing.T) {
	testCases := []struct {
		name            string
		query           string
		isInactiveKey   bool
		given           func(fixture gradeBacktestRouterFixture)
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "period backwards", query: "?completedFrom=2026-10-31&completedTo=2026-10-01", expectedStatus: http.StatusBadRequest, expectedCode: "backtest_period_invalid", expectedMessage: "期間不正確"},
		{name: "symbol without category", query: "?symbol=2330", expectedStatus: http.StatusBadRequest, expectedCode: "symbol_and_category_required_together", expectedMessage: "標的與市場類別需一起提供"},
		{name: "unsupported category", query: "?symbol=2330&category=futures", expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "blank symbol", query: "?symbol=%20&category=twStock", expectedStatus: http.StatusBadRequest, expectedCode: "symbol_required", expectedMessage: "標的為必填"},
		{name: "more than five hundred analyses", given: func(fixture gradeBacktestRouterFixture) {
			fixture.analysisResultRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return(make([]entities.AnalysisResult, 501), nil)
		}, expectedStatus: http.StatusBadRequest, expectedCode: "backtest_analysis_limit_exceeded", expectedMessage: "涵蓋分析超過 500 筆，請縮小期間"},
		{name: "price service unreachable", given: func(fixture gradeBacktestRouterFixture) {
			fixture.analysisResultRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return([]entities.AnalysisResult{{Symbol: "2330", Category: "twStock", Grade: "bullish", TimeHorizon: "short"}}, nil)
			fixture.priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.Anything).Return(nil, errors.New("connection refused")).Maybe()
		}, expectedStatus: http.StatusServiceUnavailable, expectedCode: "price_service_unavailable", expectedMessage: "價格服務暫時無法使用"},
		{name: "storage unavailable", given: func(fixture gradeBacktestRouterFixture) {
			fixture.analysisResultRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return(nil, errDatabaseDown)
		}, expectedStatus: http.StatusServiceUnavailable, expectedCode: "service_unavailable", expectedMessage: "服務暫時無法使用"},
		{name: "inactive key", isInactiveKey: true, expectedStatus: http.StatusForbidden, expectedCode: "api_key_inactive", expectedMessage: "API key 尚未啟用"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createGradeBacktestRouter(t)
			fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: !testCase.isInactiveKey}, nil)
			if testCase.given != nil {
				testCase.given(fixture)
			}

			recorder := send(fixture.router, http.MethodGet, "/grade-backtest"+testCase.query, presentedApiKey, "")

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
		})
	}
}
