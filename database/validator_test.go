package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemValidator_ValidateItem(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		item           Item
		server         string
		expectedErrors int
		errorTypes     []string
	}{
		{
			name: "valid item with origin",
			item: Item{
				TypeID: "minecraft:diamond_sword",
				Amount: 1,
				Lore:   []string{"Origin: server1"},
			},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name: "missing type",
			item: Item{
				Amount: 1,
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"missing_type"},
		},
		{
			name: "invalid amount",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 0,
			},
			server:         "server1",
			expectedErrors: 2, // invalid_amount + missing_origin
			errorTypes:     []string{"invalid_amount", "missing_origin"},
		},
		{
			name: "stack too large",
			item: Item{
				TypeID: "minecraft:diamond_sword",
				Amount: 64, // swords should stack to 1
				Lore:   []string{"Origin: server1"},
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"stack_too_large"},
		},
		{
			name: "wrong origin",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 1,
				Lore:   []string{"Origin: server2"},
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"wrong_origin"},
		},
		{
			name: "missing origin",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 1,
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"missing_origin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateItem(&tt.item, tt.server, 0)
			assert.Len(t, errors, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestItemValidator_ValidateEnchantments(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		enchantments   []map[string]any
		expectedErrors int
		errorTypes     []string
	}{
		{
			name: "valid enchantments",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 5},
				{"type": "minecraft:unbreaking", "level": 3},
			},
			expectedErrors: 0,
		},
		{
			name: "invalid level too high",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 10}, // max is 5
			},
			expectedErrors: 1,
			errorTypes:     []string{"invalid_enchantment_level"},
		},
		{
			name: "invalid level zero",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 0},
			},
			expectedErrors: 1,
			errorTypes:     []string{"invalid_enchantment_level"},
		},
		{
			name: "unknown enchantment",
			enchantments: []map[string]any{
				{"type": "minecraft:unknown_enchant", "level": 1},
			},
			expectedErrors: 1,
			errorTypes:     []string{"unknown_enchantment"},
		},
		{
			name: "incompatible enchantments",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 1},
				{"type": "minecraft:smite", "level": 1},
			},
			expectedErrors: 1,
			errorTypes:     []string{"incompatible_enchantments"},
		},
		{
			name: "duplicate enchantments",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 1},
				{"type": "minecraft:sharpness", "level": 2},
			},
			expectedErrors: 1,
			errorTypes:     []string{"duplicate_enchantment"},
		},
		{
			name: "missing type",
			enchantments: []map[string]any{
				{"level": 1},
			},
			expectedErrors: 1,
			errorTypes:     []string{"invalid_enchantment"},
		},
		{
			name: "invalid level type",
			enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": "invalid"},
			},
			expectedErrors: 1,
			errorTypes:     []string{"invalid_enchantment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateEnchantments(tt.enchantments, 0)
			assert.Len(t, errors, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestItemValidator_ValidateDurability(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		durability     map[string]any
		itemType       string
		expectedErrors int
		errorTypes     []string
	}{
		{
			name: "valid durability",
			durability: map[string]any{
				"damage":        100,
				"maxDurability": 1561,
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 0,
		},
		{
			name: "negative damage",
			durability: map[string]any{
				"damage": -10,
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 1,
			errorTypes:     []string{"negative_durability"},
		},
		{
			name: "damage exceeds max",
			durability: map[string]any{
				"damage":        2000,
				"maxDurability": 1561,
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 1,
			errorTypes:     []string{"durability_exceeds_max"},
		},
		{
			name: "invalid max durability",
			durability: map[string]any{
				"damage":        100,
				"maxDurability": 9999, // should be 1561 for diamond sword
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 1,
			errorTypes:     []string{"invalid_max_durability"},
		},
		{
			name: "invalid damage type",
			durability: map[string]any{
				"damage": "invalid",
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 1,
			errorTypes:     []string{"invalid_durability"},
		},
		{
			name: "invalid max durability type",
			durability: map[string]any{
				"maxDurability": "invalid",
			},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 1,
			errorTypes:     []string{"invalid_durability"},
		},
		{
			name:           "no durability data",
			durability:     map[string]any{},
			itemType:       "minecraft:diamond_sword",
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateDurability(tt.durability, tt.itemType, 0)
			assert.Len(t, errors, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestItemValidator_ValidateOrigin(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		lore           []string
		server         string
		expectedErrors int
		errorTypes     []string
	}{
		{
			name:           "valid origin",
			lore:           []string{"Some lore", "Origin: server1", "More lore"},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name:           "missing origin",
			lore:           []string{"Some lore", "More lore"},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"missing_origin"},
		},
		{
			name:           "wrong origin",
			lore:           []string{"Origin: server2"},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"wrong_origin"},
		},
		{
			name:           "empty lore",
			lore:           []string{},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"missing_origin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateOrigin(tt.lore, tt.server, 0)
			assert.Len(t, errors, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestItemValidator_ValidateInventory(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		inventoryJSON  string
		server         string
		player         string
		expectedErrors int
		errorTypes     []string
	}{
		{
			name: "valid inventory",
			inventoryJSON: `[
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 1,
					"lore": ["Origin: server1"]
				},
				null,
				{
					"typeId": "minecraft:bread",
					"amount": 64,
					"lore": ["Origin: server1"]
				}
			]`,
			server:         "server1",
			player:         "player1",
			expectedErrors: 0,
		},
		{
			name:           "invalid JSON",
			inventoryJSON:  "invalid json",
			server:         "server1",
			player:         "player1",
			expectedErrors: 1,
			errorTypes:     []string{"invalid_inventory"},
		},
		{
			name: "mixed valid and invalid items",
			inventoryJSON: `[
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 1,
					"lore": ["Origin: server1"]
				},
				{
					"typeId": "minecraft:diamond_sword",
					"amount": 64,
					"lore": ["Origin: server1"]
				}
			]`,
			server:         "server1",
			player:         "player1",
			expectedErrors: 1,
			errorTypes:     []string{"stack_too_large"},
		},
		{
			name: "unparseable item",
			inventoryJSON: `[
				"invalid_item_data"
			]`,
			server:         "server1",
			player:         "player1",
			expectedErrors: 1,
			errorTypes:     []string{"unparseable_item"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateInventory([]byte(tt.inventoryJSON), tt.server, tt.player)
			assert.Len(t, errors, tt.expectedErrors)

			// Check that all errors have correct player and server
			for _, err := range errors {
				assert.Equal(t, tt.player, err.Player)
				assert.Equal(t, tt.server, err.Server)
			}

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestItemValidator_ValidateShulkerContents(t *testing.T) {
	validator := NewItemValidator()

	shulkerContents := []any{
		map[string]any{
			"typeId": "minecraft:diamond",
			"amount": 1,
			"lore":   []any{"Origin: server1"},
		},
		map[string]any{
			"typeId": "minecraft:diamond_sword",
			"amount": 64, // Invalid stack size
			"lore":   []any{"Origin: server1"},
		},
		"invalid_item",
	}

	errors := validator.validateShulkerContents(shulkerContents, "server1", 0)

	// Should have errors for invalid stack size and invalid shulker content
	assert.Len(t, errors, 2)

	errorTypes := make(map[string]bool)
	for _, err := range errors {
		errorTypes[err.ErrorType] = true
	}

	assert.True(t, errorTypes["stack_too_large"])
	assert.True(t, errorTypes["invalid_shulker_content"])
}

func TestItemValidator_AddOriginToItem(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		item           Item
		server         string
		expectedModify bool
		expectedLore   []string
	}{
		{
			name: "add origin to item without lore",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 1,
			},
			server:         "server1",
			expectedModify: true,
			expectedLore:   []string{"Origin: server1"},
		},
		{
			name: "add origin to item with existing lore",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 1,
				Lore:   []string{"Some existing lore"},
			},
			server:         "server1",
			expectedModify: true,
			expectedLore:   []string{"Some existing lore", "Origin: server1"},
		},
		{
			name: "don't add origin if already exists",
			item: Item{
				TypeID: "minecraft:diamond",
				Amount: 1,
				Lore:   []string{"Origin: server2"},
			},
			server:         "server1",
			expectedModify: false,
			expectedLore:   []string{"Origin: server2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := validator.AddOriginToItem(&tt.item, tt.server)
			assert.Equal(t, tt.expectedModify, modified)
			assert.Equal(t, tt.expectedLore, tt.item.Lore)
		})
	}
}

func TestItemValidator_HasOriginFromServer(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name     string
		item     Item
		server   string
		expected bool
	}{
		{
			name: "has origin from server",
			item: Item{
				Lore: []string{"Some lore", "Origin: server1", "More lore"},
			},
			server:   "server1",
			expected: true,
		},
		{
			name: "has origin from different server",
			item: Item{
				Lore: []string{"Origin: server2"},
			},
			server:   "server1",
			expected: false,
		},
		{
			name: "no origin",
			item: Item{
				Lore: []string{"Some lore"},
			},
			server:   "server1",
			expected: false,
		},
		{
			name:     "no lore",
			item:     Item{},
			server:   "server1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.HasOriginFromServer(&tt.item, tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestItemValidator_ComplexScenarios(t *testing.T) {
	validator := NewItemValidator()

	t.Run("item with multiple validation errors", func(t *testing.T) {
		item := Item{
			TypeID: "minecraft:diamond_sword",
			Amount: 64, // Invalid stack size
			Enchantments: []map[string]any{
				{"type": "minecraft:sharpness", "level": 10}, // Invalid level
				{"type": "minecraft:smite", "level": 1},      // Incompatible with sharpness
			},
			Durability: map[string]any{
				"damage": -50, // Negative damage
			},
			Lore: []string{"Origin: server2"}, // Wrong server
		}

		errors := validator.ValidateItem(&item, "server1", 0)

		// Should have multiple errors
		assert.Greater(t, len(errors), 3)

		errorTypes := make(map[string]bool)
		for _, err := range errors {
			errorTypes[err.ErrorType] = true
		}

		assert.True(t, errorTypes["stack_too_large"])
		assert.True(t, errorTypes["invalid_enchantment_level"])
		assert.True(t, errorTypes["incompatible_enchantments"])
		assert.True(t, errorTypes["negative_durability"])
		assert.True(t, errorTypes["wrong_origin"])
	})

	t.Run("nested shulker validation", func(t *testing.T) {
		item := Item{
			TypeID: "minecraft:shulker_box",
			Amount: 1,
			Lore:   []string{"Origin: server1"},
			ShulkerContents: []any{
				map[string]any{
					"typeId": "minecraft:diamond_sword",
					"amount": 64, // Invalid stack size
					"lore":   []any{"Origin: server1"},
				},
				map[string]any{
					"typeId": "minecraft:shulker_box",
					"amount": 1,
					"lore":   []any{"Origin: server1"},
					"shulkerContents": []any{
						map[string]any{
							"typeId": "minecraft:bread",
							"amount": 0, // Invalid amount
							"lore":   []any{"Origin: server1"},
						},
					},
				},
			},
		}

		errors := validator.ValidateItem(&item, "server1", 0)

		// Should have errors from nested validation
		assert.Greater(t, len(errors), 1)

		errorTypes := make(map[string]bool)
		for _, err := range errors {
			errorTypes[err.ErrorType] = true
		}

		assert.True(t, errorTypes["stack_too_large"])
		assert.True(t, errorTypes["invalid_amount"])
	})
}

func TestMaxStackSizes(t *testing.T) {
	// Test that our max stack sizes are reasonable
	assert.Equal(t, 64, maxStackSizes["minecraft:diamond"])
	assert.Equal(t, 1, maxStackSizes["minecraft:diamond_sword"])
	assert.Equal(t, 16, maxStackSizes["minecraft:ender_pearl"])
}

func TestMaxEnchantmentLevels(t *testing.T) {
	// Test that our max enchantment levels are correct
	assert.Equal(t, 5, maxEnchantmentLevels["minecraft:sharpness"])
	assert.Equal(t, 1, maxEnchantmentLevels["minecraft:silk_touch"])
	assert.Equal(t, 3, maxEnchantmentLevels["minecraft:unbreaking"])
}

func TestIncompatibleEnchantments(t *testing.T) {
	// Test that incompatible enchantments are properly defined
	sharpnessIncompatible := incompatibleEnchantments["minecraft:sharpness"]
	assert.Contains(t, sharpnessIncompatible, "minecraft:smite")
	assert.Contains(t, sharpnessIncompatible, "minecraft:bane_of_arthropods")

	silkTouchIncompatible := incompatibleEnchantments["minecraft:silk_touch"]
	assert.Contains(t, silkTouchIncompatible, "minecraft:fortune")
}

func TestDefaultMaxDurability(t *testing.T) {
	// Test that our default max durability values are correct
	assert.Equal(t, 1561, defaultMaxDurability["minecraft:diamond_sword"])
	assert.Equal(t, 250, defaultMaxDurability["minecraft:iron_sword"])
	assert.Equal(t, 384, defaultMaxDurability["minecraft:bow"])
	
	// Test netherite durability values
	assert.Equal(t, 2031, defaultMaxDurability["minecraft:netherite_sword"])
	assert.Equal(t, 2031, defaultMaxDurability["minecraft:netherite_pickaxe"])
	assert.Equal(t, 407, defaultMaxDurability["minecraft:netherite_helmet"])
	assert.Equal(t, 592, defaultMaxDurability["minecraft:netherite_chestplate"])
}

func TestNetheriteItemValidation(t *testing.T) {
	validator := NewItemValidator()

	tests := []struct {
		name           string
		item           Item
		server         string
		expectedErrors int
		errorTypes     []string
	}{
		{
			name: "valid netherite sword",
			item: Item{
				TypeID: "minecraft:netherite_sword",
				Amount: 1,
				Lore:   []string{"Origin: server1"},
				Durability: map[string]any{
					"damage":        500,
					"maxDurability": 2031,
				},
			},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name: "invalid netherite sword stack",
			item: Item{
				TypeID: "minecraft:netherite_sword",
				Amount: 64, // Should be 1
				Lore:   []string{"Origin: server1"},
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"stack_too_large"},
		},
		{
			name: "netherite ingot valid stack",
			item: Item{
				TypeID: "minecraft:netherite_ingot",
				Amount: 64, // Should be valid
				Lore:   []string{"Origin: server1"},
			},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name: "netherite scrap valid stack",
			item: Item{
				TypeID: "minecraft:netherite_scrap",
				Amount: 64, // Should be valid
				Lore:   []string{"Origin: server1"},
			},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name: "netherite armor valid",
			item: Item{
				TypeID: "minecraft:netherite_chestplate",
				Amount: 1,
				Lore:   []string{"Origin: server1"},
				Durability: map[string]any{
					"damage":        100,
					"maxDurability": 592,
				},
			},
			server:         "server1",
			expectedErrors: 0,
		},
		{
			name: "netherite armor invalid durability",
			item: Item{
				TypeID: "minecraft:netherite_helmet",
				Amount: 1,
				Lore:   []string{"Origin: server1"},
				Durability: map[string]any{
					"damage":        100,
					"maxDurability": 9999, // Should be 407
				},
			},
			server:         "server1",
			expectedErrors: 1,
			errorTypes:     []string{"invalid_max_durability"},
		},
		{
			name: "netherite tool with valid enchantments",
			item: Item{
				TypeID: "minecraft:netherite_pickaxe",
				Amount: 1,
				Lore:   []string{"Origin: server1"},
				Enchantments: []map[string]any{
					{"type": "minecraft:efficiency", "level": 5},
					{"type": "minecraft:unbreaking", "level": 3},
					{"type": "minecraft:mending", "level": 1},
				},
				Durability: map[string]any{
					"damage":        1000,
					"maxDurability": 2031,
				},
			},
			server:         "server1",
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateItem(&tt.item, tt.server, 0)
			assert.Len(t, errors, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				errorTypeMap := make(map[string]bool)
				for _, err := range errors {
					errorTypeMap[err.ErrorType] = true
				}

				for _, expectedType := range tt.errorTypes {
					assert.True(t, errorTypeMap[expectedType], "Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func TestNetheriteStackSizes(t *testing.T) {
	// Test that netherite items have correct stack sizes
	assert.Equal(t, 64, maxStackSizes["minecraft:netherite_ingot"])
	assert.Equal(t, 64, maxStackSizes["minecraft:netherite_scrap"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_sword"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_pickaxe"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_axe"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_shovel"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_hoe"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_helmet"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_chestplate"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_leggings"])
	assert.Equal(t, 1, maxStackSizes["minecraft:netherite_boots"])
}

// TestFutureProofing tests how the validator handles unknown items from mods or future updates
func TestFutureProofing(t *testing.T) {
	validator := NewItemValidator()

	t.Run("unknown mod items with default behavior", func(t *testing.T) {
		// Test a hypothetical mod item
		modItem := Item{
			TypeID: "modname:custom_sword",
			Amount: 1,
			Lore:   []string{"Origin: server1"},
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		// Should pass validation - unknown items get default stack size of 64
		assert.Len(t, errors, 0)
	})

	t.Run("unknown mod items with large stack", func(t *testing.T) {
		// Test mod item with reasonable stack size
		modItem := Item{
			TypeID: "modname:custom_material",
			Amount: 64, // Should be allowed (default max stack)
			Lore:   []string{"Origin: server1"},
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		assert.Len(t, errors, 0)
	})

	t.Run("unknown mod items with excessive stack", func(t *testing.T) {
		// Test mod item with excessive stack size
		modItem := Item{
			TypeID: "modname:custom_material",
			Amount: 128, // Should be rejected (exceeds default max of 64)
			Lore:   []string{"Origin: server1"},
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		assert.Len(t, errors, 1)
		assert.Equal(t, "stack_too_large", errors[0].ErrorType)
	})

	t.Run("unknown enchantments are rejected", func(t *testing.T) {
		// Test item with unknown enchantment
		item := Item{
			TypeID: "minecraft:diamond_sword",
			Amount: 1,
			Lore:   []string{"Origin: server1"},
			Enchantments: []map[string]any{
				{"type": "modname:custom_enchant", "level": 1},
			},
		}

		errors := validator.ValidateItem(&item, "server1", 0)
		assert.Len(t, errors, 1)
		assert.Equal(t, "unknown_enchantment", errors[0].ErrorType)
	})

	t.Run("unknown durability items are flexible", func(t *testing.T) {
		// Test mod item with custom durability - should not validate against known values
		modItem := Item{
			TypeID: "modname:custom_tool",
			Amount: 1,
			Lore:   []string{"Origin: server1"},
			Durability: map[string]any{
				"damage":        50,
				"maxDurability": 9999, // Custom durability should be allowed
			},
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		// Should pass - unknown items don't have durability validation
		assert.Len(t, errors, 0)
	})

	t.Run("future minecraft items work with defaults", func(t *testing.T) {
		// Test hypothetical future Minecraft item
		futureItem := Item{
			TypeID: "minecraft:future_item",
			Amount: 32, // Within default stack limit
			Lore:   []string{"Origin: server1"},
		}

		errors := validator.ValidateItem(&futureItem, "server1", 0)
		assert.Len(t, errors, 0)
	})

	t.Run("mod items still require origin validation", func(t *testing.T) {
		// Test that mod items still need proper origin
		modItem := Item{
			TypeID: "modname:custom_item",
			Amount: 1,
			// Missing origin lore
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		assert.Len(t, errors, 1)
		assert.Equal(t, "missing_origin", errors[0].ErrorType)
	})

	t.Run("mod items with wrong origin are rejected", func(t *testing.T) {
		// Test that mod items with wrong origin are rejected
		modItem := Item{
			TypeID: "modname:custom_item",
			Amount: 1,
			Lore:   []string{"Origin: server2"}, // Wrong server
		}

		errors := validator.ValidateItem(&modItem, "server1", 0)
		assert.Len(t, errors, 1)
		assert.Equal(t, "wrong_origin", errors[0].ErrorType)
	})
}

// Benchmark tests
func BenchmarkItemValidator_ValidateItem(b *testing.B) {
	validator := NewItemValidator()
	item := Item{
		TypeID: "minecraft:diamond_sword",
		Amount: 1,
		Lore:   []string{"Origin: server1"},
		Enchantments: []map[string]any{
			{"type": "minecraft:sharpness", "level": 5},
			{"type": "minecraft:unbreaking", "level": 3},
		},
		Durability: map[string]any{
			"damage":        100,
			"maxDurability": 1561,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateItem(&item, "server1", 0)
	}
}

func BenchmarkItemValidator_ValidateInventory(b *testing.B) {
	validator := NewItemValidator()
	inventoryJSON := `[
		{"typeId": "minecraft:diamond_sword", "amount": 1, "lore": ["Origin: server1"]},
		{"typeId": "minecraft:bread", "amount": 64, "lore": ["Origin: server1"]},
		{"typeId": "minecraft:diamond", "amount": 32, "lore": ["Origin: server1"]},
		null, null, null
	]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateInventory([]byte(inventoryJSON), "server1", "player1")
	}
}
