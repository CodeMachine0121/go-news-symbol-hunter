package main

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var defaultSessionWeights = vo.SessionWeightsVo{PreMarket: 0.3, Intraday: 0.2, AfterMarket: 0.5}

func TestLoadServerConfig(t *testing.T) {
	testCases := []struct {
		name                      string
		serverPort                string
		databaseUrl               string
		aiAnalysisModel           string
		aiAnalysisEffort          string
		maxConcurrency            string
		backgroundJobsEnabled     string
		tradingSessionJobInterval string
		preMarketWeight           string
		intradayWeight            string
		afterMarketWeight         string
		expectedConfig            ServerConfig
		expectedError             error
	}{
		{name: "defaults the port, analysis and background job settings", databaseUrl: "host=db", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: "host=db", AiAnalysisModel: "claude-sonnet-5-5", AiAnalysisEffort: "medium", AiAnalysisMaximumConcurrency: 4, BackgroundJobsEnabled: true, TradingSessionJobInterval: time.Minute, SessionWeights: defaultSessionWeights}},
		{name: "reads the environment", serverPort: "9090", databaseUrl: "host=db", aiAnalysisModel: "claude-opus-5-5", aiAnalysisEffort: "high", maxConcurrency: "8", backgroundJobsEnabled: "false", tradingSessionJobInterval: "30", preMarketWeight: "0.4", intradayWeight: "0.1", afterMarketWeight: "0.6", expectedConfig: ServerConfig{ServerPort: "9090", DatabaseUrl: "host=db", AiAnalysisModel: "claude-opus-5-5", AiAnalysisEffort: "high", AiAnalysisMaximumConcurrency: 8, BackgroundJobsEnabled: false, TradingSessionJobInterval: 30 * time.Second, SessionWeights: vo.SessionWeightsVo{PreMarket: 0.4, Intraday: 0.1, AfterMarket: 0.6}}},
		{name: "replaces unusable values with the defaults", databaseUrl: "host=db", aiAnalysisEffort: "extreme", maxConcurrency: "0", backgroundJobsEnabled: "maybe", tradingSessionJobInterval: "soon", preMarketWeight: "heavy", intradayWeight: "-1", afterMarketWeight: "0", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: "host=db", AiAnalysisModel: "claude-sonnet-5-5", AiAnalysisEffort: "medium", AiAnalysisMaximumConcurrency: 4, BackgroundJobsEnabled: true, TradingSessionJobInterval: time.Minute, SessionWeights: defaultSessionWeights}},
		{name: "keeps a zero interval so the job stays disabled", databaseUrl: "host=db", tradingSessionJobInterval: "0", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: "host=db", AiAnalysisModel: "claude-sonnet-5-5", AiAnalysisEffort: "medium", AiAnalysisMaximumConcurrency: 4, BackgroundJobsEnabled: true, TradingSessionJobInterval: 0, SessionWeights: defaultSessionWeights}},
		{name: "fails fast without a database url", serverPort: "9090", expectedError: errDatabaseUrlMissing},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("SERVER_PORT", testCase.serverPort)
			t.Setenv("DATABASE_URL", testCase.databaseUrl)
			t.Setenv("AI_ANALYSIS_MODEL", testCase.aiAnalysisModel)
			t.Setenv("AI_ANALYSIS_EFFORT", testCase.aiAnalysisEffort)
			t.Setenv("AI_ANALYSIS_MAX_CONCURRENCY", testCase.maxConcurrency)
			t.Setenv("BACKGROUND_JOBS_ENABLED", testCase.backgroundJobsEnabled)
			t.Setenv("TRADING_SESSION_JOB_INTERVAL_SECONDS", testCase.tradingSessionJobInterval)
			t.Setenv("SESSION_WEIGHT_PRE_MARKET", testCase.preMarketWeight)
			t.Setenv("SESSION_WEIGHT_INTRADAY", testCase.intradayWeight)
			t.Setenv("SESSION_WEIGHT_AFTER_MARKET", testCase.afterMarketWeight)

			serverConfig, err := loadServerConfig()

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedConfig, serverConfig)
		})
	}
}
