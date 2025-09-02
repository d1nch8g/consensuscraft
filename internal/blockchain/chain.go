package blockchain

import (
	"encoding/json"
	"sync"
	"time"
)

type Blockchain struct {
	Chain      []*Block `json:"chain"`
	Difficulty int      `json:"difficulty"`
	mu         sync.RWMutex
}

func NewBlockchain(difficulty int) *Blockchain {
	genesis := &Block{
		Index:         0,
		Timestamp:     time.Now(),
		PlayerUUID:    "genesis",
		InventoryData: []byte(`{"genesis": "block"}`),
		PreviousHash:  "",
		Hash:          "",
		Nonce:         0,
	}
	genesis.Hash = genesis.calculateHash()

	return &Blockchain{
		Chain:      []*Block{genesis},
		Difficulty: difficulty,
	}
}

func (bc *Blockchain) AddBlock(playerUUID string, inventoryData []byte) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	lastBlock := bc.Chain[len(bc.Chain)-1]
	newBlock := NewBlock(lastBlock.Index+1, playerUUID, inventoryData, lastBlock.Hash, bc.Difficulty)
	bc.Chain = append(bc.Chain, newBlock)
}

func (bc *Blockchain) GetPlayerInventory(playerUUID string) ([]byte, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i := len(bc.Chain) - 1; i >= 0; i-- {
		if bc.Chain[i].PlayerUUID == playerUUID {
			return bc.Chain[i].InventoryData, true
		}
	}
	return nil, false
}

func (bc *Blockchain) IsValid() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i := 1; i < len(bc.Chain); i++ {
		currentBlock := bc.Chain[i]
		previousBlock := bc.Chain[i-1]

		if currentBlock.Hash != currentBlock.calculateHash() {
			return false
		}

		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func (bc *Blockchain) ToJSON() ([]byte, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return json.Marshal(bc)
}
