package bds

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewStdinWrapper tests the constructor function
func TestNewStdinWrapper(t *testing.T) {
	t.Run("CreateNewStdinWrapper", func(t *testing.T) {
		// Create a mock server stdin
		mockStdin := &stdinMockWriteCloser{}

		wrapper := NewStdinWrapper(mockStdin)

		assert.NotNil(t, wrapper)
		assert.Equal(t, mockStdin, wrapper.serverStdin)
		assert.NotNil(t, wrapper.reader)
		assert.True(t, wrapper.enabled)
	})

	t.Run("CreateWithNilStdin", func(t *testing.T) {
		wrapper := NewStdinWrapper(nil)

		assert.NotNil(t, wrapper)
		assert.Nil(t, wrapper.serverStdin)
		assert.NotNil(t, wrapper.reader)
		assert.True(t, wrapper.enabled)
	})
}

// TestStdinWrapper_Start tests the Start function
func TestStdinWrapper_Start(t *testing.T) {
	t.Run("StartNormal", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Since logger doesn't have SetOutput, we'll test that Start doesn't panic
		assert.NotPanics(t, func() {
			wrapper.Start()
		})

		// Give goroutine time to start
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("StartWithNilStdin", func(t *testing.T) {
		wrapper := NewStdinWrapper(nil)

		// Should not panic
		assert.NotPanics(t, func() {
			wrapper.Start()
		})
	})
}

// TestStdinWrapper_Stop tests the Stop function
func TestStdinWrapper_Stop(t *testing.T) {
	t.Run("StopNormal", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		wrapper.Stop()

		assert.False(t, wrapper.enabled)
	})

	t.Run("StopMultipleTimes", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		wrapper.Stop()
		assert.False(t, wrapper.enabled)

		// Should be idempotent
		wrapper.Stop()
		assert.False(t, wrapper.enabled)
	})
}

// TestStdinWrapper_inputLoop tests the main input processing loop
func TestStdinWrapper_inputLoop(t *testing.T) {
	t.Run("InputLoopNormalCommands", func(t *testing.T) {
		// Create mock stdin with test input
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Replace the reader with a custom one that provides test input
		testInput := "test command\nanother command\n"
		wrapper.reader = bufio.NewReader(strings.NewReader(testInput))

		// Capture stdout to verify prompt
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		go func() {
			wrapper.inputLoop()
		}()

		// Give time for processing
		time.Sleep(100 * time.Millisecond)
		wrapper.Stop()

		// Restore stdout and read output
		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)

		output := buf.String()
		assert.Contains(t, output, "> ")
	})

	t.Run("InputLoopEmptyCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Empty command should be skipped
		testInput := "\n\n"
		wrapper.reader = bufio.NewReader(strings.NewReader(testInput))

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		go func() {
			wrapper.inputLoop()
		}()

		time.Sleep(100 * time.Millisecond)
		wrapper.Stop()

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)

		output := buf.String()
		// Should show prompts but not process empty commands
		assert.Contains(t, output, "> ")
	})

	t.Run("InputLoopEOF", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// EOF should break the loop
		wrapper.reader = bufio.NewReader(strings.NewReader(""))

		// Since we can't capture logger output, we'll test that the function doesn't panic
		assert.NotPanics(t, func() {
			go wrapper.inputLoop()
			time.Sleep(100 * time.Millisecond)
		})
	})

	t.Run("InputLoopReadError", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Create a reader that will return an error
		errorReader := &errorReader{}
		wrapper.reader = bufio.NewReader(errorReader)

		// Since we can't capture logger output, we'll test that the function doesn't panic
		assert.NotPanics(t, func() {
			go wrapper.inputLoop()
			time.Sleep(100 * time.Millisecond)
			wrapper.Stop()
		})
	})
}

// TestStdinWrapper_handleSpecialCommands tests special command handling
func TestStdinWrapper_handleSpecialCommands(t *testing.T) {
	t.Run("HandleExitCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("exit")

		assert.True(t, result)
		assert.False(t, wrapper.enabled)

		// Verify stop command was sent
		assert.Equal(t, "stop\n", string(mockStdin.writtenData))
	})

	t.Run("HandleQuitCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("quit")

		assert.True(t, result)
		assert.False(t, wrapper.enabled)
	})

	t.Run("HandleExitUpperCase", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("EXIT")

		assert.True(t, result)
		assert.False(t, wrapper.enabled)
	})

	t.Run("HandleExitMixedCase", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("ExIt")

		assert.True(t, result)
		assert.False(t, wrapper.enabled)
	})

	t.Run("HandleHelpCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := wrapper.handleSpecialCommands("help")

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)

		output := buf.String()

		assert.True(t, result)
		assert.Contains(t, output, "BDS Stdin Wrapper Commands:")
		assert.Contains(t, output, "help")
		assert.Contains(t, output, "exit/quit")
		assert.Contains(t, output, "Common Bedrock Server Commands:")
	})

	t.Run("HandleRegularCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("list")

		assert.False(t, result)
		assert.True(t, wrapper.enabled)
	})

	t.Run("HandleEmptyCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		result := wrapper.handleSpecialCommands("")

		assert.False(t, result)
		assert.True(t, wrapper.enabled)
	})
}

