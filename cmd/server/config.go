package main

import (
	"errors"
	"os"
	"slices"
	"strconv"
)

const (
	defaultServerPort                   = "8080"
	defaultAiAnalysisModel              = "claude-opus-5-5"
	defaultAiAnalysisEffort             = "high"
	defaultAiAnalysisMaximumConcurrency = 4
)

var supportedAiAnalysisEfforts = []string{"low", "medium", "high", "xhigh", "max"}

var errDatabaseUrlMissing = errors.New("DATABASE_URL is required")

type ServerConfig struct {
	ServerPort                   string
	DatabaseUrl                  string
	AiAnalysisModel              string
	AiAnalysisEffort             string
	AiAnalysisMaximumConcurrency int
}

func loadServerConfig() (ServerConfig, error) {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return ServerConfig{}, errDatabaseUrlMissing
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = defaultServerPort
	}
	aiAnalysisModel := os.Getenv("AI_ANALYSIS_MODEL")
	if aiAnalysisModel == "" {
		aiAnalysisModel = defaultAiAnalysisModel
	}
	aiAnalysisEffort := os.Getenv("AI_ANALYSIS_EFFORT")
	if !slices.Contains(supportedAiAnalysisEfforts, aiAnalysisEffort) {
		aiAnalysisEffort = defaultAiAnalysisEffort
	}
	aiAnalysisMaximumConcurrency, err := strconv.Atoi(os.Getenv("AI_ANALYSIS_MAX_CONCURRENCY"))
	if err != nil || aiAnalysisMaximumConcurrency <= 0 {
		aiAnalysisMaximumConcurrency = defaultAiAnalysisMaximumConcurrency
	}
	return ServerConfig{
		ServerPort:                   serverPort,
		DatabaseUrl:                  databaseUrl,
		AiAnalysisModel:              aiAnalysisModel,
		AiAnalysisEffort:             aiAnalysisEffort,
		AiAnalysisMaximumConcurrency: aiAnalysisMaximumConcurrency,
	}, nil
}
