package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var analysisStartedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type symbolAnalysisFixture struct {
	symbolAnalysisApplication *application.SymbolAnalysisApplication
	listedCompanyProxy        *mocks.MockIListedCompanyProxy
	cryptocurrencyProxy       *mocks.MockICryptocurrencyProxy
	cryptoNewsProxy           *mocks.MockINewsProxy
	usStockNewsProxy          *mocks.MockINewsProxy
	analystProxy              *mocks.MockIAnalystProxy
	analysisEventRepository   *mocks.MockIAnalysisEventRepository
	analysisResultRepository  *mocks.MockIAnalysisResultRepository
	priceQuotesBySymbol       map[string]vo.PriceQuoteVo
}

func createSymbolAnalysisFixture(t *testing.T) symbolAnalysisFixture {
	return createSymbolAnalysisFixtureWithCapacity(t, 4)
}

func createSymbolAnalysisFixtureWithCapacity(t *testing.T, maximumConcurrentAnalyses int) symbolAnalysisFixture {
	fixture := symbolAnalysisFixture{
		listedCompanyProxy:       mocks.NewMockIListedCompanyProxy(t),
		cryptocurrencyProxy:      mocks.NewMockICryptocurrencyProxy(t),
		cryptoNewsProxy:          createNewsProxy(t, "CoinDesk"),
		usStockNewsProxy:         createNewsProxy(t, "Yahoo 財經"),
		analystProxy:             mocks.NewMockIAnalystProxy(t),
		analysisEventRepository:  mocks.NewMockIAnalysisEventRepository(t),
		analysisResultRepository: mocks.NewMockIAnalysisResultRepository(t),
		priceQuotesBySymbol:      map[string]vo.PriceQuoteVo{},
	}
	fixture.analystProxy.EXPECT().ModelName().Return("claude-opus-5-5").Maybe()
	newPriceProxy := func() *mocks.MockIPriceProxy {
		priceProxy := mocks.NewMockIPriceProxy(t)
		priceProxy.EXPECT().FetchPrice(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, symbol string) (vo.PriceQuoteVo, error) {
			priceQuote, priced := fixture.priceQuotesBySymbol[symbol]
			if !priced {
				return vo.PriceQuoteVo{}, errDatabaseDown
			}
			return priceQuote, nil
		}).Maybe()
		return priceProxy
	}
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{TwStock: []interfaces.IPriceProxy{newPriceProxy()}, UsStock: []interfaces.IPriceProxy{newPriceProxy()}, Crypto: []interfaces.IPriceProxy{newPriceProxy()}})
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(analysisStartedAt).Maybe()
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{fixture.listedCompanyProxy}, Crypto: fixture.cryptocurrencyProxy})
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{
		Crypto:  []dto.NewsProviderDto{{NewsProxy: fixture.cryptoNewsProxy}},
		UsStock: []dto.NewsProviderDto{{NewsProxy: fixture.usStockNewsProxy}},
	})
	fixture.symbolAnalysisApplication = application.NewSymbolAnalysisApplication(service.NewSymbolAnalysisService(
		symbolResolutionService, newsSearchService, priceSnapshotService, fixture.analystProxy, fixture.analysisEventRepository, fixture.analysisResultRepository, clockProxy,
	), maximumConcurrentAnalyses)
	return fixture
}

func finishedAgo(duration time.Duration) *time.Time {
	finishedAt := analysisStartedAt.Add(-duration)
	return &finishedAt
}

func (fixture symbolAnalysisFixture) givenBitcoin() {
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("Bitcoin", true, nil)
}

func TestStartSymbolAnalysis_CreatesARunningAnalysisAndAnalyzesInTheBackground(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil).Once()
	createdAnalysisEvent := entities.AnalysisEvent{}
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 11
		createdAnalysisEvent = *analysisEvent
		return nil
	})
	analyzed := make(chan vo.AnalystRequestVo, 1)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(11)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		return &createdAnalysisEvent, nil
	})
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, request vo.AnalystRequestVo, _ []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		analyzed <- request
		return vo.AnalystTurnVo{IsRefused: true}, nil
	})
	finished := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, *entities.AnalysisEvent) error {
		close(finished)
		return nil
	})

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{ApiKeyID: 5, Symbol: "btc", Category: "crypto"})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, dto.AnalysisEventDto{AnalysisEventID: 11, Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}, startedSymbolAnalysis.AnalysisEvent)
	assert.Equal(t, entities.AnalysisEvent{ID: 11, ApiKeyID: 5, Symbol: "BTC", Category: "crypto", Status: "running", Model: "claude-opus-5-5", StartedAt: analysisStartedAt}, createdAnalysisEvent)
	assert.Equal(t, vo.AnalystRequestVo{Symbol: "BTC", Category: "crypto", SearchKeyword: "Bitcoin"}, <-analyzed)
	<-finished
}

