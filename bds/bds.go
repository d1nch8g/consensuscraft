package bds

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InventoryCallback defines the callback function type for inventory operations
type InventoryCallback func(playerName string) ([]byte, error)

// InventoryUpdate represents an inventory update event
type InventoryUpdate struct {
	PlayerName string
	Inventory  []byte
}

// Parameters defines the configuration parameters for the BDS
type Parameters struct {
	BedrockServerPort int
	BedrockMaxThreads int
	MaxPlayers        int
	PlayerIdleTimeout int
	ViewDistance      int
	InventoryCallback InventoryCallback
	StartTrigger      chan struct{}
}

// Bds represents the Bedrock Dedicated Server instance
type Bds struct {
	// Public channel for inventory updates
	InventoryUpdate chan InventoryUpdate

	// Public channels for player events
	PlayerLogin  chan string
	PlayerLogout chan string
}

// New creates a new Bedrock Dedicated Server instance and starts the management loop
func New(params Parameters) (*Bds, error) {
	if params.InventoryCallback == nil {
		return nil, fmt.Errorf("inventory callback cannot be nil")
	}

	if params.StartTrigger == nil {
		return nil, fmt.Errorf("start trigger channel cannot be nil")
	}

	// Ensure server is properly set up
	if err := ensureServerSetup(); err != nil {
		return nil, fmt.Errorf("failed to setup server: %w", err)
	}

	// Determine server path
	serverPath := filepath.Join("server", "bedrock_server")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("bedrock server executable not found at %s", serverPath)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bds := &Bds{
		InventoryUpdate: make(chan InventoryUpdate, 100),
		PlayerLogin:     make(chan string, 100),
		PlayerLogout:    make(chan string, 100),
	}

	// Start the management loop in a goroutine
	go func() {
		defer cancel()
		defer close(bds.InventoryUpdate)
		defer close(bds.PlayerLogin)
		defer close(bds.PlayerLogout)

		var serverProcess *exec.Cmd

		log.Println("BDS: Starting management loop")

		for {
			select {
			case <-ctx.Done():
				log.Println("BDS: Context cancelled, shutting down")
				if serverProcess != nil {
					log.Println("BDS: Stopping server process")
					if err := serverProcess.Process.Signal(os.Interrupt); err != nil {
						log.Printf("BDS: Failed to send interrupt signal: %v", err)
						if killErr := serverProcess.Process.Kill(); killErr != nil {
							log.Printf("BDS: Failed to kill server process: %v", killErr)
						}
					}
				}
				log.Println("BDS: Shutdown complete")
				return

			case <-params.StartTrigger:
				if serverProcess != nil {
					log.Println("BDS: Server is already running")
					continue
				}

				log.Println("BDS: Starting Bedrock Dedicated Server")

				// Apply configuration parameters to server.properties
				if err := applyServerConfig(params); err != nil {
					log.Printf("BDS: Warning - failed to apply server configuration: %v", err)
				}

				// Prepare server command - use absolute path to avoid path issues
				absServerPath, err := filepath.Abs(serverPath)
				if err != nil {
					log.Printf("BDS: Failed to get absolute path for server: %v", err)
					serverProcess = nil
					continue
				}
				
				serverProcess = exec.CommandContext(ctx, absServerPath)
				serverProcess.Dir = "server"

				// Set up pipes for stdout and stderr to capture logs
				stdout, err := serverProcess.StdoutPipe()
				if err != nil {
					log.Printf("BDS: Failed to create stdout pipe: %v", err)
					serverProcess = nil
					continue
				}

				stderr, err := serverProcess.StderrPipe()
				if err != nil {
					log.Printf("BDS: Failed to create stderr pipe: %v", err)
					serverProcess = nil
					continue
				}

				// Set up stdin pipe for sending commands
				stdin, err := serverProcess.StdinPipe()
				if err != nil {
					log.Printf("BDS: Failed to create stdin pipe: %v", err)
					serverProcess = nil
					continue
				}

				// Start the server process
				if err := serverProcess.Start(); err != nil {
					log.Printf("BDS: Failed to start server process: %v", err)
					serverProcess = nil
					continue
				}

				log.Printf("BDS: Server started with PID %d", serverProcess.Process.Pid)

				// Start log monitoring goroutines
				go monitorServerLogs(stdout, bds, params, stdin)
				go monitorServerLogs(stderr, bds, params, stdin)

				// Execute gamerule command 10 seconds after startup
				go func(stdinPipe io.WriteCloser) {
					log.Println("BDS: Scheduling gamerule showcoordinates command for 10 seconds after startup")
					select {
					case <-ctx.Done():
						return
					case <-time.After(10 * time.Second):
						command := "gamerule showcoordinates true\n"
						if _, err := stdinPipe.Write([]byte(command)); err != nil {
							log.Printf("BDS: Failed to send gamerule showcoordinates command: %v", err)
						} else {
							log.Println("BDS: Successfully sent gamerule showcoordinates true command")
						}
					}
				}(stdin)

				// Monitor server process in a separate goroutine
				go func(proc *exec.Cmd) {
					err := proc.Wait()
					serverProcess = nil

					if err != nil {
						log.Printf("BDS: Server process exited unexpectedly: %v", err)
					} else {
						log.Println("BDS: Server process exited")
					}
				}(serverProcess)
			}
		}
	}()

	// Send initial start trigger
	select {
	case params.StartTrigger <- struct{}{}:
		log.Println("BDS: Initial start trigger sent")
	default:
		log.Println("BDS: Start trigger channel full")
	}

	return bds, nil
}

