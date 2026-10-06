package main

import (
	"context"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestOpenDatabase_RejectsAnUnreachableDatabase(t *testing.T) {
	_, err := openDatabase(ServerConfig{DatabaseUrl: "host=127.0.0.1 port=1 user=nobody dbname=none sslmode=disable connect_timeout=1"})

	assert.Error(t, err)
}

func TestRegisteredRoutes_ServeHealthAndGuardProtectedRoutes(t *testing.T) {
	// never connects: the routes exercised here answer before reaching the database
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 port=1 dbname=unused_test"}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRoutes(router, buildControllers(database, ServerConfig{AiAnalysisModel: "claude-opus-5-5", AiAnalysisEffort: "high"}))

	healthRecorder := httptest.NewRecorder()
	router.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	issueRecorder := httptest.NewRecorder()
	router.ServeHTTP(issueRecorder, httptest.NewRequest(http.MethodPost, "/api-keys", nil))
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, httptest.NewRequest(http.MethodGet, "/api-keys/me", nil))
	revokeRecorder := httptest.NewRecorder()
	router.ServeHTTP(revokeRecorder, httptest.NewRequest(http.MethodDelete, "/api-keys/me", nil))
	newsRecorder := httptest.NewRecorder()
	router.ServeHTTP(newsRecorder, httptest.NewRequest(http.MethodGet, "/news?symbol=BTC&category=crypto", nil))
	startAnalysisRecorder := httptest.NewRecorder()
	router.ServeHTTP(startAnalysisRecorder, httptest.NewRequest(http.MethodPost, "/analysis-events", nil))
	getAnalysisRecorder := httptest.NewRecorder()
	router.ServeHTTP(getAnalysisRecorder, httptest.NewRequest(http.MethodGet, "/analysis-events/1", nil))

	assert.Equal(t, http.StatusOK, healthRecorder.Code)
	assert.Equal(t, http.StatusBadRequest, issueRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, statusRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, revokeRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, newsRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, startAnalysisRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, getAnalysisRecorder.Code)
}

func TestBuildNewsProvidersByCategory_AssignsProvidersAndLocalesPerMarket(t *testing.T) {
	receivedGoogleLanguages := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if language := request.URL.Query().Get("hl"); language != "" {
			receivedGoogleLanguages <- language + "|" + request.URL.Query().Get("gl") + "|" + request.URL.Query().Get("ceid")
		}
		_, _ = writer.Write([]byte(`<rss><channel></channel></rss>`))
	}))
	defer server.Close()
	newsProviderCatalog := buildNewsProviderCatalog(httpfetch.NewHttpBodyReader(server.Client()), ExternalSourceUrls{GoogleNewsSearch: server.URL})
	newsProvidersByCategory := map[string][]dto.NewsProviderDto{"twStock": newsProviderCatalog.TwStock, "usStock": newsProviderCatalog.UsStock, "crypto": newsProviderCatalog.Crypto}

	googleLocaleOf := func(category string) string {
		for _, newsProvider := range newsProvidersByCategory[category] {
			if newsProvider.NewsProxy.ProviderName() == "Google 新聞" {
				_, err := newsProvider.NewsProxy.FetchNews(context.Background(), "keyword")
				require.NoError(t, err)
				return <-receivedGoogleLanguages
			}
		}
		return ""
	}
	providerNamesOf := func(category string) []string {
		providerNames := []string{}
		for _, newsProvider := range newsProvidersByCategory[category] {
			providerNames = append(providerNames, newsProvider.NewsProxy.ProviderName())
		}
		return providerNames
	}
	relevanceFiltersOf := func(category string) []bool {
		relevanceFilters := []bool{}
		for _, newsProvider := range newsProvidersByCategory[category] {
			relevanceFilters = append(relevanceFilters, newsProvider.RequiresRelevanceFilter)
		}
		return relevanceFilters
	}
	assert.Equal(t, []string{"鉅亨網", "Google 新聞"}, providerNamesOf("twStock"))
	assert.Equal(t, []string{"Yahoo 財經", "Google 新聞"}, providerNamesOf("usStock"))
	assert.Equal(t, []string{"CoinDesk", "Cointelegraph", "Google 新聞"}, providerNamesOf("crypto"))
	assert.Equal(t, []bool{false, false}, relevanceFiltersOf("twStock"))
	assert.Equal(t, []bool{false, false}, relevanceFiltersOf("usStock"))
	assert.Equal(t, []bool{true, true, false}, relevanceFiltersOf("crypto"))
	assert.Equal(t, "zh-TW|TW|TW:zh-Hant", googleLocaleOf("twStock"))
	assert.Equal(t, "en-US|US|US:en", googleLocaleOf("usStock"))
	assert.Equal(t, "en-US|US|US:en", googleLocaleOf("crypto"))
}

func TestNewExternalHttpClient_GivesUpAfterTenSeconds(t *testing.T) {
	assert.Equal(t, 10*time.Second, newExternalHttpClient().Timeout)
}

func TestPrepareRouter_FailsInterruptedAnalysesBeforeServing(t *testing.T) {
	// never connects, so the startup sweep is the only step that can fail
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 port=1 dbname=unused_test connect_timeout=1"}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)

	router, err := prepareRouter(context.Background(), buildControllers(database, ServerConfig{AiAnalysisModel: "claude-opus-5-5", AiAnalysisEffort: "high"}))

	assert.ErrorContains(t, err, "fail interrupted analysis events")
	assert.Nil(t, router)
}
