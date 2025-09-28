package bds

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/d1nch8g/consensuscraft/logger"
)

// OutputParser handles server log monitoring, parsing, and inventory operations
type OutputParser struct {
	// Compiled regex patterns for log parsing
	playerSpawnedRegex *regexp.Regexp
	enderChestRegex    *regexp.Regexp

	// Inventory callbacks
	receiveCallback InventoryReceiveCallback
	updateCallback  InventoryUpdateCallback
}

// NewOutputParser creates a new output parser
func NewOutputParser(rc InventoryReceiveCallback, uc InventoryUpdateCallback) *OutputParser {
	return &OutputParser{
		playerSpawnedRegex: regexp.MustCompile(`Player Spawned: ([^,\s]+)`),
		enderChestRegex:    regexp.MustCompile(`\[X_ENDER_CHEST\]\[([^\]]+)\]\[(.+)\]`),
		receiveCallback:    rc,
		updateCallback:     uc,
	}
}

// Start starts monitoring server logs with flexible I/O handling
// It can handle both direct I/O piping and separate pipes for parsing
func (op *OutputParser) Start(serverProcess *exec.Cmd, bds *Bds, params Parameters, pipes ...interface{}) {
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
	go op.monitorServerLogs(stdout, bds, params, stdin)
	go op.monitorServerLogs(stderr, bds, params, stdin)
}

// monitorServerLogs monitors server output and processes events
func (op *OutputParser) monitorServerLogs(reader io.Reader, bds *Bds, params Parameters, stdin io.WriteCloser) {
	// Create a TeeReader to duplicate output to stdout while parsing
	teeReader := io.TeeReader(reader, os.Stdout)
	scanner := bufio.NewScanner(teeReader)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse player spawned events - trigger inventory restoration
		if matches := op.playerSpawnedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			logger.Printf("Player spawned: %s", playerName)

			// Get inventory data from callback and restore it via tags
			go func(name string) {
				if inventoryData, err := params.InventoryReceiveCallback(name); err == nil {
					if err := op.restorePlayerInventory(name, inventoryData, stdin); err != nil {
						logger.Printf("Failed to restore inventory for %s: %v", name, err)
					}
				} else {
					logger.Printf("Failed to get inventory data for %s: %v", name, err)
				}
			}(playerName)
		}

		// Parse ender chest inventory updates
		if matches := op.enderChestRegex.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			inventoryData := matches[2]

			logger.Printf("Inventory update for %s", playerName)

			// The inventory data is already a valid JSON array from JavaScript
			// Don't wrap it in additional brackets
			jsonInventoryData := inventoryData

			op.updatePlayerInventory(playerName, []byte(jsonInventoryData))

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

// restorePlayerInventory restores a player's inventory using server commands
func (op *OutputParser) restorePlayerInventory(playerName string, inventoryData []byte, stdin io.WriteCloser) error {
	if len(inventoryData) == 0 {
		return nil // No inventory to restore
	}

	// Convert inventory data to string
	inventoryStr := string(inventoryData)

	// Chunk the inventory data for player tags (max 1500 chars per tag)
	const maxChunkSize = 1500
	chunks := []string{}

	for i := 0; i < len(inventoryStr); i += maxChunkSize {
		end := min(i+maxChunkSize, len(inventoryStr))
		chunks = append(chunks, inventoryStr[i:end])
	}

	// Send commands to add inventory tags
	for i, chunk := range chunks {
		// Escape quotes in the chunk
		escapedChunk := strings.ReplaceAll(chunk, `"`, `\"`)

		// Create the tag command
		tagCommand := fmt.Sprintf(`tag "%s" add "restore_inv_%d_%s"`+"\n", playerName, i, escapedChunk)

		// Send command to server
		if _, err := stdin.Write([]byte(tagCommand)); err != nil {
			return fmt.Errorf("failed to send tag command: %w", err)
		}

		logger.Printf("Added inventory tag %d for player %s", i, playerName)
	}

	return nil
}

func (op *OutputParser) updatePlayerInventory(playerName string, inventoryData []byte) error {
	if op.updateCallback != nil {
		return op.updateCallback(playerName, inventoryData)
	}
	return nil
}
