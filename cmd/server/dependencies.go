package main

import (
	"context"
	"errors"
	"fmt"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/binance"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/claude"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/tpex"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/yahoofinance"
	"github.com/anthropics/anthropic-sdk-go"
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/coingecko"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/gotrading"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/news"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/system"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/twse"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/job"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const externalRequestTimeout = 10 * time.Second

type ExternalSourceUrls struct {
	CnyesSearch          string
	CnyesArticle         string
	GoogleNewsSearch     string
	YahooFinanceHeadline string
	CoinDeskFeed         string
	CointelegraphFeed    string
	TwseListedCompanies  string
	CoinGeckoSearch      string
	BinanceTickerPrice   string
	YahooFinanceChart    string
	TwseDailyClosing     string
	TpexDailyClosing     string
}

var productionExternalSourceUrls = ExternalSourceUrls{
	CnyesSearch:          "https://ess.api.cnyes.com/ess/api/v1/news/keyword",
	CnyesArticle:         "https://news.cnyes.com/news/id",
	GoogleNewsSearch:     "https://news.google.com/rss/search",
	YahooFinanceHeadline: "https://feeds.finance.yahoo.com/rss/2.0/headline",
	CoinDeskFeed:         "https://www.coindesk.com/arc/outboundfeeds/rss/",
	CointelegraphFeed:    "https://cointelegraph.com/rss",
	TwseListedCompanies:  "https://openapi.twse.com.tw/v1/opendata/t187ap03_L",
	CoinGeckoSearch:      "https://api.coingecko.com/api/v3/search",
	BinanceTickerPrice:   "https://api.binance.com/api/v3/ticker/price",
	YahooFinanceChart:    "https://query1.finance.yahoo.com/v8/finance/chart",
	TwseDailyClosing:     "https://openapi.twse.com.tw/v1/exchangeReport/STOCK_DAY_ALL",
	TpexDailyClosing:     "https://www.tpex.org.tw/openapi/v1/tpex_mainboard_daily_close_quotes",
}

func newExternalHttpClient() *http.Client {
	return &http.Client{Timeout: externalRequestTimeout}
}

type Controllers struct {
	healthController             *controller.HealthController
	apiKeyController             *controller.ApiKeyController
	newsController               *controller.NewsController
	analysisEventController      *controller.AnalysisEventController
	trackedSymbolGradeController *controller.TrackedSymbolGradeController
	gradeBacktestController      *controller.GradeBacktestController
	symbolAnalysisApplication    *application.SymbolAnalysisApplication
	sessionGradeApplication      *application.SessionGradeApplication
}

func openDatabase(serverConfig ServerConfig) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(serverConfig.DatabaseUrl), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(&entities.ApiKey{}, &entities.AnalysisEvent{}, &entities.GradeRecord{}, &entities.TrackedSymbol{}, &entities.SessionGrade{}, &entities.SessionRun{}); err != nil {
		return nil, err
	}
	// a database from before grade records still holds analysis results and combined grades in their own tables
	if err := persistence.NewGradeRecordRepository(database).ImportLegacyGrades(context.Background()); err != nil {
		return nil, fmt.Errorf("move legacy grades: %w", err)
	}
	return database, nil
}

func buildNewsProviderCatalog(httpBodyReader *httpfetch.HttpBodyReader, externalSourceUrls ExternalSourceUrls, yahooFinanceProxy *yahoofinance.YahooFinanceProxy) dto.NewsProviderCatalogDto {
	rssNewsReader := news.NewRssNewsReader(httpBodyReader, utilities.NewRssFeedParser())
	traditionalChineseGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsTraditionalChineseLocale)
	englishGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsEnglishLocale)
	return dto.NewsProviderCatalogDto{
		TwStock: []dto.NewsProviderDto{
			{NewsProxy: news.NewCnyesNewsProxy(httpBodyReader, externalSourceUrls.CnyesSearch, externalSourceUrls.CnyesArticle)},
			{NewsProxy: traditionalChineseGoogleNewsProxy},
		},
		UsStock: []dto.NewsProviderDto{
			{NewsProxy: yahooFinanceProxy},
			{NewsProxy: englishGoogleNewsProxy},
		},
		Crypto: []dto.NewsProviderDto{
			{NewsProxy: news.NewCoinDeskNewsProxy(rssNewsReader, externalSourceUrls.CoinDeskFeed), RequiresRelevanceFilter: true},
			{NewsProxy: news.NewCointelegraphNewsProxy(rssNewsReader, externalSourceUrls.CointelegraphFeed), RequiresRelevanceFilter: true},
			{NewsProxy: englishGoogleNewsProxy},
		},
	}
}

