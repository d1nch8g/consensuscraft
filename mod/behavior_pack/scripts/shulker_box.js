import { world, system, ItemStack, EnchantmentTypes } from "@minecraft/server";

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

function isShulkerBox(typeId) {
    return SHULKER_BOX_TYPES.includes(typeId);
}

// Generate UUID v4 for shulker box contents
function generateShulkerId(contents) {
    // Generate UUID v4
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

// Get shulker ID from item (stored in lore)
function getShulkerIdFromItem(item) {
    try {
        const lore = item.getLore();
        for (const line of lore) {
            if (line.startsWith("ID: ")) {
                const id = line.substring("ID: ".length).trim();
                return id;
            }
        }
    } catch (e) { }
    return null;
}

// Set shulker ID on item (stored in lore)
function setShulkerIdOnItem(item, shulkerId) {
    try {
        let lore = item.getLore() || [];
        // Remove any existing ID line
        lore = lore.filter(line => !line.startsWith("ID: "));
        // Add new ID line at the end
        lore.push(`ID: ${shulkerId}`);
        item.setLore(lore);
        return true;
    } catch (e) {
        return false;
    }
}

function serializeItem(item) {
    if (!item) return null;

    const serialized = {
        typeId: item.typeId,
        amount: item.amount,
    };

    if (item.nameTag) serialized.nameTag = item.nameTag;

    try {
        const lore = item.getLore();
        if (lore && lore.length > 0) serialized.lore = lore;
    } catch (e) { }

    try {
        const durability = item.getComponent("minecraft:durability");
        if (durability) {
            serialized.durability = {
                damage: durability.damage,
                maxDurability: durability.maxDurability,
            };
        }
    } catch (e) { }

    try {
        const enchantable = item.getComponent("minecraft:enchantable");
        if (enchantable) {
            const enchantments = enchantable.getEnchantments();
            if (enchantments && enchantments.length > 0) {
                serialized.enchantments = enchantments.map((ench) => ({
                    type: ench.type.id,
                    level: ench.level,
                }));
            }
        }
    } catch (e) { }

    // For shulker boxes, include their stored contents
    if (isShulkerBox(item.typeId)) {
        try {
            const shulkerId = getShulkerIdFromItem(item);
            if (shulkerId) {
                const contentsData = world.getDynamicProperty(shulkerId);
                if (contentsData) {
                    const contents = JSON.parse(contentsData);
                    serialized.shulker_contents = contents;
                }
            }
        } catch (e) { }
    }

    return serialized;
}

function deserializeItem(itemData) {
    if (!itemData) return null;

    try {
        const item = new ItemStack(itemData.typeId, itemData.amount);

        if (itemData.nameTag) item.nameTag = itemData.nameTag;
        if (itemData.lore && itemData.lore.length > 0) item.setLore(itemData.lore);

        if (itemData.durability) {
            const durabilityComponent = item.getComponent("minecraft:durability");
            if (durabilityComponent) {
                durabilityComponent.damage = itemData.durability.damage;
            }
        }

        if (itemData.enchantments && itemData.enchantments.length > 0) {
            const enchantable = item.getComponent("minecraft:enchantable");
            if (enchantable) {
                itemData.enchantments.forEach((enchData) => {
                    try {
                        const enchantmentType = EnchantmentTypes.get(enchData.type);
                        if (enchantmentType) {
                            enchantable.addEnchantment({
                                type: enchantmentType,
                                level: enchData.level
                            });
                        }
                    } catch (e) { }
                });
            }
        }

        return item;
    } catch (error) {
        return null;
    }
}

// Restore shulker box contents from stored data
function restoreShulkerContents(shulkerItem, container) {
    try {
        const shulkerId = getShulkerIdFromItem(shulkerItem);
        if (!shulkerId) {
            return;
        }

        restoreShulkerContentsByID(shulkerId, container);
    } catch (error) { }
}

// Restore shulker box contents by ID
function restoreShulkerContentsByID(shulkerId, container) {
    try {
        const contentsData = world.getDynamicProperty(shulkerId);
        if (!contentsData) {
            return;
        }

        const contents = JSON.parse(contentsData);

        // Restore items to container
        for (let slot = 0; slot < Math.min(contents.length, container.size); slot++) {
            if (contents[slot]) {
                const item = deserializeItem(contents[slot]);
                if (item) {
                    container.setItem(slot, item);
                }
            }
        }
    } catch (error) { }
}

// Shulker box persistence system is now active!

// Replace dropped shulker box with one that has proper lore
function replaceDroppedShulkerWithLore(watchData) {
    const { location, typeId, shulkerId, dimension } = watchData;
    const searchRadius = 1.5;

    // Wait exactly 1 tick, then perform the replacement transaction
    system.runTimeout(() => {
        // Search for dropped shulker box item entities
        const nearbyItems = dimension.getEntities({
            type: "minecraft:item",
            location: location,
            maxDistance: searchRadius
        });

        // Count candidates: untagged shulkers of the correct type
        const candidates = [];
        for (const itemEntity of nearbyItems) {
            const itemStack = itemEntity.getComponent("minecraft:item")?.itemStack;
            if (itemStack && itemStack.typeId === typeId && !getShulkerIdFromItem(itemStack)) {
                candidates.push(itemEntity);
            }
        }

        // Safety check: only proceed if exactly 1 candidate
        if (candidates.length === 1) {
            const itemEntity = candidates[0];
            const itemStack = itemEntity.getComponent("minecraft:item").itemStack;

            // Create a new ItemStack with the same properties but add lore
            const newItemStack = new ItemStack(itemStack.typeId, itemStack.amount);

            // Copy over existing properties
            if (itemStack.nameTag) newItemStack.nameTag = itemStack.nameTag;

            // Set the shulker ID in lore
            if (setShulkerIdOnItem(newItemStack, shulkerId)) {
                // Get the entity's position and motion
                const entityLocation = itemEntity.location;
                const entityVelocity = itemEntity.getVelocity();

                // Remove the old entity
                itemEntity.remove();

                // Spawn new item entity at the same location
                const newItemEntity = dimension.spawnItem(newItemStack, entityLocation);

                // Apply the same velocity if it had any
                if (entityVelocity && (entityVelocity.x !== 0 || entityVelocity.y !== 0 || entityVelocity.z !== 0)) {
                    newItemEntity.applyImpulse(entityVelocity);
                }
            }
        }
    }, 1); // Execute after exactly 1 tick
}

// SHULKER BOX PLACED - restore contents and add emerald to last slot
world.afterEvents.playerPlaceBlock.subscribe((event) => {
    const { player, block } = event;

    if (isShulkerBox(block.typeId)) {
        system.runTimeout(() => {
            const inventory = block.getComponent("minecraft:inventory");
            if (inventory && inventory.container) {
                const container = inventory.container;

                let foundShulkerId = null;
                const playerInventory = player.getComponent("minecraft:inventory").container;

                // First: Check remaining shulker boxes in inventory for lore tags
                for (let slot = 0; slot < playerInventory.size; slot++) {
                    const item = playerInventory.getItem(slot);
                    if (item && item.typeId === block.typeId) {
                        const shulkerId = getShulkerIdFromItem(item);
                        if (shulkerId) {
                            foundShulkerId = shulkerId;
                            // Remove the lore from this item since it was placed
                            let lore = item.getLore() || [];
                            lore = lore.filter(line => !line.startsWith("ID: "));
                            item.setLore(lore);
                            playerInventory.setItem(slot, item);
                            break;
                        }
                    }
                }

                // Fallback: Look for recent shulker data (extended time window for cross-session)
                if (!foundShulkerId) {

                    try {
                        const allProps = world.getDynamicPropertyIds();
                        for (const propId of allProps) {
                            // Check if it's a valid UUID v4 format
                            if (/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(propId)) {
                                const data = world.getDynamicProperty(propId);
                                if (data) {
                                    // For UUID-based IDs, we'll just use the most recent one
                                    // Since we can't extract timestamp from UUID, we'll check all recent ones
                                    foundShulkerId = propId;
                                    break; // Use the first valid one found
                                }
                            }
                        }
                    } catch (e) { }
                }

                // Restore contents if shulker ID found
                if (foundShulkerId) {
                    restoreShulkerContentsByID(foundShulkerId, container);
                }
            }
        }, 5);
    }
});

// SHULKER BOX BROKEN - capture contents and store in item
world.beforeEvents.playerBreakBlock.subscribe((event) => {
    const { player, block } = event;

    if (isShulkerBox(block.typeId)) {
        const inventory = block.getComponent("minecraft:inventory");
        if (inventory && inventory.container) {
            const container = inventory.container;
            const contents = [];

            for (let slot = 0; slot < container.size; slot++) {
                const item = container.getItem(slot);
                contents[slot] = serializeItem(item);
            }

            // Generate UUID v4 ID for this shulker box
            const shulkerId = generateShulkerId(contents);

            // Store contents in world dynamic properties
            try {
                world.setDynamicProperty(shulkerId, JSON.stringify(contents));

                // Replace the dropped item with one that has proper lore
                replaceDroppedShulkerWithLore({
                    location: block.location,
                    typeId: block.typeId,
                    shulkerId: shulkerId,
                    player: player,
                    dimension: player.dimension
                });

            } catch (error) { }
        }
    }
});

// Old afterBreakBlock approach removed - now using the watcher system

// Export for use in other modules
export { serializeItem, deserializeItem, isShulkerBox, restoreShulkerContents, getShulkerIdFromItem };
