package bds

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/d1nch8g/consensuscraft/logger"
)

// LogMonitor handles server log monitoring and parsing
type LogMonitor struct {
	// Compiled regex patterns for log parsing
	playerSpawnedRegex *regexp.Regexp
	enderChestRegex    *regexp.Regexp
}

// NewLogMonitor creates a new log monitor
func NewLogMonitor() *LogMonitor {
	return &LogMonitor{
		playerSpawnedRegex: regexp.MustCompile(`Player Spawned: ([^,\s]+)`),
		enderChestRegex:    regexp.MustCompile(`\[X_ENDER_CHEST\]\[([^\]]+)\]\[(.+)\]`),
	}
}

// Start starts monitoring server logs with flexible I/O handling
// It can handle both direct I/O piping and separate pipes for parsing
func (lm *LogMonitor) Start(serverProcess *exec.Cmd, bds *Bds, params Parameters, pipes ...interface{}) {
	var stdout, stderr io.ReadCloser
	var stdin io.WriteCloser

	if len(pipes) >= 3 {
		// Use provided pipes (stdout, stderr, stdin)
		var ok bool
		if stdout, ok = pipes[0].(io.ReadCloser); !ok {
			logger.Println("Invalid stdout pipe type")
			return
		}
		if stderr, ok = pipes[1].(io.ReadCloser); !ok {
			logger.Println("Invalid stderr pipe type")
			return
		}
		if stdin, ok = pipes[2].(io.WriteCloser); !ok {
			logger.Println("Invalid stdin pipe type")
			return
		}
		logger.Println("Log monitoring started with separate pipes")
	} else {
		// Use direct I/O piping to os.Stdout/Stderr
		// This is a trade-off for requirement #5 (pipe to process stdin/stdout/stderr)
		// Player events cannot be parsed with direct I/O piping
		logger.Println("Log monitoring started with direct I/O piping")
		logger.Println("Note - Player events cannot be parsed with direct I/O piping")
		return
	}

	// Start monitoring stdout and stderr in separate goroutines
	go lm.monitorServerLogs(stdout, bds, params, stdin)
	go lm.monitorServerLogs(stderr, bds, params, stdin)
}

// monitorServerLogs monitors server output and processes events
func (lm *LogMonitor) monitorServerLogs(reader io.Reader, bds *Bds, params Parameters, stdin io.WriteCloser) {
	// Create a TeeReader to duplicate output to stdout while parsing
	teeReader := io.TeeReader(reader, os.Stdout)
	scanner := bufio.NewScanner(teeReader)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse player spawned events - trigger inventory restoration
		if matches := lm.playerSpawnedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			logger.Printf("Player spawned: %s", playerName)

			// Get inventory data from callback and restore it via tags
			go func(name string) {
				if inventoryData, err := params.InventoryReceiveCallback(name); err == nil {
					if err := bds.inventory.RestorePlayerInventory(name, inventoryData, stdin); err != nil {
						logger.Printf("Failed to restore inventory for %s: %v", name, err)
					}
				} else {
					logger.Printf("Failed to get inventory data for %s: %v", name, err)
				}
			}(playerName)
		}

		// Parse ender chest inventory updates
		if matches := lm.enderChestRegex.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			inventoryData := matches[2]

			logger.Printf("Inventory update for %s", playerName)

			// The inventory data is already a valid JSON array from JavaScript
			// Don't wrap it in additional brackets
			jsonInventoryData := inventoryData

			bds.inventory.UpdatePlayerInventory(playerName, []byte(jsonInventoryData))

			select {
			case bds.InventoryUpdate <- InventoryUpdate{
				PlayerName: playerName,
				Inventory:  []byte(jsonInventoryData),
			}:
			default:
				logger.Printf("InventoryUpdate channel full, dropping event for %s", playerName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("Error reading server logs: %v", err)
	}
}
