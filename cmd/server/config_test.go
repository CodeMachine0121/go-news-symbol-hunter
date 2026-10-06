package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadServerConfig(t *testing.T) {
	testCases := []struct {
		name             string
		serverPort       string
		databaseUrl      string
		aiAnalysisModel  string
		aiAnalysisEffort string
		maxConcurrency   string
		expectedConfig   ServerConfig
		expectedError    error
	}{
		{name: "defaults the port and analysis settings", databaseUrl: "host=db", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: "host=db", AiAnalysisModel: "claude-opus-5-5", AiAnalysisEffort: "high", AiAnalysisMaximumConcurrency: 4}},
		{name: "reads the environment", serverPort: "9090", databaseUrl: "host=db", aiAnalysisModel: "claude-sonnet-5-5", aiAnalysisEffort: "medium", maxConcurrency: "8", expectedConfig: ServerConfig{ServerPort: "9090", DatabaseUrl: "host=db", AiAnalysisModel: "claude-sonnet-5-5", AiAnalysisEffort: "medium", AiAnalysisMaximumConcurrency: 8}},
		{name: "replaces an unsupported effort with the default", databaseUrl: "host=db", aiAnalysisEffort: "extreme", maxConcurrency: "0", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: "host=db", AiAnalysisModel: "claude-opus-5-5", AiAnalysisEffort: "high", AiAnalysisMaximumConcurrency: 4}},
		{name: "fails fast without a database url", serverPort: "9090", expectedError: errDatabaseUrlMissing},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("SERVER_PORT", testCase.serverPort)
			t.Setenv("DATABASE_URL", testCase.databaseUrl)
			t.Setenv("AI_ANALYSIS_MODEL", testCase.aiAnalysisModel)
			t.Setenv("AI_ANALYSIS_EFFORT", testCase.aiAnalysisEffort)
			t.Setenv("AI_ANALYSIS_MAX_CONCURRENCY", testCase.maxConcurrency)

			serverConfig, err := loadServerConfig()

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedConfig, serverConfig)
		})
	}
}
