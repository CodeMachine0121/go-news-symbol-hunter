package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
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
	registerRoutes(router, buildControllers(database))

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

	assert.Equal(t, http.StatusOK, healthRecorder.Code)
	assert.Equal(t, http.StatusBadRequest, issueRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, statusRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, revokeRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, newsRecorder.Code)
}

func TestBuildNewsProvidersByCategory_AssignsProvidersPerMarket(t *testing.T) {
	newsProvidersByCategory := buildNewsProvidersByCategory(utilities.NewHttpBodyReader(http.DefaultClient))

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
	assert.Equal(t, []bool{true, true, false}, relevanceFiltersOf("crypto"))
	assert.Equal(t, []bool{false, false}, relevanceFiltersOf("twStock"))
}
