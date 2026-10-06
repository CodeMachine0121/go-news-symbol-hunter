package main

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/claude"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/anthropics/anthropic-sdk-go"
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/coingecko"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/news"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/system"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/twse"
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
}

func newExternalHttpClient() *http.Client {
	return &http.Client{Timeout: externalRequestTimeout}
}

type Controllers struct {
	healthController          *controller.HealthController
	apiKeyController          *controller.ApiKeyController
	newsController            *controller.NewsController
	analysisEventController   *controller.AnalysisEventController
	symbolAnalysisApplication *application.SymbolAnalysisApplication
}

func openDatabase(serverConfig ServerConfig) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(serverConfig.DatabaseUrl), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(&entities.ApiKey{}, &entities.AnalysisEvent{}, &entities.AnalysisResult{}); err != nil {
		return nil, err
	}
	return database, nil
}

func buildNewsProviderCatalog(httpBodyReader *httpfetch.HttpBodyReader, externalSourceUrls ExternalSourceUrls) dto.NewsProviderCatalogDto {
	rssNewsReader := news.NewRssNewsReader(httpBodyReader, utilities.NewRssFeedParser())
	traditionalChineseGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsTraditionalChineseLocale)
	englishGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsEnglishLocale)
	return dto.NewsProviderCatalogDto{
		TwStock: []dto.NewsProviderDto{
			{NewsProxy: news.NewCnyesNewsProxy(httpBodyReader, externalSourceUrls.CnyesSearch, externalSourceUrls.CnyesArticle)},
			{NewsProxy: traditionalChineseGoogleNewsProxy},
		},
		UsStock: []dto.NewsProviderDto{
			{NewsProxy: news.NewYahooFinanceNewsProxy(rssNewsReader, externalSourceUrls.YahooFinanceHeadline)},
			{NewsProxy: englishGoogleNewsProxy},
		},
		Crypto: []dto.NewsProviderDto{
			{NewsProxy: news.NewCoinDeskNewsProxy(rssNewsReader, externalSourceUrls.CoinDeskFeed), RequiresRelevanceFilter: true},
			{NewsProxy: news.NewCointelegraphNewsProxy(rssNewsReader, externalSourceUrls.CointelegraphFeed), RequiresRelevanceFilter: true},
			{NewsProxy: englishGoogleNewsProxy},
		},
	}
}

func buildControllers(database *gorm.DB, serverConfig ServerConfig) Controllers {
	clockProxy := system.NewSystemClockProxy()
	httpBodyReader := httpfetch.NewHttpBodyReader(newExternalHttpClient())
	apiKeyService := service.NewApiKeyService(persistence.NewApiKeyRepository(database), clockProxy, system.NewCryptoRandomProxy())
	symbolResolutionService := service.NewSymbolResolutionService(
		twse.NewTwseListedCompanyProxy(httpBodyReader, clockProxy, productionExternalSourceUrls.TwseListedCompanies),
		coingecko.NewCoinGeckoCryptocurrencyProxy(httpBodyReader, clockProxy, productionExternalSourceUrls.CoinGeckoSearch),
	)
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, buildNewsProviderCatalog(httpBodyReader, productionExternalSourceUrls))
	symbolAnalysisService := service.NewSymbolAnalysisService(
		symbolResolutionService,
		newsSearchService,
		claude.NewClaudeAnalystProxy(anthropic.NewClient(), serverConfig.AiAnalysisModel, serverConfig.AiAnalysisEffort),
		persistence.NewAnalysisEventRepository(database),
		persistence.NewAnalysisResultRepository(database),
		clockProxy,
	)
	symbolAnalysisApplication := application.NewSymbolAnalysisApplication(symbolAnalysisService, serverConfig.AiAnalysisMaximumConcurrency)
	return Controllers{
		healthController:          controller.NewHealthController(),
		apiKeyController:          controller.NewApiKeyController(application.NewApiKeyApplication(apiKeyService)),
		newsController:            controller.NewNewsController(application.NewNewsSearchApplication(newsSearchService)),
		analysisEventController:   controller.NewAnalysisEventController(symbolAnalysisApplication),
		symbolAnalysisApplication: symbolAnalysisApplication,
	}
}

func prepareRouter(ctx context.Context, controllers Controllers) (*gin.Engine, error) {
	if err := controllers.symbolAnalysisApplication.FailInterruptedAnalysisEvents(ctx); err != nil {
		return nil, fmt.Errorf("fail interrupted analysis events: %w", err)
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
}
