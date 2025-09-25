package inventories

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/d1nch8g/consensuscraft/database"
)

// ItemData represents a Minecraft item with all its properties
type ItemData struct {
	TypeId          string            `json:"typeId"`
	Amount          int               `json:"amount"`
	NameTag         string            `json:"nameTag,omitempty"`
	Lore            []string          `json:"lore,omitempty"`
	Enchantments    []EnchantmentData `json:"enchantments,omitempty"`
	Durability      *DurabilityData   `json:"durability,omitempty"`
	ShulkerContents []any             `json:"shulker_contents,omitempty"`
}

// EnchantmentData represents an enchantment on an item
type EnchantmentData struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// DurabilityData represents durability information
type DurabilityData struct {
	Damage        int `json:"damage"`
	MaxDurability int `json:"maxDurability"`
}

// OriginInfo represents parsed origin information from lore
type OriginInfo struct {
	Server string
}

// originRegex matches "Origin: server_name" format
var originRegex = regexp.MustCompile(`^Origin:\s+(\S+)$`)

// ParseOriginFromLore extracts origin information from item lore
func ParseOriginFromLore(lore []string) *OriginInfo {
	for _, line := range lore {
		matches := originRegex.FindStringSubmatch(line)
		if len(matches) == 2 {
			serverName := matches[1]
			return &OriginInfo{
				Server: serverName,
			}
		}
	}
	return nil
}

// HasOriginFromServer checks if an item originates from a specific server
func HasOriginFromServer(item map[string]any, serverName string) bool {
	loreInterface, exists := item["lore"]
	if !exists {
		return false
	}

	loreSlice, ok := loreInterface.([]any)
	if !ok {
		return false
	}

	var lore []string
	for _, l := range loreSlice {
		if str, ok := l.(string); ok {
			lore = append(lore, str)
		}
	}

	origin := ParseOriginFromLore(lore)
	return origin != nil && origin.Server == serverName
}

// CleanInventoryFromServer removes all items originating from a specific server
func CleanInventoryFromServer(inventoryJSON []byte, serverName string) ([]byte, bool, error) {
	if len(inventoryJSON) == 0 {
		return inventoryJSON, false, nil
	}

	var items []any
	if err := json.Unmarshal(inventoryJSON, &items); err != nil {
		return inventoryJSON, false, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}

	modified := false

	for i, itemInterface := range items {
		if itemInterface == nil {
			continue
		}

		itemMap, ok := itemInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if this item should be removed
		if HasOriginFromServer(itemMap, serverName) {
			items[i] = nil
			modified = true
			continue
		}

		// Check nested shulker contents
		if shulkerContents, exists := itemMap["shulker_contents"]; exists {
			if shulkerSlice, ok := shulkerContents.([]any); ok {
				shulkerModified := cleanShulkerContents(shulkerSlice, serverName)
				if shulkerModified {
					itemMap["shulker_contents"] = shulkerSlice
					items[i] = itemMap
					modified = true
				}
			}
		}
	}

	if !modified {
		return inventoryJSON, false, nil
	}

	cleanedJSON, err := json.Marshal(items)
	if err != nil {
		return inventoryJSON, false, fmt.Errorf("failed to marshal cleaned inventory: %w", err)
	}

	return cleanedJSON, true, nil
}

// cleanShulkerContents recursively cleans shulker box contents
func cleanShulkerContents(shulkerItems []any, serverName string) bool {
	modified := false

	for i, shulkerItemInterface := range shulkerItems {
		if shulkerItemInterface == nil {
			continue
		}

		shulkerItemMap, ok := shulkerItemInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if this shulker item should be removed
		if HasOriginFromServer(shulkerItemMap, serverName) {
			shulkerItems[i] = nil
			modified = true
			continue
		}

		// Recursively check nested shulker contents
		if nestedShulkerContents, exists := shulkerItemMap["shulker_contents"]; exists {
			if nestedShulkerSlice, ok := nestedShulkerContents.([]any); ok {
				if cleanShulkerContents(nestedShulkerSlice, serverName) {
					shulkerItemMap["shulker_contents"] = nestedShulkerSlice
					shulkerItems[i] = shulkerItemMap
					modified = true
				}
			}
		}
	}

	return modified
}

// EnhancedDB wraps the database with inventory cleaning capabilities
type EnhancedDB struct {
	*database.DB
}

// NewEnhancedDB creates a new enhanced database wrapper
func NewEnhancedDB(db *database.DB) *EnhancedDB {
	return &EnhancedDB{DB: db}
}

// DeleteWithInventoryCleanup performs force deletion and cleans inventories
func (edb *EnhancedDB) DeleteWithInventoryCleanup(serverName string, force bool) error {
	// First perform the standard database deletion
	if err := edb.DB.Delete(serverName, force); err != nil {
		return fmt.Errorf("failed to perform database deletion: %w", err)
	}

	// Now clean remaining inventories
	return edb.cleanAllInventories(serverName)
}

