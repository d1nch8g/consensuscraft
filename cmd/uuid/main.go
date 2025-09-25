package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
)

type Manifest struct {
	FormatVersion int      `json:"format_version"`
	Metadata      Metadata `json:"metadata"`
	Header        Header   `json:"header"`
	Modules       []Module `json:"modules"`
	Dependencies  []Dependency `json:"dependencies,omitempty"`
}

type Metadata struct {
	Authors       []string     `json:"authors"`
	GeneratedWith GeneratedWith `json:"generated_with"`
}

type GeneratedWith struct {
	Bridge []string `json:"bridge"`
	Dash   []string `json:"dash"`
}

type Header struct {
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	MinEngineVersion []int     `json:"min_engine_version"`
	UUID             string    `json:"uuid"`
	Version          []int     `json:"version"`
}

type Module struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
	Version  []int  `json:"version"`
	Language string `json:"language,omitempty"`
	Entry    string `json:"entry,omitempty"`
}

type Dependency struct {
	ModuleName string `json:"module_name"`
	Version    string `json:"version"`
}

func generateUUID() string {
	return uuid.New().String()
}

func updateManifestUUIDs(manifestPath string) error {
	// Read the manifest file
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Parse JSON
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	// Generate new UUIDs
	headerUUID := generateUUID()
	manifest.Header.UUID = headerUUID

	fmt.Printf("Generated UUIDs for %s:\n", manifestPath)
	fmt.Printf("  Header: %s\n", headerUUID)

	// Update module UUIDs
	for i := range manifest.Modules {
		moduleUUID := generateUUID()
		manifest.Modules[i].UUID = moduleUUID
		fmt.Printf("  Module %d (%s): %s\n", i, manifest.Modules[i].Type, moduleUUID)
	}

	// Marshal back to JSON with proper formatting
	updatedData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated manifest: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(manifestPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	fmt.Printf("✓ Updated %s\n", manifestPath)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: uuid <manifest_path> [manifest_path2] ...")
		fmt.Println("Example: uuid mod/behavior_pack/manifest.json mod/resource_pack/manifest.json")
		os.Exit(1)
	}

	fmt.Println("Refreshing UUIDs in addon manifests...")

	for _, manifestPath := range os.Args[1:] {
		if err := updateManifestUUIDs(manifestPath); err != nil {
			log.Printf("Error updating %s: %v", manifestPath, err)
			os.Exit(1)
		}
	}

	fmt.Println("✓ All UUIDs refreshed successfully")
}
