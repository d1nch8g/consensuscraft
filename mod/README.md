# BDS InvSee - Programmable Ender Chest System

A programmable ender chest system for Minecraft Bedrock Edition servers that provides 45-slot personal storage with server-side inventory management, restoration capabilities, and reworked shulker box functionality.

## Features

- **Programmable X Ender Chest**: 45-slot per-player persistent storage system
- **Server-Side Inventory Control**: Restore player inventories from backup data via player tags
- **Enhanced Shulker Boxes**: Persistent storage with nested inventory support
- **Full Item Serialization**: Support for enchantments, durability, lore, custom names, and nested containers

## Server-Side Interface

### Inventory Restoration System

The addon can restore complex player inventories from server-side data using player tags. This is particularly useful for:

- Server migrations
- Inventory backups and restoration
- Cross-server player transfers
- Emergency inventory recovery

#### Tag Format

Inventory data is stored in player tags with the format: `restore_inv_{part}_{data}`

- `{part}`: Sequential number (0, 1, 2, ...) for chunked data
- `{data}`: Base64 or JSON string containing inventory data

#### Tag Size Limitations

**Important**: Minecraft Bedrock Edition has character limits for player tags that must be considered:

- **Per-tag character limit**: While not officially documented, practical testing shows tags should be kept under **32,000 characters** to ensure reliability across all platforms
- **Recommended chunk size**: **1,000-2,000 characters** per tag for optimal compatibility
- **Tag format**: Must be either a single word (no spaces) or a double-quoted string with escape characters
- **Maximum tags per entity**: While Java Edition limits entities to 1024 tags, Bedrock Edition limits are less documented but should be considered finite

**Tag Chunking Strategy**:

```javascript
// Recommended approach for large inventories
const maxChunkSize = 1500; // Conservative limit for cross-platform compatibility
const jsonString = JSON.stringify(inventoryData);
const chunks = [];

for (let i = 0; i < jsonString.length; i += maxChunkSize) {
  chunks.push(jsonString.substring(i, i + maxChunkSize));
}

// Apply chunked tags
chunks.forEach((chunk, index) => {
  const tagName = `restore_inv_${index}_${chunk}`;
  // Use server command: tag @p add "restore_inv_{index}_{chunk}"
});
```

#### Supported Item Properties

The system supports comprehensive item serialization including:

- **Basic Properties**: `typeId`, `amount`, `nameTag`
- **Enchantments**: All vanilla enchantments with levels
- **Durability**: Current damage and max durability
- **Lore**: Multi-line item descriptions
- **Shulker Contents**: Nested inventory data for shulker boxes

### Example Inventory Structures

#### Simple Item Example

```json
{
  "typeId": "minecraft:diamond_sword",
  "amount": 1,
  "nameTag": "Legendary Blade",
  "lore": ["§6Epic Weapon", "§7Forged by ancient smiths"],
  "enchantments": [
    { "type": "minecraft:sharpness", "level": 5 },
    { "type": "minecraft:unbreaking", "level": 3 },
    { "type": "minecraft:mending", "level": 1 }
  ],
  "durability": {
    "damage": 150,
    "maxDurability": 1561
  }
}
```

#### Complex Inventory with Shulker Boxes

