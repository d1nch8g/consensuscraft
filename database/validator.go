package database

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Minecraft item validation constants and maps
var (
	// Maximum stack sizes for different item types
	maxStackSizes = map[string]int{
		// Basic materials
		"minecraft:diamond":           64,
		"minecraft:iron_ingot":        64,
		"minecraft:gold_ingot":        64,
		"minecraft:netherite_ingot":   64,
		"minecraft:netherite_scrap":   64,
		"minecraft:coal":              64,
		"minecraft:bread":             64,
		"minecraft:apple":             64,
		// Swords
		"minecraft:diamond_sword":     1,
		"minecraft:iron_sword":        1,
		"minecraft:golden_sword":      1,
		"minecraft:wooden_sword":      1,
		"minecraft:stone_sword":       1,
		"minecraft:netherite_sword":   1,
		// Pickaxes
		"minecraft:diamond_pickaxe":   1,
		"minecraft:iron_pickaxe":      1,
		"minecraft:golden_pickaxe":    1,
		"minecraft:wooden_pickaxe":    1,
		"minecraft:stone_pickaxe":     1,
		"minecraft:netherite_pickaxe": 1,
		// Axes
		"minecraft:diamond_axe":       1,
		"minecraft:iron_axe":          1,
		"minecraft:golden_axe":        1,
		"minecraft:wooden_axe":        1,
		"minecraft:stone_axe":         1,
		"minecraft:netherite_axe":     1,
		// Shovels
		"minecraft:diamond_shovel":    1,
		"minecraft:iron_shovel":       1,
		"minecraft:golden_shovel":     1,
		"minecraft:wooden_shovel":     1,
		"minecraft:stone_shovel":      1,
		"minecraft:netherite_shovel":  1,
		// Hoes
		"minecraft:diamond_hoe":       1,
		"minecraft:iron_hoe":          1,
		"minecraft:golden_hoe":        1,
		"minecraft:wooden_hoe":        1,
		"minecraft:stone_hoe":         1,
		"minecraft:netherite_hoe":     1,
		// Armor
		"minecraft:netherite_helmet":     1,
		"minecraft:netherite_chestplate": 1,
		"minecraft:netherite_leggings":   1,
		"minecraft:netherite_boots":      1,
		// Other tools
		"minecraft:bow":               1,
		"minecraft:crossbow":          1,
		"minecraft:shield":            1,
		"minecraft:bucket":            16,
		"minecraft:water_bucket":      1,
		"minecraft:lava_bucket":       1,
		"minecraft:milk_bucket":       1,
		"minecraft:ender_pearl":       16,
		"minecraft:snowball":          16,
		"minecraft:egg":               16,
		"minecraft:potion":            1,
		"minecraft:splash_potion":     1,
		"minecraft:lingering_potion":  1,
	}

	// Valid enchantments and their maximum levels
	maxEnchantmentLevels = map[string]int{
		"minecraft:sharpness":          5,
		"minecraft:smite":              5,
		"minecraft:bane_of_arthropods": 5,
		"minecraft:knockback":          2,
		"minecraft:fire_aspect":        2,
		"minecraft:looting":            3,
		"minecraft:sweeping":           3,
		"minecraft:efficiency":         5,
		"minecraft:silk_touch":         1,
		"minecraft:unbreaking":         3,
		"minecraft:fortune":            3,
		"minecraft:power":              5,
		"minecraft:punch":              2,
		"minecraft:flame":              1,
		"minecraft:infinity":           1,
		"minecraft:luck_of_the_sea":    3,
		"minecraft:lure":               3,
		"minecraft:loyalty":            3,
		"minecraft:impaling":           5,
		"minecraft:riptide":            3,
		"minecraft:channeling":         1,
		"minecraft:multishot":          1,
		"minecraft:quick_charge":       3,
		"minecraft:piercing":           4,
		"minecraft:mending":            1,
		"minecraft:protection":         4,
		"minecraft:fire_protection":    4,
		"minecraft:feather_falling":    4,
		"minecraft:blast_protection":   4,
		"minecraft:projectile_protection": 4,
		"minecraft:respiration":        3,
		"minecraft:aqua_affinity":      1,
		"minecraft:thorns":             3,
		"minecraft:depth_strider":      3,
		"minecraft:frost_walker":       2,
		"minecraft:soul_speed":         3,
		"minecraft:swift_sneak":        3,
	}

	// Incompatible enchantment groups
	incompatibleEnchantments = map[string][]string{
		"minecraft:sharpness":          {"minecraft:smite", "minecraft:bane_of_arthropods"},
		"minecraft:smite":              {"minecraft:sharpness", "minecraft:bane_of_arthropods"},
		"minecraft:bane_of_arthropods": {"minecraft:sharpness", "minecraft:smite"},
		"minecraft:silk_touch":         {"minecraft:fortune"},
		"minecraft:fortune":            {"minecraft:silk_touch"},
		"minecraft:infinity":           {"minecraft:mending"},
		"minecraft:mending":            {"minecraft:infinity"},
		"minecraft:loyalty":            {"minecraft:riptide"},
		"minecraft:riptide":            {"minecraft:loyalty"},
		"minecraft:multishot":          {"minecraft:piercing"},
		"minecraft:piercing":           {"minecraft:multishot"},
		"minecraft:protection":         {"minecraft:fire_protection", "minecraft:blast_protection", "minecraft:projectile_protection"},
		"minecraft:fire_protection":    {"minecraft:protection", "minecraft:blast_protection", "minecraft:projectile_protection"},
		"minecraft:blast_protection":   {"minecraft:protection", "minecraft:fire_protection", "minecraft:projectile_protection"},
		"minecraft:projectile_protection": {"minecraft:protection", "minecraft:fire_protection", "minecraft:blast_protection"},
		"minecraft:depth_strider":      {"minecraft:frost_walker"},
		"minecraft:frost_walker":       {"minecraft:depth_strider"},
	}

	// Default maximum durability for items
	defaultMaxDurability = map[string]int{
		// Swords
		"minecraft:diamond_sword":     1561,
		"minecraft:iron_sword":        250,
		"minecraft:golden_sword":      32,
		"minecraft:wooden_sword":      59,
		"minecraft:stone_sword":       131,
		"minecraft:netherite_sword":   2031,
		// Pickaxes
		"minecraft:diamond_pickaxe":   1561,
		"minecraft:iron_pickaxe":      250,
		"minecraft:golden_pickaxe":    32,
		"minecraft:wooden_pickaxe":    59,
		"minecraft:stone_pickaxe":     131,
		"minecraft:netherite_pickaxe": 2031,
		// Axes
		"minecraft:diamond_axe":       1561,
		"minecraft:iron_axe":          250,
		"minecraft:golden_axe":        32,
		"minecraft:wooden_axe":        59,
		"minecraft:stone_axe":         131,
		"minecraft:netherite_axe":     2031,
		// Shovels
		"minecraft:diamond_shovel":    1561,
		"minecraft:iron_shovel":       250,
		"minecraft:golden_shovel":     32,
		"minecraft:wooden_shovel":     59,
		"minecraft:stone_shovel":      131,
		"minecraft:netherite_shovel":  2031,
		// Hoes
		"minecraft:diamond_hoe":       1561,
		"minecraft:iron_hoe":          250,
		"minecraft:golden_hoe":        32,
		"minecraft:wooden_hoe":        59,
		"minecraft:stone_hoe":         131,
		"minecraft:netherite_hoe":     2031,
		// Armor - Diamond
		"minecraft:diamond_helmet":    363,
		"minecraft:diamond_chestplate": 528,
		"minecraft:diamond_leggings":  495,
		"minecraft:diamond_boots":     429,
		// Armor - Iron
		"minecraft:iron_helmet":       165,
		"minecraft:iron_chestplate":   240,
		"minecraft:iron_leggings":     225,
		"minecraft:iron_boots":        195,
		// Armor - Netherite
		"minecraft:netherite_helmet":     407,
		"minecraft:netherite_chestplate": 592,
		"minecraft:netherite_leggings":   555,
		"minecraft:netherite_boots":      481,
		// Other tools
		"minecraft:bow":               384,
		"minecraft:crossbow":          326,
		"minecraft:shield":            336,
	}
)

