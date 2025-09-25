package bds

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/d1nch8g/consensuscraft/logger"
)

// StdinWrapper handles interactive stdin input for the bedrock server
type StdinWrapper struct {
	serverStdin io.WriteCloser
	reader      *bufio.Reader
	enabled     bool
}

// NewStdinWrapper creates a new stdin wrapper
func NewStdinWrapper(serverStdin io.WriteCloser) *StdinWrapper {
	return &StdinWrapper{
		serverStdin: serverStdin,
		reader:      bufio.NewReader(os.Stdin),
		enabled:     true,
	}
}

// Start begins the stdin wrapper loop
func (sw *StdinWrapper) Start() {
	logger.Println("Starting stdin wrapper - type commands and press Enter to send to server")
	logger.Println("Type 'exit' or 'quit' to stop the server")
	
	go sw.inputLoop()
}

// Stop disables the stdin wrapper
func (sw *StdinWrapper) Stop() {
	sw.enabled = false
	logger.Println("Stdin wrapper stopped")
}

// inputLoop handles the main input processing loop
func (sw *StdinWrapper) inputLoop() {
	for sw.enabled {
		// Print prompt
		fmt.Print("> ")
		
		// Read line from stdin
		input, err := sw.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Println("EOF received, stopping stdin wrapper")
				break
			}
			logger.Printf("Error reading from stdin: %v", err)
			continue
		}
		
		// Trim whitespace
		command := strings.TrimSpace(input)
		
		// Skip empty commands
		if command == "" {
			continue
		}
		
		// Handle special commands
		if sw.handleSpecialCommands(command) {
			continue
		}
		
		// Send command to server
		if err := sw.sendCommand(command); err != nil {
			logger.Printf("Failed to send command to server: %v", err)
		} else {
			logger.Printf("Sent command: %s", command)
		}
	}
}

// handleSpecialCommands processes special wrapper commands
func (sw *StdinWrapper) handleSpecialCommands(command string) bool {
	switch strings.ToLower(command) {
	case "exit", "quit":
		logger.Println("Exit command received, stopping server...")
		sw.enabled = false
		// Send stop command to server
		sw.sendCommand("stop")
		return true
	case "help":
		sw.showHelp()
		return true
	default:
		return false
	}
}

// sendCommand sends a command to the bedrock server
func (sw *StdinWrapper) sendCommand(command string) error {
	if sw.serverStdin == nil {
		return fmt.Errorf("server stdin is not available")
	}
	
	// Add newline if not present
	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}
	
	// Write command to server stdin
	_, err := sw.serverStdin.Write([]byte(command))
	return err
}

// showHelp displays help information
func (sw *StdinWrapper) showHelp() {
	fmt.Println("BDS Stdin Wrapper Commands:")
	fmt.Println("  help          - Show this help message")
	fmt.Println("  exit/quit     - Stop the server and exit")
	fmt.Println("  <any command> - Send command directly to bedrock server")
	fmt.Println("")
	fmt.Println("Common Bedrock Server Commands:")
	fmt.Println("  list          - List connected players")
	fmt.Println("  say <message> - Send message to all players")
	fmt.Println("  stop          - Stop the server")
	fmt.Println("  kick <player> - Kick a player")
	fmt.Println("  ban <player>  - Ban a player")
	fmt.Println("  op <player>   - Give operator privileges")
	fmt.Println("  deop <player> - Remove operator privileges")
	fmt.Println("  gamerule <rule> <value> - Set game rule")
	fmt.Println("  scoreboard <args> - Manage scoreboards")
}