```json
[
  {
    "typeId": "minecraft:red_shulker_box",
    "amount": 1,
    "nameTag": "§cWeapons Storage",
    "lore": ["§7Contains legendary weapons"],
    "shulker_contents": [
      {
        "typeId": "minecraft:netherite_sword",
        "amount": 1,
        "nameTag": "§5Void Slayer",
        "enchantments": [
          { "type": "minecraft:sharpness", "level": 5 },
          { "type": "minecraft:looting", "level": 3 },
          { "type": "minecraft:unbreaking", "level": 3 },
          { "type": "minecraft:mending", "level": 1 },
          { "type": "minecraft:fire_aspect", "level": 2 }
        ],
        "durability": { "damage": 0, "maxDurability": 2031 }
      },
      {
        "typeId": "minecraft:bow",
        "amount": 1,
        "nameTag": "§bFrost Bow",
        "lore": ["§7Shoots frozen arrows", "§3+50% Ice Damage"],
        "enchantments": [
          { "type": "minecraft:power", "level": 5 },
          { "type": "minecraft:infinity", "level": 1 },
          { "type": "minecraft:flame", "level": 1 }
        ],
        "durability": { "damage": 23, "maxDurability": 384 }
      },
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null
    ]
  },
  {
    "typeId": "minecraft:blue_shulker_box",
    "amount": 1,
    "nameTag": "§9Potion Storage",
    "shulker_contents": [
      {
        "typeId": "minecraft:potion",
        "amount": 64,
        "nameTag": "§dHealing Potion II"
      },
      {
        "typeId": "minecraft:splash_potion",
        "amount": 32,
        "nameTag": "§cStrength Potion"
      },
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null
    ]
  },
  {
    "typeId": "minecraft:diamond_pickaxe",
    "amount": 1,
    "nameTag": "§bMaster Miner",
    "lore": ["§7The ultimate mining tool", "§6Efficiency: Maximum"],
    "enchantments": [
      { "type": "minecraft:efficiency", "level": 5 },
      { "type": "minecraft:fortune", "level": 3 },
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 }
    ],
    "durability": { "damage": 456, "maxDurability": 1561 }
  },
  {
    "typeId": "minecraft:elytra",
    "amount": 1,
    "nameTag": "§5Wings of Freedom",
    "lore": ["§7Soar through the skies", "§dEnchanted with ancient magic"],
    "enchantments": [
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 }
    ],
    "durability": { "damage": 100, "maxDurability": 432 }
  },
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null
]
```

#### Armor Set Example

```json
[
  {
    "typeId": "minecraft:netherite_helmet",
    "amount": 1,
    "nameTag": "§4Crown of the Nether",
    "lore": ["§7Forged in the depths of hell", "§c+25% Fire Resistance"],
    "enchantments": [
      { "type": "minecraft:protection", "level": 4 },
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 },
      { "type": "minecraft:respiration", "level": 3 },
      { "type": "minecraft:aqua_affinity", "level": 1 }
    ],
    "durability": { "damage": 50, "maxDurability": 407 }
  },
  {
    "typeId": "minecraft:netherite_chestplate",
    "amount": 1,
    "nameTag": "§4Chestplate of Valor",
    "enchantments": [
      { "type": "minecraft:protection", "level": 4 },
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 },
      { "type": "minecraft:thorns", "level": 3 }
    ],
    "durability": { "damage": 0, "maxDurability": 592 }
  },
  {
    "typeId": "minecraft:netherite_leggings",
    "amount": 1,
    "nameTag": "§4Leggings of Endurance",
    "enchantments": [
      { "type": "minecraft:protection", "level": 4 },
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 }
    ],
    "durability": { "damage": 25, "maxDurability": 555 }
  },
  {
    "typeId": "minecraft:netherite_boots",
    "amount": 1,
    "nameTag": "§4Boots of Swift Travel",
    "lore": ["§7Walk on any terrain", "§b+15% Movement Speed"],
    "enchantments": [
      { "type": "minecraft:protection", "level": 4 },
      { "type": "minecraft:unbreaking", "level": 3 },
      { "type": "minecraft:mending", "level": 1 },
      { "type": "minecraft:feather_falling", "level": 4 },
      { "type": "minecraft:depth_strider", "level": 3 }
    ],
    "durability": { "damage": 75, "maxDurability": 481 }
  }
]
```

### Implementation Examples

#### Setting Player Inventory Tags (Server Command)

```bash
# Simple example - single item
tag @p add "restore_inv_0_[{\"typeId\":\"minecraft:diamond_sword\",\"amount\":1,\"nameTag\":\"Test Sword\"}]"

# Complex example with multiple items (split across multiple tags due to length limits)
tag @p add "restore_inv_0_[{\"typeId\":\"minecraft:red_shulker_box\",\"amount\":1,\"shulker_contents\":[{\"typeId\":\"minecraft:diamond\",\"amount\":64},"
tag @p add "restore_inv_1_null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null]}]"
```

