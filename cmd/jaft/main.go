package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/d1nch8g/jaft/internal/blockchain"
	"github.com/d1nch8g/jaft/internal/config"
	"github.com/d1nch8g/jaft/internal/crypto"
	"github.com/d1nch8g/jaft/internal/fuse"
	"github.com/d1nch8g/jaft/internal/minecraft"
	"github.com/d1nch8g/jaft/internal/monitor"
	"github.com/d1nch8g/jaft/internal/network"
)

func main() {
	// Validate environment
	if err := monitor.ValidateEnvironment(); err != nil {
		log.Fatalf("Environment validation failed: %v", err)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize key manager
	keyMgr := crypto.NewKeyManager()
	if err := keyMgr.GenerateKey(); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}
	defer keyMgr.Destroy()

	// Initialize blockchain with storage
	_ = blockchain.NewBlockchain(cfg.BlockchainDifficulty)

	// Initialize attestation manager
	attestMgr, err := crypto.NewAttestationManager()
	if err != nil {
		log.Fatalf("Failed to initialize attestation: %v", err)
	}
	defer attestMgr.Close()

	// Setup file monitoring with restart function
	restartFunc := func() {
		log.Println("Restarting due to tampering...")
		os.Exit(1)
	}
	fileMon := monitor.NewFileMonitor(restartFunc)
	procMon := monitor.NewProcessMonitor()

	// Allow our own process
	procMon.AllowProcess(os.Getpid())

	// Mount encrypted filesystem
	fuseServer, err := fuse.Mount(cfg.FUSEMountPoint, cfg.DataDir, keyMgr.GetKey())
	if err != nil {
		log.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer fuseServer.Unmount()

	// Initialize minecraft server
	mcServer := minecraft.NewServer(cfg.MinecraftServerURL, cfg.FUSEMountPoint)

	// Download and start minecraft server
	log.Println("Downloading Minecraft server...")
	if err := mcServer.Download(); err != nil {
		log.Fatalf("Failed to download Minecraft server: %v", err)
	}

	log.Println("Starting Minecraft server...")
	if err := mcServer.Start(); err != nil {
		log.Fatalf("Failed to start Minecraft server: %v", err)
	}
	defer mcServer.Cleanup()

	// Start monitoring
	fileMon.AddPath(cfg.FUSEMountPoint)
	fileMon.Start()
	procMon.Start()
	defer fileMon.Stop()
	defer procMon.Stop()

	// Initialize and start network
	peerNet := network.NewPeerNetwork()
	for _, peer := range cfg.NetworkPeers {
		peerNet.AddPeer(peer)
	}

	if err := peerNet.Start(cfg.Port); err != nil {
		log.Fatalf("Failed to start peer network: %v", err)
	}

	log.Printf("JAFT node started on port %d", cfg.Port)

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")
}