func TestStartSymbolAnalysis_ReusesRecentOrRunningAnalyses(t *testing.T) {
	testCases := []struct {
		name          string
		latestEvent   entities.AnalysisEvent
		expectsResult bool
	}{
		{name: "succeeded 5h59m ago", latestEvent: entities.AnalysisEvent{ID: 3, Symbol: "BTC", Category: "crypto", Status: "succeeded", FinishedAt: finishedAgo(5*time.Hour + 59*time.Minute)}, expectsResult: true},
		{name: "still running", latestEvent: entities.AnalysisEvent{ID: 3, Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&testCase.latestEvent, nil)
			if testCase.expectsResult {
				fixture.analysisResultRepository.EXPECT().FindByAnalysisEventID(mock.Anything, uint(3)).Return(&entities.AnalysisResult{AnalysisEventID: 3, Grade: "bullish"}, nil)
			}

			startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

			require.NoError(t, err)
			assert.False(t, startedSymbolAnalysis.IsNew)
			assert.Equal(t, uint(3), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
			assert.Equal(t, testCase.expectsResult, startedSymbolAnalysis.AnalysisEvent.Result != nil)
		})
	}
}

func TestStartSymbolAnalysis_StartsANewAnalysisWhenNothingIsReusable(t *testing.T) {
	testCases := []struct {
		name        string
		latestEvent *entities.AnalysisEvent
	}{
		{name: "succeeded 6h1m ago", latestEvent: &entities.AnalysisEvent{ID: 3, Status: "succeeded", FinishedAt: finishedAgo(6*time.Hour + time.Minute)}},
		{name: "failed 1h ago", latestEvent: &entities.AnalysisEvent{ID: 3, Status: "failed", FinishedAt: finishedAgo(time.Hour)}},
		{name: "never analyzed", latestEvent: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(testCase.latestEvent, nil)
			fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			analysisDone := make(chan struct{})
			fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(0)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
				close(analysisDone)
				return nil, nil
			})

			startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

			require.NoError(t, err)
			assert.True(t, startedSymbolAnalysis.IsNew)
			<-analysisDone
		})
	}
}

func TestStartSymbolAnalysis_LooksUpReuseByMarketCategory(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "usStock").Return(nil, nil)
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	analysisDone := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(0)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisDone)
		return nil, nil
	})

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "usStock"})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	<-analysisDone
}

func TestStartSymbolAnalysis_ReturnsTheConcurrentlyStartedAnalysis(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil).Once()
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(service.ErrAnalysisAlreadyRunning)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 8, Status: "running", StartedAt: analysisStartedAt}, nil).Once()

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	require.NoError(t, err)
	assert.False(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(8), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
}

func TestStartSymbolAnalysis_Rejections(t *testing.T) {
	testCases := []struct {
		name          string
		symbol        string
		category      string
		given         func(fixture symbolAnalysisFixture)
		expectedError error
	}{
		{name: "unsupported market", symbol: "0700", category: "hk", expectedError: service.ErrMarketCategoryUnsupported},
		{name: "missing symbol", symbol: "", category: "crypto", expectedError: service.ErrSymbolRequired},
		{name: "unlisted taiwan stock", symbol: "9999", category: "twStock", given: func(fixture symbolAnalysisFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "9999").Return("", false, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "storage unavailable on lookup", symbol: "BTC", category: "crypto", given: func(fixture symbolAnalysisFixture) {
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, errDatabaseDown)
		}, expectedError: service.ErrAnalysisStorageUnavailable},
		{name: "storage unavailable on create", symbol: "BTC", category: "crypto", given: func(fixture symbolAnalysisFixture) {
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil)
			fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}, expectedError: service.ErrAnalysisStorageUnavailable},
		{name: "concurrent analysis vanished", symbol: "BTC", category: "crypto", given: func(fixture symbolAnalysisFixture) {
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil)
			fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(service.ErrAnalysisAlreadyRunning).Times(3)
		}, expectedError: service.ErrAnalysisStorageUnavailable},
		{name: "reused result unavailable", symbol: "BTC", category: "crypto", given: func(fixture symbolAnalysisFixture) {
			fixture.givenBitcoin()
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 3, Status: "succeeded", FinishedAt: finishedAgo(time.Hour)}, nil)
			fixture.analysisResultRepository.EXPECT().FindByAnalysisEventID(mock.Anything, uint(3)).Return(nil, errDatabaseDown)
		}, expectedError: service.ErrAnalysisStorageUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			if testCase.given != nil {
				testCase.given(fixture)
			}

			_, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: testCase.symbol, Category: testCase.category})

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func runningBitcoinAnalysis() *entities.AnalysisEvent {
	return &entities.AnalysisEvent{ID: 21, ApiKeyID: 5, Symbol: "BTC", Category: "crypto", Status: "running", Model: "claude-opus-5-5", StartedAt: analysisStartedAt}
}

