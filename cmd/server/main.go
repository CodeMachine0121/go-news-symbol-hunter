package main

import (
	"context"
	"log"

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
	router, err := prepareRouter(context.Background(), buildControllers(database, serverConfig))
	if err != nil {
		log.Fatalf("prepare router: %v", err)
	}
	if err := router.Run(":" + serverConfig.ServerPort); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
