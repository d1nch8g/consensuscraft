import { world, system } from "@minecraft/server";

// Track processed players to ensure one-time restoration only
const processedPlayers = new Set();

// Listen for player spawn events (login)
world.afterEvents.playerSpawn.subscribe((event) => {
    const player = event.player;
    const playerId = player.id;


    // Run restoration exactly 1 second after login
    system.runTimeout(() => {
        // Only process each player once per session
        if (processedPlayers.has(playerId)) {
            return;
        }

        processedPlayers.add(playerId);
        performOneTimeRestoration(player);

    }, 20); // 1 second = 20 ticks
});

// Clean up processed players when they leave
world.afterEvents.playerLeave.subscribe((event) => {
    const playerId = event.playerId;
    processedPlayers.delete(playerId);
});

/**
 * Perform one-time restoration check for a player
 * This runs exactly once, 1 second after player login
 */
function performOneTimeRestoration(player) {
    try {

        const tags = player.getTags();
        const inventoryTags = tags.filter(tag => tag.startsWith("restore_inv_"));

        if (inventoryTags.length > 0) {
            restoreInventoryFromTags(player, inventoryTags);
        } else {
            cleanPlayerEnderChest(player);
        }

    } catch (error) {
        console.log(`Error in one-time restoration for ${player.name}:`, error.message);
    }
}

/**
 * Restore inventory from player tags and create virtual shulker storage
 */
function restoreInventoryFromTags(player, inventoryTags) {
    try {
        // Sort tags by part number (restore_inv_0, restore_inv_1, etc.)
        const sortedTags = inventoryTags.sort((a, b) => {
            const aNum = parseInt(a.match(/restore_inv_(\d+)_/)?.[1] || "0");
            const bNum = parseInt(b.match(/restore_inv_(\d+)_/)?.[1] || "0");
            return aNum - bNum;
        });

        // Reconstruct the JSON string from chunks
        let jsonString = "";
        for (const tag of sortedTags) {
            // Extract data part after restore_inv_{number}_
            const parts = tag.split('_');
            if (parts.length >= 3) {
                // Join everything after the third part (restore_inv_{number}_DATA)
                const dataPart = parts.slice(3).join('_');
                jsonString += dataPart;
            }
        }

        // Clean up the JSON string - remove any trailing commas or invalid characters
        jsonString = jsonString.trim();

        // Ensure the JSON string is properly formatted
        if (!jsonString.startsWith('[')) {
            jsonString = '[' + jsonString;
        }
        if (!jsonString.endsWith(']')) {
            // Remove trailing comma if present
            jsonString = jsonString.replace(/,$/, '');
            jsonString = jsonString + ']';
        }

        console.log(`Attempting to parse JSON for ${player.name}: ${jsonString.substring(0, 200)}...`);

        // Parse the inventory data
        const inventoryData = JSON.parse(jsonString);

        // Validate that we got an array
        if (!Array.isArray(inventoryData)) {
            throw new Error("Parsed data is not an array");
        }

        // Process the inventory and create virtual shulker storage
        const processedInventory = processInventoryWithShulkers(inventoryData);

        // Save to ender chest storage
        const playerId = player.id;
        world.setDynamicProperty(`enderchest_${playerId}`, JSON.stringify(processedInventory));

        // Clean up restoration tags
        for (const tag of inventoryTags) {
            player.removeTag(tag);
        }

        // Notify player
        player.sendMessage(`§aYour ender chest inventory has been restored from backup!`);
        console.log(`Successfully restored inventory for ${player.name} with ${inventoryData.length} slots`);

    } catch (error) {
        console.log(`Error restoring inventory for ${player.name}: ${error.message}`);
        console.log(`Available tags: ${inventoryTags.join(', ')}`);

        // Clean up tags even on error
        try {
            for (const tag of inventoryTags) {
                player.removeTag(tag);
            }
        } catch (e) {
            console.log(`Error cleaning up tags: ${e.message}`);
        }

        // Set empty inventory as fallback
        try {
            const playerId = player.id;
            const emptyInventory = new Array(45).fill(null);
            world.setDynamicProperty(`enderchest_${playerId}`, JSON.stringify(emptyInventory));
            player.sendMessage(`§cInventory restoration failed, starting with empty ender chest.`);
        } catch (fallbackError) {
            console.log(`Error setting fallback inventory: ${fallbackError.message}`);
        }
    }
}

/**
 * Process inventory data and create virtual shulker storage for nested shulker boxes
 */
function processInventoryWithShulkers(inventoryData) {
    const processedInventory = new Array(45).fill(null);

    if (!Array.isArray(inventoryData)) {
        return processedInventory;
    }

    for (let i = 0; i < Math.min(inventoryData.length, 45); i++) {
        const item = inventoryData[i];

        if (!item || !item.typeId) {
            processedInventory[i] = null;
            continue;
        }

        // Convert to ender chest format
        const formattedItem = {
            typeId: item.typeId,
            amount: item.amount || 1,
            nameTag: item.nameTag || undefined,
            lore: item.lore || [],
            enchantments: item.enchantments || [],
            durability: item.durability || null
        };

        // Handle shulker boxes - create virtual storage
        if (item.shulkerContents && isShulkerBox(item.typeId)) {
            const shulkerId = generateShulkerId();

            // Store shulker contents in world dynamic properties (same as shulker_box.js)
            world.setDynamicProperty(shulkerId, JSON.stringify(item.shulkerContents));

            // Add shulker ID to lore so it can be retrieved when placed
            if (!formattedItem.lore) formattedItem.lore = [];
            formattedItem.lore.push(`ID: ${shulkerId}`);

        }

        processedInventory[i] = formattedItem;
    }

    return processedInventory;
}

/**
 * Clean player's ender chest (called when no restoration tags found)
 */
function cleanPlayerEnderChest(player) {
    try {
        const playerId = player.id;
        const emptyInventory = new Array(45).fill(null);

        world.setDynamicProperty(`enderchest_${playerId}`, JSON.stringify(emptyInventory));

        player.sendMessage(`§7Your ender chest has been cleaned.`);

    } catch (error) {
        console.log(`Error cleaning ender chest for ${player.name}:`, error.message);
    }
}

/**
 * Check if item type is a shulker box
 */
function isShulkerBox(typeId) {
    const SHULKER_BOX_TYPES = [
        "minecraft:shulker_box",
        "minecraft:undyed_shulker_box",
        "minecraft:white_shulker_box",
        "minecraft:orange_shulker_box",
        "minecraft:magenta_shulker_box",
        "minecraft:light_blue_shulker_box",
        "minecraft:yellow_shulker_box",
        "minecraft:lime_shulker_box",
        "minecraft:pink_shulker_box",
        "minecraft:gray_shulker_box",
        "minecraft:light_gray_shulker_box",
        "minecraft:cyan_shulker_box",
        "minecraft:purple_shulker_box",
        "minecraft:blue_shulker_box",
        "minecraft:brown_shulker_box",
        "minecraft:green_shulker_box",
        "minecraft:red_shulker_box",
        "minecraft:black_shulker_box",
    ];
    return SHULKER_BOX_TYPES.includes(typeId);
}

/**
 * Generate UUID v4 for shulker box storage (same as shulker_box.js)
 */
function generateShulkerId() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

/**
 * Count non-null items in inventory
 */
function countItems(inventory) {
    return inventory.filter(item => item !== null).length;
}
