package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
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

type directoryAnswer struct {
	shortName string
	found     bool
	err       error
}

func searchTaiwanStockNews(t *testing.T, stockCode string, twseAnswer directoryAnswer, tpexAnswer *directoryAnswer) (dto.SymbolNewsDto, []string, error) {
	twseDirectory := mocks.NewMockIListedCompanyProxy(t)
	twseDirectory.EXPECT().FindCompanyShortName(mock.Anything, stockCode).Return(twseAnswer.shortName, twseAnswer.found, twseAnswer.err)
	if tpexAnswer == nil {
		tpexAnswer = &directoryAnswer{}
	}
	tpexDirectory := mocks.NewMockIListedCompanyProxy(t)
	tpexDirectory.EXPECT().FindCompanyShortName(mock.Anything, stockCode).Return(tpexAnswer.shortName, tpexAnswer.found, tpexAnswer.err)
	searchedKeywords := []string{}
	newsProxy := mocks.NewMockINewsProxy(t)
	newsProxy.EXPECT().ProviderName().Return("鉅亨網").Maybe()
	newsProxy.EXPECT().FetchNews(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, searchKeyword string) ([]vo.NewsVo, error) {
		searchedKeywords = append(searchedKeywords, searchKeyword)
		return []vo.NewsVo{}, nil
	}).Maybe()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)).Maybe()
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{
		TwStock: []interfaces.IListedCompanyProxy{twseDirectory, tpexDirectory},
		Crypto:  mocks.NewMockICryptocurrencyProxy(t),
	})
	newsSearchApplication := application.NewNewsSearchApplication(service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{
		TwStock: []dto.NewsProviderDto{{NewsProxy: newsProxy}},
	}))

	symbolNews, err := newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: stockCode, Category: "twStock"})
	return symbolNews, searchedKeywords, err
}

func TestTaiwanStockResolution_ListedStocksStillComeFromTwse(t *testing.T) {
	symbolNews, searchedKeywords, err := searchTaiwanStockNews(t, "2330", directoryAnswer{shortName: "台積電", found: true}, &directoryAnswer{shortName: "同代號的上櫃證券", found: true})

	require.NoError(t, err)
	assert.Equal(t, "2330", symbolNews.Symbol)
	assert.Equal(t, []string{"台積電"}, searchedKeywords)
}

func TestTaiwanStockResolution_FallsBackToTpexForOtcStocks(t *testing.T) {
	_, searchedKeywords, err := searchTaiwanStockNews(t, "6182", directoryAnswer{}, &directoryAnswer{shortName: "合晶", found: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"合晶"}, searchedKeywords)
}

func TestTaiwanStockResolution_UsesTpexWhenTwseIsUnavailable(t *testing.T) {
	_, searchedKeywords, err := searchTaiwanStockNews(t, "6182", directoryAnswer{err: errDatabaseDown}, &directoryAnswer{shortName: "合晶", found: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"合晶"}, searchedKeywords)
}

func TestTaiwanStockResolution_Rejections(t *testing.T) {
	testCases := []struct {
		name          string
		stockCode     string
		twseAnswer    directoryAnswer
		tpexAnswer    directoryAnswer
		expectedError error
	}{
		{name: "neither exchange lists the stock", stockCode: "9999", expectedError: service.ErrSymbolNotFound},
		{name: "twse unavailable and tpex does not list it", stockCode: "2330", twseAnswer: directoryAnswer{err: errDatabaseDown}, expectedError: service.ErrNewsProvidersUnavailable},
		{name: "twse does not list it and tpex unavailable", stockCode: "6182", tpexAnswer: directoryAnswer{err: errDatabaseDown}, expectedError: service.ErrNewsProvidersUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, searchedKeywords, err := searchTaiwanStockNews(t, testCase.stockCode, testCase.twseAnswer, &testCase.tpexAnswer)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Empty(t, searchedKeywords)
		})
	}
}

func TestCapturePrice_FallsBackToTheNextSourceOfTheMarket(t *testing.T) {
	tradingDate := time.Date(2026, 10, 6, 0, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*60*60))
	tpexQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString("128.00"), "TWD", tradingDate, "櫃買中心")
	require.NoError(t, err)
	twseQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString("2575.00"), "TWD", tradingDate, "證交所")
	require.NoError(t, err)
	twsePrices := mocks.NewMockIPriceProxy(t)
	twsePrices.EXPECT().FetchPrice(mock.Anything, "6182").Return(vo.PriceQuoteVo{}, errDatabaseDown)
	twsePrices.EXPECT().FetchPrice(mock.Anything, "2330").Return(twseQuote, nil)
	twsePrices.EXPECT().FetchPrice(mock.Anything, "9999").Return(vo.PriceQuoteVo{}, errDatabaseDown)
	tpexPrices := mocks.NewMockIPriceProxy(t)
	tpexPrices.EXPECT().FetchPrice(mock.Anything, "6182").Return(tpexQuote, nil)
	tpexPrices.EXPECT().FetchPrice(mock.Anything, "9999").Return(vo.PriceQuoteVo{}, errDatabaseDown)
	tpexPrices.EXPECT().FetchPrice(mock.Anything, "2330").Return(tpexQuote, nil)
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{TwStock: []interfaces.IPriceProxy{twsePrices, tpexPrices}})

	otcPrice := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "6182", Category: "twStock"})
	listedPrice := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "2330", Category: "twStock"})
	missingPrice := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "9999", Category: "twStock"})

	require.NotNil(t, otcPrice)
	assert.Equal(t, tpexQuote, *otcPrice)
	require.NotNil(t, listedPrice)
	assert.Equal(t, "證交所", listedPrice.Source)
	assert.Nil(t, missingPrice)
}

