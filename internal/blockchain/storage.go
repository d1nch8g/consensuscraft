package blockchain

import (
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
)

type Storage struct {
	db *leveldb.DB
}

func NewStorage(path string) (*Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveBlock(block *Block) error {
	key := fmt.Sprintf("block_%d", block.Index)
	data, err := json.Marshal(block)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(key), data, nil)
}

func (s *Storage) LoadBlock(index int) (*Block, error) {
	key := fmt.Sprintf("block_%d", index)
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		return nil, err
	}
	
	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, err
	}
	return &block, nil
}

func (s *Storage) SaveChainHeight(height int) error {
	return s.db.Put([]byte("chain_height"), []byte(fmt.Sprintf("%d", height)), nil)
}

func (s *Storage) LoadChainHeight() (int, error) {
	data, err := s.db.Get([]byte("chain_height"), nil)
	if err != nil {
		return 0, err
	}
	var height int
	if _, err := fmt.Sscanf(string(data), "%d", &height); err != nil {
		return 0, err
	}
	return height, nil
}
