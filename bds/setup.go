package bds

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/d1nch8g/consensuscraft/logger"
)

// Setup handles server setup scenarios
type Setup struct{}

// NewSetup creates a new setup manager
func NewSetup() *Setup {
	return &Setup{}
}

// Platform-specific constants
var (
	serverZipFile     string
	serverDownloadURL string
	serverExecutable  string
)

// init initializes platform-specific constants based on the operating system
func init() {
	switch runtime.GOOS {
	case "windows":
		serverZipFile = "bedrock-server-1.21.102.1.zip"
		serverDownloadURL = "https://www.minecraft.net/bedrockdedicatedserver/bin-win/bedrock-server-1.21.102.1.zip"
		serverExecutable = "bedrock_server.exe"
	default: // linux and other unix-like systems
		serverZipFile = "bedrock-server-1.21.102.1.zip"
		serverDownloadURL = "https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server-1.21.102.1.zip"
		serverExecutable = "bedrock_server"
	}
}

// EnsureServer ensures the bedrock server is available based on current directory state
func (s *Setup) EnsureServer() (string, error) {
	logger.Println("Checking server setup scenarios...")

	var serverPath string

	// Scenario 2.1: Check if server executable exists in current directory
	if path := s.checkCurrentDirectory(); path != "" {
		logger.Printf("Found server in current directory: %s", path)
		serverPath = path
	} else if path := s.checkZipArchive(); path != "" {
		// Scenario 2.2: Check if there's a zip archive with server
		logger.Printf("Found server zip archive, extracting...")
		if err := s.extractServer(); err != nil {
			return "", fmt.Errorf("failed to extract server: %w", err)
		}
		// Return the path to the extracted server executable in current directory
		logger.Printf("Server extracted to: %s", serverExecutable)
		serverPath = serverExecutable
	} else {
		// Scenario 2.3: Nothing in current directory - download and setup
		logger.Println("No server found, downloading minecraft server...")
		if err := s.downloadAndSetup(); err != nil {
			return "", fmt.Errorf("failed to download and setup server: %w", err)
		}

		// Return the path to the downloaded and extracted server executable in current directory
		logger.Printf("Server downloaded and extracted to: %s", serverExecutable)
		serverPath = serverExecutable
	}

	// Always ensure mcpack is installed on server startup
	logger.Println("Ensuring x_ender_chest mcpack is installed...")
	mcpackInstaller := NewMcpackInstaller()
	if err := mcpackInstaller.EnsureMcpackInstalled(); err != nil {
		logger.Printf("Warning - failed to install mcpack: %v", err)
		// Don't fail server startup if mcpack installation fails
	}

	return serverPath, nil
}

// checkCurrentDirectory checks if bedrock_server executable exists in current directory
func (s *Setup) checkCurrentDirectory() string {
	// Check for platform-specific executable in current directory
	if _, err := os.Stat(serverExecutable); err == nil {
		return serverExecutable
	}

	// Check for platform-specific executable in server subdirectory
	serverPath := filepath.Join("server", serverExecutable)
	if _, err := os.Stat(serverPath); err == nil {
		return serverPath
	}

	// Fallback: check for both possible executable names (for cross-platform compatibility)
	executables := []string{"bedrock_server", "bedrock_server.exe"}
	for _, exe := range executables {
		if _, err := os.Stat(exe); err == nil {
			return exe
		}
		serverPath := filepath.Join("server", exe)
		if _, err := os.Stat(serverPath); err == nil {
			return serverPath
		}
	}

	return ""
}

// checkZipArchive checks if there's a bedrock server zip file
func (s *Setup) checkZipArchive() string {
	// Check for the specific version zip file
	if _, err := os.Stat(serverZipFile); err == nil {
		return serverZipFile
	}

	// Check for any bedrock server zip files
	files, err := filepath.Glob("bedrock-server*.zip")
	if err == nil && len(files) > 0 {
		return files[0]
	}

	return ""
}

// downloadAndSetup downloads the minecraft server and sets it up
func (s *Setup) downloadAndSetup() error {
	// Download the server zip
	if err := s.downloadServerZip(); err != nil {
		return fmt.Errorf("failed to download server: %w", err)
	}

	// Extract server to current directory
	if err := s.extractServer(); err != nil {
		return fmt.Errorf("failed to extract server: %w", err)
	}

	logger.Println("Server download and setup complete")
	return nil
}

// downloadServerZip downloads the bedrock server zip from the official URL
func (s *Setup) downloadServerZip() error {
	logger.Printf("Downloading server from %s...", serverDownloadURL)

	// Create a custom HTTP client with proper headers
	client := &http.Client{}
	req, err := http.NewRequest("GET", serverDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers that are required by the Minecraft download server
	req.Header.Set("User-Agent", "Wget/1.21.3")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "Keep-Alive")

	// Execute the request
	resp, err := client.Do(req)
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

	logger.Println("Server download complete")
	return nil
}

// extractServer extracts the bedrock server zip to the current directory
func (s *Setup) extractServer() error {
	logger.Println("Extracting server...")

	// Find the zip file to extract
	zipFile := s.checkZipArchive()
	if zipFile == "" {
		return fmt.Errorf("no zip file found to extract")
	}

	// Open zip file
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Extract files directly to current directory
	for _, file := range reader.File {
		path := file.Name

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
		if err := s.extractFile(file, path); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	// Make server executable (only needed on Unix-like systems)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(serverExecutable, 0755); err != nil {
			return fmt.Errorf("failed to make server executable: %w", err)
		}
	}

	logger.Println("Server extraction complete")
	return nil
}

// extractFile extracts a single file from the zip archive
func (s *Setup) extractFile(file *zip.File, destPath string) error {
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
