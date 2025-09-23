package bds

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds the BDS configuration loaded from .env file
type Config struct {
	// Server configuration
	BedrockServerPort int
	BedrockMaxThreads int
	MaxPlayers        int
	PlayerIdleTimeout int
	ViewDistance      int

	// Hardcoded security settings (enforced by wrapper)
	AllowCheats         bool
	ForceGamemode       bool
	TexturepackRequired bool
	Gamemode            string
	Difficulty          string
	ServerName          string
	LevelName           string
	LevelSeed           string
}

// LoadConfig loads configuration from .env file with defaults
func LoadConfig() (*Config, error) {
	config := &Config{
		// Default values
		BedrockServerPort: 19132,
		BedrockMaxThreads: 8,
		MaxPlayers:        10,
		PlayerIdleTimeout: 30,
		ViewDistance:      32,

		// Hardcoded security settings (enforced by wrapper)
		TexturepackRequired: true,
		AllowCheats:         true,
		Difficulty:          "normal",
		ForceGamemode:       true,
		Gamemode:            "survival",
		LevelSeed:           "",
		ServerName:          "ConsensusCraft Node",
		LevelName:           "Bedrock level",
	}

	// Load from .env file if it exists
	if err := config.loadFromEnv(); err != nil {
		log.Printf("BDS: Warning - failed to load .env file: %v", err)
		log.Println("BDS: Using default configuration")
	}

	return config, nil
}

// loadFromEnv loads configuration from .env file
func (c *Config) loadFromEnv() error {
	file, err := os.Open(".env")
	if err != nil {
		if os.IsNotExist(err) {
			// Create default .env file
			return c.createDefaultEnv()
		}
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		if err := c.setConfigValue(key, value); err != nil {
			log.Printf("BDS: Warning - invalid config value %s=%s: %v", key, value, err)
		}
	}

	return scanner.Err()
}

// setConfigValue sets a configuration value by key
func (c *Config) setConfigValue(key, value string) error {
	switch key {
	case "BEDROCK_SERVER_PORT":
		if port, err := strconv.Atoi(value); err == nil {
			c.BedrockServerPort = port
		} else {
			return fmt.Errorf("invalid port: %s", value)
		}
	case "BEDROCK_MAX_THREADS":
		if threads, err := strconv.Atoi(value); err == nil {
			c.BedrockMaxThreads = threads
		} else {
			return fmt.Errorf("invalid max threads: %s", value)
		}
	case "MAX_PLAYERS":
		if players, err := strconv.Atoi(value); err == nil {
			c.MaxPlayers = players
		} else {
			return fmt.Errorf("invalid max players: %s", value)
		}
	case "PLAYER_IDLE_TIMEOUT":
		if timeout, err := strconv.Atoi(value); err == nil {
			c.PlayerIdleTimeout = timeout
		} else {
			return fmt.Errorf("invalid idle timeout: %s", value)
		}
	case "VIEW_DISTANCE":
		if distance, err := strconv.Atoi(value); err == nil {
			c.ViewDistance = distance
		} else {
			return fmt.Errorf("invalid view distance: %s", value)
		}
	case "SERVER_NAME":
		c.ServerName = value
	case "LEVEL_NAME":
		c.LevelName = value
	case "LEVEL_SEED":
		c.LevelSeed = value
	case "DIFFICULTY":
		if value == "peaceful" || value == "easy" || value == "normal" || value == "hard" {
			c.Difficulty = value
		} else {
			return fmt.Errorf("invalid difficulty: %s", value)
		}
	case "GAMEMODE":
		if value == "survival" || value == "creative" || value == "adventure" {
			c.Gamemode = value
		} else {
			return fmt.Errorf("invalid gamemode: %s", value)
		}
	}
	return nil
}

// createDefaultEnv creates a default .env file
func (c *Config) createDefaultEnv() error {
	file, err := os.Create(".env")
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	envContent := `# ConsensusCraft BDS Configuration
# This file contains configuration for the Bedrock Dedicated Server
# Modify these values as needed - they will not overwrite server.properties

# Server Network Configuration
BEDROCK_SERVER_PORT=19132
BEDROCK_MAX_THREADS=8

# Player Configuration
MAX_PLAYERS=10
PLAYER_IDLE_TIMEOUT=30
VIEW_DISTANCE=32

# World Configuration
SERVER_NAME="ConsensusCraft Node"
LEVEL_NAME="Bedrock level"
LEVEL_SEED=""
DIFFICULTY=normal
GAMEMODE=survival

# Note: Security settings (online-mode, xbox-auth, etc.) are hardcoded for network security
# and cannot be changed through this configuration file.
`

	_, err = file.WriteString(envContent)
	if err != nil {
		return fmt.Errorf("failed to write default .env content: %w", err)
	}

	log.Println("BDS: Created default .env configuration file")
	return nil
}


// GetEnvVars returns environment variables for the server process
func (c *Config) GetEnvVars() []string {
	return []string{
		fmt.Sprintf("BEDROCK_SERVER_PORT=%d", c.BedrockServerPort),
		fmt.Sprintf("BEDROCK_MAX_THREADS=%d", c.BedrockMaxThreads),
		fmt.Sprintf("MAX_PLAYERS=%d", c.MaxPlayers),
		fmt.Sprintf("PLAYER_IDLE_TIMEOUT=%d", c.PlayerIdleTimeout),
		fmt.Sprintf("VIEW_DISTANCE=%d", c.ViewDistance),
		fmt.Sprintf("SERVER_NAME=%s", c.ServerName),
		fmt.Sprintf("LEVEL_NAME=%s", c.LevelName),
		fmt.Sprintf("LEVEL_SEED=%s", c.LevelSeed),
		fmt.Sprintf("DIFFICULTY=%s", c.Difficulty),
		fmt.Sprintf("GAMEMODE=%s", c.Gamemode),
	}
}