// cleanAllInventories iterates through all remaining inventories and cleans them
func (edb *EnhancedDB) cleanAllInventories(serverName string) error {
	iter := edb.DB.NewIterator()
	if iter == nil {
		return fmt.Errorf("failed to create database iterator")
	}
	defer iter.Release()

	var updates []struct {
		player    string
		inventory []byte
	}

	// Collect all inventories that need cleaning
	for iter.Next() {
		player := string(iter.Key())
		data := iter.Value()

		var playerInv database.PlayerInventories
		if err := json.Unmarshal(data, &playerInv); err != nil {
			continue // Skip corrupted entries
		}

		// Process each inventory entry
		for i, entry := range playerInv.Entries {
			cleanedInventory, modified, err := CleanInventoryFromServer(entry.Inventory, serverName)
			if err != nil {
				continue // Skip entries that can't be processed
			}

			if modified {
				playerInv.Entries[i].Inventory = cleanedInventory
			}
		}

		// Re-marshal the player inventories
		updatedData, err := json.Marshal(playerInv)
		if err != nil {
			continue
		}

		// Check if anything actually changed
		if string(updatedData) != string(data) {
			updates = append(updates, struct {
				player    string
				inventory []byte
			}{player, updatedData})
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	// Apply all updates
	for _, update := range updates {
		// Use Put to update the player's inventory data
		// We need to extract the latest entry to maintain the database structure
		var playerInv database.PlayerInventories
		if err := json.Unmarshal(update.inventory, &playerInv); err != nil {
			continue
		}

		if len(playerInv.Entries) > 0 {
			// Get the latest entry and update it
			latestEntry := playerInv.Entries[0]
			if err := edb.DB.Put(update.player, latestEntry.Inventory, latestEntry.Server); err != nil {
				return fmt.Errorf("failed to update inventory for player %s: %w", update.player, err)
			}
		}
	}

	return nil
}

// AddOriginToInventory adds origin lore to all items in an inventory that don't have it
func AddOriginToInventory(inventoryJSON []byte, serverName string) ([]byte, bool, error) {
	if len(inventoryJSON) == 0 {
		return inventoryJSON, false, nil
	}

	var items []any
	if err := json.Unmarshal(inventoryJSON, &items); err != nil {
		return inventoryJSON, false, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}

	modified := false
	originLine := fmt.Sprintf("Origin: %s", serverName)

	for i, itemInterface := range items {
		if itemInterface == nil {
			continue
		}

		itemMap, ok := itemInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if item already has origin
		if HasOriginFromServer(itemMap, serverName) {
			continue
		}

		// Add origin to lore
		loreInterface, exists := itemMap["lore"]
		var lore []any
		if exists {
			if loreSlice, ok := loreInterface.([]any); ok {
				lore = loreSlice
			}
		}

		// Check if any origin already exists
		hasAnyOrigin := false
		for _, l := range lore {
			if str, ok := l.(string); ok && strings.HasPrefix(str, "Origin: ") {
				hasAnyOrigin = true
				break
			}
		}

		// Only add origin if no origin exists
		if !hasAnyOrigin {
			lore = append(lore, originLine)
			itemMap["lore"] = lore
			items[i] = itemMap
			modified = true
		}

		// Handle nested shulker contents
		if shulkerContents, exists := itemMap["shulker_contents"]; exists {
			if shulkerSlice, ok := shulkerContents.([]any); ok {
				shulkerModified := addOriginToShulkerContents(shulkerSlice, serverName)
				if shulkerModified {
					itemMap["shulker_contents"] = shulkerSlice
					items[i] = itemMap
					modified = true
				}
			}
		}
	}

	if !modified {
		return inventoryJSON, false, nil
	}

	updatedJSON, err := json.Marshal(items)
	if err != nil {
		return inventoryJSON, false, fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	return updatedJSON, true, nil
}

// addOriginToShulkerContents recursively adds origin to shulker box contents
func addOriginToShulkerContents(shulkerItems []any, serverName string) bool {
	modified := false
	originLine := fmt.Sprintf("Origin: %s", serverName)

	for i, shulkerItemInterface := range shulkerItems {
		if shulkerItemInterface == nil {
			continue
		}

		shulkerItemMap, ok := shulkerItemInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if item already has any origin
		loreInterface, exists := shulkerItemMap["lore"]
		var lore []any
		if exists {
			if loreSlice, ok := loreInterface.([]any); ok {
				lore = loreSlice
			}
		}

		// Check if any origin already exists
		hasAnyOrigin := false
		for _, l := range lore {
			if str, ok := l.(string); ok && strings.HasPrefix(str, "Origin: ") {
				hasAnyOrigin = true
				break
			}
		}

		// Only add origin if no origin exists
		if !hasAnyOrigin {
			lore = append(lore, originLine)
			shulkerItemMap["lore"] = lore
			shulkerItems[i] = shulkerItemMap
			modified = true
		}

		// Recursively handle nested shulker contents
		if nestedShulkerContents, exists := shulkerItemMap["shulker_contents"]; exists {
			if nestedShulkerSlice, ok := nestedShulkerContents.([]any); ok {
				if addOriginToShulkerContents(nestedShulkerSlice, serverName) {
					shulkerItemMap["shulker_contents"] = nestedShulkerSlice
					shulkerItems[i] = shulkerItemMap
					modified = true
				}
			}
		}
	}

	return modified
}
