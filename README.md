# Consensuscraft - Decentralized Anti-Cheat Bedrock Network

A trustless, decentralized Minecraft Bedrock server network with shared cross-network inventories and ender chests, featuring statistical anti-xray/cheat validation and bot detection.

## Core Concept

Consensuscraft creates a network of autonomous Minecraft Bedrock servers that collectively validate player behavior and server integrity through:

- **Shared player inventories** with atomic claiming system preventing duplication
- **Cross-server bot detection** through human verification challenges
- **Cryptographic player authentication** via public key infrastructure
- **Distributed database synchronization** for trustless data storage

## Architecture

### Player Authentication System

**Decentralized Key Registration:**

- Players register their public keys on any network node
- Authentication requires signing server-provided challenge words
- Response must be posted in chat to prove private key ownership
- Prevents server operators from impersonating wealthy players
- Nodes can only register players currently online; excessive registrations trigger cross-node challenges

### Player Inventory Management

**Atomic Claiming System:**

- Player inventories stored in decentralized network database
- Servers must "claim" players before they can join (atomic operation)
- Only one server can claim a player at a time (prevents duplication)
- Lease-based system handles server crashes gracefully
- Players released back to network when they disconnect

**Database Synchronization:**

- New nodes sync full database on startup via streaming key-value pairs
- LevelDB-based storage with eventual consistency
- All inventory changes replicated across network nodes

### Bot Detection Network

**Cross-Node Verification:**

- Random checks every 12-24 hours on network nodes
- Each node tests 100% of target server's population
- Simple chat-based questions with 4 options requiring human reasoning
- Requires 70%+ human response rate to pass

**Enforcement:**

- Failed checks reported as violations to network
- Servers with >50% violations in 12h window are banned
- Network automatically ejects malicious nodes

## Security Features

### Server Validation

- **Online Mode**: Mandatory Xbox Live authentication
- **Xbox Auth**: Verify legitimate Mojang/Microsoft accounts only
- **Server Authority**: All game actions validated server-side
- **Vanilla Enforcement**: Modified clients rejected automatically

### Network Integrity

- **Atomic Operations**: Prevent race conditions in player claiming
- **Cryptographic Proofs**: All node communications signed
- **Distributed Consensus**: No single point of failure

## Configuration

### Hardcoded Server Settings

```toml
online-mode=true # Xbox Live authentication required
xbox-auth=true # Legitimate accounts only
texturepack-required=true # Server-provided resource pack
server-authoritative-movement=true
server-authoritative-block-breaking=true
allow-cheats=true # Required for inventory management
correct-player-movement=true
difficulty=normal
force-gamemode=true
gamemode=survival
level-seed=randomized # Each node uses unique world gen
```

### Client Validation

- **Official Accounts Only**: Mojang/Xbox authentication mandatory
- **Vanilla Clients**: Modified clients blocked by server validation
- **Server Authority**: Client cannot override server decisions

## Network Protocol

### Core Services (gRPC Streaming)

```protobuf
service ConsensusCraftService {
  rpc NodeStream(stream Message) returns (stream Message);
}
```

### Message Types

**Network Management:**

- `DiscoverNodesRequest/Response` - Find other network nodes
- `JoinRequest/Response` - Join the network with cryptographic proof
- `SyncDatabaseRequest/Data/Response` - Replicate database state

**Player Authentication:**

- `RegisterPlayerRequest/Response` - Register public keys for players

**Inventory Management:**

- `ClaimPlayerRequest/Response` - Atomically claim player from network
- `ReleasePlayerRequest/Response` - Return player inventory to network

**Bot Detection:**

- `BotCheckRequest` - Request bot check on target server
- `BotViolationReport` - Report failed bot check to network

### Data Flow

1. **Node Startup**: Discover peers → Sync database → Join network
2. **Player Join**: Claim player → Load inventory → Player connects
3. **Player Leave**: Save inventory → Release player → Network stores data
4. **Bot Checks**: Random timer → Test target → Report violations → Ban if needed

## Development Status

🚧 **Project in Development**

This is an experimental anti-cheat system pushing the boundaries of decentralized gaming infrastructure. The goal is to create a trustless network where players can enjoy fair gameplay without relying on centralized authorities, keeping their inventories and ender chests saved in a decentralized way.

## Future Enhancements

- **Statistical Analysis**: Item complexity validation and anomaly detection
- **Machine Learning**: Pattern recognition for sophisticated cheat detection
- **Appeal System**: Democratic review process for disputed bans
- **Economic Modeling**: Advanced statistical models for item flow analysis
