package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDatabase_RejectsAnUnreachableDatabase(t *testing.T) {
	_, err := openDatabase(ServerConfig{DatabaseUrl: "host=127.0.0.1 port=1 user=nobody dbname=none sslmode=disable connect_timeout=1"})

	assert.Error(t, err)
}

func TestRegisteredRoutes_ServeHealthAndApiKeyIssuing(t *testing.T) {
	databaseUrl := os.Getenv("TEST_POSTGRES_DSN")
	if databaseUrl == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	database, err := openDatabase(ServerConfig{DatabaseUrl: databaseUrl})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRoutes(router, buildControllers(database))

	healthRecorder := httptest.NewRecorder()
	router.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	issueRecorder := httptest.NewRecorder()
	router.ServeHTTP(issueRecorder, httptest.NewRequest(http.MethodPost, "/api-keys", strings.NewReader(`{"name":"整合測試"}`)))
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, httptest.NewRequest(http.MethodGet, "/api-keys/me", nil))
	revokeRecorder := httptest.NewRecorder()
	router.ServeHTTP(revokeRecorder, httptest.NewRequest(http.MethodDelete, "/api-keys/me", nil))

	assert.Equal(t, http.StatusOK, healthRecorder.Code)
	assert.Equal(t, http.StatusCreated, issueRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, statusRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, revokeRecorder.Code)
}
