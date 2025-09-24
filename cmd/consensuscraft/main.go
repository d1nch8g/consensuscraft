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

	for _, bn := range cfg.BannedNodes {
		inventories.Delete(bn, true)
	}

	runBDS := make(chan struct{})

	bds, err := bds.New(bds.Parameters{
		InventoryReceiveCallback: func(playerName string) ([]byte, error) {
			return inventories.Get(playerName)
		},
		InventoryUpdateCallback: func(playerName string, inventory []byte) error {
			return inventories.Put(playerName, inventory, cfg.WebAddress)
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