func (fixture symbolAnalysisFixture) captureFinishedEvent() *entities.AnalysisEvent {
	finishedAnalysisEvent := &entities.AnalysisEvent{}
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		*finishedAnalysisEvent = *analysisEvent
		return nil
	})
	return finishedAnalysisEvent
}

func TestAnalyzeSymbol_SearchesThenSavesTheNormalizedConclusion(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.givenBitcoin()
	fixture.cryptoNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").Return([]vo.NewsVo{
		{Title: "Bitcoin ETF inflows", Link: "https://news/1", PublishedAt: analysisStartedAt.Add(-time.Hour), ProviderName: "CoinDesk"},
		{Title: "Miners sell", Link: "https://news/2", PublishedAt: analysisStartedAt.Add(-2 * time.Hour), ProviderName: "CoinDesk"},
	}, nil)
	fixture.usStockNewsProxy.EXPECT().FetchNews(mock.Anything, "MSTR").Return([]vo.NewsVo{
		{Title: "MicroStrategy buys", Link: "https://news/3", PublishedAt: analysisStartedAt.Add(-3 * time.Hour), ProviderName: "Yahoo 財經"},
		{Title: "Bitcoin ETF inflows again", Link: "https://news/1", PublishedAt: analysisStartedAt.Add(-time.Hour), ProviderName: "Yahoo 財經"},
	}, nil)
	receivedExchanges := [][]vo.AnalystExchangeVo{}
	fixture.analystProxy.EXPECT().Respond(mock.Anything, vo.AnalystRequestVo{Symbol: "BTC", Category: "crypto", SearchKeyword: "Bitcoin"}, mock.Anything).RunAndReturn(func(_ context.Context, _ vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		receivedExchanges = append(receivedExchanges, exchanges)
		if len(exchanges) == 0 {
			return vo.AnalystTurnVo{Reply: "reply-1", Usage: vo.AnalystUsageVo{InputTokens: 100, OutputTokens: 10}, NewsSearches: []vo.AnalystNewsSearchVo{
				{ToolCallID: "search-btc", Symbol: "BTC", Category: "crypto"},
				{ToolCallID: "search-mstr", Symbol: "MSTR", Category: "usStock"},
				{ToolCallID: "search-bad", Symbol: "XYZ", Category: "hk"},
			}}, nil
		}
		return vo.AnalystTurnVo{Reply: "reply-2", Usage: vo.AnalystUsageVo{InputTokens: 200, OutputTokens: 20}, Conclusion: &vo.RawAnalysisConclusionVo{
			Grade: "bullish", Confidence: 72, TimeHorizon: "short", Reason: "ETF 資金流入",
			KeyEventLinks: []string{"https://news/1", "https://invented"}, RiskFactors: []string{"礦工賣壓"},
		}}, nil
	})
	savedAnalysisResult := entities.AnalysisResult{}
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisResult *entities.AnalysisResult) error {
		savedAnalysisResult = *analysisResult
		return nil
	})
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21, SearchKeyword: "Bitcoin"})

	require.Len(t, receivedExchanges, 2)
	toolResults := receivedExchanges[1][0].ToolResults
	assert.Equal(t, "reply-1", receivedExchanges[1][0].Reply)
	require.Len(t, toolResults, 3)
	assert.Equal(t, "search-btc", toolResults[0].ToolCallID)
	assert.False(t, toolResults[0].IsError)
	var searchResult dto.SymbolNewsDto
	require.NoError(t, json.Unmarshal([]byte(toolResults[0].Content), &searchResult))
	assert.Equal(t, "BTC", searchResult.Symbol)
	assert.Len(t, searchResult.News, 2)
	assert.Equal(t, vo.AnalystToolResultVo{ToolCallID: "search-bad", Content: "市場類別只能是 crypto、twStock、usStock", IsError: true}, toolResults[2])
	assert.Equal(t, entities.AnalysisResult{
		AnalysisEventID: 21, Symbol: "BTC", Category: "crypto", Grade: "bullish", Confidence: 72, TimeHorizon: "short", Reason: "ETF 資金流入",
		KeyEvents:   []entities.AnalysisKeyEvent{{Title: "Bitcoin ETF inflows", Link: "https://news/1", PublishedAt: analysisStartedAt.Add(-time.Hour)}},
		RiskFactors: []string{"礦工賣壓"},
		Evidence: []entities.AnalysisEvidence{
			{Title: "Bitcoin ETF inflows", Link: "https://news/1", PublishedAt: analysisStartedAt.Add(-time.Hour), ProviderName: "CoinDesk"},
			{Title: "Miners sell", Link: "https://news/2", PublishedAt: analysisStartedAt.Add(-2 * time.Hour), ProviderName: "CoinDesk"},
			{Title: "MicroStrategy buys", Link: "https://news/3", PublishedAt: analysisStartedAt.Add(-3 * time.Hour), ProviderName: "Yahoo 財經"},
		},
		CreatedAt: analysisStartedAt,
	}, savedAnalysisResult)
	finishedAt := analysisStartedAt
	assert.Equal(t, entities.AnalysisEvent{ID: 21, ApiKeyID: 5, Symbol: "BTC", Category: "crypto", Status: "succeeded", Model: "claude-opus-5-5", InputTokens: 300, OutputTokens: 30, StartedAt: analysisStartedAt, FinishedAt: &finishedAt}, *finishedAnalysisEvent)
}

