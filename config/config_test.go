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
	assert.Empty(t, config.BannedNodes, "BannedNodes should be empty by default")
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

func TestBannedNodes(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single_node",
			envValue: "192.168.1.100:19132",
			expected: []string{"192.168.1.100:19132"},
		},
		{
			name:     "multiple_nodes",
			envValue: "192.168.1.100:19132,10.0.0.5:19132,example.com:19132",
			expected: []string{"192.168.1.100:19132", "10.0.0.5:19132", "example.com:19132"},
		},
		{
			name:     "nodes_with_whitespace",
			envValue: " 192.168.1.100:19132 , 10.0.0.5:19132 , example.com:19132 ",
			expected: []string{"192.168.1.100:19132", "10.0.0.5:19132", "example.com:19132"},
		},
		{
			name:     "empty_entries_filtered",
			envValue: "192.168.1.100:19132,,10.0.0.5:19132, ,example.com:19132",
			expected: []string{"192.168.1.100:19132", "10.0.0.5:19132", "example.com:19132"},
		},
		{
			name:     "empty_string",
			envValue: "",
			expected: []string{},
		},
		{
			name:     "only_commas_and_spaces",
			envValue: " , , ",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("BANNED_NODES", tc.envValue)

			config := New()
			assert.Equal(t, tc.expected, config.BannedNodes, "BannedNodes should be parsed correctly for case: %s", tc.name)
		})
	}
}

func TestBannedNodesNotSet(t *testing.T) {
	// Test when BANNED_NODES environment variable is not set
	os.Clearenv()

	config := New()
	assert.Empty(t, config.BannedNodes, "BannedNodes should be empty when env var not set")
}
