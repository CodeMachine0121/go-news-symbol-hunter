package main

import (
	"errors"
	"fmt"
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

var (
	errDatabaseUrlMissing = errors.New("DATABASE_URL is required")
	errSettingUnreadable  = errors.New("setting cannot be read")
)

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
	// background job settings change what gets analyzed and how grades are weighted, so a typo stops startup
	// instead of silently falling back to a default; an unset value still takes its default
	backgroundJobsEnabled := true
	if rawBackgroundJobsEnabled := os.Getenv("BACKGROUND_JOBS_ENABLED"); rawBackgroundJobsEnabled != "" {
		backgroundJobsEnabled, err = strconv.ParseBool(rawBackgroundJobsEnabled)
		if err != nil {
			return ServerConfig{}, fmt.Errorf("%w: BACKGROUND_JOBS_ENABLED=%q", errSettingUnreadable, rawBackgroundJobsEnabled)
		}
	}
	tradingSessionJobInterval := defaultTradingSessionJobInterval
	if rawTradingSessionJobInterval := os.Getenv("TRADING_SESSION_JOB_INTERVAL_SECONDS"); rawTradingSessionJobInterval != "" {
		tradingSessionJobIntervalSeconds, parseError := strconv.Atoi(rawTradingSessionJobInterval)
		if parseError != nil {
			return ServerConfig{}, fmt.Errorf("%w: TRADING_SESSION_JOB_INTERVAL_SECONDS=%q", errSettingUnreadable, rawTradingSessionJobInterval)
		}
		tradingSessionJobInterval = time.Duration(tradingSessionJobIntervalSeconds) * time.Second
	}
	// an unset weight reads as zero, which the session weights replace with its default
	sessionWeightsBySetting := map[string]float64{}
	for _, sessionWeightSetting := range []string{"SESSION_WEIGHT_PRE_MARKET", "SESSION_WEIGHT_INTRADAY", "SESSION_WEIGHT_AFTER_MARKET"} {
		rawSessionWeight := os.Getenv(sessionWeightSetting)
		if rawSessionWeight == "" {
			continue
		}
		sessionWeight, parseError := strconv.ParseFloat(rawSessionWeight, 64)
		if parseError != nil {
			return ServerConfig{}, fmt.Errorf("%w: %s=%q", errSettingUnreadable, sessionWeightSetting, rawSessionWeight)
		}
		sessionWeightsBySetting[sessionWeightSetting] = sessionWeight
	}
	return ServerConfig{
		ServerPort:                   serverPort,
		DatabaseUrl:                  databaseUrl,
		AiAnalysisModel:              aiAnalysisModel,
		AiAnalysisEffort:             aiAnalysisEffort,
		AiAnalysisMaximumConcurrency: aiAnalysisMaximumConcurrency,
		BackgroundJobsEnabled:        backgroundJobsEnabled,
		TradingSessionJobInterval:    tradingSessionJobInterval,
		SessionWeights:               vo.NewSessionWeightsVo(sessionWeightsBySetting["SESSION_WEIGHT_PRE_MARKET"], sessionWeightsBySetting["SESSION_WEIGHT_INTRADAY"], sessionWeightsBySetting["SESSION_WEIGHT_AFTER_MARKET"]),
	}, nil
}
