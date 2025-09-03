# JAFT

A secure, trustless network for running Minecraft Bedrock servers with encrypted distributed inventory management and execution isolation. The system ensures tamper-proof gameplay while maintaining player inventory persistence across different server instances using a distributed encrypted database.

## Key Features

- **Distributed Database**: Player inventories and ender chest contents stored in encrypted distributed database
- **Trustless Architecture**: No trust required between network participants
- **Encrypted Execution**: FUSE-based file system encryption for server files
- **Tamper Detection**: Continuous monitoring for file system modifications
- **Process Isolation**: Secure process spawning with inherited file descriptors
- **Remote Attestation**: Cryptographic verification of node integrity

## Architecture

### Core Components

- **Go Application**: Main orchestrator running inside Ubuntu Docker containers
- **Minecraft Bedrock Server**: Downloaded directly from Mojang, configured for legitimate clients only
- **FUSE File System**: Encrypted file operations for server data
- **Distributed Database**: Handles encrypted inventory persistence and peer synchronization

### Security Features

- **Client Authentication**: Only legitimate Mojang/Xbox accounts allowed, vanilla clients enforced
- **Dynamic Server Placement**: Randomized server file locations to prevent mounting attacks
- **Process Monitoring**: Automatic termination of untrusted spawned processes
- **Memory Protection**: Encryption keys managed with [memguard](https://github.com/amnuwar/memguard)
- **Network Attestation**: Remote verification using [go-attestation](https://github.com/google/go-attestation)

### Distributed Database Architecture

- **Simple Key-Value Store**: Player inventories stored as encrypted JSON data
- **Peer Replication**: Data automatically synced across network nodes
- **Session-Based Operations**:
  - Player joins → Inventory loaded from distributed database
  - Player leaves → Inventory saved back to distributed database
- **Conflict Resolution**: Last-write-wins with timestamp-based versioning

### Key Distribution

- Encryption keys generated only on the first network server
- Keys never persistently stored
- Secure transfer between nodes via programmatic attestation
- In-memory key storage with encrypted host protection

## Security Model

The system operates on a zero-trust principle where:

- Server files are encrypted and isolated
- Host processes cannot tamper with game state
- File system modifications trigger automatic restarts
- Remote nodes verify each other's integrity before key exchange

### Version Verification Process

**Binary Hash Verification**: All nodes verify they're running identical code before sharing encryption keys, and keys are shared to the nodes running latest version of jaft:

1. **Startup Check**: Node compares its binary hash with latest version from `github.com/d1nch8g/jaft`
2. **Peer Communication**: Before key recovery, nodes verify each other's binary hashes
3. **Security Exit**: Process terminates if versions cannot be synchronized

**Update Flow**:

```
Node A requests key from Node B
→ Node B checks: "Is A running same binary hash as me?"
→ If NO: Node A automatically updates to latest version
→ Hash re-check: "Are we now running identical code?"
→ If YES: Key sharing proceeds
→ If NO: Connection rejected for security
```

This ensures all network participants run identical, verified code before any cryptographic operations.

## Client Security Requirements

**Minecraft Bedrock Server Configuration**: JAFT enforces strict client authentication to prevent cheating and ensure game integrity:

### Hardcoded Settings:

- **`online-mode=true`**: Mandatory Xbox Live authentication
- **`xbox-auth=true`**: Verify legitimate Mojang/Microsoft accounts
- **`texturepack-required=true`**: Forces all players to use server-provided resource pack
- **`server-authoritative-movement=true`**: Server controls player movement
- **`server-authoritative-block-breaking=true`**: Server validates all block interactions
- **`allow-cheats=true`**: Required for decentralized inventory management
- **`correct-player-movement=true`**: Ensures movement validation consistency
- **`difficulty=normal`**: Standard difficulty level across all nodes
- **`force-gamemode=true`**: Enforces consistent gamemode across network
- **`gamemode=survival`**: All players must play in survival mode
- **`level-seed=randomized`**: Each server instance uses randomized world generation

### Client Validation:

- **Official Accounts Only**: Players must authenticate with legitimate Mojang/Xbox accounts
- **Vanilla Clients Enforced**: Modified clients are rejected by server-side validation
- **X-ray Protection**: Server enforces textures for commonly X-rayed blocks (stone, dirt, grass, etc.) to prevent cheat resource packs
- **Server Authority**: All game actions validated server-side, client cannot override

This configuration ensures only legitimate, unmodified Minecraft clients can participate in the trustless network.

---

Project in development
