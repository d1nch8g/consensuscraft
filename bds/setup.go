package bds

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	log.Println("BDS: Checking server setup scenarios...")

	var serverPath string

	// Scenario 2.1: Check if server executable exists in current directory
	if path := s.checkCurrentDirectory(); path != "" {
		log.Printf("BDS: Found server in current directory: %s", path)
		serverPath = path
	} else if path := s.checkZipArchive(); path != "" {
		// Scenario 2.2: Check if there's a zip archive with server
		log.Printf("BDS: Found server zip archive, extracting...")
		if err := s.extractServer(); err != nil {
			return "", fmt.Errorf("failed to extract server: %w", err)
		}
		// Return the path to the extracted server executable
		extractedPath := filepath.Join("server", serverExecutable)
		log.Printf("BDS: Server extracted to: %s", extractedPath)
		serverPath = extractedPath
	} else {
		// Scenario 2.3: Nothing in current directory - download and setup
		log.Println("BDS: No server found, downloading minecraft server...")
		if err := s.downloadAndSetup(); err != nil {
			return "", fmt.Errorf("failed to download and setup server: %w", err)
		}

		// Return the path to the downloaded and extracted server executable
		downloadedPath := filepath.Join("server", serverExecutable)
		log.Printf("BDS: Server downloaded and extracted to: %s", downloadedPath)
		serverPath = downloadedPath
	}

	// Always ensure mcpack is installed on server startup
	log.Println("BDS: Ensuring x_ender_chest mcpack is installed...")
	mcpackInstaller := NewMcpackInstaller()
	if err := mcpackInstaller.EnsureMcpackInstalled(); err != nil {
		log.Printf("BDS: Warning - failed to install mcpack: %v", err)
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

	// Extract server to server directory
	if err := s.extractServer(); err != nil {
		return fmt.Errorf("failed to extract server: %w", err)
	}

	log.Println("BDS: Server download and setup complete")
	return nil
}

// downloadServerZip downloads the bedrock server zip from the official URL
func (s *Setup) downloadServerZip() error {
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
func (s *Setup) extractServer() error {
	log.Println("BDS: Extracting server...")

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
		if err := s.extractFile(file, path); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	// Make server executable (only needed on Unix-like systems)
	if runtime.GOOS != "windows" {
		serverPath := filepath.Join("server", serverExecutable)
		if err := os.Chmod(serverPath, 0755); err != nil {
			return fmt.Errorf("failed to make server executable: %w", err)
		}
	}

	log.Println("BDS: Server extraction complete")
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