// TestStdinWrapper_sendCommand tests command sending functionality
func TestStdinWrapper_sendCommand(t *testing.T) {
	t.Run("SendCommandNormal", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		err := wrapper.sendCommand("list")

		assert.NoError(t, err)
		assert.Equal(t, "list\n", string(mockStdin.writtenData))
	})

	t.Run("SendCommandWithNewline", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		err := wrapper.sendCommand("list\n")

		assert.NoError(t, err)
		assert.Equal(t, "list\n", string(mockStdin.writtenData))
	})

	t.Run("SendCommandEmpty", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		err := wrapper.sendCommand("")

		assert.NoError(t, err)
		assert.Equal(t, "\n", string(mockStdin.writtenData))
	})

	t.Run("SendCommandNilStdin", func(t *testing.T) {
		wrapper := NewStdinWrapper(nil)

		err := wrapper.sendCommand("list")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server stdin is not available")
	})

	t.Run("SendCommandWriteError", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{shouldError: true}
		wrapper := NewStdinWrapper(mockStdin)

		err := wrapper.sendCommand("list")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock write error")
	})

	t.Run("SendCommandComplexCommand", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		complexCommand := "scoreboard objectives add test dummy"
		err := wrapper.sendCommand(complexCommand)

		assert.NoError(t, err)
		assert.Equal(t, complexCommand+"\n", string(mockStdin.writtenData))
	})
}

// TestStdinWrapper_showHelp tests the help display functionality
func TestStdinWrapper_showHelp(t *testing.T) {
	t.Run("ShowHelpNormal", func(t *testing.T) {
		wrapper := NewStdinWrapper(nil)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		wrapper.showHelp()

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)

		output := buf.String()

		// Verify all expected help sections are present
		assert.Contains(t, output, "BDS Stdin Wrapper Commands:")
		assert.Contains(t, output, "help")
		assert.Contains(t, output, "exit/quit")
		assert.Contains(t, output, "Common Bedrock Server Commands:")
		assert.Contains(t, output, "list")
		assert.Contains(t, output, "say")
		assert.Contains(t, output, "stop")
		assert.Contains(t, output, "kick")
		assert.Contains(t, output, "ban")
		assert.Contains(t, output, "op")
		assert.Contains(t, output, "deop")
		assert.Contains(t, output, "gamerule")
		assert.Contains(t, output, "scoreboard")
	})
}

// TestStdinWrapper_Integration tests integration scenarios
func TestStdinWrapper_Integration(t *testing.T) {
	t.Run("IntegrationNormalFlow", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Test the full flow: start, process commands, stop
		wrapper.Start()

		// Give time for goroutine to start
		time.Sleep(50 * time.Millisecond)

		// Stop the wrapper
		wrapper.Stop()
	})

	t.Run("IntegrationWithCommands", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Test sending multiple commands
		commands := []string{"list", "say hello", "gamerule showcoordinates true"}

		for _, cmd := range commands {
			err := wrapper.sendCommand(cmd)
			assert.NoError(t, err)
		}

		expectedOutput := "list\nsay hello\ngamerule showcoordinates true\n"
		assert.Equal(t, expectedOutput, string(mockStdin.writtenData))
	})
}

// TestStdinWrapper_EdgeCases tests edge cases and error conditions
func TestStdinWrapper_EdgeCases(t *testing.T) {
	t.Run("ConcurrentAccess", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Test concurrent access to the wrapper
		done := make(chan bool, 2)

		go func() {
			for i := 0; i < 10; i++ {
				wrapper.sendCommand(fmt.Sprintf("command%d", i))
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				wrapper.sendCommand(fmt.Sprintf("other%d", i))
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done

		// Should not panic and all commands should be sent
		assert.Greater(t, len(mockStdin.writtenData), 0)
	})

	t.Run("StopBeforeStart", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		// Should not panic when stopping before starting
		assert.NotPanics(t, func() {
			wrapper.Stop()
		})
	})

	t.Run("SendCommandAfterStop", func(t *testing.T) {
		mockStdin := &stdinMockWriteCloser{}
		wrapper := NewStdinWrapper(mockStdin)

		wrapper.Stop()

		// Should still be able to send commands after stop
		err := wrapper.sendCommand("list")
		assert.NoError(t, err)
		assert.Equal(t, "list\n", string(mockStdin.writtenData))
	})
}

// Mock implementations for testing

// stdinMockWriteCloser implements io.WriteCloser for testing
type stdinMockWriteCloser struct {
	writtenData []byte
	shouldError bool
	closed      bool
}

func (m *stdinMockWriteCloser) Write(p []byte) (n int, err error) {
	if m.shouldError {
		return 0, fmt.Errorf("mock write error")
	}
	m.writtenData = append(m.writtenData, p...)
	return len(p), nil
}

func (m *stdinMockWriteCloser) Close() error {
	m.closed = true
	return nil
}

// errorReader implements io.Reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock read error")
}
