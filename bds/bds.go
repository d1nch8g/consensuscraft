package bds

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/d1nch8g/consensuscraft/logger"
)

// InventoryReceiveCallback defines the callback function type for inventory operations
type InventoryReceiveCallback func(playerName string) ([]byte, error)
type InventoryUpdateCallback func(playerName string, inventory []byte) error

// InventoryUpdate represents an inventory update event
type InventoryUpdate struct {
	PlayerName string
	Inventory  []byte
	Server     string
}

// Parameters defines the configuration parameters for the BDS
type Parameters struct {
	InventoryReceiveCallback InventoryReceiveCallback
	InventoryUpdateCallback  InventoryUpdateCallback
	StartTrigger             chan struct{}
	WebAddress               string // Server web address for origin tracking
}

// Bds represents the Bedrock Dedicated Server instance
type Bds struct {
	// Public channel for inventory updates
	InventoryUpdate chan InventoryUpdate

	// Public channels for player events
	PlayerLogin  chan string
	PlayerLogout chan string

	// Internal components
	server      *Server
	config      *Config
	inventory   *InventoryManager
	logs        *LogMonitor
	stdinWrapper *StdinWrapper
}

// New creates a new Bedrock Dedicated Server instance and starts the management loop
func New(params Parameters) (*Bds, error) {
	if params.InventoryReceiveCallback == nil {
		return nil, fmt.Errorf("inventory callback cannot be nil")
	}

	if params.StartTrigger == nil {
		return nil, fmt.Errorf("start trigger channel cannot be nil")
	}

	// Load configuration from .env file
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup server based on current directory state
	setup := NewSetup()
	serverPath, err := setup.EnsureServer()
	if err != nil {
		return nil, fmt.Errorf("failed to setup server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bds := &Bds{
		InventoryUpdate: make(chan InventoryUpdate, 100),
		PlayerLogin:     make(chan string, 100),
		PlayerLogout:    make(chan string, 100),
		config:          config,
		inventory: NewInventoryManager(
			params.InventoryReceiveCallback,
			params.InventoryUpdateCallback,
		),
		logs: NewLogMonitor(),
	}

	// Create server manager with WebAddress for origin tracking
	bds.server = NewServer(serverPath, config, ctx, cancel, params.WebAddress)

	// Start the management loop in a goroutine
	go func() {
		defer cancel()
		defer close(bds.InventoryUpdate)
		defer close(bds.PlayerLogin)
		defer close(bds.PlayerLogout)

		var serverProcess *exec.Cmd

		logger.Println("Starting management loop")

		for {
			select {
			case <-ctx.Done():
				logger.Println("Context cancelled, shutting down")
				if bds.stdinWrapper != nil {
					bds.stdinWrapper.Stop()
					bds.stdinWrapper = nil
				}
				if serverProcess != nil {
					bds.server.Stop(serverProcess)
				}
				logger.Println("Shutdown complete")
				return

			case <-params.StartTrigger:
				if serverProcess != nil {
					logger.Println("Server is already running")
					continue
				}

				logger.Println("Starting Bedrock Dedicated Server")

				// For requirement #5 (pipe stdin/stdout/stderr), we use StartWithPipes
				// to enable both direct I/O piping AND log parsing for player events
				var stdin io.WriteCloser
				var stdout, stderr io.ReadCloser

				serverProcess, stdin, stdout, stderr, err = bds.server.StartWithPipes()
				if err != nil {
					logger.Printf("Failed to start server: %v", err)
					serverProcess = nil
					continue
				}

				logger.Printf("Server started with PID %d", serverProcess.Process.Pid)

				// Start log monitoring with pipes that also output to stdout/stderr
				bds.logs.StartWithPipes(stdout, stderr, stdin, bds, params)

				// Start stdin wrapper for interactive command input
				bds.stdinWrapper = NewStdinWrapper(stdin)
				bds.stdinWrapper.Start()

				// Monitor server process in a separate goroutine
				go func(proc *exec.Cmd) {
					err := proc.Wait()
					serverProcess = nil

					// Stop stdin wrapper when server exits
					if bds.stdinWrapper != nil {
						bds.stdinWrapper.Stop()
						bds.stdinWrapper = nil
					}

					if err != nil {
						logger.Printf("Server process exited unexpectedly: %v", err)
					} else {
						logger.Println("Server process exited")
					}
				}(serverProcess)
			}
		}
	}()

	// Send initial start trigger
	select {
	case params.StartTrigger <- struct{}{}:
		logger.Println("Initial start trigger sent")
	default:
		logger.Println("Start trigger channel full")
	}

	return bds, nil
}
