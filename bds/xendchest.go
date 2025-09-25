package bds

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/d1nch8g/consensuscraft/gen/xendchest"
)

// PackEntry represents a pack entry in world pack configuration
type PackEntry struct {
	PackID  string `json:"pack_id"`
	Version []int  `json:"version"`
}

// Manifest represents the structure of a Minecraft pack manifest
type Manifest struct {
	Header struct {
		UUID    string `json:"uuid"`
		Version []int  `json:"version"`
	} `json:"header"`
}

// McpackInstaller handles mcpack installation and activation
type McpackInstaller struct {
	behaviorPackUUID string
	resourcePackUUID string
}

// NewMcpackInstaller creates a new mcpack installer
func NewMcpackInstaller() *McpackInstaller {
	return &McpackInstaller{}
}

// getPackUUIDs extracts UUIDs from the embedded mcpack
func (mi *McpackInstaller) getPackUUIDs() error {
	if mi.behaviorPackUUID != "" && mi.resourcePackUUID != "" {
		// Already loaded
		return nil
	}

	// Get the embedded mcpack data
	mcpackData, err := xendchest.Asset("x_ender_chest.mcpack")
	if err != nil {
		return fmt.Errorf("failed to get embedded mcpack: %w", err)
	}

	// Create temporary file for reading
	tempFile, err := os.CreateTemp("", "x_ender_chest_*.mcpack")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write mcpack data to temp file
	if _, err := tempFile.Write(mcpackData); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tempFile.Close()

	// Open the mcpack file (it's a zip file)
	reader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open mcpack file: %w", err)
	}
	defer reader.Close()

	// Extract UUIDs from manifest files
	for _, file := range reader.File {
		var manifest Manifest
		var isTarget bool

		if file.Name == "behavior_pack/manifest.json" {
			isTarget = true
		} else if file.Name == "resource_pack/manifest.json" {
			isTarget = true
		}

		if isTarget {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", file.Name, err)
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", file.Name, err)
			}

			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("failed to parse %s: %w", file.Name, err)
			}

			if file.Name == "behavior_pack/manifest.json" {
				mi.behaviorPackUUID = manifest.Header.UUID
				log.Printf("BDS: Found behavior pack UUID: %s", mi.behaviorPackUUID)
			} else if file.Name == "resource_pack/manifest.json" {
				mi.resourcePackUUID = manifest.Header.UUID
				log.Printf("BDS: Found resource pack UUID: %s", mi.resourcePackUUID)
			}
		}
	}

	if mi.behaviorPackUUID == "" {
		return fmt.Errorf("failed to find behavior pack UUID in mcpack")
	}
	if mi.resourcePackUUID == "" {
		return fmt.Errorf("failed to find resource pack UUID in mcpack")
	}

	return nil
}

// InstallMcpack installs the embedded mcpack to the server
func (mi *McpackInstaller) InstallMcpack() error {
	log.Println("BDS: Installing x_ender_chest mcpack...")

	// Get the embedded mcpack data
	mcpackData, err := xendchest.Asset("x_ender_chest.mcpack")
	if err != nil {
		return fmt.Errorf("failed to get embedded mcpack: %w", err)
	}

	// Extract and activate the mcpack
	if err := mi.ExtractAndActivateMcpack(mcpackData); err != nil {
		return fmt.Errorf("failed to extract and activate mcpack: %w", err)
	}

	return nil
}

// ExtractAndActivateMcpack extracts the mcpack and activates it in worlds
func (mi *McpackInstaller) ExtractAndActivateMcpack(mcpackData []byte) error {
	log.Println("BDS: Extracting and activating mcpack...")

	// Create temporary file for extraction
	tempFile, err := os.CreateTemp("", "x_ender_chest_*.mcpack")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write mcpack data to temp file
	if _, err := tempFile.Write(mcpackData); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tempFile.Close()

	// Extract mcpack (it's a zip file)
	if err := mi.extractMcpack(tempFile.Name()); err != nil {
		return fmt.Errorf("failed to extract mcpack: %w", err)
	}

	// Activate in worlds
	if err := mi.activateInWorlds(); err != nil {
		return fmt.Errorf("failed to activate in worlds: %w", err)
	}

	return nil
}

