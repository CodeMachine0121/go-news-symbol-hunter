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

func buildNewsProvidersByCategory(httpBodyReader *utilities.HttpBodyReader, externalSourceUrls ExternalSourceUrls) map[string][]dto.NewsProviderDto {
	rssNewsReader := news.NewRssNewsReader(httpBodyReader, utilities.NewRssFeedParser())
	traditionalChineseGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsTraditionalChineseLocale)
	englishGoogleNewsProxy := news.NewGoogleNewsProxy(rssNewsReader, externalSourceUrls.GoogleNewsSearch, news.GoogleNewsEnglishLocale)
	return map[string][]dto.NewsProviderDto{
		vo.MarketCategoryTwStock: {
			{NewsProxy: news.NewCnyesNewsProxy(httpBodyReader, externalSourceUrls.CnyesSearch, externalSourceUrls.CnyesArticle)},
			{NewsProxy: traditionalChineseGoogleNewsProxy},
		},
		vo.MarketCategoryUsStock: {
			{NewsProxy: news.NewYahooFinanceNewsProxy(rssNewsReader, externalSourceUrls.YahooFinanceHeadline)},
			{NewsProxy: englishGoogleNewsProxy},
		},
		vo.MarketCategoryCrypto: {
			{NewsProxy: news.NewCoinDeskNewsProxy(rssNewsReader, externalSourceUrls.CoinDeskFeed), RequiresRelevanceFilter: true},
			{NewsProxy: news.NewCointelegraphNewsProxy(rssNewsReader, externalSourceUrls.CointelegraphFeed), RequiresRelevanceFilter: true},
			{NewsProxy: englishGoogleNewsProxy},
		},
	}
}

func buildControllers(database *gorm.DB) Controllers {
	clockProxy := system.NewSystemClockProxy()
	httpBodyReader := utilities.NewHttpBodyReader(newExternalHttpClient())
	apiKeyService := service.NewApiKeyService(persistence.NewApiKeyRepository(database), clockProxy, system.NewCryptoRandomProxy())
	symbolResolutionService := service.NewSymbolResolutionService(
		twse.NewTwseListedCompanyProxy(httpBodyReader, clockProxy, productionExternalSourceUrls.TwseListedCompanies),
		coingecko.NewCoinGeckoCryptocurrencyProxy(httpBodyReader, clockProxy, productionExternalSourceUrls.CoinGeckoSearch),
	)
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, buildNewsProvidersByCategory(httpBodyReader, productionExternalSourceUrls))
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