#### Server-Side Inventory Management

```javascript
// Example: Creating inventory data programmatically
const inventoryData = [
  {
    typeId: "minecraft:diamond_sword",
    amount: 1,
    nameTag: "§6Legendary Blade",
    lore: ["§7A weapon of great power"],
    enchantments: [
      { type: "minecraft:sharpness", level: 5 },
      { type: "minecraft:unbreaking", level: 3 },
    ],
    durability: { damage: 0, maxDurability: 1561 },
  },
  // ... more items
];

// Convert to JSON and split into chunks for player tags
const jsonString = JSON.stringify(inventoryData);
const chunkSize = 1000; // Adjust based on tag length limits
const chunks = [];

for (let i = 0; i < jsonString.length; i += chunkSize) {
  chunks.push(jsonString.substring(i, i + chunkSize));
}

// Apply tags to player
chunks.forEach((chunk, index) => {
  // Use server commands or API to add tags
  // tag @p add "restore_inv_{index}_{chunk}"
});
```

#### Practical Size Considerations

**Small Inventory Example** (~500 characters):

```bash
# Single tag for simple inventory
tag @p add "restore_inv_0_[{\"typeId\":\"minecraft:diamond_sword\",\"amount\":1},{\"typeId\":\"minecraft:bread\",\"amount\":64}]"
```

**Medium Inventory Example** (~2,000 characters, requires chunking):

```bash
# First chunk
tag @p add "restore_inv_0_[{\"typeId\":\"minecraft:netherite_sword\",\"amount\":1,\"nameTag\":\"§5Legendary Blade\",\"enchantments\":[{\"type\":\"minecraft:sharpness\",\"level\":5}]},{\"typeId\":\"minecraft:red_shulker_box\",\"amount\":1,\"shulker_contents\":[{\"typeId\":\"minecraft:diamond\",\"amount\":64},"

# Second chunk
tag @p add "restore_inv_1_null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null]}]"
```

**Large Inventory Example** (10,000+ characters, multiple chunks):

```bash
# For very large inventories with multiple shulker boxes containing many items
# This would require 7-10 tags depending on content complexity
tag @p add "restore_inv_0_[{\"typeId\":\"minecraft:red_shulker_box\",\"amount\":1,\"shulker_contents\":[{\"typeId\":\"minecraft:netherite_sword\",\"amount\":1,\"nameTag\":\"§5Void Slayer\",\"enchantments\":[{\"type\":\"minecraft:sharpness\",\"level\":5},{\"type\":\"minecraft:looting\",\"level\":3}]},{\"typeId\":\"minecraft:bow\",\"amount\":1,\"nameTag\":\"§bFrost Bow\",\"enchantments\":[{\"type\":\"minecraft:power\",\"level\":5}]},"
tag @p add "restore_inv_1_{\"typeId\":\"minecraft:diamond_pickaxe\",\"amount\":1,\"enchantments\":[{\"type\":\"minecraft:efficiency\",\"level\":5}]},null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null]},{\"typeId\":\"minecraft:blue_shulker_box\",\"amount\":1,\"shulker_contents\":[{\"typeId\":\"minecraft:potion\",\"amount\":64},"
# ... continue with additional chunks as needed
```

**Size Estimation Guidelines**:

- **Basic item**: ~50-100 characters
- **Item with enchantments**: ~150-300 characters
- **Item with lore and enchantments**: ~200-400 characters
- **Empty shulker box**: ~100 characters
- **Full shulker box (27 items)**: ~3,000-8,000 characters
- **Complete 45-slot inventory**: ~2,000-20,000+ characters (depending on complexity)

### Enchantment Reference

All vanilla Minecraft enchantments are supported:

#### Weapon Enchantments

- `minecraft:sharpness` (1-5)
- `minecraft:smite` (1-5)
- `minecraft:bane_of_arthropods` (1-5)
- `minecraft:looting` (1-3)
- `minecraft:fire_aspect` (1-2)
- `minecraft:knockback` (1-2)
- `minecraft:sweeping` (1-3)

#### Tool Enchantments