func TestStartSymbolAnalysis_AcceptsAnOtcStock(t *testing.T) {
	twseDirectory := mocks.NewMockIListedCompanyProxy(t)
	twseDirectory.EXPECT().FindCompanyShortName(mock.Anything, "6182").Return("", false, nil)
	tpexDirectory := mocks.NewMockIListedCompanyProxy(t)
	tpexDirectory.EXPECT().FindCompanyShortName(mock.Anything, "6182").Return("合晶", true, nil)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)).Maybe()
	analystProxy := mocks.NewMockIAnalystProxy(t)
	analystProxy.EXPECT().ModelName().Return("claude-sonnet-5-5").Maybe()
	analysisEventRepository := mocks.NewMockIAnalysisEventRepository(t)
	analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "6182", "twStock").Return(nil, nil)
	analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		analysisEvent.ID = 61
		return nil
	})
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{twseDirectory, tpexDirectory}})
	symbolAnalysisService := service.NewSymbolAnalysisService(
		symbolResolutionService,
		service.NewAnalystConsultationService(analystProxy, service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{})),
		service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{}),
		analysisEventRepository, mocks.NewMockIAnalysisResultRepository(t), clockProxy,
	)

	startedSymbolAnalysis, err := symbolAnalysisService.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "6182", Category: "twStock", AllowsNewAnalysis: true})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(61), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
	assert.Equal(t, "running", startedSymbolAnalysis.AnalysisEvent.Status)
	assert.Equal(t, "合晶", startedSymbolAnalysis.SearchKeyword)
}

func TestTaiwanStockResolution_AsksBothExchangesAtOnce(t *testing.T) {
	bothAsked := make(chan struct{}, 2)
	waitForTheOther := func() {
		bothAsked <- struct{}{}
		for len(bothAsked) < 2 {
			time.Sleep(time.Millisecond)
		}
	}
	twseDirectory := mocks.NewMockIListedCompanyProxy(t)
	twseDirectory.EXPECT().FindCompanyShortName(mock.Anything, "6182").RunAndReturn(func(context.Context, string) (string, bool, error) {
		waitForTheOther()
		return "", false, nil
	})
	tpexDirectory := mocks.NewMockIListedCompanyProxy(t)
	tpexDirectory.EXPECT().FindCompanyShortName(mock.Anything, "6182").RunAndReturn(func(context.Context, string) (string, bool, error) {
		waitForTheOther()
		return "合晶", true, nil
	})
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{twseDirectory, tpexDirectory}})

	resolvedSymbol, err := symbolResolutionService.ResolveSymbol(context.Background(), dto.ResolveSymbolDto{Symbol: "6182", Category: "twStock"})

	require.NoError(t, err)
	assert.Equal(t, "合晶", resolvedSymbol.SearchKeyword)
}

func TestCapturePrice_AsksEverySourceOfTheMarketAtOnce(t *testing.T) {
	tradingDate := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	tpexQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString("128.00"), "TWD", tradingDate, "櫃買中心")
	require.NoError(t, err)
	bothAsked := make(chan struct{}, 2)
	waitForTheOther := func() {
		bothAsked <- struct{}{}
		for len(bothAsked) < 2 {
			time.Sleep(time.Millisecond)
		}
	}
	twsePrices := mocks.NewMockIPriceProxy(t)
	twsePrices.EXPECT().FetchPrice(mock.Anything, "6182").RunAndReturn(func(context.Context, string) (vo.PriceQuoteVo, error) {
		waitForTheOther()
		return vo.PriceQuoteVo{}, errDatabaseDown
	})
	tpexPrices := mocks.NewMockIPriceProxy(t)
	tpexPrices.EXPECT().FetchPrice(mock.Anything, "6182").RunAndReturn(func(context.Context, string) (vo.PriceQuoteVo, error) {
		waitForTheOther()
		return tpexQuote, nil
	})
	priceSnapshotService := service.NewPriceSnapshotService(dto.PriceProviderCatalogDto{TwStock: []interfaces.IPriceProxy{twsePrices, tpexPrices}})

	priceQuote := priceSnapshotService.CapturePrice(context.Background(), dto.CapturePriceDto{Symbol: "6182", Category: "twStock"})

	require.NotNil(t, priceQuote)
	assert.Equal(t, "櫃買中心", priceQuote.Source)
}