type ExternalSourceProxies struct {
	twseOpenDataProxy *twse.TwseOpenDataProxy
	tpexOpenDataProxy *tpex.TpexOpenDataProxy
	yahooFinanceProxy *yahoofinance.YahooFinanceProxy
	binancePriceProxy *binance.BinancePriceProxy
}

func buildExternalSourceProxies(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, externalSourceUrls ExternalSourceUrls) ExternalSourceProxies {
	return ExternalSourceProxies{
		twseOpenDataProxy: twse.NewTwseOpenDataProxy(httpBodyReader, clockProxy, twse.TwseOpenDataUrls{ListedCompanies: externalSourceUrls.TwseListedCompanies, DailyClosing: externalSourceUrls.TwseDailyClosing}),
		tpexOpenDataProxy: tpex.NewTpexOpenDataProxy(httpBodyReader, clockProxy, externalSourceUrls.TpexDailyClosing),
		yahooFinanceProxy: yahoofinance.NewYahooFinanceProxy(httpBodyReader, news.NewRssNewsReader(httpBodyReader, utilities.NewRssFeedParser()), yahoofinance.YahooFinanceUrls{HeadlineFeed: externalSourceUrls.YahooFinanceHeadline, Chart: externalSourceUrls.YahooFinanceChart}),
		binancePriceProxy: binance.NewBinancePriceProxy(httpBodyReader, clockProxy, externalSourceUrls.BinanceTickerPrice),
	}
}

func buildSymbolDirectoryCatalog(externalSourceProxies ExternalSourceProxies, cryptocurrencyProxy interfaces.ICryptocurrencyProxy) dto.SymbolDirectoryCatalogDto {
	return dto.SymbolDirectoryCatalogDto{
		TwStock: []interfaces.IListedCompanyProxy{externalSourceProxies.twseOpenDataProxy, externalSourceProxies.tpexOpenDataProxy},
		UsStock: externalSourceProxies.yahooFinanceProxy,
		Crypto:  cryptocurrencyProxy,
	}
}

func buildPriceProviderCatalog(externalSourceProxies ExternalSourceProxies) dto.PriceProviderCatalogDto {
	return dto.PriceProviderCatalogDto{
		TwStock: []interfaces.IPriceProxy{externalSourceProxies.twseOpenDataProxy, externalSourceProxies.tpexOpenDataProxy},
		UsStock: []interfaces.IPriceProxy{externalSourceProxies.yahooFinanceProxy},
		Crypto:  []interfaces.IPriceProxy{externalSourceProxies.binancePriceProxy},
	}
}

// go-trading has no US stocks, so they stay without a price source and cannot be backtested
func buildPriceHistoryCatalog(priceHistoryProxy interfaces.IPriceHistoryProxy) dto.PriceHistoryCatalogDto {
	return dto.PriceHistoryCatalogDto{TwStock: priceHistoryProxy, Crypto: priceHistoryProxy}
}

