import { system, world, ItemStack, EnchantmentTypes } from "@minecraft/server";
import { serializeItem, deserializeItem, isShulkerBox, getShulkerIdFromItem } from "./shulker_box.js";

const chests = ["x_ender_chest"];

// Per-player ender chest storage
const enderChestStorage = new Map();

// Save player data to world dynamic properties
function savePlayerData(playerId, items) {
    try {
        const itemsData = items.map((item) => {
            if (!item) return null;
            return {
                typeId: item.typeId,
                amount: item.amount,
                nameTag: item.nameTag,
                lore: item.getLore(),
                enchantments: (() => {
                    const enchantable = item.getComponent("enchantable");
                    if (enchantable) {
                        const enchantments = enchantable.getEnchantments();
                        return enchantments.map((ench) => ({
                            type: ench.type.id,
                            level: ench.level,
                        }));
                    }
                    return [];
                })(),
                durability: (() => {
                    const durability = item.getComponent("minecraft:durability");
                    return durability ? {
                        damage: durability.damage,
                        maxDurability: durability.maxDurability,
                    } : null;
                })(),
            };
        });
        world.setDynamicProperty(
            `enderchest_${playerId}`,
            JSON.stringify(itemsData)
        );
    } catch (error) { }
}

// Load player data from world dynamic properties
function loadPlayerData(playerId) {
    try {
        const data = world.getDynamicProperty(`enderchest_${playerId}`);
        if (!data) return new Array(45).fill(null);

        const itemsData = JSON.parse(data);
        return itemsData.map((itemData) => {
            return deserializeItem(itemData);
        });
    } catch (error) {
        return new Array(45).fill(null);
    }
}

system.beforeEvents.startup.subscribe((initEvent) => {
    initEvent.blockComponentRegistry.registerCustomComponent(
        "x_ender_chest:x_ender_chest",
        {
            onTick(event) {
                const block = event.block;
                const dimension = event.dimension;

                const findChest = chests.find(
                    (chest) => block.typeId === `x_ender_chest:${chest}`
                );
                const entities = dimension.getEntities({
                    location: block.center(),
                    type: `x_ender_chest:${findChest}`,
                    maxDistance: 0.75,
                });
                const chestEntity = entities[0];
                if (!chestEntity) return;

                // Set directional tags for the chest entity
                const direction = block.permutation.getState(
                    "minecraft:cardinal_direction"
                );
                if (direction) {
                    switch (direction) {
                        case "north":
                            chestEntity.addTag("north");
                            break;
                        case "south":
                            chestEntity.addTag("south");
                            break;
                        case "west":
                            chestEntity.addTag("west");
                            break;
                        case "east":
                            chestEntity.addTag("east");
                            break;
                    }
                }
            },
        }
    );
});

// Handle script events for chest opening/closing
system.afterEvents.scriptEventReceive.subscribe((event) => {
    const { id, message, sourceEntity } = event;

    const block = sourceEntity.dimension.getBlock({
        x: Math.floor(sourceEntity.location.x),
        y: Math.floor(sourceEntity.location.y),
        z: Math.floor(sourceEntity.location.z),
    });

    const findChest = chests.find(
        (chest) => block.typeId === `x_ender_chest:${chest}`
    );
    if (!findChest) return;

    if (id == "x_ender_chest:open" && findChest) {
        sourceEntity.setProperty("x_ender_chest:opened", true);
    }

    if (id == "x_ender_chest:close" && sourceEntity.hasTag("chestOpened")) {
        // Save the current chest contents to the player's storage before closing
        const players = sourceEntity.dimension.getPlayers();
        const nearbyPlayer = players.find((p) => {
            const distance = Math.sqrt(
                Math.pow(p.location.x - sourceEntity.location.x, 2) +
                Math.pow(p.location.y - sourceEntity.location.y, 2) +
                Math.pow(p.location.z - sourceEntity.location.z, 2)
            );
            return distance <= 5 && p.hasTag("interacted");
        });

        if (nearbyPlayer) {
            const playerStorage = getPlayerEnderStorage(nearbyPlayer.id);
            savePlayerItemsFromChest(sourceEntity, playerStorage, nearbyPlayer.id);
            nearbyPlayer.removeTag("interacted");
        }

        sourceEntity.removeTag("chestOpened");
        sourceEntity.dimension.playSound(
            "random.enderchestclosed",
            sourceEntity.location
        );

        sourceEntity.addTag("closed");

        sourceEntity.dimension.runCommand(
            `execute positioned ${sourceEntity.location.x} ${sourceEntity.location.y} ${sourceEntity.location.z} run tag @e[tag=top,r=7] remove interacted`
        );
        sourceEntity.dimension.runCommand(
            `execute positioned ${sourceEntity.location.x} ${sourceEntity.location.y} ${sourceEntity.location.z} run playanimation @e[family=top,r=1,c=1] animation.chest.close`
        );

        system.runTimeout(() => {
            sourceEntity.dimension.runCommand(
                `execute positioned ${sourceEntity.location.x} ${sourceEntity.location.y} ${sourceEntity.location.z} run event entity @e[family=top,r=0,c=1] x_ender_chest:despawn_chest`
            );
            block.setPermutation(block.permutation.withState("x_ender_chest:top", 0));
        }, 8);
    }
});

