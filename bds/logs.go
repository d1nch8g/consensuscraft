package bds

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// LogMonitor handles server log monitoring and parsing
type LogMonitor struct {
	// Compiled regex patterns for log parsing
	playerConnectedRegex    *regexp.Regexp
	playerSpawnedRegex      *regexp.Regexp
	playerDisconnectedRegex *regexp.Regexp
	enderChestRegex         *regexp.Regexp
}

// NewLogMonitor creates a new log monitor
func NewLogMonitor() *LogMonitor {
	return &LogMonitor{
		playerConnectedRegex:    regexp.MustCompile(`Player connected: ([^,]+),`),
		playerSpawnedRegex:      regexp.MustCompile(`Player Spawned: ([^,\s]+)`),
		playerDisconnectedRegex: regexp.MustCompile(`Player disconnected: ([^,]+),`),
		enderChestRegex:         regexp.MustCompile(`\[X_ENDER_CHEST\]\[([^\]]+)\]\[(.+)\]`),
	}
}

// Start starts monitoring server logs with direct I/O piping
func (lm *LogMonitor) Start(serverProcess *exec.Cmd, bds *Bds, params Parameters) {
	// Since we're using direct I/O piping to os.Stdout/Stderr,
	// we can't intercept the logs for parsing player events.
	// This is a trade-off for requirement #5 (pipe to process stdin/stdout/stderr)

	log.Println("BDS: Log monitoring started with direct I/O piping")
	log.Println("BDS: Note - Player events cannot be parsed with direct I/O piping")
}

// StartWithPipes starts monitoring server logs with separate pipes for parsing
func (lm *LogMonitor) StartWithPipes(stdout, stderr io.ReadCloser, stdin io.WriteCloser, bds *Bds, params Parameters) {
	// Start monitoring stdout and stderr in separate goroutines
	go lm.monitorServerLogs(stdout, bds, params, stdin)
	go lm.monitorServerLogs(stderr, bds, params, stdin)

	log.Println("BDS: Log monitoring started with separate pipes")
}

// monitorServerLogs monitors server output and processes events
func (lm *LogMonitor) monitorServerLogs(reader io.Reader, bds *Bds, params Parameters, stdin io.WriteCloser) {
	// Create a TeeReader to duplicate output to stdout while parsing
	teeReader := io.TeeReader(reader, os.Stdout)
	scanner := bufio.NewScanner(teeReader)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse player connected events
		if matches := lm.playerConnectedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player connected: %s", playerName)

			select {
			case bds.PlayerLogin <- playerName:
			default:
				log.Printf("BDS: PlayerLogin channel full, dropping event for %s", playerName)
			}
		}

		// Parse player spawned events - trigger inventory restoration
		if matches := lm.playerSpawnedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player spawned: %s", playerName)

			// Get inventory data from callback and restore it via tags
			go func(name string) {
				if inventoryData, err := params.InventoryReceiveCallback(name); err == nil {
					if err := bds.inventory.RestorePlayerInventory(name, inventoryData, stdin); err != nil {
						log.Printf("BDS: Failed to restore inventory for %s: %v", name, err)
					}
				} else {
					log.Printf("BDS: Failed to get inventory data for %s: %v", name, err)
				}
			}(playerName)
		}

		// Parse player disconnected events
		if matches := lm.playerDisconnectedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player disconnected: %s", playerName)

			select {
			case bds.PlayerLogout <- playerName:
			default:
				log.Printf("BDS: PlayerLogout channel full, dropping event for %s", playerName)
			}
		}

		// Parse ender chest inventory updates
		if matches := lm.enderChestRegex.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			inventoryData := matches[2]

			log.Printf("BDS: Inventory update for %s", playerName)

			bds.inventory.UpdatePlayerInventory(playerName, []byte(inventoryData))

			select {
			case bds.InventoryUpdate <- InventoryUpdate{
				PlayerName: playerName,
				Inventory:  []byte(inventoryData),
			}:
			default:
				log.Printf("BDS: InventoryUpdate channel full, dropping event for %s", playerName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("BDS: Error reading server logs: %v", err)
	}
}