func TestAnalyzeSymbol_RecordsWhyAnAnalysisFailed(t *testing.T) {
	conclusionWithoutReason := &vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: " "}
	testCases := []struct {
		name                  string
		respond               func(exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error)
		givenResultStorage    func(fixture symbolAnalysisFixture)
		expectedFailureReason string
		expectedRounds        int
	}{
		{name: "analyst unavailable", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{}, errDatabaseDown
		}, expectedFailureReason: "AI 服務暫時無法使用", expectedRounds: 1},
		{name: "analyst refused", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{IsRefused: true}, nil
		}, expectedFailureReason: "AI 拒絕分析此標的", expectedRounds: 1},
		{name: "conclusion without reason", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{Conclusion: conclusionWithoutReason}, nil
		}, expectedFailureReason: "AI 未提供完整分析", expectedRounds: 1},
		{name: "neither searches nor conclusion", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{Reply: "plain text"}, nil
		}, expectedFailureReason: "AI 未提供完整分析", expectedRounds: 1},
		{name: "five rounds without a conclusion", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{NewsSearches: []vo.AnalystNewsSearchVo{{ToolCallID: "x", Symbol: "", Category: "crypto"}}}, nil
		}, expectedFailureReason: "AI 未在限制內完成分析", expectedRounds: 5},
		{name: "result could not be saved", respond: func([]vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
			return vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: "r"}}, nil
		}, givenResultStorage: func(fixture symbolAnalysisFixture) {
			fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}, expectedFailureReason: "分析結果保存失敗", expectedRounds: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
			rounds := 0
			fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
				rounds++
				return testCase.respond(exchanges)
			})
			if testCase.givenResultStorage != nil {
				testCase.givenResultStorage(fixture)
			}
			finishedAnalysisEvent := fixture.captureFinishedEvent()

			fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21, SearchKeyword: "Bitcoin"})

			assert.Equal(t, testCase.expectedRounds, rounds)
			assert.Equal(t, "failed", finishedAnalysisEvent.Status)
			assert.Equal(t, testCase.expectedFailureReason, finishedAnalysisEvent.FailureReason)
		})
	}
}

