package main

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/controller"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Controllers struct {
	healthController *controller.HealthController
	apiKeyController *controller.ApiKeyController
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

func buildControllers(database *gorm.DB) Controllers {
	apiKeyService := service.NewApiKeyService(persistence.NewApiKeyRepository(database))
	return Controllers{
		healthController: controller.NewHealthController(),
		apiKeyController: controller.NewApiKeyController(application.NewApiKeyApplication(apiKeyService)),
	}
}

func registerRoutes(router *gin.Engine, controllers Controllers) {
	router.GET("/health", controllers.healthController.GetHealth)
	router.POST("/api-keys", controllers.apiKeyController.IssueApiKey)
	router.GET("/api-keys/me", controllers.apiKeyController.GetApiKeyStatus)
	router.DELETE("/api-keys/me", controllers.apiKeyController.RevokeApiKey)
}
