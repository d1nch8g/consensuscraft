package bds

import (
	"fmt"
	"io"
	"log"
	"strings"
)

// InventoryManager handles player inventory operations
type InventoryManager struct {
	callback InventoryCallback
}

// NewInventoryManager creates a new inventory manager
func NewInventoryManager(callback InventoryCallback) *InventoryManager {
	return &InventoryManager{
		callback: callback,
	}
}

// RestorePlayerInventory restores a player's inventory using server commands
func (im *InventoryManager) RestorePlayerInventory(playerName string, inventoryData []byte, stdin io.WriteCloser) error {
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
