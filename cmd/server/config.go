package main

import (
	"errors"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	defaultServerPort                   = "8080"
	defaultAiAnalysisModel              = "claude-sonnet-5-5"
	defaultAiAnalysisEffort             = "medium"
	defaultAiAnalysisMaximumConcurrency = 4
	defaultTradingSessionJobInterval    = time.Minute
)

var supportedAiAnalysisEfforts = []string{"low", "medium", "high", "xhigh", "max"}

var errDatabaseUrlMissing = errors.New("DATABASE_URL is required")

type ServerConfig struct {
	ServerPort                   string
	DatabaseUrl                  string
	AiAnalysisModel              string
	AiAnalysisEffort             string
	AiAnalysisMaximumConcurrency int
	BackgroundJobsEnabled        bool
	// non-positive disables the trading session job
	TradingSessionJobInterval time.Duration
	SessionWeights            vo.SessionWeightsVo
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
	backgroundJobsEnabled, err := strconv.ParseBool(os.Getenv("BACKGROUND_JOBS_ENABLED"))
	if err != nil {
		backgroundJobsEnabled = true
	}
	tradingSessionJobIntervalSeconds, err := strconv.Atoi(os.Getenv("TRADING_SESSION_JOB_INTERVAL_SECONDS"))
	tradingSessionJobInterval := time.Duration(tradingSessionJobIntervalSeconds) * time.Second
	if err != nil {
		tradingSessionJobInterval = defaultTradingSessionJobInterval
	}
	// unparsable weights read as zero, which the session weights replace with their defaults
	preMarketSessionWeight, _ := strconv.ParseFloat(os.Getenv("SESSION_WEIGHT_PRE_MARKET"), 64)
	intradaySessionWeight, _ := strconv.ParseFloat(os.Getenv("SESSION_WEIGHT_INTRADAY"), 64)
	afterMarketSessionWeight, _ := strconv.ParseFloat(os.Getenv("SESSION_WEIGHT_AFTER_MARKET"), 64)
	return ServerConfig{
		ServerPort:                   serverPort,
		DatabaseUrl:                  databaseUrl,
		AiAnalysisModel:              aiAnalysisModel,
		AiAnalysisEffort:             aiAnalysisEffort,
		AiAnalysisMaximumConcurrency: aiAnalysisMaximumConcurrency,
		BackgroundJobsEnabled:        backgroundJobsEnabled,
		TradingSessionJobInterval:    tradingSessionJobInterval,
		SessionWeights:               vo.NewSessionWeightsVo(preMarketSessionWeight, intradaySessionWeight, afterMarketSessionWeight),
	}, nil
}