- `minecraft:efficiency` (1-5)
- `minecraft:fortune` (1-3)
- `minecraft:silk_touch` (1)

#### Armor Enchantments

- `minecraft:protection` (1-4)
- `minecraft:fire_protection` (1-4)
- `minecraft:blast_protection` (1-4)
- `minecraft:projectile_protection` (1-4)
- `minecraft:thorns` (1-3)
- `minecraft:respiration` (1-3)
- `minecraft:aqua_affinity` (1)
- `minecraft:depth_strider` (1-3)
- `minecraft:frost_walker` (1-2)
- `minecraft:feather_falling` (1-4)

#### Bow Enchantments

- `minecraft:power` (1-5)
- `minecraft:punch` (1-2)
- `minecraft:flame` (1)
- `minecraft:infinity` (1)

#### Universal Enchantments

- `minecraft:unbreaking` (1-3)
- `minecraft:mending` (1)
- `minecraft:curse_of_vanishing` (1)
- `minecraft:curse_of_binding` (1)

### Shulker Box System

The addon provides enhanced shulker box functionality with persistent storage:

#### Features

- **Persistent Contents**: Shulker box contents are preserved when broken and placed
- **Unique IDs**: Each shulker box gets a unique UUID for content tracking
- **Nested Storage**: Support for shulker boxes containing other shulker boxes
- **Cross-Session Persistence**: Contents survive server restarts

#### Supported Shulker Box Types

- `minecraft:shulker_box`
- `minecraft:undyed_shulker_box`
- All 16 colored variants (white, orange, magenta, etc.)

### X Ender Chest System

Programmable ender chest implementation with 45-slot extended storage:

#### Features

- **45-Slot Storage**: Extended capacity beyond vanilla ender chests
- **Per-Player Storage**: Each player has their own personal inventory space
- **Persistent Storage**: Contents saved to world dynamic properties
- **Cross-Session Support**: Inventory persists across server restarts
- **Comprehensive Logging**: Full inventory serialization for debugging
- **Server-Side Control**: Programmable via player tags and commands

#### Usage

1. Place X Ender Chest block
2. Right-click to open personal 45-slot storage
3. Items are automatically saved when chest is closed
4. Access from any X Ender Chest in the world
5. Server can programmatically restore inventories via player tags

### Console Logging

The addon provides comprehensive logging for ender chest inventory contents:

```
[X_ENDER_CHEST][PlayerName][{"typeId":"minecraft:diamond_sword","amount":1,...}]
```

Logs include full item serialization with all properties for easy debugging and backup purposes.

### Installation

1. Download the addon files
2. Place in your Minecraft Bedrock server's behavior pack folder
3. Enable the behavior pack in your world settings
4. Restart the server

### Important Notes

- **UUID Persistence**: Do not update addon UUIDs after release to prevent ender chest content loss
- **Tag Limits**: Player tags have character limits; large inventories may need chunking
- **Performance**: Large nested shulker inventories may impact performance
- **Backup**: Always backup world data before major updates

### API Reference

#### Core Functions

- `serializeItem(item)`: Convert ItemStack to JSON data
- `deserializeItem(itemData)`: Convert JSON data to ItemStack
- `isShulkerBox(typeId)`: Check if item is a shulker box
- `getShulkerIdFromItem(item)`: Extract shulker UUID from item lore

#### Events

- `playerSpawn`: Triggers inventory restoration check
- `playerInteractWithEntity`: Opens X Ender Chest
- `playerPlaceBlock`: Restores shulker box contents
- `playerBreakBlock`: Saves shulker box contents

### Troubleshooting

#### Debug Commands

```bash
# Check player tags
tag @p list

# Clear restoration tags
tag @p remove restore_inv_0_data

# Check dynamic properties (server console)
# Properties are logged automatically during operations
```

### Contributing

This addon is designed for server administrators and developers who need programmable ender chest systems with extended storage capabilities. Contributions and improvements are welcome.

### License

This project is open source. Please respect the original authors when modifying or redistributing.

<!--
TODO:

- Don't update addon RP/BP uuid's after release, not to shred contents of ender chests.

 -->
