package controller_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type trackedSymbolGradeRouterFixture struct {
	router                  *gin.Engine
	apiKeyRepository        *mocks.MockIApiKeyRepository
	sessionGradeRepository  *mocks.MockISessionGradeRepository
	combinedGradeRepository *mocks.MockICombinedGradeRepository
}

func createTrackedSymbolGradeRouter(t *testing.T) trackedSymbolGradeRouterFixture {
	gin.SetMode(gin.TestMode)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(analysisStartedAt).Maybe()
	fixture := trackedSymbolGradeRouterFixture{
		apiKeyRepository:        mocks.NewMockIApiKeyRepository(t),
		sessionGradeRepository:  mocks.NewMockISessionGradeRepository(t),
		combinedGradeRepository: mocks.NewMockICombinedGradeRepository(t),
	}
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{}})
	sessionGradeService := service.NewSessionGradeService(
		symbolResolutionService,
		service.NewAnalystConsultationService(mocks.NewMockIAnalystProxy(t), service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{})),
		mocks.NewMockITrackedSymbolRepository(t),
		fixture.sessionGradeRepository,
		fixture.combinedGradeRepository,
		mocks.NewMockISessionRunRepository(t),
		clockProxy,
		vo.NewSessionWeightsVo(0, 0, 0),
	)
	apiKeyController := controller.NewApiKeyController(application.NewApiKeyApplication(service.NewApiKeyService(fixture.apiKeyRepository, clockProxy, mocks.NewMockIRandomProxy(t))))
	trackedSymbolGradeController := controller.NewTrackedSymbolGradeController(application.NewSessionGradeApplication(sessionGradeService))
	fixture.router = gin.New()
	fixture.router.Group("/", apiKeyController.RequireActiveApiKey()).GET("/tracked-symbol-grades", trackedSymbolGradeController.GetTrackedSymbolGrades)
	return fixture
}

func TestGetTrackedSymbolGrades_ReturnsTheGradesAsJson(t *testing.T) {
	fixture := createTrackedSymbolGradeRouter(t)
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: true}, nil)
	updatedAt := time.Date(2026, 10, 7, 4, 31, 0, 0, time.UTC)
	fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "2330", "twStock").Return([]entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "bullish", CombinedScore: 1, Confidence: 60, UpdatedAt: updatedAt}}, nil)
	fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, "2330", "twStock", "2026-10-07").Return([]entities.SessionGrade{{Session: "preMarket", Grade: "bullish", Confidence: 60, Reason: "法說會", CreatedAt: updatedAt}}, nil)

	recorder := send(fixture.router, http.MethodGet, "/tracked-symbol-grades?symbol=2330&category=twStock", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[{"symbol":"2330","category":"twStock","tradingDay":"2026-10-07","grade":"bullish","combinedScore":1,"confidence":60,"updatedAt":"2026-10-07T04:31:00Z",
		"sessionGrades":[{"session":"preMarket","grade":"bullish","confidence":60,"reason":"法說會","keyEvents":[],"riskFactors":[],"evidence":[],"createdAt":"2026-10-07T04:31:00Z"}]}]`, recorder.Body.String())
}

func TestGetTrackedSymbolGrades_ReturnsAnEmptyListWhenNothingIsGraded(t *testing.T) {
	fixture := createTrackedSymbolGradeRouter(t)
	fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: true}, nil)
	fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return([]entities.CombinedGrade{}, nil)

	recorder := send(fixture.router, http.MethodGet, "/tracked-symbol-grades", presentedApiKey, "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}

func TestGetTrackedSymbolGrades_ErrorResponses(t *testing.T) {
	testCases := []struct {
		name            string
		query           string
		isInactiveKey   bool
		given           func(fixture trackedSymbolGradeRouterFixture)
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{name: "symbol without category", query: "?symbol=2330", expectedStatus: http.StatusBadRequest, expectedCode: "symbol_and_category_required_together", expectedMessage: "標的與市場類別需一起提供"},
		{name: "unsupported category", query: "?symbol=0700&category=hk", expectedStatus: http.StatusBadRequest, expectedCode: "market_category_unsupported", expectedMessage: "市場類別只能是 crypto、twStock、usStock"},
		{name: "blank symbol", query: "?symbol=%20&category=twStock", expectedStatus: http.StatusBadRequest, expectedCode: "symbol_required", expectedMessage: "標的為必填"},
		{name: "inactive key", isInactiveKey: true, expectedStatus: http.StatusForbidden, expectedCode: "api_key_inactive", expectedMessage: "API key 尚未啟用"},
		{name: "storage unavailable", given: func(fixture trackedSymbolGradeRouterFixture) {
			fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return(nil, errDatabaseDown)
		}, expectedStatus: http.StatusServiceUnavailable, expectedCode: "service_unavailable", expectedMessage: "服務暫時無法使用"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createTrackedSymbolGradeRouter(t)
			fixture.apiKeyRepository.EXPECT().FindBySecretHash(hashOf(presentedApiKey)).Return(&entities.ApiKey{ID: 12, IsActive: !testCase.isInactiveKey}, nil)
			if testCase.given != nil {
				testCase.given(fixture)
			}

			recorder := send(fixture.router, http.MethodGet, "/tracked-symbol-grades"+testCase.query, presentedApiKey, "")

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Equal(t, controller.ErrorDetail{Code: testCase.expectedCode, Message: testCase.expectedMessage}, decodeError(t, recorder))
		})
	}
}
