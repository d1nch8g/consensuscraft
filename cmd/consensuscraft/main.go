package main

import (
	"time"

	"github.com/d1nch8g/consensuscraft/bds"
	"github.com/d1nch8g/consensuscraft/config"
	"github.com/d1nch8g/consensuscraft/database"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.New()

	inventories, err := database.New("inventories.ldb")
	if err != nil {
		logrus.Fatalf("unable to open inventories database: %v", err)
	}

	runBDS := make(chan struct{})

	bds, err := bds.New(bds.Parameters{
		BedrockServerPort: cfg.BedrockServerPort,
		BedrockMaxThreads: cfg.BedrockMaxThreads,
		MaxPlayers:        cfg.MaxPlayers,
		PlayerIdleTimeout: cfg.PlayerIdleTimeout,
		ViewDistance:      cfg.ViewDistance,
		InventoryCallback: func(playerName string) ([]byte, error) {
			return inventories.Get([]byte(playerName))
		},
		StartTrigger: runBDS,
	})
	if err != nil {
		logrus.Fatalf("unable to launch bedrock dedicated server: %v", err)
	}

	runBDS <- struct{}{}

	_ = bds

	for {
		time.Sleep(time.Hour * 284)
	}
}
