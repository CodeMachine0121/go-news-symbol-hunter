package main

import (
	"errors"
	"os"
)

const defaultServerPort = "8080"

var errDatabaseUrlMissing = errors.New("DATABASE_URL is required")

type ServerConfig struct {
	ServerPort  string
	DatabaseUrl string
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
	return ServerConfig{ServerPort: serverPort, DatabaseUrl: databaseUrl}, nil
}
