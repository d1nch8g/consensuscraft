package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ConnectedNode     string
	ServerName        string
	BedrockServerPort int
	GRPCPort          int
	BedrockMaxThreads int
	MaxPlayers        int
	PlayerIdleTimeout int
	ViewDistance      int
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	return &Config{
		ConnectedNode:     getEnvString("CONNECTED_NODE", ""),
		ServerName:        getEnvString("SERVER_NAME", "ConsensusCraft"),
		BedrockServerPort: getEnvInt("BEDROCK_SERVER_PORT", 19132),
		GRPCPort:          getEnvInt("GRPC_PORT", 32842),
		BedrockMaxThreads: getEnvInt("BEDROCK_MAX_THREADS", 8),
		MaxPlayers:        getEnvInt("MAX_PLAYERS", 10),
		PlayerIdleTimeout: getEnvInt("PLAYER_IDLE_TIMEOUT", 30),
		ViewDistance:      getEnvInt("VIEW_DISTANCE", 10),
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}