// Get or create per-player ender chest storage
function getPlayerEnderStorage(playerId) {
    if (!enderChestStorage.has(playerId)) {
        const loadedData = loadPlayerData(playerId);
        enderChestStorage.set(playerId, loadedData);
    }
    return enderChestStorage.get(playerId);
}

// Copy items from storage array to chest entity
function loadPlayerItemsToChest(chest, playerItems) {
    const container = chest.getComponent("inventory").container;
    // Clear the chest inventory first to prevent item duplication
    for (let i = 0; i < 45; i++) {
        container.setItem(i, null);
    }
    // Load player's items from storage
    for (let i = 0; i < Math.min(45, playerItems.length); i++) {
        if (playerItems[i]) {
            container.setItem(i, playerItems[i]);
        }
    }
}

// Save items from chest entity to storage array
function savePlayerItemsFromChest(chest, playerItems, playerId) {
    const container = chest.getComponent("inventory").container;
    for (let i = 0; i < 45; i++) {
        const item = container.getItem(i);
        playerItems[i] = item || null;
    }
    // Persist to world storage
    savePlayerData(playerId, playerItems);

    // Log final ender chest state with full serialization
    logEnderChestContents(playerId, playerItems);
}

// Log comprehensive ender chest contents including shulker box dynamic properties
function logEnderChestContents(playerId, playerItems) {
    try {
        const player = world.getPlayers().find((p) => p.id === playerId);
        const playerName = player ? player.name : playerId;

        const serializedContents = [];
        for (let i = 0; i < playerItems.length; i++) {
            const item = playerItems[i];

            // Silent processing for shulker boxes

            serializedContents[i] = serializeItem(item);
        }

        console.log(`[X_ENDER_CHEST][${playerName}]${JSON.stringify(serializedContents)}`);
    } catch (e) { }
}

// Handle player interaction with chest
world.afterEvents.playerInteractWithEntity.subscribe((event) => {
    const player = event.player;
    const target = event.target;

    const findChest = chests.find(
        (chest) => target.typeId === `x_ender_chest:${chest}`
    );
    if (!findChest) return;

    const pos = target.location;
    const block = target.dimension.getBlock({
        x: Math.floor(pos.x),
        y: Math.floor(pos.y),
        z: Math.floor(pos.z),
    });

    const tags = target.getTags();
    const directions = {
        north: "180",
        south: "0",
        west: "90",
        east: "-90",
    };

    const directionKeys = Object.keys(directions);
    const chestTag = tags.find((tag) =>
        directionKeys.some((dir) => tag.startsWith(dir))
    );
    const chestRot = directions[chestTag] ?? 0;

    function SpawnEntity() {
        target.dimension.runCommand(
            `execute positioned ${pos.x} ${pos.y} ${pos.z} run summon x_ender_chest:${findChest}_top ~ ~ ~ ${chestRot}`
        );
    }

    if (target.typeId === `x_ender_chest:${findChest}`) {
        target.addTag("chestOpened");
        player.addTag("interacted");
        target.dimension.playSound("random.enderchestopen", pos);

        // Load player's personal items into this chest
        const playerStorage = getPlayerEnderStorage(player.id);
        loadPlayerItemsToChest(target, playerStorage);

        block.setPermutation(block.permutation.withState("x_ender_chest:top", 1));
        SpawnEntity();

        // Start closed immediately, then play open animation
        // First set to closed idle to override the default opened state
        target.dimension.runCommand(
            `execute positioned ${pos.x} ${pos.y} ${pos.z} run playanimation @e[family=top,r=2,c=1] closed_idle`
        );

        system.runTimeout(() => {
            // Play open animation
            target.dimension.runCommand(
                `execute positioned ${pos.x} ${pos.y} ${pos.z} run playanimation @e[family=top,r=2,c=1] animation.chest.open`
            );
        }, 2);
    }
});

// Save player data when they leave the world
world.afterEvents.playerLeave.subscribe((event) => {
    const playerId = event.playerId;
    if (enderChestStorage.has(playerId)) {
        const playerStorage = enderChestStorage.get(playerId);
        savePlayerData(playerId, playerStorage);
        enderChestStorage.delete(playerId); // Clean up memory
    }
});
