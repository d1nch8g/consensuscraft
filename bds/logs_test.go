package bds

import (
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewLogMonitor tests the constructor function
func TestNewLogMonitor(t *testing.T) {
	t.Run("CreateNewLogMonitor", func(t *testing.T) {
		lm := NewLogMonitor()
		
		assert.NotNil(t, lm)
		assert.NotNil(t, lm.playerSpawnedRegex)
		assert.NotNil(t, lm.enderChestRegex)
		
		// Test player spawned regex
		matches := lm.playerSpawnedRegex.FindStringSubmatch("Player Spawned: TestPlayer")
		assert.Len(t, matches, 2)
		assert.Equal(t, "TestPlayer", matches[1])
		
		// Test ender chest regex
		matches = lm.enderChestRegex.FindStringSubmatch("[X_ENDER_CHEST][TestPlayer][[{\"item\":\"stone\"}]]")
		assert.Len(t, matches, 3)
		assert.Equal(t, "TestPlayer", matches[1])
		assert.Equal(t, "[{\"item\":\"stone\"}]", matches[2])
	})
}

// TestLogMonitor_Start tests the unified Start method
func TestLogMonitor_Start(t *testing.T) {
	t.Run("StartWithDirectIO", func(t *testing.T) {
		lm := NewLogMonitor()
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
		}
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		cmd := &exec.Cmd{}
		
		// Should log direct I/O message and return early
		lm.Start(cmd, bds, params)
		// No pipes provided, so it should use direct I/O and return
	})

	t.Run("StartWithPipes", func(t *testing.T) {
		lm := NewLogMonitor()
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
		}
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		cmd := &exec.Cmd{}
		
		// Create mock pipes
		stdoutReader, stdoutWriter := io.Pipe()
		stderrReader, stderrWriter := io.Pipe()
		stdinReader, stdinWriter := io.Pipe()
		
		// Start monitoring with pipes
		lm.Start(cmd, bds, params, stdoutReader, stderrReader, stdinWriter)
		
		// Close pipes to clean up
		stdoutReader.Close()
		stderrReader.Close()
		stdinWriter.Close()
		stdinReader.Close()
		stdoutWriter.Close()
		stderrWriter.Close()
	})

	t.Run("StartWithInvalidPipes", func(t *testing.T) {
		lm := NewLogMonitor()
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
		}
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		cmd := &exec.Cmd{}
		
		// Should handle invalid pipe types gracefully
		lm.Start(cmd, bds, params, "invalid", "pipes", "here")
	})

	t.Run("StartWithInsufficientPipes", func(t *testing.T) {
		lm := NewLogMonitor()
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
		}
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		cmd := &exec.Cmd{}
		
		// Should use direct I/O with insufficient pipes
		stdoutReader, stdoutWriter := io.Pipe()
		defer stdoutReader.Close()
		defer stdoutWriter.Close()
		
		lm.Start(cmd, bds, params, stdoutReader)
	})
}

// TestLogMonitor_monitorServerLogs tests the log monitoring functionality
func TestLogMonitor_monitorServerLogs(t *testing.T) {
	t.Run("MonitorPlayerSpawnedEvent", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create mock BDS and parameters
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			InventoryReceiveCallback: func(playerName string) ([]byte, error) {
				assert.Equal(t, "TestPlayer", playerName)
				return []byte(`[{"item":"stone"}]`), nil
			},
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create mock stdin
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Create input with player spawned event
		input := "Player Spawned: TestPlayer\n"
		reader := strings.NewReader(input)
		
		// Start monitoring in a goroutine
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		
		// Give it time to process
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("MonitorEnderChestEvent", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create mock BDS and parameters
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create mock stdin
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Create input with ender chest event
		input := "[X_ENDER_CHEST][TestPlayer][[{\"item\":\"stone\"}]]\n"
		reader := strings.NewReader(input)
		
		// Start monitoring in a goroutine
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		
		// Wait for inventory update
		select {
		case update := <-bds.InventoryUpdate:
			assert.Equal(t, "TestPlayer", update.PlayerName)
			assert.Equal(t, `[{"item":"stone"}]`, string(update.Inventory))
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for inventory update")
		}
	})

	t.Run("MonitorMultipleEvents", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create mock BDS and parameters
		inventoryCallbackCalled := false
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			InventoryReceiveCallback: func(playerName string) ([]byte, error) {
				inventoryCallbackCalled = true
				assert.Equal(t, "Player1", playerName)
				return []byte(`[{"item":"stone"}]`), nil
			},
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create mock stdin
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Create input with multiple events
		input := `Player Spawned: Player1
[X_ENDER_CHEST][Player2][[{"item":"diamond"}]]
Regular log message
`
		reader := strings.NewReader(input)
		
		// Start monitoring in a goroutine
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		
		// Wait for inventory updates
		updates := 0
		for updates < 1 {
			select {
			case update := <-bds.InventoryUpdate:
				assert.Equal(t, "Player2", update.PlayerName)
				assert.Equal(t, `[{"item":"diamond"}]`, string(update.Inventory))
				updates++
			case <-time.After(200 * time.Millisecond):
				break
			}
		}
		
		// Give callback time to be called
		time.Sleep(100 * time.Millisecond)
		assert.True(t, inventoryCallbackCalled, "Inventory receive callback should have been called")
	})

	t.Run("MonitorWithScannerError", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create mock BDS and parameters
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create mock stdin
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Create a reader that will cause scanner error
		reader := &logsErrorReader{}
		
		// Start monitoring - should handle scanner error gracefully
		lm.monitorServerLogs(reader, bds, params, stdinWriter)
	})

	t.Run("MonitorWithFullChannel", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create BDS with full channel
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 1), // Small buffer
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		// Fill the channel
		bds.InventoryUpdate <- InventoryUpdate{
			PlayerName: "Dummy",
			Inventory:  []byte("[]"),
		}
		
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create mock stdin
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Create input with multiple ender chest events
		input := `[X_ENDER_CHEST][Player1][[{"item":"stone"}]]
[X_ENDER_CHEST][Player2][[{"item":"diamond"}]]
`
		reader := strings.NewReader(input)
		
		// Start monitoring - should handle full channel gracefully
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		
		// Give it time to process
		time.Sleep(100 * time.Millisecond)
	})
}