func TestAnalyzeSymbol_WithoutNewsConcludesNeutralWithZeroConfidence(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 80, TimeHorizon: "short", Reason: "r"}}, nil)
	savedAnalysisResult := entities.AnalysisResult{}
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisResult *entities.AnalysisResult) error {
		savedAnalysisResult = *analysisResult
		return nil
	})
	fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "neutral", savedAnalysisResult.Grade)
	assert.Equal(t, 0, savedAnalysisResult.Confidence)
}

func TestAnalyzeSymbol_StopsWhenTheEventCannotBeLoaded(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		storedEvent  *entities.AnalysisEvent
		storageError error
	}{
		{name: "missing", storedEvent: nil},
		{name: "storage unavailable", storageError: errDatabaseDown},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(testCase.storedEvent, testCase.storageError)

			fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})
		})
	}
}

func TestAnalyzeSymbol_ReportsAFinishedEventThatCannotBeSaved(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{IsRefused: true}, nil)
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).Return(errDatabaseDown)

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})
}

func TestGetAnalysisEvent(t *testing.T) {
	finishedAt := analysisStartedAt.Add(time.Minute)
	testCases := []struct {
		name           string
		storedEvent    *entities.AnalysisEvent
		storedResult   *entities.AnalysisResult
		expectedStatus string
		expectsResult  bool
		expectedReason string
		expectedError  error
	}{
		{name: "succeeded", storedEvent: &entities.AnalysisEvent{ID: 4, Status: "succeeded", FinishedAt: &finishedAt}, storedResult: &entities.AnalysisResult{AnalysisEventID: 4, Grade: "bearish"}, expectedStatus: "succeeded", expectsResult: true},
		{name: "running", storedEvent: &entities.AnalysisEvent{ID: 4, Status: "running"}, expectedStatus: "running"},
		{name: "failed", storedEvent: &entities.AnalysisEvent{ID: 4, Status: "failed", FailureReason: "AI 服務暫時無法使用", FinishedAt: &finishedAt}, expectedStatus: "failed", expectedReason: "AI 服務暫時無法使用"},
		{name: "missing", storedEvent: nil, expectedError: service.ErrAnalysisEventNotFound},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(4)).Return(testCase.storedEvent, nil)
			if testCase.storedResult != nil {
				fixture.analysisResultRepository.EXPECT().FindByAnalysisEventID(mock.Anything, uint(4)).Return(testCase.storedResult, nil)
			}

			analysisEvent, err := fixture.symbolAnalysisApplication.GetAnalysisEvent(context.Background(), 4)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedStatus, analysisEvent.Status)
			assert.Equal(t, testCase.expectedReason, analysisEvent.FailureReason)
			assert.Equal(t, testCase.expectsResult, analysisEvent.Result != nil)
		})
	}
}

func TestGetAnalysisEvent_StorageUnavailable(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(4)).Return(nil, errDatabaseDown)

	_, err := fixture.symbolAnalysisApplication.GetAnalysisEvent(context.Background(), 4)

	assert.ErrorIs(t, err, service.ErrAnalysisStorageUnavailable)
}

func TestFailInterruptedAnalysisEvents(t *testing.T) {
	for _, storageError := range []error{nil, errDatabaseDown} {
		t.Run(fmt.Sprint(storageError), func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			fixture.analysisEventRepository.EXPECT().FailAllRunning(mock.Anything, "服務重新啟動，分析中斷", analysisStartedAt).Return(storageError)

			err := fixture.symbolAnalysisApplication.FailInterruptedAnalysisEvents(context.Background())

			if storageError == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, service.ErrAnalysisStorageUnavailable)
			}
		})
	}
}

func TestStartSymbolAnalysis_RespondsBeforeTheAnalystAnswers(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil)
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 11
		return nil
	})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(11)).Return(runningBitcoinAnalysis(), nil)
	analystCalled := make(chan struct{})
	releaseAnalyst := make(chan struct{})
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(context.Context, vo.AnalystRequestVo, []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		close(analystCalled)
		<-releaseAnalyst
		return vo.AnalystTurnVo{IsRefused: true}, nil
	})
	finished := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, *entities.AnalysisEvent) error {
		close(finished)
		return nil
	})

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})
	<-analystCalled

	require.NoError(t, err)
	assert.Equal(t, "running", startedSymbolAnalysis.AnalysisEvent.Status)
	close(releaseAnalyst)
	<-finished
}