// extractMcpack extracts the mcpack file to appropriate directories
func (mi *McpackInstaller) extractMcpack(mcpackPath string) error {
	log.Println("BDS: Extracting mcpack contents...")

	// Open the mcpack file (it's a zip file)
	reader, err := zip.OpenReader(mcpackPath)
	if err != nil {
		return fmt.Errorf("failed to open mcpack file: %w", err)
	}
	defer reader.Close()

	// Create base directories
	behaviorDir := filepath.Join("behavior_packs", "x_ender_chest")
	resourceDir := filepath.Join("resource_packs", "x_ender_chest")

	if err := os.MkdirAll(behaviorDir, 0755); err != nil {
		return fmt.Errorf("failed to create behavior pack directory: %w", err)
	}

	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create resource pack directory: %w", err)
	}

	// Extract files from the mcpack
	for _, file := range reader.File {
		// Determine destination based on file path
		var destPath string

		if strings.HasPrefix(file.Name, "behavior_pack/") {
			// Extract to behavior_packs directory
			relativePath := strings.TrimPrefix(file.Name, "behavior_pack/")
			destPath = filepath.Join(behaviorDir, relativePath)
		} else if strings.HasPrefix(file.Name, "resource_pack/") {
			// Extract to resource_packs directory
			relativePath := strings.TrimPrefix(file.Name, "resource_pack/")
			destPath = filepath.Join(resourceDir, relativePath)
		} else {
			// Skip files that don't belong to either pack
			continue
		}

		// Create directory if this is a directory entry
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Create parent directories for files
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
		}

		// Extract the file
		if err := mi.extractFile(file, destPath); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	log.Printf("BDS: Successfully extracted mcpack contents to behavior_packs and resource_packs")
	return nil
}

// extractFile extracts a single file from the zip archive
func (mi *McpackInstaller) extractFile(file *zip.File, destPath string) error {
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

// activateInWorlds activates the mcpack in all existing worlds
func (mi *McpackInstaller) activateInWorlds() error {
	log.Println("BDS: Activating mcpack in worlds...")

	// Check if worlds directory exists
	worldsDir := "worlds"
	if _, err := os.Stat(worldsDir); os.IsNotExist(err) {
		log.Println("BDS: No worlds directory found, creating default world configuration...")
		// Create worlds directory and default world
		if err := os.MkdirAll(filepath.Join(worldsDir, "Bedrock level"), 0755); err != nil {
			return fmt.Errorf("failed to create default world directory: %w", err)
		}
		// Activate in the default world
		return mi.activateInWorld(filepath.Join(worldsDir, "Bedrock level"))
	}

	// List all world directories
	worlds, err := os.ReadDir(worldsDir)
	if err != nil {
		return fmt.Errorf("failed to read worlds directory: %w", err)
	}

	// For each world, ensure the mcpack is activated
	for _, world := range worlds {
		if world.IsDir() {
			worldPath := filepath.Join(worldsDir, world.Name())
			if err := mi.activateInWorld(worldPath); err != nil {
				log.Printf("BDS: Warning - failed to activate mcpack in world %s: %v", world.Name(), err)
				// Continue with other worlds
			}
		}
	}

	return nil
}

// activateInWorld activates the mcpack in a specific world
func (mi *McpackInstaller) activateInWorld(worldPath string) error {
	// Ensure we have the UUIDs loaded
	if err := mi.getPackUUIDs(); err != nil {
		return fmt.Errorf("failed to get pack UUIDs: %w", err)
	}

	behaviorPacksFile := filepath.Join(worldPath, "world_behavior_packs.json")
	resourcePacksFile := filepath.Join(worldPath, "world_resource_packs.json")

	// Handle behavior packs
	if err := mi.addPackToWorldConfig(behaviorPacksFile, mi.behaviorPackUUID, [3]int{1, 0, 0}); err != nil {
		return fmt.Errorf("failed to add behavior pack to world config: %w", err)
	}

	// Handle resource packs
	if err := mi.addPackToWorldConfig(resourcePacksFile, mi.resourcePackUUID, [3]int{1, 0, 0}); err != nil {
		return fmt.Errorf("failed to add resource pack to world config: %w", err)
	}

	log.Printf("BDS: Activated mcpack in world: %s", filepath.Base(worldPath))
	return nil
}

// addPackToWorldConfig adds a pack to world configuration if it doesn't already exist
func (mi *McpackInstaller) addPackToWorldConfig(configFile string, packUUID string, version [3]int) error {
	var packs []PackEntry

	// Read existing configuration if it exists
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &packs); err != nil {
			log.Printf("BDS: Warning - failed to parse existing %s: %v", configFile, err)
			// Continue with empty packs slice
			packs = []PackEntry{}
		}
	}

	// Check if our pack is already in the configuration
	for _, pack := range packs {
		if pack.PackID == packUUID {
			log.Printf("BDS: Pack %s already exists in %s", packUUID, filepath.Base(configFile))
			return nil
		}
	}

	// Add our pack to the configuration
	newPack := PackEntry{
		PackID:  packUUID,
		Version: []int{version[0], version[1], version[2]},
	}
	packs = append(packs, newPack)

	// Write the updated configuration
	data, err := json.MarshalIndent(packs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pack configuration: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write pack configuration: %w", err)
	}

	log.Printf("BDS: Added pack %s to %s", packUUID, filepath.Base(configFile))
	return nil
}

