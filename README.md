## JAFT

A secure, trustless network for running Minecraft Bedrock servers with blockchain-based inventory management and encrypted execution isolation. The system ensures tamper-proof gameplay while maintaining player inventory persistence across different server instances.

## Key Features

- **Blockchain Integration**: Player inventories and ender chest contents stored on blockchain
- **Trustless Architecture**: No trust required between network participants
- **Encrypted Execution**: FUSE-based file system encryption for server files
- **Tamper Detection**: Continuous monitoring for file system modifications
- **Process Isolation**: Secure process spawning with inherited file descriptors
- **Remote Attestation**: Cryptographic verification of node integrity

## Architecture

### Core Components

- **Go Application**: Main orchestrator running inside Ubuntu Docker containers
- **Minecraft Bedrock Server**: Downloaded directly from Mojang over https
- **FUSE File System**: Encrypted file operations for server data
- **Blockchain Interface**: Handles inventory persistence and synchronization

### Security Features

- **Dynamic Server Placement**: Randomized server file locations to prevent mounting attacks
- **Process Monitoring**: Automatic termination of untrusted spawned processes
- **Memory Protection**: Encryption keys managed with [memguard](https://github.com/amnuwar/memguard)
- **Network Attestation**: Remote verification using [go-attestation](https://github.com/google/go-attestation)

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

---

Project in development
