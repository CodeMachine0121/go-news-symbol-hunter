package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestRegisteredRoutes_ServeHealthAndGuardApiKeyRoutes(t *testing.T) {
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

	assert.Equal(t, http.StatusOK, healthRecorder.Code)
	assert.Equal(t, http.StatusBadRequest, issueRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, statusRecorder.Code)
	assert.Equal(t, http.StatusUnauthorized, revokeRecorder.Code)
}