// ItemValidator provides validation functionality for Minecraft items
type ItemValidator struct{}

// NewItemValidator creates a new item validator
func NewItemValidator() *ItemValidator {
	return &ItemValidator{}
}

// ValidateInventory validates an entire inventory for a specific server
func (v *ItemValidator) ValidateInventory(inventoryData []byte, server, player string) []ValidationError {
	var inventory []any
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		return []ValidationError{{
			Player:    player,
			Server:    server,
			ItemIndex: -1,
			ErrorType: "invalid_inventory",
			Message:   "Failed to parse inventory JSON",
		}}
	}

	var allErrors []ValidationError
	for i, slot := range inventory {
		if slot == nil {
			continue
		}

		// Try to parse as Item
		slotBytes, err := json.Marshal(slot)
		if err != nil {
			allErrors = append(allErrors, ValidationError{
				Player:    player,
				Server:    server,
				ItemIndex: i,
				ErrorType: "invalid_item_data",
				Message:   "Item contains invalid data",
			})
			continue
		}

		var item Item
		if err := json.Unmarshal(slotBytes, &item); err != nil {
			allErrors = append(allErrors, ValidationError{
				Player:    player,
				Server:    server,
				ItemIndex: i,
				ErrorType: "unparseable_item",
				Message:   "Item data cannot be parsed",
			})
			continue
		}

		// Validate the item
		itemErrors := v.ValidateItem(&item, server, i)
		for _, itemError := range itemErrors {
			itemError.Player = player
			itemError.Server = server
			allErrors = append(allErrors, itemError)
		}
	}

	return allErrors
}

