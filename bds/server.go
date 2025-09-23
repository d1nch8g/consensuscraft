package bds

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Server manages the bedrock server process
type Server struct {
	serverPath string
	config     *Config
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer creates a new server manager
func NewServer(serverPath string, config *Config, ctx context.Context, cancel context.CancelFunc) *Server {
	return &Server{
		serverPath: serverPath,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
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

	// Schedule gamerule command 10 seconds after startup
	go s.scheduleGameruleCommand(serverProcess)

	return serverProcess, nil
}

// Stop stops the server process gracefully
func (s *Server) Stop(serverProcess *exec.Cmd) {
	if serverProcess == nil || serverProcess.Process == nil {
		return
	}

	log.Println("BDS: Stopping server process")
	
	// Try to send interrupt signal first
	if err := serverProcess.Process.Signal(os.Interrupt); err != nil {
		log.Printf("BDS: Failed to send interrupt signal: %v", err)
		
		// If interrupt fails, try to kill the process
		if killErr := serverProcess.Process.Kill(); killErr != nil {
			log.Printf("BDS: Failed to kill server process: %v", killErr)
		}
	}
}

// scheduleGameruleCommand sends the gamerule showcoordinates command after startup
func (s *Server) scheduleGameruleCommand(serverProcess *exec.Cmd) {
	log.Println("BDS: Scheduling gamerule showcoordinates command for 10 seconds after startup")
	
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(10 * time.Second):
		// Since we're piping directly to os.Stdin, we need to write to the process stdin
		// But since we set it to os.Stdin, we can't write to it programmatically
		// We'll need to modify this approach
		log.Println("BDS: Note - gamerule command should be sent manually or through a different mechanism")
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

// scheduleGameruleCommandWithPipe sends the gamerule command through the stdin pipe
func (s *Server) scheduleGameruleCommandWithPipe(stdin io.WriteCloser) {
	log.Println("BDS: Scheduling gamerule showcoordinates command for 10 seconds after startup")
	
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(10 * time.Second):
		command := "gamerule showcoordinates true\n"
		if _, err := stdin.Write([]byte(command)); err != nil {
			log.Printf("BDS: Failed to send gamerule showcoordinates command: %v", err)
		} else {
			log.Println("BDS: Successfully sent gamerule showcoordinates true command")
		}
	}
}
