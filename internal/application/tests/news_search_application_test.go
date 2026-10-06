package application_test

import (
	"context"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var newsSearchedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type newsSearchFixture struct {
	newsSearchApplication *application.NewsSearchApplication
	listedCompanyProxy    *mocks.MockIListedCompanyProxy
	cryptocurrencyProxy   *mocks.MockICryptocurrencyProxy
	cnyesNewsProxy        *mocks.MockINewsProxy
	twGoogleNewsProxy     *mocks.MockINewsProxy
	yahooNewsProxy        *mocks.MockINewsProxy
	usGoogleNewsProxy     *mocks.MockINewsProxy
	coinDeskNewsProxy     *mocks.MockINewsProxy
	cryptoGoogleNewsProxy *mocks.MockINewsProxy
}

func createNewsProxy(t *testing.T, providerName string) *mocks.MockINewsProxy {
	newsProxy := mocks.NewMockINewsProxy(t)
	newsProxy.EXPECT().ProviderName().Return(providerName).Maybe()
	return newsProxy
}

func createNewsSearchFixture(t *testing.T) newsSearchFixture {
	fixture := newsSearchFixture{
		listedCompanyProxy:    mocks.NewMockIListedCompanyProxy(t),
		cryptocurrencyProxy:   mocks.NewMockICryptocurrencyProxy(t),
		cnyesNewsProxy:        createNewsProxy(t, "鉅亨網"),
		twGoogleNewsProxy:     createNewsProxy(t, "Google 新聞"),
		yahooNewsProxy:        createNewsProxy(t, "Yahoo 財經"),
		usGoogleNewsProxy:     createNewsProxy(t, "Google 新聞"),
		coinDeskNewsProxy:     createNewsProxy(t, "CoinDesk"),
		cryptoGoogleNewsProxy: createNewsProxy(t, "Google 新聞"),
	}
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(newsSearchedAt).Maybe()
	newsSearchService := service.NewNewsSearchService(
		service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{fixture.listedCompanyProxy}, Crypto: fixture.cryptocurrencyProxy}),
		clockProxy,
		dto.NewsProviderCatalogDto{
			TwStock: []dto.NewsProviderDto{{NewsProxy: fixture.cnyesNewsProxy}, {NewsProxy: fixture.twGoogleNewsProxy}},
			UsStock: []dto.NewsProviderDto{{NewsProxy: fixture.yahooNewsProxy}, {NewsProxy: fixture.usGoogleNewsProxy}},
			Crypto:  []dto.NewsProviderDto{{NewsProxy: fixture.coinDeskNewsProxy, RequiresRelevanceFilter: true}, {NewsProxy: fixture.cryptoGoogleNewsProxy}},
		},
	)
	fixture.newsSearchApplication = application.NewNewsSearchApplication(newsSearchService)
	return fixture
}

func titlesOfNews(newsDtos []dto.NewsDto) []string {
	titles := []string{}
	for _, newsDto := range newsDtos {
		titles = append(titles, newsDto.Title)
	}
	return titles
}

func TestSearchSymbolNews_TaiwanStockSearchesByCompanyShortName(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "2330").Return("台積電", true, nil)
	fixture.cnyesNewsProxy.EXPECT().FetchNews(mock.Anything, "台積電").Return([]vo.NewsVo{{Title: "台積電法說會", ProviderName: "鉅亨網", PublishedAt: newsSearchedAt.Add(-2 * time.Hour)}}, nil)
	fixture.twGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "台積電").Return([]vo.NewsVo{{Title: "台積電法說會", ProviderName: "Google 新聞", PublishedAt: newsSearchedAt.Add(-3 * time.Hour)}, {Title: "台積電擴產", ProviderName: "Google 新聞", PublishedAt: newsSearchedAt.Add(-time.Hour)}}, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "2330", Category: "twStock"})

	require.NoError(t, err)
	assert.Equal(t, "2330", symbolNews.Symbol)
	assert.Equal(t, "twStock", symbolNews.Category)
	assert.Equal(t, []string{"台積電擴產", "台積電法說會"}, titlesOfNews(symbolNews.News))
	assert.Equal(t, "鉅亨網", symbolNews.News[1].ProviderName)
	assert.Equal(t, []string{}, symbolNews.FailedNewsProviders)
}

func TestSearchSymbolNews_UsStockSearchesByUpperCasedSymbol(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.yahooNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").Return([]vo.NewsVo{{Title: "Apple earnings", PublishedAt: newsSearchedAt}}, nil)
	fixture.usGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").Return([]vo.NewsVo{}, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: " aapl ", Category: "usStock"})

	require.NoError(t, err)
	assert.Equal(t, "AAPL", symbolNews.Symbol)
	assert.Equal(t, []string{"Apple earnings"}, titlesOfNews(symbolNews.News))
}

func TestSearchSymbolNews_UsStockWithoutNewsReturnsAnEmptyList(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.yahooNewsProxy.EXPECT().FetchNews(mock.Anything, "ZZZZ").Return([]vo.NewsVo{}, nil)
	fixture.usGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "ZZZZ").Return(nil, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "ZZZZ", Category: "usStock"})

	require.NoError(t, err)
	assert.Equal(t, []dto.NewsDto{}, symbolNews.News)
	assert.Equal(t, []string{}, symbolNews.FailedNewsProviders)
}

