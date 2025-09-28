package bds

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/d1nch8g/consensuscraft/logger"
)

// Server manages the bedrock server process
type Server struct {
	serverPath    string
	config        *Config
	ctx           context.Context
	cancel        context.CancelFunc
	webAddress    string
	scheduleDelay time.Duration // Configurable delay for scheduled commands
}

// NewServer creates a new server manager
func NewServer(serverPath string, config *Config, ctx context.Context, cancel context.CancelFunc, webAddress string) *Server {
	return &Server{
		serverPath:    serverPath,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		webAddress:    webAddress,
		scheduleDelay: 15 * time.Second, // Default 15 seconds for production
	}
}

// Start starts the bedrock server process with proper I/O piping
func (s *Server) Start() (*exec.Cmd, error) {
	// Get absolute path to avoid path issues
	absServerPath, err := filepath.Abs(s.serverPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for server: %w", err)
	}

	// Create server command
	serverProcess := exec.CommandContext(s.ctx, absServerPath)

	// Set working directory
	if filepath.Dir(s.serverPath) != "." {
		serverProcess.Dir = filepath.Dir(s.serverPath)
	}

	// Set environment variables from config
	serverProcess.Env = append(os.Environ(), s.config.GetEnvVars()...)

	// Pipe stdin, stdout, stderr directly to process stdin, stdout, stderr
	serverProcess.Stdin = os.Stdin
	serverProcess.Stdout = os.Stdout
	serverProcess.Stderr = os.Stderr

	// Start the server process
	if err := serverProcess.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server process: %w", err)
	}

	return serverProcess, nil
}

// Stop stops the server process gracefully
func (s *Server) Stop(serverProcess *exec.Cmd) {
	if serverProcess == nil || serverProcess.Process == nil {
		return
	}

	logger.Println("Stopping server process")

	// Try to send interrupt signal first
	if err := serverProcess.Process.Signal(os.Interrupt); err != nil {
		logger.Printf("Failed to send interrupt signal: %v", err)

		// If interrupt fails, try to kill the process
		if killErr := serverProcess.Process.Kill(); killErr != nil {
			logger.Printf("Failed to kill server process: %v", killErr)
		}
	}
}

// StartWithPipes starts the server with separate pipes for monitoring (alternative approach)
func (s *Server) StartWithPipes() (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	// Get absolute path to avoid path issues
	absServerPath, err := filepath.Abs(s.serverPath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get absolute path for server: %w", err)
	}

	// Create server command
	serverProcess := exec.CommandContext(s.ctx, absServerPath)

	// Set working directory
	if filepath.Dir(s.serverPath) != "." {
		serverProcess.Dir = filepath.Dir(s.serverPath)
	}

	// Set environment variables from config
	serverProcess.Env = append(os.Environ(), s.config.GetEnvVars()...)

	// Create pipes for stdin, stdout, stderr
	stdin, err := serverProcess.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := serverProcess.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := serverProcess.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the server process
	if err := serverProcess.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to start server process: %w", err)
	}

	// Schedule gamerule command with access to stdin
	go s.scheduleGameruleCommandWithPipe(stdin)

	return serverProcess, stdin, stdout, stderr, nil
}

// scheduleGameruleCommandWithPipe sends the gamerule and scoreboard commands through the stdin pipe
func (s *Server) scheduleGameruleCommandWithPipe(stdin io.WriteCloser) {
	logger.Printf("Scheduling gamerule showcoordinates and scoreboard commands for %v after startup", s.scheduleDelay)

	select {
	case <-s.ctx.Done():
		return
	case <-time.After(s.scheduleDelay):
		// Send gamerule command
		gameruleCommand := "gamerule showcoordinates true\n"
		if _, err := stdin.Write([]byte(gameruleCommand)); err != nil {
			logger.Printf("Failed to send gamerule showcoordinates command: %v", err)
		} else {
			logger.Println("Successfully sent gamerule showcoordinates true command")
		}

		// Wait a moment before sending scoreboard commands (use shorter delay for tests)
		time.Sleep(100 * time.Millisecond)

		// Send scoreboard setup commands
		scoreboardObjectiveCommand := "scoreboard objectives add serverName dummy\n"
		if _, err := stdin.Write([]byte(scoreboardObjectiveCommand)); err != nil {
			logger.Printf("Failed to send scoreboard objectives command: %v", err)
		} else {
			logger.Println("Successfully sent scoreboard objectives add serverName dummy command")
		}

		// Wait a moment before setting the server name (use shorter delay for tests)
		time.Sleep(50 * time.Millisecond)

		// Set the server name in scoreboard (use WebAddress if available, otherwise use a default)
		serverName := s.webAddress
		if serverName == "" {
			serverName = "unknown-server"
		}

		scoreboardSetCommand := fmt.Sprintf("scoreboard players set \"%s\" serverName 1\n", serverName)
		if _, err := stdin.Write([]byte(scoreboardSetCommand)); err != nil {
			logger.Printf("Failed to send scoreboard players set command: %v", err)
		} else {
			logger.Printf("Successfully set server name in scoreboard: %s", serverName)
		}
	}
}
