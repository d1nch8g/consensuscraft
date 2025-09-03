package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	config := Load()

	// Test default values
	assert.Empty(t, config.ConnectedNode, "ConnectedNode should be empty by default")
	assert.Equal(t, 19132, config.BedrockServerPort, "BedrockServerPort should default to 19132")
	assert.Equal(t, 8080, config.JaftHTTPPort, "JaftHTTPPort should default to 8080")
	assert.Equal(t, 8, config.BedrockMaxThreads, "BedrockMaxThreads should default to 8")
	assert.Equal(t, 10, config.MaxPlayers, "MaxPlayers should default to 10")
	assert.Equal(t, 30*time.Minute, config.PlayerIdleTimeout, "PlayerIdleTimeout should default to 30m")
	assert.Equal(t, "JAFT Server", config.ServerName, "ServerName should default to 'JAFT Server'")
	assert.Equal(t, 10, config.ViewDistance, "ViewDistance should default to 10")
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("CONNECTED_NODE", "192.168.1.100:8080")
	os.Setenv("BEDROCK_SERVER_PORT", "25565")
	os.Setenv("JAFT_HTTP_PORT", "9090")
	os.Setenv("BEDROCK_MAX_THREADS", "16")
	os.Setenv("MAX_PLAYERS", "20")
	os.Setenv("PLAYER_IDLE_TIMEOUT", "1h")
	os.Setenv("SERVER_NAME", "Test Server")
	os.Setenv("VIEW_DISTANCE", "16")

	defer os.Clearenv()

	config := Load()

	// Test environment values
	assert.Equal(t, "192.168.1.100:8080", config.ConnectedNode, "ConnectedNode should load from env")
	assert.Equal(t, 25565, config.BedrockServerPort, "BedrockServerPort should load from env")
	assert.Equal(t, 9090, config.JaftHTTPPort, "JaftHTTPPort should load from env")
	assert.Equal(t, 16, config.BedrockMaxThreads, "BedrockMaxThreads should load from env")
	assert.Equal(t, 20, config.MaxPlayers, "MaxPlayers should load from env")
	assert.Equal(t, time.Hour, config.PlayerIdleTimeout, "PlayerIdleTimeout should load from env")
	assert.Equal(t, "Test Server", config.ServerName, "ServerName should load from env")
	assert.Equal(t, 16, config.ViewDistance, "ViewDistance should load from env")
}

func TestInvalidValues(t *testing.T) {
	// Test invalid integer values (should fallback to defaults)
	os.Setenv("BEDROCK_SERVER_PORT", "invalid")
	os.Setenv("PLAYER_IDLE_TIMEOUT", "invalid")

	defer os.Clearenv()

	config := Load()

	// Should use default values when invalid
	assert.Equal(t, 19132, config.BedrockServerPort, "BedrockServerPort should fallback to default on invalid value")
	assert.Equal(t, 30*time.Minute, config.PlayerIdleTimeout, "PlayerIdleTimeout should fallback to default on invalid value")
}

func TestValidDurationFormats(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{"seconds", "45s", 45 * time.Second},
		{"minutes", "15m", 15 * time.Minute},
		{"hours", "2h", 2 * time.Hour},
		{"combined", "1h30m", time.Hour + 30*time.Minute},
		{"complex", "2h15m30s", 2*time.Hour + 15*time.Minute + 30*time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("PLAYER_IDLE_TIMEOUT", tc.value)

			config := Load()
			assert.Equal(t, tc.expected, config.PlayerIdleTimeout, "Duration format %s should parse correctly", tc.value)
		})
	}
}

func TestPortRangeValues(t *testing.T) {
	testCases := []struct {
		name     string
		port     string
		expected int
	}{
		{"low_port", "1024", 1024},
		{"high_port", "65535", 65535},
		{"minecraft_default", "25565", 25565},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("BEDROCK_SERVER_PORT", tc.port)

			config := Load()
			assert.Equal(t, tc.expected, config.BedrockServerPort, "Port %s should be parsed correctly", tc.port)
		})
	}
}

func TestViewDistanceValues(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected int
	}{
		{"minimum", "1", 1},
		{"maximum", "32", 32},
		{"common", "16", 16},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("VIEW_DISTANCE", tc.value)

			config := Load()
			assert.Equal(t, tc.expected, config.ViewDistance, "ViewDistance %s should be parsed correctly", tc.value)
		})
	}
}

func TestEmptyStringHandling(t *testing.T) {
	os.Clearenv()
	// Set empty strings explicitly
	os.Setenv("CONNECTED_NODE", "")
	os.Setenv("SERVER_NAME", "")

	config := Load()

	// Empty ConnectedNode should remain empty
	assert.Empty(t, config.ConnectedNode, "Empty CONNECTED_NODE should remain empty")
	// Empty ServerName should use default
	assert.Equal(t, "JAFT Server", config.ServerName, "Empty SERVER_NAME should use default")
}
