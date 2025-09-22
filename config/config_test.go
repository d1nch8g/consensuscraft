package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	config := New()

	// Test default values
	assert.Empty(t, config.ConnectedNode, "ConnectedNode should be empty by default")
	assert.Equal(t, 19132, config.BedrockServerPort, "BedrockServerPort should default to 19132")
	assert.Equal(t, 32842, config.GRPCPort, "GRPCPort should default to 32842")
	assert.Equal(t, 8, config.BedrockMaxThreads, "BedrockMaxThreads should default to 8")
	assert.Equal(t, 10, config.MaxPlayers, "MaxPlayers should default to 10")
	assert.Equal(t, 30, config.PlayerIdleTimeout, "PlayerIdleTimeout should default to 30m")
	assert.Equal(t, "ConsensusCraft", config.ServerName, "ServerName should default to 'JAFT Server'")
	assert.Equal(t, 10, config.ViewDistance, "ViewDistance should default to 10")
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("CONNECTED_NODE", "192.168.1.100:8080")
	os.Setenv("BEDROCK_SERVER_PORT", "25565")
	os.Setenv("GRPC_PORT", "32842")
	os.Setenv("BEDROCK_MAX_THREADS", "16")
	os.Setenv("MAX_PLAYERS", "20")
	os.Setenv("PLAYER_IDLE_TIMEOUT", "1h")
	os.Setenv("SERVER_NAME", "Test Server")
	os.Setenv("VIEW_DISTANCE", "16")

	defer os.Clearenv()

	config := New()

	// Test environment values
	assert.Equal(t, "192.168.1.100:8080", config.ConnectedNode, "ConnectedNode should load from env")
	assert.Equal(t, 25565, config.BedrockServerPort, "BedrockServerPort should load from env")
	assert.Equal(t, 32842, config.GRPCPort, "GRPCPort should load from env")
	assert.Equal(t, 16, config.BedrockMaxThreads, "BedrockMaxThreads should load from env")
	assert.Equal(t, 20, config.MaxPlayers, "MaxPlayers should load from env")
	assert.Equal(t, 30, config.PlayerIdleTimeout, "PlayerIdleTimeout should load from env")
	assert.Equal(t, "Test Server", config.ServerName, "ServerName should load from env")
	assert.Equal(t, 16, config.ViewDistance, "ViewDistance should load from env")
}

func TestInvalidValues(t *testing.T) {
	// Test invalid integer values (should fallback to defaults)
	os.Setenv("BEDROCK_SERVER_PORT", "invalid")
	os.Setenv("PLAYER_IDLE_TIMEOUT", "invalid")

	defer os.Clearenv()

	config := New()

	// Should use default values when invalid
	assert.Equal(t, 19132, config.BedrockServerPort, "BedrockServerPort should fallback to default on invalid value")
	assert.Equal(t, 30, config.PlayerIdleTimeout, "PlayerIdleTimeout should fallback to default on invalid value")
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

			config := New()
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

			config := New()
			assert.Equal(t, tc.expected, config.ViewDistance, "ViewDistance %s should be parsed correctly", tc.value)
		})
	}
}

func TestEmptyStringHandling(t *testing.T) {
	os.Clearenv()
	// Set empty strings explicitly
	os.Setenv("CONNECTED_NODE", "")
	os.Setenv("SERVER_NAME", "")

	config := New()

	// Empty ConnectedNode should remain empty
	assert.Empty(t, config.ConnectedNode, "Empty CONNECTED_NODE should remain empty")
	// Empty ServerName should use default
	assert.Equal(t, "ConsensusCraft", config.ServerName, "Empty SERVER_NAME should use default")
}