func TestAnalyzeSymbol_TellsTheAnalystWhenASearchFailsAndContinues(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.givenBitcoin()
	fixture.cryptoNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").Return(nil, errDatabaseDown)
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "NOTACOIN").Return("", false, nil)
	receivedToolResults := []vo.AnalystToolResultVo{}
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		if len(exchanges) == 0 {
			return vo.AnalystTurnVo{NewsSearches: []vo.AnalystNewsSearchVo{
				{ToolCallID: "all-providers-down", Symbol: "BTC", Category: "crypto"},
				{ToolCallID: "unknown-related", Symbol: "NOTACOIN", Category: "crypto"},
			}}, nil
		}
		receivedToolResults = exchanges[0].ToolResults
		return vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 60, TimeHorizon: "short", Reason: "r"}}, nil
	})
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, []vo.AnalystToolResultVo{
		{ToolCallID: "all-providers-down", Content: "新聞來源暫時無法使用", IsError: true},
		{ToolCallID: "unknown-related", Content: "找不到此標的", IsError: true},
	}, receivedToolResults)
	assert.Equal(t, "succeeded", finishedAnalysisEvent.Status)
}

func TestAnalyzeSymbol_RecordsTheModelThatActuallyAnswered(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{ModelName: "claude-opus-4-8", Conclusion: &vo.RawAnalysisConclusionVo{Grade: "neutral", Reason: "r"}}, nil)
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "claude-opus-4-8", finishedAnalysisEvent.Model)
}

func TestAnalyzeSymbol_DoesNotSearchInTheFinalRound(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.givenBitcoin()
	fixture.cryptoNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").Return([]vo.NewsVo{}, nil).Times(4)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{NewsSearches: []vo.AnalystNewsSearchVo{{ToolCallID: "x", Symbol: "BTC", Category: "crypto"}}}, nil).Times(5)
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "AI 未在限制內完成分析", finishedAnalysisEvent.FailureReason)
}

func TestStartSymbolAnalysis_ReplacesAStaleRunningAnalysis(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	staleAnalysisEvent := entities.AnalysisEvent{ID: 3, Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt.Add(-16 * time.Minute), InputTokens: 50}
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&staleAnalysisEvent, nil).Once()
	timedOutAnalysisEvent := entities.AnalysisEvent{}
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		timedOutAnalysisEvent = *analysisEvent
		return nil
	}).Once()
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 4
		return nil
	})
	analysisDone := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(4)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisDone)
		return nil, nil
	})

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(4), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
	assert.Equal(t, "failed", timedOutAnalysisEvent.Status)
	assert.Equal(t, "分析逾時", timedOutAnalysisEvent.FailureReason)
	assert.Equal(t, int64(50), timedOutAnalysisEvent.InputTokens)
	<-analysisDone
}

func TestStartSymbolAnalysis_ReusesARunningAnalysisWithinFifteenMinutes(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 3, Status: "running", StartedAt: analysisStartedAt.Add(-14 * time.Minute)}, nil)

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	require.NoError(t, err)
	assert.False(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(3), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
}

func TestStartSymbolAnalysis_FailsWhenAStaleAnalysisCannotBeClosed(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 3, Status: "running", StartedAt: analysisStartedAt.Add(-time.Hour)}, nil)
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).Return(errDatabaseDown)

	_, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	assert.ErrorIs(t, err, service.ErrAnalysisStorageUnavailable)
}

func TestStartSymbolAnalysis_RetriesWhenTheConcurrentWinnerAlreadyFinished(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil)
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(service.ErrAnalysisAlreadyRunning).Once()
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 9
		return nil
	}).Once()
	analysisDone := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(9)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisDone)
		return nil, nil
	})

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(9), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
	<-analysisDone
}

func TestStartSymbolAnalysis_LimitsConcurrentAnalysesButStillReuses(t *testing.T) {
	fixture := createSymbolAnalysisFixtureWithCapacity(t, 1)
	fixture.givenBitcoin()
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "ETH").Return("Ethereum", true, nil)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil).Once()
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 11
		return nil
	}).Once()
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(11)).Return(runningBitcoinAnalysis(), nil)
	releaseAnalyst := make(chan struct{})
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(context.Context, vo.AnalystRequestVo, []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		<-releaseAnalyst
		return vo.AnalystTurnVo{IsRefused: true}, nil
	})
	finished := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, *entities.AnalysisEvent) error {
		close(finished)
		return nil
	})
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "ETH", "crypto").Return(nil, nil).Once()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(&entities.AnalysisEvent{ID: 11, Status: "running", StartedAt: analysisStartedAt}, nil).Once()

	_, firstError := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})
	_, saturatedError := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "ETH", Category: "crypto"})
	reusedAnalysis, reusedError := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})
	close(releaseAnalyst)
	<-finished

	require.NoError(t, firstError)
	assert.ErrorIs(t, saturatedError, service.ErrAnalysisCapacityReached)
	require.NoError(t, reusedError)
	assert.Equal(t, uint(11), reusedAnalysis.AnalysisEvent.AnalysisEventID)
}

