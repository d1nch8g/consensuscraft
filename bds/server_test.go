package bds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServer tests the constructor function
func TestNewServer(t *testing.T) {
	t.Run("CreateNewServer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "test_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		assert.NotNil(t, server)
		assert.Equal(t, serverPath, server.serverPath)
		assert.Equal(t, config, server.config)
		assert.Equal(t, ctx, server.ctx)
		assert.NotNil(t, server.cancel) // Can't compare cancel functions directly
		assert.Equal(t, webAddress, server.webAddress)
	})

	t.Run("CreateNewServerWithEmptyWebAddress", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "test_server"

		server := NewServer(serverPath, config, ctx, cancel, "")

		assert.NotNil(t, server)
		assert.Equal(t, serverPath, server.serverPath)
		assert.Equal(t, config, server.config)
		assert.Equal(t, ctx, server.ctx)
		assert.NotNil(t, server.cancel) // Can't compare cancel functions directly
		assert.Equal(t, "", server.webAddress)
	})
}

// TestServer_Start tests the server start functionality
func TestServer_Start(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("StartWithMockServer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable
		err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'mock server started'\nsleep 1"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, err := server.Start()
		assert.NoError(t, err)
		assert.NotNil(t, process)

		// Clean up
		server.Stop(process)
	})

	t.Run("StartWithNonExistentServer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "non_existent_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, err := server.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start server process")
		assert.Nil(t, process)
	})

	t.Run("StartWithRelativePath", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "./mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable
		err := os.WriteFile("mock_server", []byte("#!/bin/bash\necho 'mock server'"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, err := server.Start()
		assert.NoError(t, err)
		assert.NotNil(t, process)

		// Clean up
		server.Stop(process)
	})
}

// TestServer_Stop tests the server stop functionality
func TestServer_Stop(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("StopNilProcess", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		// Should not panic when stopping nil process
		assert.NotPanics(t, func() {
			server.Stop(nil)
		})
	})

	t.Run("StopProcessWithNilProcess", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		// Create a command but don't start it
		cmd := exec.Command("echo", "test")

		// Should not panic when stopping process with nil Process
		assert.NotPanics(t, func() {
			server.Stop(cmd)
		})
	})
}

// TestServer_StartWithPipes tests the server start with pipes functionality
func TestServer_StartWithPipes(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("StartWithPipesSuccess", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable
		err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'mock server with pipes'\nsleep 1"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, stdin, stdout, stderr, err := server.StartWithPipes()
		assert.NoError(t, err)
		assert.NotNil(t, process)
		assert.NotNil(t, stdin)
		assert.NotNil(t, stdout)
		assert.NotNil(t, stderr)

		// Clean up
		stdin.Close()
		stdout.Close()
		stderr.Close()
		server.Stop(process)
	})

	t.Run("StartWithPipesNonExistentServer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "non_existent_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, stdin, stdout, stderr, err := server.StartWithPipes()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start server process")
		assert.Nil(t, process)
		assert.Nil(t, stdin)
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
	})
}