// TestLogMonitor_Integration tests integration scenarios
func TestLogMonitor_Integration(t *testing.T) {
	t.Run("IntegrationWithRealPipes", func(t *testing.T) {
		lm := NewLogMonitor()
		
		// Create mock BDS and parameters
		eventsProcessed := 0
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			InventoryReceiveCallback: func(playerName string) ([]byte, error) {
				eventsProcessed++
				return []byte(`[{"item":"stone"}]`), nil
			},
			StartTrigger: make(chan struct{}, 1),
		}
		
		// Create real pipes
		stdoutReader, stdoutWriter := io.Pipe()
		stderrReader, stderrWriter := io.Pipe()
		stdinReader, stdinWriter := io.Pipe()
		
		// Start monitoring
		cmd := &exec.Cmd{}
		lm.Start(cmd, bds, params, stdoutReader, stderrReader, stdinWriter)
		
		// Send test data through pipes
		go func() {
			stdoutWriter.Write([]byte("Player Spawned: IntegrationPlayer\n"))
			stderrWriter.Write([]byte("[X_ENDER_CHEST][IntegrationPlayer][[{\"item\":\"emerald\"}]]\n"))
			
			// Close pipes after sending data
			time.Sleep(100 * time.Millisecond)
			stdoutWriter.Close()
			stderrWriter.Close()
			stdinWriter.Close()
		}()
		
		// Wait for events to be processed
		time.Sleep(200 * time.Millisecond)
		
		// Clean up
		stdinReader.Close()
		stdoutReader.Close()
		stderrReader.Close()
		
		// At least one event should be processed
		assert.Greater(t, eventsProcessed, 0, "At least one event should be processed")
	})
}

// TestLogMonitor_EdgeCases tests edge cases and error conditions
func TestLogMonitor_EdgeCases(t *testing.T) {
	t.Run("EmptyPlayerName", func(t *testing.T) {
		lm := NewLogMonitor()
		
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Test with empty player name in ender chest event
		input := "[X_ENDER_CHEST][][[{\"item\":\"stone\"}]]\n"
		reader := strings.NewReader(input)
		
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("MalformedEnderChestEvent", func(t *testing.T) {
		lm := NewLogMonitor()
		
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			StartTrigger: make(chan struct{}, 1),
		}
		
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Test with malformed ender chest event
		input := "[X_ENDER_CHEST][Player1]\n" // Missing inventory data
		reader := strings.NewReader(input)
		
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("InventoryCallbackError", func(t *testing.T) {
		lm := NewLogMonitor()
		
		bds := &Bds{
			InventoryUpdate: make(chan InventoryUpdate, 100),
			inventory: NewInventoryManager(
				func(playerName string) ([]byte, error) {
					return []byte(`[{"item":"stone"}]`), nil
				},
				func(playerName string, inventory []byte) error {
					return nil
				},
			),
		}
		
		params := Parameters{
			InventoryReceiveCallback: func(playerName string) ([]byte, error) {
				return nil, assert.AnError // Simulate callback error
			},
			StartTrigger: make(chan struct{}, 1),
		}
		
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()
		
		// Test with player spawned event that triggers callback error
		input := "Player Spawned: ErrorPlayer\n"
		reader := strings.NewReader(input)
		
		go lm.monitorServerLogs(reader, bds, params, stdinWriter)
		time.Sleep(100 * time.Millisecond)
	})
}

// logsErrorReader implements io.Reader that always returns an error
type logsErrorReader struct{}

func (e *logsErrorReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}
