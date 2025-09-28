package bds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/d1nch8g/consensuscraft/gen/xendchest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMcpackInstaller_RealWorld tests the McpackInstaller functionality in real-world scenarios
func TestMcpackInstaller_RealWorld(t *testing.T) {
	t.Run("NewMcpackInstaller", func(t *testing.T) {
		installer := NewMcpackInstaller()
		assert.NotNil(t, installer)
		assert.Empty(t, installer.behaviorPackUUID)
		assert.Empty(t, installer.resourcePackUUID)
	})

	t.Run("GetPackUUIDs_FromRealAsset", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Test that we can extract UUIDs from the real embedded mcpack
		err := installer.getPackUUIDs()
		assert.NoError(t, err)

		// Verify UUIDs are properly extracted (they should be valid UUIDs)
		assert.NotEmpty(t, installer.behaviorPackUUID)
		assert.NotEmpty(t, installer.resourcePackUUID)
		assert.Contains(t, installer.behaviorPackUUID, "-") // UUIDs contain hyphens
		assert.Contains(t, installer.resourcePackUUID, "-") // UUIDs contain hyphens
	})

	t.Run("InstallMcpack_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Install the mcpack using real embedded assets
		err := installer.InstallMcpack()
		assert.NoError(t, err)

		// Verify files were extracted to correct Minecraft server directories
		behaviorManifest := filepath.Join("behavior_packs", "x_ender_chest", "manifest.json")
		resourceManifest := filepath.Join("resource_packs", "x_ender_chest", "manifest.json")

		assert.FileExists(t, behaviorManifest)
		assert.FileExists(t, resourceManifest)

		// Verify manifest contents contain valid UUIDs
		var behaviorManifestData map[string]interface{}
		behaviorData, err := os.ReadFile(behaviorManifest)
		require.NoError(t, err)
		err = json.Unmarshal(behaviorData, &behaviorManifestData)
		require.NoError(t, err)

		var resourceManifestData map[string]interface{}
		resourceData, err := os.ReadFile(resourceManifest)
		require.NoError(t, err)
		err = json.Unmarshal(resourceData, &resourceManifestData)
		require.NoError(t, err)

		// Verify UUIDs in extracted manifests match what we expect
		behaviorHeader := behaviorManifestData["header"].(map[string]interface{})
		resourceHeader := resourceManifestData["header"].(map[string]interface{})

		assert.NotEmpty(t, behaviorHeader["uuid"])
		assert.NotEmpty(t, resourceHeader["uuid"])
	})

	t.Run("ExtractAndActivateMcpack_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real mcpack data from embedded assets
		mcpackData, err := xendchest.Asset("x_ender_chest.mcpack")
		require.NoError(t, err)

		// Extract and activate using real data
		err = installer.ExtractAndActivateMcpack(mcpackData)
		assert.NoError(t, err)

		// Verify files were extracted to correct directories
		behaviorManifest := filepath.Join("behavior_packs", "x_ender_chest", "manifest.json")
		resourceManifest := filepath.Join("resource_packs", "x_ender_chest", "manifest.json")

		assert.FileExists(t, behaviorManifest)
		assert.FileExists(t, resourceManifest)

		// Verify world activation
		defaultWorldDir := filepath.Join("worlds", "Bedrock level")
		behaviorConfig := filepath.Join(defaultWorldDir, "world_behavior_packs.json")
		resourceConfig := filepath.Join(defaultWorldDir, "world_resource_packs.json")

		assert.FileExists(t, behaviorConfig)
		assert.FileExists(t, resourceConfig)

		// Verify world configuration contains the pack UUIDs
		var behaviorPacks []PackEntry
		behaviorData, err := os.ReadFile(behaviorConfig)
		require.NoError(t, err)
		err = json.Unmarshal(behaviorData, &behaviorPacks)
		require.NoError(t, err)
		assert.Len(t, behaviorPacks, 1)

		var resourcePacks []PackEntry
		resourceData, err := os.ReadFile(resourceConfig)
		require.NoError(t, err)
		err = json.Unmarshal(resourceData, &resourcePacks)
		require.NoError(t, err)
		assert.Len(t, resourcePacks, 1)
	})

	t.Run("ExtractMcpack_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real mcpack data and save to temp file
		mcpackData, err := xendchest.Asset("x_ender_chest.mcpack")
		require.NoError(t, err)

		tempFile, err := os.CreateTemp("", "real_mcpack_*.mcpack")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.Write(mcpackData)
		require.NoError(t, err)
		tempFile.Close()

		// Extract the real mcpack
		err = installer.extractMcpack(tempFile.Name())
		assert.NoError(t, err)

		// Verify extraction to correct Minecraft directories
		behaviorManifest := filepath.Join("behavior_packs", "x_ender_chest", "manifest.json")
		resourceManifest := filepath.Join("resource_packs", "x_ender_chest", "manifest.json")

		assert.FileExists(t, behaviorManifest)
		assert.FileExists(t, resourceManifest)

		// Verify additional files were extracted (not just manifests)
		behaviorScripts := filepath.Join("behavior_packs", "x_ender_chest", "scripts")
		resourceTextures := filepath.Join("resource_packs", "x_ender_chest", "textures")

		assert.DirExists(t, behaviorScripts)
		assert.DirExists(t, resourceTextures)
	})

	t.Run("ExtractMcpack_InvalidFile", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Create an invalid file
		err := os.WriteFile("invalid.mcpack", []byte("not a zip file"), 0644)
		require.NoError(t, err)

		err = installer.extractMcpack("invalid.mcpack")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open mcpack file")
	})

	t.Run("ActivateInWorlds_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// First get the real UUIDs
		err := installer.getPackUUIDs()
		require.NoError(t, err)

		// Create test world directories
		worldDir1 := filepath.Join("worlds", "TestWorld1")
		worldDir2 := filepath.Join("worlds", "TestWorld2")
		err = os.MkdirAll(worldDir1, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(worldDir2, 0755)
		require.NoError(t, err)

		err = installer.activateInWorlds()
		assert.NoError(t, err)

		// Verify world configuration files were created in all worlds
		for _, worldDir := range []string{worldDir1, worldDir2} {
			behaviorConfig := filepath.Join(worldDir, "world_behavior_packs.json")
			resourceConfig := filepath.Join(worldDir, "world_resource_packs.json")

			assert.FileExists(t, behaviorConfig)
			assert.FileExists(t, resourceConfig)

			// Verify configuration contains the correct UUIDs
			var behaviorPacks []PackEntry
			behaviorData, err := os.ReadFile(behaviorConfig)
			require.NoError(t, err)
			err = json.Unmarshal(behaviorData, &behaviorPacks)
			require.NoError(t, err)
			assert.Len(t, behaviorPacks, 1)
			assert.Equal(t, installer.behaviorPackUUID, behaviorPacks[0].PackID)

			var resourcePacks []PackEntry
			resourceData, err := os.ReadFile(resourceConfig)
			require.NoError(t, err)
			err = json.Unmarshal(resourceData, &resourcePacks)
			require.NoError(t, err)
			assert.Len(t, resourcePacks, 1)
			assert.Equal(t, installer.resourcePackUUID, resourcePacks[0].PackID)
		}
	})

	t.Run("ActivateInWorlds_NoWorldsDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real UUIDs
		err := installer.getPackUUIDs()
		require.NoError(t, err)

		// No worlds directory exists - should create default
		err = installer.activateInWorlds()
		assert.NoError(t, err)

		// Verify default world was created and configured
		defaultWorldDir := filepath.Join("worlds", "Bedrock level")
		behaviorConfig := filepath.Join(defaultWorldDir, "world_behavior_packs.json")
		resourceConfig := filepath.Join(defaultWorldDir, "world_resource_packs.json")

		assert.FileExists(t, behaviorConfig)
		assert.FileExists(t, resourceConfig)
	})

	t.Run("ActivateInWorld_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real UUIDs
		err := installer.getPackUUIDs()
		require.NoError(t, err)

		// Create test world directory
		worldDir := "CustomWorld"
		err = os.MkdirAll(worldDir, 0755)
		require.NoError(t, err)

		err = installer.activateInWorld(worldDir)
		assert.NoError(t, err)

		// Verify world configuration files were created
		behaviorConfig := filepath.Join(worldDir, "world_behavior_packs.json")
		resourceConfig := filepath.Join(worldDir, "world_resource_packs.json")

		assert.FileExists(t, behaviorConfig)
		assert.FileExists(t, resourceConfig)

		// Verify configuration contains correct UUIDs
		var behaviorPacks []PackEntry
		behaviorData, err := os.ReadFile(behaviorConfig)
		require.NoError(t, err)
		err = json.Unmarshal(behaviorData, &behaviorPacks)
		require.NoError(t, err)
		assert.Len(t, behaviorPacks, 1)
		assert.Equal(t, installer.behaviorPackUUID, behaviorPacks[0].PackID)

		var resourcePacks []PackEntry
		resourceData, err := os.ReadFile(resourceConfig)
		require.NoError(t, err)
		err = json.Unmarshal(resourceData, &resourcePacks)
		require.NoError(t, err)
		assert.Len(t, resourcePacks, 1)
		assert.Equal(t, installer.resourcePackUUID, resourcePacks[0].PackID)
	})

	t.Run("AddPackToWorldConfig_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real UUIDs
		err := installer.getPackUUIDs()
		require.NoError(t, err)

		configFile := "test_world_config.json"
		version := [3]int{1, 0, 0}

		// Test adding to new config file
		err = installer.addPackToWorldConfig(configFile, installer.behaviorPackUUID, version)
		assert.NoError(t, err)

		// Verify file was created with correct content
		var packs []PackEntry
		data, err := os.ReadFile(configFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &packs)
		require.NoError(t, err)

		assert.Len(t, packs, 1)
		assert.Equal(t, installer.behaviorPackUUID, packs[0].PackID)
		assert.Equal(t, []int{1, 0, 0}, packs[0].Version)

		// Test adding to existing config file (should not duplicate)
		err = installer.addPackToWorldConfig(configFile, installer.behaviorPackUUID, version)
		assert.NoError(t, err)

		// Verify no duplication occurred
		data, err = os.ReadFile(configFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &packs)
		require.NoError(t, err)

		assert.Len(t, packs, 1) // Still only one entry
	})

	t.Run("AddPackToWorldConfig_InvalidJSON", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// Get real UUIDs
		err := installer.getPackUUIDs()
		require.NoError(t, err)

		configFile := "test_config.json"
		version := [3]int{1, 0, 0}

		// Create invalid JSON file
		err = os.WriteFile(configFile, []byte("invalid json"), 0644)
		require.NoError(t, err)

		err = installer.addPackToWorldConfig(configFile, installer.behaviorPackUUID, version)
		assert.NoError(t, err) // Should handle gracefully and create new config

		// Verify file was overwritten with valid JSON
		var packs []PackEntry
		data, err := os.ReadFile(configFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &packs)
		require.NoError(t, err)

		assert.Len(t, packs, 1)
		assert.Equal(t, installer.behaviorPackUUID, packs[0].PackID)
	})

	t.Run("EnsureMcpackInstalled_RealWorld", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// First installation
		err := installer.EnsureMcpackInstalled()
		assert.NoError(t, err)

		// Verify files were installed in correct directories
		behaviorManifest := filepath.Join("behavior_packs", "x_ender_chest", "manifest.json")
		resourceManifest := filepath.Join("resource_packs", "x_ender_chest", "manifest.json")

		assert.FileExists(t, behaviorManifest)
		assert.FileExists(t, resourceManifest)

		// Verify world activation
		defaultWorldDir := filepath.Join("worlds", "Bedrock level")
		behaviorConfig := filepath.Join(defaultWorldDir, "world_behavior_packs.json")
		resourceConfig := filepath.Join(defaultWorldDir, "world_resource_packs.json")

		assert.FileExists(t, behaviorConfig)
		assert.FileExists(t, resourceConfig)

		// Second installation (should detect already installed)
		err = installer.EnsureMcpackInstalled()
		assert.NoError(t, err)
	})

	t.Run("EnsureMcpackInstalled_UUIDMismatch", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		installer := NewMcpackInstaller()

		// First installation
		err := installer.EnsureMcpackInstalled()
		assert.NoError(t, err)

		// Manually change the UUIDs in the extracted manifests to simulate mismatch
		behaviorManifest := filepath.Join("behavior_packs", "x_ender_chest", "manifest.json")
		resourceManifest := filepath.Join("resource_packs", "x_ender_chest", "manifest.json")

		// Read current manifests
		behaviorData, err := os.ReadFile(behaviorManifest)
		require.NoError(t, err)
		resourceData, err := os.ReadFile(resourceManifest)
		require.NoError(t, err)

		var behaviorManifestData map[string]interface{}
		err = json.Unmarshal(behaviorData, &behaviorManifestData)
		require.NoError(t, err)

		var resourceManifestData map[string]interface{}
		err = json.Unmarshal(resourceData, &resourceManifestData)
		require.NoError(t, err)

		// Change UUIDs to simulate different version
		behaviorHeader := behaviorManifestData["header"].(map[string]interface{})
		resourceHeader := resourceManifestData["header"].(map[string]interface{})
		behaviorHeader["uuid"] = "different-behavior-uuid"
		resourceHeader["uuid"] = "different-resource-uuid"

		// Write modified manifests back
		modifiedBehaviorData, err := json.MarshalIndent(behaviorManifestData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(behaviorManifest, modifiedBehaviorData, 0644)
		require.NoError(t, err)

		modifiedResourceData, err := json.MarshalIndent(resourceManifestData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(resourceManifest, modifiedResourceData, 0644)
		require.NoError(t, err)

		// Should reinstall due to UUID mismatch
		err = installer.EnsureMcpackInstalled()
		assert.NoError(t, err)

		// Verify files were reinstalled with correct UUIDs
		finalBehaviorData, err := os.ReadFile(behaviorManifest)
		require.NoError(t, err)
		var finalBehaviorManifest map[string]interface{}
		err = json.Unmarshal(finalBehaviorData, &finalBehaviorManifest)
		require.NoError(t, err)

		finalResourceData, err := os.ReadFile(resourceManifest)
		require.NoError(t, err)
		var finalResourceManifest map[string]interface{}
		err = json.Unmarshal(finalResourceData, &finalResourceManifest)
		require.NoError(t, err)

		// UUIDs should be restored to original values
		finalBehaviorHeader := finalBehaviorManifest["header"].(map[string]interface{})
		finalResourceHeader := finalResourceManifest["header"].(map[string]interface{})
		assert.NotEqual(t, "different-behavior-uuid", finalBehaviorHeader["uuid"])
		assert.NotEqual(t, "different-resource-uuid", finalResourceHeader["uuid"])
	})
}
