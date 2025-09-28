package inventories

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOriginFromLore(t *testing.T) {
	tests := []struct {
		name     string
		lore     []string
		expected *OriginInfo
	}{
		{
			name: "valid origin",
			lore: []string{
				"Some other lore",
				"Origin: server1",
				"More lore",
			},
			expected: &OriginInfo{
				Server: "server1",
			},
		},
		{
			name: "no origin",
			lore: []string{
				"Some lore",
				"More lore",
			},
			expected: nil,
		},
		{
			name: "origin with extra text",
			lore: []string{
				"Origin: server1 extra-text",
			},
			expected: nil,
		},
		{
			name:     "empty lore",
			lore:     []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseOriginFromLore(tt.lore)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Server, result.Server)
			}
		})
	}
}

func TestHasOriginFromServer(t *testing.T) {
	tests := []struct {
		name     string
		item     map[string]any
		server   string
		expected bool
	}{
		{
			name: "has origin from server",
			item: map[string]any{
				"typeId": "minecraft:diamond_sword",
				"lore": []any{
					"Some lore",
					"Origin: server1",
				},
			},
			server:   "server1",
			expected: true,
		},
		{
			name: "has origin from different server",
			item: map[string]any{
				"typeId": "minecraft:diamond_sword",
				"lore": []any{
					"Origin: server2",
				},
			},
			server:   "server1",
			expected: false,
		},
		{
			name: "no origin",
			item: map[string]any{
				"typeId": "minecraft:diamond_sword",
				"lore": []any{
					"Some lore",
				},
			},
			server:   "server1",
			expected: false,
		},
		{
			name: "no lore",
			item: map[string]any{
				"typeId": "minecraft:diamond_sword",
			},
			server:   "server1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasOriginFromServer(tt.item, tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanInventoryFromServer(t *testing.T) {
	tests := []struct {
		name         string
		inventory    string
		server       string
		expectModify bool
		expectError  bool
	}{
		{
			name: "remove items from server",
			inventory: `[
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 1,
					"lore": ["Origin: server1"]
				},
				{
					"typeId": "minecraft:bread",
					"amount": 64,
					"lore": ["Origin: server2"]
				},
				null
			]`,
			server:       "server1",
			expectModify: true,
			expectError:  false,
		},
		{
			name: "no items from server",
			inventory: `[
				{
					"typeId": "minecraft:bread",
					"amount": 64,
					"lore": ["Origin: server2"]
				},
				null
			]`,
			server:       "server1",
			expectModify: false,
			expectError:  false,
		},
		{
			name: "remove items from shulker contents",
			inventory: `[
				{
					"typeId": "minecraft:red_shulker_box",
					"amount": 1,
					"shulkerContents": [
						{
							"typeId": "minecraft:diamond",
							"amount": 64,
							"lore": ["Origin: server1"]
						},
						{
							"typeId": "minecraft:gold_ingot",
							"amount": 32,
							"lore": ["Origin: server2"]
						}
					]
				}
			]`,
			server:       "server1",
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "empty inventory",
			inventory:    "",
			server:       "server1",
			expectModify: false,
			expectError:  false,
		},
		{
			name:         "invalid json",
			inventory:    "invalid json",
			server:       "server1",
			expectModify: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified, err := CleanInventoryFromServer([]byte(tt.inventory), tt.server)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectModify, modified)

			if modified {
				// Verify the result is valid JSON
				var items []any
				err := json.Unmarshal(result, &items)
				assert.NoError(t, err)

				// Verify items from the specified server are removed
				for _, itemInterface := range items {
					if itemInterface == nil {
						continue
					}
					itemMap, ok := itemInterface.(map[string]any)
					if !ok {
						continue
					}
					assert.False(t, HasOriginFromServer(itemMap, tt.server))
				}
			}
		})
	}
}

