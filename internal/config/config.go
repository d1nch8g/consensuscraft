package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                 int
	BlockchainDifficulty int
	MinecraftServerURL   string
	MinecraftServerHash  string
	DataDir              string
	FUSEMountPoint       string
	NetworkPeers         []string
}

func Load() *Config {
	return &Config{
		Port:                 getEnvInt("JAFT_PORT", 42567),
		BlockchainDifficulty: getEnvInt("BLOCKCHAIN_DIFFICULTY", 2),
		MinecraftServerURL:   getEnv("MINECRAFT_SERVER_URL", "https://minecraft.azureedge.net/bin-linux/bedrock-server-1.20.81.01.zip"),
		MinecraftServerHash:  getEnv("MINECRAFT_SERVER_HASH", ""),
		DataDir:              getEnv("DATA_DIR", "/tmp/jaft"),
		FUSEMountPoint:       getEnv("FUSE_MOUNT", "/tmp/jaft-fuse"),
		NetworkPeers:         getEnvSlice("NETWORK_PEERS"),
	}
}

func getEnv(key, defaultValue string) string {
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
	}
	return defaultValue
}

func getEnvSlice(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return []string{}
	}
	// Simple comma-separated parsing
	peers := []string{}
	current := ""
	for _, char := range value {
		if char == ',' {
			if current != "" {
				peers = append(peers, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		peers = append(peers, current)
	}
	return peers
}
