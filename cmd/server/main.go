package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	serverConfig := loadServerConfig()
	database, err := openDatabase(serverConfig)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	router := gin.Default()
	registerRoutes(router, buildControllers(database))
	if err := router.Run(":" + serverConfig.ServerPort); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
