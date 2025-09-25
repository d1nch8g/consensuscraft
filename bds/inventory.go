package bds

import (
	"fmt"
	"io"
	"strings"

	"github.com/d1nch8g/consensuscraft/logger"
)

// InventoryManager handles player inventory operations
type InventoryManager struct {
	receiveCallback InventoryReceiveCallback
	updateCallback  InventoryUpdateCallback
}

// NewInventoryManager creates a new inventory manager
func NewInventoryManager(rc InventoryReceiveCallback, uc InventoryUpdateCallback) *InventoryManager {
	return &InventoryManager{
		receiveCallback: rc,
		updateCallback:  uc,
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

		logger.Printf("Added inventory tag %d for player %s", i, playerName)
	}

	return nil
}

func (im *InventoryManager) UpdatePlayerInventory(playerName string, inventoryData []byte) error {
	return im.updateCallback(playerName, inventoryData)
}