func TestSearchSymbolNews_CryptoFiltersOnlyWholeFeedProviders(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("Bitcoin", true, nil)
	fixture.coinDeskNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").Return([]vo.NewsVo{
		{Title: "Bitcoin rallies", PublishedAt: newsSearchedAt},
		{Title: "Ether upgrade", PublishedAt: newsSearchedAt.Add(-time.Minute)},
	}, nil)
	fixture.cryptoGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "Bitcoin").Return([]vo.NewsVo{{Title: "Crypto market wrap", PublishedAt: newsSearchedAt.Add(-2 * time.Minute)}}, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "btc", Category: "crypto"})

	require.NoError(t, err)
	assert.Equal(t, "BTC", symbolNews.Symbol)
	assert.Equal(t, []string{"Bitcoin rallies", "Crypto market wrap"}, titlesOfNews(symbolNews.News))
}

func TestSearchSymbolNews_ReportsAFailedProviderAndKeepsTheOthers(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "2330").Return("台積電", true, nil)
	fixture.cnyesNewsProxy.EXPECT().FetchNews(mock.Anything, "台積電").Return([]vo.NewsVo{{Title: "台積電法說會", PublishedAt: newsSearchedAt}}, nil)
	fixture.twGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "台積電").Return(nil, errDatabaseDown)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "2330", Category: "twStock"})

	require.NoError(t, err)
	assert.Equal(t, []string{"台積電法說會"}, titlesOfNews(symbolNews.News))
	assert.Equal(t, []string{"Google 新聞"}, symbolNews.FailedNewsProviders)
}

func TestSearchSymbolNews_Rejections(t *testing.T) {
	testCases := []struct {
		name          string
		symbol        string
		category      string
		givenProxies  func(fixture newsSearchFixture)
		expectedError error
	}{
		{name: "unsupported market", symbol: "0700", category: "港股", expectedError: service.ErrMarketCategoryUnsupported},
		{name: "missing market", symbol: "0700", category: "", expectedError: service.ErrMarketCategoryUnsupported},
		{name: "missing symbol", symbol: " ", category: "crypto", expectedError: service.ErrSymbolRequired},
		{name: "unlisted taiwan stock", symbol: "9999", category: "twStock", givenProxies: func(fixture newsSearchFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "9999").Return("", false, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "unknown coin", symbol: "NOTACOIN", category: "crypto", givenProxies: func(fixture newsSearchFixture) {
			fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "NOTACOIN").Return("", false, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "listed company directory unavailable", symbol: "2330", category: "twStock", givenProxies: func(fixture newsSearchFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "2330").Return("", false, errDatabaseDown)
		}, expectedError: service.ErrNewsProvidersUnavailable},
		{name: "coin directory unavailable", symbol: "BTC", category: "crypto", givenProxies: func(fixture newsSearchFixture) {
			fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "BTC").Return("", false, errDatabaseDown)
		}, expectedError: service.ErrNewsProvidersUnavailable},
		{name: "every news provider fails", symbol: "AAPL", category: "usStock", givenProxies: func(fixture newsSearchFixture) {
			fixture.yahooNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").Return(nil, errDatabaseDown)
			fixture.usGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").Return(nil, errDatabaseDown)
		}, expectedError: service.ErrNewsProvidersUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createNewsSearchFixture(t)
			if testCase.givenProxies != nil {
				testCase.givenProxies(fixture)
			}

			_, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: testCase.symbol, Category: testCase.category})

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestSearchSymbolNews_APanickingProviderFailsAlone(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.yahooNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").RunAndReturn(func(context.Context, string) ([]vo.NewsVo, error) {
		panic("unexpected payload")
	})
	fixture.usGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "AAPL").Return([]vo.NewsVo{{Title: "Apple earnings", PublishedAt: newsSearchedAt}}, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "AAPL", Category: "usStock"})

	require.NoError(t, err)
	assert.Equal(t, []string{"Apple earnings"}, titlesOfNews(symbolNews.News))
	assert.Equal(t, []string{"Yahoo 財經"}, symbolNews.FailedNewsProviders)
}

func TestSearchSymbolNews_CryptoRelevanceMatchesWholeWordsOnly(t *testing.T) {
	fixture := createNewsSearchFixture(t)
	fixture.cryptocurrencyProxy.EXPECT().FindCoinName(mock.Anything, "ETH").Return("Ethereum", true, nil)
	fixture.coinDeskNewsProxy.EXPECT().FetchNews(mock.Anything, "Ethereum").Return([]vo.NewsVo{
		{Title: "ETH tops $5,000", PublishedAt: newsSearchedAt},
		{Title: "Something big is coming together", PublishedAt: newsSearchedAt.Add(-time.Minute)},
		{Title: "Ethereum upgrade ships", PublishedAt: newsSearchedAt.Add(-2 * time.Minute)},
	}, nil)
	fixture.cryptoGoogleNewsProxy.EXPECT().FetchNews(mock.Anything, "Ethereum").Return(nil, nil)

	symbolNews, err := fixture.newsSearchApplication.SearchSymbolNews(context.Background(), dto.SearchSymbolNewsDto{Symbol: "ETH", Category: "crypto"})

	require.NoError(t, err)
	assert.Equal(t, []string{"ETH tops $5,000", "Ethereum upgrade ships"}, titlesOfNews(symbolNews.News))
}
