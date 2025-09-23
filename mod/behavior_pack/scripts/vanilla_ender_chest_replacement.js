import { world } from "@minecraft/server";

// Replace vanilla ender chests with custom ones when players interact with them
world.afterEvents.playerInteractWithBlock.subscribe((event) => {
    const { block, player } = event;

    // Check if the interacted block is a vanilla ender chest
    if (block.typeId === "minecraft:ender_chest") {
        // Get the direction of the vanilla chest (if it has one)
        const direction =
            block.permutation.getState("minecraft:cardinal_direction") || "north";

        // Replace the vanilla ender chest with custom ender chest
        block.setType("x_ender_chest:x_ender_chest");

        // Set the same direction for the custom chest
        const newPermutation = block.permutation.withState(
            "minecraft:cardinal_direction",
            direction
        );
        block.setPermutation(newPermutation);
    }
});
