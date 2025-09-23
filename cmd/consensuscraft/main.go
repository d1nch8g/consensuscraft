package main

import (
	"time"

	"github.com/d1nch8g/consensuscraft/bds"
	"github.com/d1nch8g/consensuscraft/database"
	"github.com/sirupsen/logrus"
)

func main() {
	inventories, err := database.New("inventories.ldb")
	if err != nil {
		logrus.Fatalf("unable to open inventories database: %v", err)
	}

	runBDS := make(chan struct{})

	bds, err := bds.New(bds.Parameters{
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
