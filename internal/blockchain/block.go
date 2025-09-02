package blockchain

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Block struct {
	Index         int       `json:"index"`
	Timestamp     time.Time `json:"timestamp"`
	PlayerUUID    string    `json:"player_uuid"`
	InventoryData []byte    `json:"inventory_data"`
	PreviousHash  string    `json:"previous_hash"`
	Hash          string    `json:"hash"`
	Nonce         int       `json:"nonce"`
}

func (b *Block) calculateHash() string {
	blockData := strconv.Itoa(b.Index) + b.Timestamp.String() + b.PlayerUUID + string(b.InventoryData) + b.PreviousHash + strconv.Itoa(b.Nonce)
	hash := sha256.Sum256([]byte(blockData))
	return fmt.Sprintf("%x", hash)
}

func (b *Block) mine(difficulty int) {
	target := strings.Repeat("0", difficulty)
	for !strings.HasPrefix(b.Hash, target) {
		b.Nonce++
		b.Hash = b.calculateHash()
	}
}

func NewBlock(index int, playerUUID string, inventoryData []byte, previousHash string, difficulty int) *Block {
	block := &Block{
		Index:         index,
		Timestamp:     time.Now(),
		PlayerUUID:    playerUUID,
		InventoryData: inventoryData,
		PreviousHash:  previousHash,
		Nonce:         0,
	}
	block.Hash = block.calculateHash()
	block.mine(difficulty)
	return block
}