// EnsureMcpackInstalled ensures the mcpack is installed and activated
func (mi *McpackInstaller) EnsureMcpackInstalled() error {
	// First, get the current UUIDs from the embedded mcpack
	if err := mi.getPackUUIDs(); err != nil {
		return fmt.Errorf("failed to get pack UUIDs: %w", err)
	}

	// Check if mcpack is already extracted and has matching UUIDs
	behaviorDir := filepath.Join("behavior_packs", "x_ender_chest")
	resourceDir := filepath.Join("resource_packs", "x_ender_chest")
	behaviorManifest := filepath.Join(behaviorDir, "manifest.json")
	resourceManifest := filepath.Join(resourceDir, "manifest.json")

	// Check if installed UUIDs match current embedded UUIDs
	needsReinstall := false
	
	if _, err := os.Stat(behaviorManifest); err == nil {
		// Check behavior pack UUID
		data, err := os.ReadFile(behaviorManifest)
		if err == nil {
			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				if manifest.Header.UUID != mi.behaviorPackUUID {
					log.Printf("BDS: Behavior pack UUID mismatch - installed: %s, current: %s", manifest.Header.UUID, mi.behaviorPackUUID)
					needsReinstall = true
				}
			} else {
				needsReinstall = true
			}
		} else {
			needsReinstall = true
		}
	} else {
		needsReinstall = true
	}

	if _, err := os.Stat(resourceManifest); err == nil {
		// Check resource pack UUID
		data, err := os.ReadFile(resourceManifest)
		if err == nil {
			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				if manifest.Header.UUID != mi.resourcePackUUID {
					log.Printf("BDS: Resource pack UUID mismatch - installed: %s, current: %s", manifest.Header.UUID, mi.resourcePackUUID)
					needsReinstall = true
				}
			} else {
				needsReinstall = true
			}
		} else {
			needsReinstall = true
		}
	} else {
		needsReinstall = true
	}

	if needsReinstall {
		log.Println("BDS: Pack UUIDs don't match or packs missing - reinstalling...")
		// Clean up old pack directories
		os.RemoveAll(behaviorDir)
		os.RemoveAll(resourceDir)
		// Install the mcpack
		return mi.InstallMcpack()
	}

	log.Println("BDS: x_ender_chest mcpack already installed with correct UUIDs")
	// Still try to activate in any new worlds
	return mi.activateInWorlds()
}