// applyServerConfig applies the configuration parameters to server.properties
func applyServerConfig(params Parameters) error {
	propertiesPath := filepath.Join("server", "server.properties")

	// Read existing properties
	properties := make(map[string]string)
	if file, err := os.Open(propertiesPath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				properties[parts[0]] = parts[1]
			}
		}
	}

	// Apply configuration parameters
	properties["server-port"] = strconv.Itoa(params.BedrockServerPort)
	properties["max-players"] = strconv.Itoa(params.MaxPlayers)
	properties["view-distance"] = strconv.Itoa(params.ViewDistance)
	properties["max-threads"] = strconv.Itoa(params.BedrockMaxThreads)
	properties["player-idle-timeout"] = strconv.Itoa(params.PlayerIdleTimeout)

	// Write updated properties
	file, err := os.Create(propertiesPath)
	if err != nil {
		return fmt.Errorf("failed to create server.properties: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header comment
	fmt.Fprintln(writer, "# Minecraft Bedrock Dedicated Server Configuration")
	fmt.Fprintln(writer, "# Generated by BDS module")
	fmt.Fprintln(writer, "")

	// Write properties
	for key, value := range properties {
		fmt.Fprintf(writer, "%s=%s\n", key, value)
	}

	log.Printf("BDS: Applied server configuration - Port: %d, MaxPlayers: %d, ViewDistance: %d, MaxThreads: %d, IdleTimeout: %d",
		params.BedrockServerPort, params.MaxPlayers, params.ViewDistance,
		params.BedrockMaxThreads, params.PlayerIdleTimeout)

	return nil
}

// Constants for server setup
const (
	// Expected SHA256 hash of bedrock-server-1.21.102.1.zip
	expectedServerHash = "87dc223a4bbdd15a0cc39d9b3630d4ae50e635898543a087ab8ef97cc0f9432e"
	serverZipFile      = "bedrock-server-1.21.102.1.zip"
	serverDownloadURL  = "https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server-1.21.102.1.zip"
)

// ensureServerSetup ensures the bedrock server is properly downloaded, validated, and extracted
func ensureServerSetup() error {
	log.Println("BDS: Ensuring server setup...")

	// Check if zip file exists and validate hash
	if err := validateServerZip(); err != nil {
		log.Printf("BDS: Server zip validation failed: %v", err)

		// Download the server zip
		if err := downloadServerZip(); err != nil {
			return fmt.Errorf("failed to download server: %w", err)
		}

		// Validate the downloaded zip
		if err := validateServerZip(); err != nil {
			return fmt.Errorf("downloaded server zip is invalid: %w", err)
		}
	}

	// Extract server to server directory
	if err := extractServer(); err != nil {
		return fmt.Errorf("failed to extract server: %w", err)
	}

	// Apply hardcoded configuration
	if err := applyHardcodedConfig(); err != nil {
		return fmt.Errorf("failed to apply hardcoded config: %w", err)
	}

	log.Println("BDS: Server setup complete")
	return nil
}

// validateServerZip checks if the server zip exists and has the correct hash
func validateServerZip() error {
	// Check if file exists
	if _, err := os.Stat(serverZipFile); os.IsNotExist(err) {
		return fmt.Errorf("server zip file not found")
	}

	// Calculate file hash
	file, err := os.Open(serverZipFile)
	if err != nil {
		return fmt.Errorf("failed to open server zip: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualHash != expectedServerHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedServerHash, actualHash)
	}

	log.Println("BDS: Server zip validation successful")
	return nil
}

// downloadServerZip downloads the bedrock server zip from the official URL
func downloadServerZip() error {
	log.Printf("BDS: Downloading server from %s...", serverDownloadURL)

	// Create HTTP request
	resp, err := http.Get(serverDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create output file
	out, err := os.Create(serverZipFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save downloaded file: %w", err)
	}

	log.Println("BDS: Server download complete")
	return nil
}

// extractServer extracts the bedrock server zip to the server directory
func extractServer() error {
	log.Println("BDS: Extracting server...")

	// Open zip file
	reader, err := zip.OpenReader(serverZipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Create server directory
	if err := os.MkdirAll("server", 0755); err != nil {
		return fmt.Errorf("failed to create server directory: %w", err)
	}

	// Extract files
	for _, file := range reader.File {
		path := filepath.Join("server", file.Name)

		// Create directory if needed
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
		}

		// Extract file
		if err := extractFile(file, path); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	// Make bedrock_server executable
	serverPath := filepath.Join("server", "bedrock_server")
	if err := os.Chmod(serverPath, 0755); err != nil {
		return fmt.Errorf("failed to make server executable: %w", err)
	}

	log.Println("BDS: Server extraction complete")
	return nil
}

// extractFile extracts a single file from the zip archive
func extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// applyHardcodedConfig applies the hardcoded configuration from README.md
func applyHardcodedConfig() error {
	log.Println("BDS: Applying hardcoded configuration...")

	propertiesPath := filepath.Join("server", "server.properties")

	// Hardcoded configuration from README.md
	hardcodedConfig := map[string]string{
		"online-mode":          "true",
		"xbox-auth":            "true",
		"texturepack-required": "true",
		"allow-cheats":         "true",
		"difficulty":           "normal",
		"force-gamemode":       "true",
		"gamemode":             "survival",
		"level-seed":           "", // Will be randomized
		"server-name":          "ConsensusCraft Node",
		"level-name":           "Bedrock level",
	}

	// Read existing properties
	properties := make(map[string]string)
	if file, err := os.Open(propertiesPath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				properties[parts[0]] = parts[1]
			}
		}
	}

	// Apply hardcoded configuration (overwrite existing values)
	maps.Copy(properties, hardcodedConfig)

	// Write updated properties
	file, err := os.Create(propertiesPath)
	if err != nil {
		return fmt.Errorf("failed to create server.properties: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header comment
	fmt.Fprintln(writer, "# Minecraft Bedrock Dedicated Server Configuration")
	fmt.Fprintln(writer, "# ConsensusCraft - Hardcoded Security Configuration")
	fmt.Fprintln(writer, "# DO NOT MODIFY - Required for network security")
	fmt.Fprintln(writer, "")

	// Write properties
	for key, value := range properties {
		fmt.Fprintf(writer, "%s=%s\n", key, value)
	}

	log.Println("BDS: Hardcoded configuration applied")
	return nil
}

// monitorServerLogs monitors server output and processes events
func monitorServerLogs(reader io.Reader, bds *Bds, params Parameters, stdin io.WriteCloser) {
	// Create a TeeReader to duplicate output to stdout while parsing
	teeReader := io.TeeReader(reader, os.Stdout)
	scanner := bufio.NewScanner(teeReader)

	// Compile regex patterns for log parsing
	playerConnectedRegex := regexp.MustCompile(`Player connected: ([^,]+),`)
	playerSpawnedRegex := regexp.MustCompile(`Player Spawned: ([^,\s]+)`)
	playerDisconnectedRegex := regexp.MustCompile(`Player disconnected: ([^,]+),`)
	enderChestRegex := regexp.MustCompile(`\[X_ENDER_CHEST\]\[([^\]]+)\]\[(.+)\]`)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse player connected events
		if matches := playerConnectedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player connected: %s", playerName)

			select {
			case bds.PlayerLogin <- playerName:
			default:
				log.Printf("BDS: PlayerLogin channel full, dropping event for %s", playerName)
			}
		}

		// Parse player spawned events - trigger inventory restoration
		if matches := playerSpawnedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player spawned: %s", playerName)

			// Get inventory data from callback and restore it via tags
			go func(name string) {
				if inventoryData, err := params.InventoryCallback(name); err == nil {
					if err := restorePlayerInventory(name, inventoryData, stdin); err != nil {
						log.Printf("BDS: Failed to restore inventory for %s: %v", name, err)
					}
				} else {
					log.Printf("BDS: Failed to get inventory data for %s: %v", name, err)
				}
			}(playerName)
		}

		// Parse player disconnected events
		if matches := playerDisconnectedRegex.FindStringSubmatch(line); len(matches) > 1 {
			playerName := strings.TrimSpace(matches[1])
			log.Printf("BDS: Player disconnected: %s", playerName)

			select {
			case bds.PlayerLogout <- playerName:
			default:
				log.Printf("BDS: PlayerLogout channel full, dropping event for %s", playerName)
			}
		}

		// Parse ender chest inventory updates
		if matches := enderChestRegex.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			inventoryData := matches[2]

			log.Printf("BDS: Inventory update for %s", playerName)

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

// restorePlayerInventory restores a player's inventory using server commands
func restorePlayerInventory(playerName string, inventoryData []byte, stdin io.WriteCloser) error {
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

		log.Printf("BDS: Added inventory tag %d for player %s", i, playerName)
	}

	return nil
}