func TestAddOriginToInventory(t *testing.T) {
	tests := []struct {
		name         string
		inventory    string
		server       string
		expectModify bool
		expectError  bool
	}{
		{
			name: "add origin to items without origin",
			inventory: `[
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 1
				},
				{
					"typeId": "minecraft:bread",
					"amount": 64,
					"lore": ["Some existing lore"]
				},
				null
			]`,
			server:       "server1",
			expectModify: true,
			expectError:  false,
		},
		{
			name: "items already have origin",
			inventory: `[
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 1,
					"lore": ["Origin: server2"]
				},
				null
			]`,
			server:       "server1",
			expectModify: false,
			expectError:  false,
		},
		{
			name: "add origin to shulker contents",
			inventory: `[
				{
					"typeId": "minecraft:red_shulker_box",
					"amount": 1,
					"shulkerContents": [
						{
							"typeId": "minecraft:diamond",
							"amount": 64
						},
						{
							"typeId": "minecraft:gold_ingot",
							"amount": 32,
							"lore": ["Origin: server2"]
						}
					]
				}
			]`,
			server:       "server1",
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "empty inventory",
			inventory:    "",
			server:       "server1",
			expectModify: false,
			expectError:  false,
		},
		{
			name:         "invalid json",
			inventory:    "invalid json",
			server:       "server1",
			expectModify: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified, err := AddOriginToInventory([]byte(tt.inventory), tt.server)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectModify, modified)

			if modified {
				// Verify the result is valid JSON
				var items []any
				err := json.Unmarshal(result, &items)
				assert.NoError(t, err)

				// Verify items without existing origin now have origin
				for _, itemInterface := range items {
					if itemInterface == nil {
						continue
					}
					itemMap, ok := itemInterface.(map[string]any)
					if !ok {
						continue
					}

					// Check if item has lore
					loreInterface, exists := itemMap["lore"]
					if !exists {
						continue
					}

					loreSlice, ok := loreInterface.([]any)
					if !ok {
						continue
					}

					// Should have at least one origin line
					hasOrigin := false
					for _, l := range loreSlice {
						if str, ok := l.(string); ok {
							if strings.HasPrefix(str, "Origin: ") {
								hasOrigin = true
								break
							}
						}
					}
					assert.True(t, hasOrigin, "Item should have origin lore")
				}
			}
		})
	}
}

func TestCleanShulkerContents(t *testing.T) {
	shulkerItems := []any{
		map[string]any{
			"typeId": "minecraft:diamond",
			"amount": 64,
			"lore":   []any{"Origin: server1"},
		},
		map[string]any{
			"typeId": "minecraft:gold_ingot",
			"amount": 32,
			"lore":   []any{"Origin: server2"},
		},
		nil,
	}

	modified := cleanShulkerContents(shulkerItems, "server1")
	assert.True(t, modified)

	// First item should be nil (removed)
	assert.Nil(t, shulkerItems[0])

	// Second item should remain
	assert.NotNil(t, shulkerItems[1])
	secondItem, ok := shulkerItems[1].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "minecraft:gold_ingot", secondItem["typeId"])
}

func TestAddOriginToShulkerContents(t *testing.T) {
	shulkerItems := []any{
		map[string]any{
			"typeId": "minecraft:diamond",
			"amount": 64,
		},
		map[string]any{
			"typeId": "minecraft:gold_ingot",
			"amount": 32,
			"lore":   []any{"Origin: server2"},
		},
		nil,
	}

	modified := addOriginToShulkerContents(shulkerItems, "server1")
	assert.True(t, modified)

	// First item should now have origin
	firstItem, ok := shulkerItems[0].(map[string]any)
	assert.True(t, ok)
	lore, exists := firstItem["lore"]
	assert.True(t, exists)
	loreSlice, ok := lore.([]any)
	assert.True(t, ok)
	assert.Len(t, loreSlice, 1)
	assert.Contains(t, loreSlice[0].(string), "Origin: server1")

	// Second item should remain unchanged (already has origin)
	secondItem, ok := shulkerItems[1].(map[string]any)
	assert.True(t, ok)
	lore2, exists := secondItem["lore"]
	assert.True(t, exists)
	loreSlice2, ok := lore2.([]any)
	assert.True(t, ok)
	assert.Len(t, loreSlice2, 1)
	assert.Contains(t, loreSlice2[0].(string), "Origin: server2")
}