// ValidateItem performs comprehensive validation on a Minecraft item
func (v *ItemValidator) ValidateItem(item *Item, server string, itemIndex int) []ValidationError {
	var errors []ValidationError

	// Validate item type
	if item.TypeID == "" {
		errors = append(errors, ValidationError{
			ItemIndex: itemIndex,
			ErrorType: "missing_type",
			Message:   "Item missing typeId",
		})
		return errors // Can't validate further without type
	}

	// Validate stack size
	if item.Amount <= 0 {
		errors = append(errors, ValidationError{
			ItemIndex: itemIndex,
			ErrorType: "invalid_amount",
			Message:   "Item amount must be positive",
		})
	} else {
		maxStack := maxStackSizes[item.TypeID]
		if maxStack == 0 {
			maxStack = 64 // Default max stack size
		}
		if item.Amount > maxStack {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "stack_too_large",
				Message:   fmt.Sprintf("Stack size %d exceeds maximum %d for %s", item.Amount, maxStack, item.TypeID),
			})
		}
	}

	// Validate enchantments
	if len(item.Enchantments) > 0 {
		enchantmentErrors := v.validateEnchantments(item.Enchantments, itemIndex)
		errors = append(errors, enchantmentErrors...)
	}

	// Validate durability
	if item.Durability != nil {
		durabilityErrors := v.validateDurability(item.Durability, item.TypeID, itemIndex)
		errors = append(errors, durabilityErrors...)
	}

	// Validate origin lore
	originErrors := v.validateOrigin(item.Lore, server, itemIndex)
	errors = append(errors, originErrors...)

	// Recursively validate shulker contents
	if len(item.ShulkerContents) > 0 {
		shulkerErrors := v.validateShulkerContents(item.ShulkerContents, server, itemIndex)
		errors = append(errors, shulkerErrors...)
	}

	return errors
}

