import { system, world } from "@minecraft/server";

const chests = ["x_ender_chest"];

system.beforeEvents.startup.subscribe((initEvent) => {
  initEvent.blockComponentRegistry.registerCustomComponent(
    "x_ender_chest:place",
    {
      onPlace(event) {
        const block = event.block;

        const findChest = chests.find(
          (chest) => block.typeId === `x_ender_chest:${chest}`
        );
        if (findChest) {
          system.run(() => {
            let entity = block.dimension.spawnEntity(
              `x_ender_chest:${findChest}`,
              block.center()
            );
            entity.nameTag = `x_ender_chest.${findChest}`;
          });
        }
      },
    }
  );
});
