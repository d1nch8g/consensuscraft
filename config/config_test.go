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
	assert.Equal(t, 32842, config.GRPCPort, "GRPCPort should default to 32842")
	assert.Equal(t, "localhost", config.WebAddress, "WebAddress should default to 'localhost'")
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("CONNECTED_NODE", "")
	os.Setenv("WEB_ADDRESS", "localhost")
	os.Setenv("GRPC_PORT", "32842")

	defer os.Clearenv()

	config := New()

	// Test environment values
	assert.Equal(t, "", config.ConnectedNode, "ConnectedNode should load from env")
	assert.Equal(t, 32842, config.GRPCPort, "GRPCPort should load from env")
	assert.Equal(t, "localhost", config.WebAddress, "WebAddress should load from env")
}

func TestInvalidValues(t *testing.T) {
	// Test invalid integer values (should fallback to defaults)
	os.Setenv("GRPC_PORT", "invalid")

	defer os.Clearenv()

	config := New()

	// Should use default values when invalid
	assert.Equal(t, 32842, config.GRPCPort, "BedrockServerPort should fallback to default on invalid value")
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
			os.Setenv("GRPC_PORT", tc.port)

			config := New()
			assert.Equal(t, tc.expected, config.GRPCPort, "Port %s should be parsed correctly", tc.port)
		})
	}
}