// validateEnchantments validates enchantment combinations and levels
func (v *ItemValidator) validateEnchantments(enchantments []map[string]any, itemIndex int) []ValidationError {
	var errors []ValidationError
	seenEnchantments := make(map[string]int)

	for enchIdx, enchant := range enchantments {
		enchType, hasType := enchant["type"].(string)
		if !hasType {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_enchantment",
				Message:   fmt.Sprintf("Enchantment %d missing type", enchIdx),
			})
			continue
		}

		var level int
		if levelFloat, ok := enchant["level"].(float64); ok {
			level = int(levelFloat)
		} else if levelInt, ok := enchant["level"].(int); ok {
			level = levelInt
		} else {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_enchantment",
				Message:   fmt.Sprintf("Enchantment %s has invalid level", enchType),
			})
			continue
		}

		// Check level bounds
		maxLevel := maxEnchantmentLevels[enchType]
		if maxLevel == 0 {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "unknown_enchantment",
				Message:   fmt.Sprintf("Unknown enchantment: %s", enchType),
			})
			continue
		}

		if level <= 0 || level > maxLevel {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_enchantment_level",
				Message:   fmt.Sprintf("Enchantment %s level %d is invalid (max: %d)", enchType, level, maxLevel),
			})
		}

		// Check for duplicates
		if _, exists := seenEnchantments[enchType]; exists {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "duplicate_enchantment",
				Message:   fmt.Sprintf("Duplicate enchantment: %s", enchType),
			})
		}
		seenEnchantments[enchType] = level

		// Check incompatible enchantments
		if incompatible, exists := incompatibleEnchantments[enchType]; exists {
			for _, incompatibleEnch := range incompatible {
				if _, hasIncompatible := seenEnchantments[incompatibleEnch]; hasIncompatible {
					errors = append(errors, ValidationError{
						ItemIndex: itemIndex,
						ErrorType: "incompatible_enchantments",
						Message:   fmt.Sprintf("Incompatible enchantments: %s and %s", enchType, incompatibleEnch),
					})
				}
			}
		}
	}

	return errors
}

// validateDurability validates item durability values
func (v *ItemValidator) validateDurability(durability map[string]any, itemType string, itemIndex int) []ValidationError {
	var errors []ValidationError

	damage, hasDamage := durability["damage"]
	maxDur, hasMaxDur := durability["maxDurability"]

	if !hasDamage && !hasMaxDur {
		return errors // No durability data to validate
	}

	var damageInt, maxDurInt int

	if hasDamage {
		if damageFloat, ok := damage.(float64); ok {
			damageInt = int(damageFloat)
		} else if damageIntVal, ok := damage.(int); ok {
			damageInt = damageIntVal
		} else {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_durability",
				Message:   "Durability damage must be a number",
			})
			return errors
		}
	}

	if hasMaxDur {
		if maxDurFloat, ok := maxDur.(float64); ok {
			maxDurInt = int(maxDurFloat)
		} else if maxDurIntVal, ok := maxDur.(int); ok {
			maxDurInt = maxDurIntVal
		} else {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_durability",
				Message:   "Durability maxDurability must be a number",
			})
			return errors
		}
	}

	// Validate damage is not negative
	if damageInt < 0 {
		errors = append(errors, ValidationError{
			ItemIndex: itemIndex,
			ErrorType: "negative_durability",
			Message:   fmt.Sprintf("Durability damage cannot be negative: %d", damageInt),
		})
	}

	// Validate max durability against known values
	if hasMaxDur {
		expectedMaxDur := defaultMaxDurability[itemType]
		if expectedMaxDur > 0 && maxDurInt != expectedMaxDur {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "invalid_max_durability",
				Message:   fmt.Sprintf("Invalid max durability %d for %s (expected: %d)", maxDurInt, itemType, expectedMaxDur),
			})
		}

		// Validate damage doesn't exceed max durability
		if damageInt > maxDurInt {
			errors = append(errors, ValidationError{
				ItemIndex: itemIndex,
				ErrorType: "durability_exceeds_max",
				Message:   fmt.Sprintf("Durability damage %d exceeds max durability %d", damageInt, maxDurInt),
			})
		}
	}

	return errors
}

