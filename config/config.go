package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ConnectedNode     string
	BedrockServerPort int
	JaftHTTPPort      int
	BedrockMaxThreads int
	MaxPlayers        int
	PlayerIdleTimeout time.Duration
	ServerName        string
	ViewDistance      int
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	return &Config{
		ConnectedNode:     getEnvString("CONNECTED_NODE", ""),
		BedrockServerPort: getEnvInt("BEDROCK_SERVER_PORT", 19132),
		JaftHTTPPort:      getEnvInt("JAFT_HTTP_PORT", 8080),
		BedrockMaxThreads: getEnvInt("BEDROCK_MAX_THREADS", 8),
		MaxPlayers:        getEnvInt("MAX_PLAYERS", 10),
		PlayerIdleTimeout: getEnvDuration("PLAYER_IDLE_TIMEOUT", "30m"),
		ServerName:        getEnvString("SERVER_NAME", "JAFT Server"),
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

func getEnvDuration(key, defaultValue string) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("Warning: Invalid duration value for %s: %s, using default: %s", key, value, defaultValue)
		duration, _ = time.ParseDuration(defaultValue)
	}

	return duration
}