// TestServer_ScheduleGameruleCommandWithPipe tests the gamerule command scheduling with pipes
func TestServer_ScheduleGameruleCommandWithPipe(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("ScheduleGameruleCommandWithPipeSuccess", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)
		server.scheduleDelay = 100 * time.Millisecond // Fast for tests

		// Create a mock stdin pipe
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()

		// Start the scheduling in a goroutine
		go server.scheduleGameruleCommandWithPipe(stdinWriter)

		// Read from the pipe to capture commands
		buf := make([]byte, 1024)
		go func() {
			time.Sleep(200 * time.Millisecond) // Wait for commands to be sent (100ms delay + buffer)
			stdinReader.Read(buf)
		}()

		// Give it a moment to start scheduling
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("ScheduleGameruleCommandWithPipeContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)
		server.scheduleDelay = 100 * time.Millisecond // Fast for tests

		// Create a mock stdin pipe
		stdinReader, stdinWriter := io.Pipe()
		defer stdinReader.Close()
		defer stdinWriter.Close()

		// Cancel context immediately
		cancel()

		// This should return immediately due to context cancellation
		server.scheduleGameruleCommandWithPipe(stdinWriter)
	})

	t.Run("ScheduleGameruleCommandWithPipeWebAddress", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		server := NewServer(serverPath, config, ctx, cancel, webAddress)
		server.scheduleDelay = 100 * time.Millisecond // Longer delay for reliable testing

		// Create a mock stdin pipe that captures written data
		var capturedData bytes.Buffer
		mockStdin := &mockWriteCloser{writer: &capturedData}

		// Start the scheduling
		go server.scheduleGameruleCommandWithPipe(mockStdin)

		// Wait for all commands to be processed (gamerule + 100ms + scoreboard + 50ms + server name)
		time.Sleep(500 * time.Millisecond) // 100ms delay + 100ms + 50ms + buffer for all commands

		// Verify the commands were sent
		output := capturedData.String()
		assert.Contains(t, output, "gamerule showcoordinates true")
		assert.Contains(t, output, "scoreboard objectives add serverName dummy")
		assert.Contains(t, output, "scoreboard players set \"test-server.example.com\" serverName 1")
	})

	t.Run("ScheduleGameruleCommandWithPipeEmptyWebAddress", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "" // Empty web address

		server := NewServer(serverPath, config, ctx, cancel, webAddress)
		server.scheduleDelay = 100 * time.Millisecond // Longer delay for reliable testing

		// Create a mock stdin pipe that captures written data
		var capturedData bytes.Buffer
		mockStdin := &mockWriteCloser{writer: &capturedData}

		// Start the scheduling
		go server.scheduleGameruleCommandWithPipe(mockStdin)

		// Wait for all commands to be processed (gamerule + 100ms + scoreboard + 50ms + server name)
		time.Sleep(500 * time.Millisecond) // 100ms delay + 100ms + 50ms + buffer for all commands

		// Verify the commands were sent with default server name
		output := capturedData.String()
		assert.Contains(t, output, "gamerule showcoordinates true")
		assert.Contains(t, output, "scoreboard objectives add serverName dummy")
		assert.Contains(t, output, "scoreboard players set \"unknown-server\" serverName 1")
	})
}

// TestServer_Integration tests integration scenarios
func TestServer_Integration(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("IntegrationStartStopCycle", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable that exits immediately
		err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'server started'\nexit 0"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		// Test regular start
		process, err := server.Start()
		assert.NoError(t, err)
		assert.NotNil(t, process)

		// Wait a bit for process to start and exit
		time.Sleep(100 * time.Millisecond)

		// Stop should handle already exited process gracefully
		assert.NotPanics(t, func() {
			server.Stop(process)
		})
	})

	t.Run("IntegrationStartWithPipesCycle", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable that runs briefly
		err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'server with pipes started'\nsleep 0.5\necho 'server exiting'"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		// Test start with pipes
		process, stdin, stdout, stderr, err := server.StartWithPipes()
		assert.NoError(t, err)
		assert.NotNil(t, process)
		assert.NotNil(t, stdin)
		assert.NotNil(t, stdout)
		assert.NotNil(t, stderr)

		// Clean up resources
		stdin.Close()
		stdout.Close()
		stderr.Close()

		// Wait for process to complete
		time.Sleep(600 * time.Millisecond)

		// Stop should handle completed process gracefully
		assert.NotPanics(t, func() {
			server.Stop(process)
		})
	})
}

// TestServer_EdgeCases tests edge cases and error conditions
func TestServer_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("StartWithInvalidWorkingDirectory", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "subdir/mock_server"
		webAddress := "test-server.example.com"

		// Create subdirectory and server
		err := os.MkdirAll("subdir", 0755)
		require.NoError(t, err)
		err = os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'server'"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		process, err := server.Start()
		assert.NoError(t, err)
		assert.NotNil(t, process)

		// Clean up
		server.Stop(process)
	})

	t.Run("StartWithPipesWriteError", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &Config{}
		serverPath := "mock_server"
		webAddress := "test-server.example.com"

		// Create a mock server executable
		err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'server'"), 0755)
		require.NoError(t, err)

		server := NewServer(serverPath, config, ctx, cancel, webAddress)

		// Test that scheduling doesn't panic even if stdin write fails
		// We can't easily simulate write failure in this test, but we can verify
		// the function handles the case gracefully
		process, stdin, stdout, stderr, err := server.StartWithPipes()
		assert.NoError(t, err)

		// Close stdin immediately to simulate write failure
		stdin.Close()

		// Give scheduling a moment to start
		time.Sleep(100 * time.Millisecond)

		// Clean up
		stdout.Close()
		stderr.Close()
		server.Stop(process)
	})
}

// mockWriteCloser implements io.WriteCloser for testing
type mockWriteCloser struct {
	writer io.Writer
	closed bool
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, fmt.Errorf("mock write closer is closed")
	}
	return m.writer.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}
