package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ConnectedNode string
	WebAddress    string
	GRPCPort      int
}

func New() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	return &Config{
		ConnectedNode: getEnvString("CONNECTED_NODE", ""),
		WebAddress:    getEnvString("WEB_ADDRESS", "localhost"),
		GRPCPort:      getEnvInt("GRPC_PORT", 32842),
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
