package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	serverConfig, err := loadServerConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	database, err := openDatabase(serverConfig)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	controllers := buildControllers(database, serverConfig)
	if err := controllers.symbolAnalysisApplication.FailInterruptedAnalysisEvents(context.Background()); err != nil {
		log.Fatalf("fail interrupted analysis events: %v", err)
	}
	router := gin.Default()
	registerRoutes(router, controllers)
	if err := router.Run(":" + serverConfig.ServerPort); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
