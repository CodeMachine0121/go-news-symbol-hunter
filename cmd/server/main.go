package main

import (
	"context"
	"log"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/job"
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
	router, err := prepareRouter(context.Background(), controllers)
	if err != nil {
		log.Fatalf("prepare router: %v", err)
	}
	if serverConfig.BackgroundJobsEnabled {
		job.NewBackgroundJobManager(buildBackgroundJobs(controllers, serverConfig)).StartAll(context.Background())
	}
	if err := router.Run(":" + serverConfig.ServerPort); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