func TestStartSymbolAnalysis_ReleasesTheSlotOfARejectedStart(t *testing.T) {
	fixture := createSymbolAnalysisFixtureWithCapacity(t, 1)
	fixture.givenBitcoin()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, errDatabaseDown).Once()
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "BTC", "crypto").Return(nil, nil).Once()
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	analysisDone := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(0)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisDone)
		return nil, nil
	})

	_, rejectedError := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})
	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "BTC", Category: "crypto"})

	assert.ErrorIs(t, rejectedError, service.ErrAnalysisStorageUnavailable)
	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	<-analysisDone
}

func TestAnalyzeSymbol_SurvivesAPanickingAnalyst(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(context.Context, vo.AnalystRequestVo, []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		panic("unexpected response")
	})

	assert.NotPanics(t, func() {
		fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})
	})
}

func TestAnalyzeSymbol_GivesTheAnalystTenMinutesThenRecordsATimeout(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, _ vo.AnalystRequestVo, _ []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		deadline, hasDeadline := ctx.Deadline()
		assert.True(t, hasDeadline)
		assert.WithinDuration(t, time.Now().Add(10*time.Minute), deadline, 5*time.Second)
		return vo.AnalystTurnVo{}, context.DeadlineExceeded
	})
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "分析逾時", finishedAnalysisEvent.FailureReason)
}

func TestAnalyzeSymbol_RunsTheSearchesOfOneTurnConcurrently(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.givenBitcoin()
	ethereumStarted := make(chan struct{})
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "ETH").Return("Ethereum", true, nil)
	fixture.cryptoNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").RunAndReturn(func(context.Context, string) ([]vo.NewsVo, error) {
		<-ethereumStarted
		return []vo.NewsVo{{Title: "btc", Link: "https://news/btc", PublishedAt: analysisStartedAt}}, nil
	})
	fixture.cryptoNewsProxy.EXPECT().FetchNews(mock.Anything, "Ethereum").RunAndReturn(func(context.Context, string) ([]vo.NewsVo, error) {
		close(ethereumStarted)
		return []vo.NewsVo{{Title: "eth", Link: "https://news/eth", PublishedAt: analysisStartedAt}}, nil
	})
	receivedToolCallIDs := []string{}
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		if len(exchanges) == 0 {
			return vo.AnalystTurnVo{NewsSearches: []vo.AnalystNewsSearchVo{{ToolCallID: "btc", Symbol: "BTC", Category: "crypto"}, {ToolCallID: "eth", Symbol: "ETH", Category: "crypto"}}}, nil
		}
		for _, toolResult := range exchanges[0].ToolResults {
			receivedToolCallIDs = append(receivedToolCallIDs, toolResult.ToolCallID)
		}
		return vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "neutral", Reason: "r"}}, nil
	})
	savedAnalysisResult := entities.AnalysisResult{}
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisResult *entities.AnalysisResult) error {
		savedAnalysisResult = *analysisResult
		return nil
	})
	fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, []string{"btc", "eth"}, receivedToolCallIDs)
	assert.Equal(t, "https://news/btc", savedAnalysisResult.Evidence[0].Link)
	assert.Equal(t, "https://news/eth", savedAnalysisResult.Evidence[1].Link)
}

func TestAnalyzeSymbol_SavesThePriceAtAnalysis(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	priceQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString("86607.62"), "USDT", analysisStartedAt, "Binance")
	require.NoError(t, err)
	fixture.priceQuotesBySymbol["BTC"] = priceQuote
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: "r"}}, nil)
	savedAnalysisResult := entities.AnalysisResult{}
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisResult *entities.AnalysisResult) error {
		savedAnalysisResult = *analysisResult
		return nil
	})
	fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "86607.62", savedAnalysisResult.Price.Decimal.String())
	assert.Equal(t, "USDT", savedAnalysisResult.PriceCurrency)
	assert.Equal(t, "Binance", savedAnalysisResult.PriceSource)
}

