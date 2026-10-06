package main

import "os"

const (
	defaultServerPort  = "8080"
	defaultDatabaseUrl = "host=localhost port=5432 user=postgres password=postgres dbname=go_symbol_news_hunter sslmode=disable"
)

type ServerConfig struct {
	ServerPort  string
	DatabaseUrl string
}

func loadServerConfig() ServerConfig {
	return ServerConfig{
		ServerPort:  readEnvironmentOrDefault("SERVER_PORT", defaultServerPort),
		DatabaseUrl: readEnvironmentOrDefault("DATABASE_URL", defaultDatabaseUrl),
	}
}

func readEnvironmentOrDefault(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}
