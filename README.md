# JAFT - Decentralized Anti-Cheat Bedrock Network

A trustless, decentralized Minecraft Bedrock server network with shared across network inventories and end chests, with statistical anti-xray/cheat validation and bot detection.

## Core Concept

JAFT creates a network of autonomous Minecraft Bedrock servers that collectively validate player behavior and server integrity through:

- **Statistical item analysis** based on obtain complexity ratings
- **Cross-server bot detection** through human verification challenges
- **Cryptographic player authentication** via public key infrastructure
- **Consensus-based enforcement** for banning cheaters and malicious servers

## Architecture

### Player Authentication System

**Decentralized Key Registration:**

- Players register their public keys on any network node
- Authentication requires signing server-provided challenge words
- Response must be posted in chat to prove private key ownership
- Prevents server operators from impersonating wealthy players
- Also nodes can only register players that are currently playing, if too many registrations occure - the players are asked to pass challenges on other network nodes (to prove registrations are real)

### Item Complexity Validation

**Statistical Analysis:**

- Each item assigned an "obtain complexity" rating
- Tracks item production vs. consumption per player/server
- Analyzes session data (login/logout times, items gained/lost)
- Flags abnormal accumulation patterns that exceed statistical norms

**Enforcement Actions:**

- Player ban: Network-wide exclusion for individuals exceeding thresholds
- Server kick: Node removal from network for suspicious activity
- Resource deletion: All items from kicked servers are purged

### Bot Detection Network

**Cross-Node Verification:**

- Random checks every 12-24 hours on network nodes
- Targets 25% of server population (minimum 1 player)
- Weighted by server size - larger servers perform more checks
- Questions asked in chat, requires 70%+ human response rate

**Challenge Types:**

- Simple chat-based questions with 4 options to choose, requiring human reasoning

## Security Features

### Server Validation

- **Online Mode**: Mandatory Xbox Live authentication
- **Xbox Auth**: Verify legitimate Mojang/Microsoft accounts only
- **Server Authority**: All game actions validated server-side
- **Vanilla Enforcement**: Modified clients rejected automatically

### Network Integrity

- **Consensus Mechanisms**: Multiple nodes must agree on violations
- **Reputation Scoring**: Nodes rated on accuracy of reports
- **Appeal System**: Process for contesting false positives
- **Cryptographic Proofs**: Verifiable item transaction records

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
level-seed randomized # Each node uses unique world gen
```

### Client Validation

- **Official Accounts Only**: Mojang/Xbox authentication mandatory
- **Vanilla Clients**: Modified clients blocked by server validation
- **Server Authority**: Client cannot override server decisions

## Network Protocol

### Data Exchange

1. **Session Tracking**: Login/logout timestamps with item deltas
2. **Statistical Reports**: Periodic complexity analysis summaries
3. **Bot Check Results**: Human verification outcomes
4. **Consensus Voting**: Multi-node agreement on enforcement actions

### Enforcement Thresholds

- **Item Anomaly**: Configurable standard deviations from expected rates
- **Bot Check Failure**: <70% human response rate triggers investigation
- **Consensus Requirement**: Minimum node agreement percentage for bans

## Development Status

🚧 **Project in Development**

This is an experimental anti-cheat system pushing the boundaries of decentralized gaming infrastructure. The goal is to create a trustless network where players can enjoy fair gameplay without relying on centralized authorities, keeping their inventories and ender chests saved on a decentralized way.

## Future Enhancements

- **Machine Learning**: Pattern recognition for sophisticated cheat detection
- **Appeal Integration**: Democratic review process for disputed bans
- **Economic Modeling**: Advanced statistical models for item flow analysis