func TestAnalyzeSymbol_CompletesWithoutAPrice(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(21)).Return(runningBitcoinAnalysis(), nil)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).Return(vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: "r"}}, nil)
	savedAnalysisResult := entities.AnalysisResult{}
	fixture.analysisResultRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisResult *entities.AnalysisResult) error {
		savedAnalysisResult = *analysisResult
		return nil
	})
	finishedAnalysisEvent := fixture.captureFinishedEvent()

	fixture.symbolAnalysisApplication.AnalyzeSymbol(context.Background(), dto.AnalyzeSymbolDto{AnalysisEventID: 21})

	assert.Equal(t, "succeeded", finishedAnalysisEvent.Status)
	assert.False(t, savedAnalysisResult.Price.Valid)
	assert.Nil(t, savedAnalysisResult.PricedAt)
}

func TestCapturePrice_UsesTheMarketsPriceSource(t *testing.T) {
	pricedAt := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	priceQuoteOf := func(source string) vo.PriceQuoteVo {
		priceQuote, _ := vo.NewPriceQuoteVo(decimal.NewFromInt(1), "X", pricedAt, source)
		return priceQuote
	}
	priceProxyReturning := func(source string) *mocks.MockIPriceProxy {
		priceProxy := mocks.NewMockIPriceProxy(t)
		priceProxy.EXPECT().FetchPrice(mock.Anything, mock.Anything).Return(priceQuoteOf(source), nil).Maybe()
		return priceProxy
	}
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{TwStock: []interfaces.IPriceProxy{priceProxyReturning("證交所")}, UsStock: []interfaces.IPriceProxy{priceProxyReturning("Yahoo 財經")}, Crypto: []interfaces.IPriceProxy{priceProxyReturning("Binance")}})

	for category, expectedSource := range map[string]string{"twStock": "證交所", "usStock": "Yahoo 財經", "crypto": "Binance"} {
		priceQuote := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "S", Category: category})

		require.NotNil(t, priceQuote)
		assert.Equal(t, expectedSource, priceQuote.Source)
	}
}

func TestCapturePrice_GivesUpAfterTenSeconds(t *testing.T) {
	stuckPriceProxy := mocks.NewMockIPriceProxy(t)
	stuckPriceProxy.EXPECT().FetchPrice(mock.Anything, "BTC").RunAndReturn(func(ctx context.Context, _ string) (vo.PriceQuoteVo, error) {
		deadline, hasDeadline := ctx.Deadline()
		assert.True(t, hasDeadline)
		assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, time.Second)
		return vo.PriceQuoteVo{}, context.DeadlineExceeded
	})
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{Crypto: []interfaces.IPriceProxy{stuckPriceProxy}})

	priceQuote := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "BTC", Category: "crypto"})

	assert.Nil(t, priceQuote)
}

func TestCapturePrice_UnknownMarketHasNoPrice(t *testing.T) {
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{TwStock: []interfaces.IPriceProxy{mocks.NewMockIPriceProxy(t)}, UsStock: []interfaces.IPriceProxy{mocks.NewMockIPriceProxy(t)}, Crypto: []interfaces.IPriceProxy{mocks.NewMockIPriceProxy(t)}})

	assert.Nil(t, priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "0700", Category: "hk"}))
}

func TestCapturePrice_HasItsOwnBudgetEvenNearTheAnalysisDeadline(t *testing.T) {
	priceProxy := mocks.NewMockIPriceProxy(t)
	priceProxy.EXPECT().FetchPrice(mock.Anything, "BTC").RunAndReturn(func(ctx context.Context, _ string) (vo.PriceQuoteVo, error) {
		deadline, _ := ctx.Deadline()
		assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, time.Second)
		return vo.NewPriceQuoteVo(decimal.NewFromInt(1), "USDT", analysisStartedAt, "Binance")
	})
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{Crypto: []interfaces.IPriceProxy{priceProxy}})
	almostExpiredContext, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	priceQuote := priceSnapshotService.CapturePrice(almostExpiredContext, dto.CapturePriceDto{Symbol: "BTC", Category: "crypto"})

	require.NotNil(t, priceQuote)
	assert.Equal(t, "Binance", priceQuote.Source)
}