// validateOrigin validates that items have proper origin lore for the server
func (v *ItemValidator) validateOrigin(lore []string, server string, itemIndex int) []ValidationError {
	var errors []ValidationError
	
	// Simple origin pattern: "Origin: server"
	originPattern := regexp.MustCompile(`^Origin:\s+(.+)$`)
	hasOrigin := false
	var originServer string

	for _, line := range lore {
		if matches := originPattern.FindStringSubmatch(line); len(matches) == 2 {
			hasOrigin = true
			originServer = strings.TrimSpace(matches[1])
			break
		}
	}

	if !hasOrigin {
		errors = append(errors, ValidationError{
			ItemIndex: itemIndex,
			ErrorType: "missing_origin",
			Message:   "Item missing origin lore",
		})
	} else if originServer != server {
		errors = append(errors, ValidationError{
			ItemIndex: itemIndex,
			ErrorType: "wrong_origin",
			Message:   fmt.Sprintf("Item origin '%s' doesn't match server '%s'", originServer, server),
		})
	}

	return errors
}

// validateShulkerContents recursively validates items in shulker boxes
func (v *ItemValidator) validateShulkerContents(contents []any, server string, parentIndex int) []ValidationError {
	var errors []ValidationError

	for i, content := range contents {
		if content == nil {
			continue
		}

		// Try to parse as Item
		contentBytes, err := json.Marshal(content)
		if err != nil {
			errors = append(errors, ValidationError{
				ItemIndex: parentIndex,
				ErrorType: "invalid_shulker_content",
				Message:   fmt.Sprintf("Shulker slot %d contains invalid data", i),
			})
			continue
		}

		var item Item
		if err := json.Unmarshal(contentBytes, &item); err != nil {
			errors = append(errors, ValidationError{
				ItemIndex: parentIndex,
				ErrorType: "invalid_shulker_content",
				Message:   fmt.Sprintf("Shulker slot %d contains unparseable item", i),
			})
			continue
		}

		// Validate the nested item
		itemErrors := v.ValidateItem(&item, server, parentIndex)
		for _, itemError := range itemErrors {
			itemError.Message = fmt.Sprintf("Shulker slot %d: %s", i, itemError.Message)
			errors = append(errors, itemError)
		}
	}

	return errors
}

// AddOriginToItem adds origin lore to an item if it doesn't have one
func (v *ItemValidator) AddOriginToItem(item *Item, server string) bool {
	// Check if item already has any origin
	originPattern := regexp.MustCompile(`^Origin:\s+(.+)$`)
	hasOrigin := false

	for _, line := range item.Lore {
		if originPattern.MatchString(line) {
			hasOrigin = true
			break
		}
	}

	if !hasOrigin {
		originLine := fmt.Sprintf("Origin: %s", server)
		item.Lore = append(item.Lore, originLine)
		return true
	}

	return false
}

// HasOriginFromServer checks if an item originates from a specific server
func (v *ItemValidator) HasOriginFromServer(item *Item, server string) bool {
	if len(item.Lore) == 0 {
		return false
	}

	// Simple origin pattern: "Origin: server"
	originPattern := regexp.MustCompile(`^Origin:\s+(.+)$`)
	for _, lore := range item.Lore {
		if matches := originPattern.FindStringSubmatch(lore); len(matches) == 2 {
			originServer := strings.TrimSpace(matches[1])
			if originServer == server {
				return true
			}
		}
	}
	return false
}
