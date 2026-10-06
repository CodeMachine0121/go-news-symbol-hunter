package main

import (
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
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

type Controllers struct {
	healthController *controller.HealthController
	apiKeyController *controller.ApiKeyController
	newsController   *controller.NewsController
}

func openDatabase(serverConfig ServerConfig) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(serverConfig.DatabaseUrl), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(&entities.ApiKey{}); err != nil {
		return nil, err
	}
	return database, nil
}

func buildNewsProvidersByCategory(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser) map[string][]dto.NewsProviderDto {
	traditionalChineseGoogleNewsProxy := news.NewGoogleNewsProxy(httpBodyReader, rssFeedParser, "https://news.google.com/rss/search", news.GoogleNewsTraditionalChineseLocale)
	englishGoogleNewsProxy := news.NewGoogleNewsProxy(httpBodyReader, rssFeedParser, "https://news.google.com/rss/search", news.GoogleNewsEnglishLocale)
	return map[string][]dto.NewsProviderDto{
		vo.MarketCategoryTwStock: {
			{NewsProxy: news.NewCnyesNewsProxy(httpBodyReader, "https://ess.api.cnyes.com/ess/api/v1/news/keyword", "https://news.cnyes.com/news/id")},
			{NewsProxy: traditionalChineseGoogleNewsProxy},
		},
		vo.MarketCategoryUsStock: {
			{NewsProxy: news.NewYahooFinanceNewsProxy(httpBodyReader, rssFeedParser, "https://feeds.finance.yahoo.com/rss/2.0/headline")},
			{NewsProxy: englishGoogleNewsProxy},
		},
		vo.MarketCategoryCrypto: {
			{NewsProxy: news.NewCoinDeskNewsProxy(httpBodyReader, rssFeedParser, "https://www.coindesk.com/arc/outboundfeeds/rss/"), RequiresRelevanceFilter: true},
			{NewsProxy: news.NewCointelegraphNewsProxy(httpBodyReader, rssFeedParser, "https://cointelegraph.com/rss"), RequiresRelevanceFilter: true},
			{NewsProxy: englishGoogleNewsProxy},
		},
	}
}

func buildControllers(database *gorm.DB) Controllers {
	clockProxy := system.NewSystemClockProxy()
	httpBodyReader := utilities.NewHttpBodyReader(&http.Client{Timeout: externalRequestTimeout})
	apiKeyService := service.NewApiKeyService(persistence.NewApiKeyRepository(database), clockProxy, system.NewCryptoRandomProxy())
	symbolResolutionService := service.NewSymbolResolutionService(
		twse.NewTwseListedCompanyProxy(httpBodyReader, clockProxy, "https://openapi.twse.com.tw/v1/opendata/t187ap03_L"),
		coingecko.NewCoinGeckoCryptocurrencyProxy(httpBodyReader, clockProxy, "https://api.coingecko.com/api/v3/search"),
	)
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, buildNewsProvidersByCategory(httpBodyReader, utilities.NewRssFeedParser()))
	return Controllers{
		healthController: controller.NewHealthController(),
		apiKeyController: controller.NewApiKeyController(application.NewApiKeyApplication(apiKeyService)),
		newsController:   controller.NewNewsController(application.NewNewsSearchApplication(newsSearchService)),
	}
}

func registerRoutes(router *gin.Engine, controllers Controllers) {
	router.GET("/health", controllers.healthController.GetHealth)
	router.POST("/api-keys", controllers.apiKeyController.IssueApiKey)
	router.GET("/api-keys/me", controllers.apiKeyController.GetApiKeyStatus)
	router.DELETE("/api-keys/me", controllers.apiKeyController.RevokeApiKey)
	protectedRoutes := router.Group("/", controllers.apiKeyController.RequireActiveApiKey())
	protectedRoutes.GET("/news", controllers.newsController.SearchSymbolNews)
}
