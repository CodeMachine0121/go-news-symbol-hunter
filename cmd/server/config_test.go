package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadServerConfig(t *testing.T) {
	testCases := []struct {
		name           string
		serverPort     string
		databaseUrl    string
		expectedConfig ServerConfig
	}{
		{name: "falls back to defaults", expectedConfig: ServerConfig{ServerPort: "8080", DatabaseUrl: defaultDatabaseUrl}},
		{name: "reads the environment", serverPort: "9090", databaseUrl: "host=db", expectedConfig: ServerConfig{ServerPort: "9090", DatabaseUrl: "host=db"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("SERVER_PORT", testCase.serverPort)
			t.Setenv("DATABASE_URL", testCase.databaseUrl)

			assert.Equal(t, testCase.expectedConfig, loadServerConfig())
		})
	}
}