func buildControllers(database *gorm.DB, serverConfig ServerConfig) Controllers {
	clockProxy := system.NewSystemClockProxy()
	httpBodyReader := httpfetch.NewHttpBodyReader(newExternalHttpClient())
	externalSourceProxies := buildExternalSourceProxies(httpBodyReader, clockProxy, productionExternalSourceUrls)
	apiKeyService := service.NewApiKeyService(persistence.NewApiKeyRepository(database), clockProxy, system.NewCryptoRandomProxy())
	symbolResolutionService := service.NewSymbolResolutionService(buildSymbolDirectoryCatalog(externalSourceProxies, coingecko.NewCoinGeckoCryptocurrencyProxy(httpBodyReader, clockProxy, productionExternalSourceUrls.CoinGeckoSearch)))
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, buildNewsProviderCatalog(httpBodyReader, productionExternalSourceUrls, externalSourceProxies.yahooFinanceProxy))
	analystConsultationService := service.NewAnalystConsultationService(claude.NewClaudeAnalystProxy(anthropic.NewClient(), serverConfig.AiAnalysisModel, serverConfig.AiAnalysisEffort), newsSearchService)
	symbolAnalysisService := service.NewSymbolAnalysisService(
		symbolResolutionService,
		analystConsultationService,
		service.NewPriceSnapshotService(buildPriceProviderCatalog(externalSourceProxies)),
		persistence.NewAnalysisEventRepository(database),
		persistence.NewGradeRecordRepository(database),
		clockProxy,
	)
	symbolAnalysisApplication := application.NewSymbolAnalysisApplication(symbolAnalysisService, serverConfig.AiAnalysisMaximumConcurrency)
	gradeBacktestService := service.NewGradeBacktestService(
		persistence.NewGradeRecordRepository(database),
		clockProxy,
		buildPriceHistoryCatalog(gotrading.NewGoTradingPriceHistoryProxy(newExternalHttpClient(), serverConfig.PriceServiceAccount)),
	)
	sessionGradeApplication := application.NewSessionGradeApplication(service.NewSessionGradeService(
		symbolResolutionService,
		analystConsultationService,
		persistence.NewTrackedSymbolRepository(database),
		persistence.NewSessionGradeRepository(database),
		persistence.NewGradeRecordRepository(database),
		persistence.NewSessionRunRepository(database),
		clockProxy,
		serverConfig.SessionWeights,
	))
	return Controllers{
		healthController:             controller.NewHealthController(),
		apiKeyController:             controller.NewApiKeyController(application.NewApiKeyApplication(apiKeyService)),
		newsController:               controller.NewNewsController(application.NewNewsSearchApplication(newsSearchService)),
		analysisEventController:      controller.NewAnalysisEventController(symbolAnalysisApplication),
		trackedSymbolGradeController: controller.NewTrackedSymbolGradeController(sessionGradeApplication),
		gradeBacktestController:      controller.NewGradeBacktestController(application.NewGradeBacktestApplication(gradeBacktestService)),
		symbolAnalysisApplication:    symbolAnalysisApplication,
		sessionGradeApplication:      sessionGradeApplication,
	}
}

func prepareRouter(ctx context.Context, controllers Controllers) (*gin.Engine, error) {
	// both sweeps run even when one fails, so a restart reports every kind of run it could not close
	if err := errors.Join(
		controllers.symbolAnalysisApplication.FailInterruptedAnalysisEvents(ctx),
		controllers.sessionGradeApplication.FailInterruptedSessionRuns(ctx),
	); err != nil {
		return nil, fmt.Errorf("fail interrupted runs: %w", err)
	}
	router := gin.Default()
	registerRoutes(router, controllers)
	return router, nil
}

func registerRoutes(router *gin.Engine, controllers Controllers) {
	router.GET("/health", controllers.healthController.GetHealth)
	router.POST("/api-keys", controllers.apiKeyController.IssueApiKey)
	router.GET("/api-keys/me", controllers.apiKeyController.GetApiKeyStatus)
	router.DELETE("/api-keys/me", controllers.apiKeyController.RevokeApiKey)
	protectedRoutes := router.Group("/", controllers.apiKeyController.RequireActiveApiKey())
	protectedRoutes.GET("/news", controllers.newsController.SearchSymbolNews)
	protectedRoutes.POST("/analysis-events", controllers.analysisEventController.StartSymbolAnalysis)
	protectedRoutes.GET("/analysis-events/:analysisEventId", controllers.analysisEventController.GetAnalysisEvent)
	protectedRoutes.GET("/tracked-symbol-grades", controllers.trackedSymbolGradeController.GetTrackedSymbolGrades)
	protectedRoutes.GET("/grade-backtest", controllers.gradeBacktestController.GetGradeBacktest)
}

func buildBackgroundJobs(controllers Controllers, serverConfig ServerConfig) []interfaces.IBackgroundJob {
	return []interfaces.IBackgroundJob{
		job.NewTradingSessionJob(controllers.sessionGradeApplication, serverConfig.TradingSessionJobInterval),
	}
}
