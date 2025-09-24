package database

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/d1nch8g/consensuscraft/gen/pb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// InventoryEntry represents a single inventory update
type InventoryEntry struct {
	Inventory []byte    `json:"inventory"`
	Server    string    `json:"server"`
	Timestamp time.Time `json:"timestamp"`
}

// PlayerInventories represents all inventory entries for a player
type PlayerInventories struct {
	Entries []InventoryEntry `json:"entries"`
}

type ChangeEntry struct {
	player    string
	entry     InventoryEntry
	timestamp time.Time
	deleted   bool
}

type DB struct {
	leveldb   *leveldb.DB
	mu        sync.RWMutex
	changeLog []ChangeEntry
	closed    bool
}

var ErrClosed = errors.New("database is closed")
var ErrPlayerNotFound = errors.New("player not found")

func New(path string) (*DB, error) {
	err := os.RemoveAll(filepath.Join(path, "LOCK"))
	if err != nil {
		return nil, err
	}

	ldb, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &DB{
		leveldb:   ldb,
		changeLog: make([]ChangeEntry, 0),
	}, nil
}

// Put adds a new inventory entry for a player
func (db *DB) Put(player string, inventory []byte, server string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}

	// Create new inventory entry
	newEntry := InventoryEntry{
		Inventory: append([]byte(nil), inventory...),
		Server:    server,
		Timestamp: time.Now(),
	}

	// Get existing inventories for player
	var playerInv PlayerInventories
	key := []byte(player)
	
	existingData, err := db.leveldb.Get(key, nil)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	}
	
	if err == nil {
		// Player exists, unmarshal existing data
		if err := json.Unmarshal(existingData, &playerInv); err != nil {
			return err
		}
	}

	// Add new entry
	playerInv.Entries = append(playerInv.Entries, newEntry)

	// Sort entries by timestamp (newest first)
	sort.Slice(playerInv.Entries, func(i, j int) bool {
		return playerInv.Entries[i].Timestamp.After(playerInv.Entries[j].Timestamp)
	})

	// Marshal and store
	data, err := json.Marshal(playerInv)
	if err != nil {
		return err
	}

	err = db.leveldb.Put(key, data, nil)
	if err != nil {
		return err
	}

	// Log change for concurrent streaming
	db.changeLog = append(db.changeLog, ChangeEntry{
		player:    player,
		entry:     newEntry,
		timestamp: time.Now(),
		deleted:   false,
	})

	// Keep change log bounded (last 1000 entries)
	if len(db.changeLog) > 1000 {
		db.changeLog = db.changeLog[len(db.changeLog)-1000:]
	}

	return nil
}

// Get returns the latest inventory for a player from all servers
func (db *DB) Get(player string) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	key := []byte(player)
	data, err := db.leveldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}

	var playerInv PlayerInventories
	if err := json.Unmarshal(data, &playerInv); err != nil {
		return nil, err
	}

	if len(playerInv.Entries) == 0 {
		return nil, ErrPlayerNotFound
	}

	// Entries are already sorted by timestamp (newest first)
	return playerInv.Entries[0].Inventory, nil
}

// Delete removes all inventory entries from a specific server for all players
// If force is true, it also removes all entries that came after the server's entries
func (db *DB) Delete(server string, force bool) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}

	// Iterate through all players
	iter := db.leveldb.NewIterator(util.BytesPrefix(nil), nil)
	defer iter.Release()

	for iter.Next() {
		player := string(iter.Key())
		data := iter.Value()

		var playerInv PlayerInventories
		if err := json.Unmarshal(data, &playerInv); err != nil {
			continue // Skip corrupted entries
		}

		originalCount := len(playerInv.Entries)
		var newEntries []InventoryEntry
		var serverTimestamp time.Time
		
		// Find the latest timestamp from the server to be deleted
		for _, entry := range playerInv.Entries {
			if entry.Server == server {
				if entry.Timestamp.After(serverTimestamp) {
					serverTimestamp = entry.Timestamp
				}
			}
		}

		// Filter entries
		for _, entry := range playerInv.Entries {
			if entry.Server == server {
				// Remove all entries from this server
				continue
			}
			
			if force && !serverTimestamp.IsZero() && entry.Timestamp.After(serverTimestamp) {
				// Remove entries that came after the server's latest entry
				continue
			}
			
			newEntries = append(newEntries, entry)
		}

		// Only update if something changed
		if len(newEntries) != originalCount {
			if len(newEntries) == 0 {
				// No entries left, delete the player entirely
				err := db.leveldb.Delete(iter.Key(), nil)
				if err != nil {
					return err
				}
			} else {
				// Update with filtered entries
				playerInv.Entries = newEntries
				
				// Sort entries by timestamp (newest first)
				sort.Slice(playerInv.Entries, func(i, j int) bool {
					return playerInv.Entries[i].Timestamp.After(playerInv.Entries[j].Timestamp)
				})

				newData, err := json.Marshal(playerInv)
				if err != nil {
					return err
				}

				err = db.leveldb.Put(iter.Key(), newData, nil)
				if err != nil {
					return err
				}
			}

			// Log deletion for concurrent streaming
			db.changeLog = append(db.changeLog, ChangeEntry{
				player:    player,
				entry:     InventoryEntry{Server: server},
				timestamp: time.Now(),
				deleted:   true,
			})
		}
	}

	if err := iter.Error(); err != nil {
		return err
	}

	// Keep change log bounded
	if len(db.changeLog) > 1000 {
		db.changeLog = db.changeLog[len(db.changeLog)-1000:]
	}

	return nil
}

// GetPlayerInventories returns all inventory entries for a player
func (db *DB) GetPlayerInventories(player string) ([]InventoryEntry, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	key := []byte(player)
	data, err := db.leveldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}

	var playerInv PlayerInventories
	if err := json.Unmarshal(data, &playerInv); err != nil {
		return nil, err
	}

	return playerInv.Entries, nil
}

func (db *DB) StreamAll() <-chan *pb.SyncDatabaseData {
	ch := make(chan *pb.SyncDatabaseData, 100)

	go func() {
		defer close(ch)

		// Mark sync start point
		syncStart := time.Now()

		// Take snapshot for consistent read
		snapshot, err := db.leveldb.GetSnapshot()
		if err != nil {
			return
		}
		defer snapshot.Release()

		// Stream all snapshot data
		iter := snapshot.NewIterator(util.BytesPrefix(nil), nil)
		defer iter.Release()

		for iter.Next() {
			// Copy data to avoid reference issues
			key := append([]byte(nil), iter.Key()...)
			value := append([]byte(nil), iter.Value()...)

			select {
			case ch <- &pb.SyncDatabaseData{
				Key:   key,
				Value: value,
			}:
			default:
				// Channel full, continue but note potential data loss
				continue
			}
		}

		if err := iter.Error(); err != nil {
			return
		}

		// Stream changes that happened during snapshot read
		db.mu.RLock()
		for _, change := range db.changeLog {
			if change.timestamp.After(syncStart) {
				if change.deleted {
					// Send deletion marker (empty value)
					select {
					case ch <- &pb.SyncDatabaseData{
						Key:   []byte(change.player),
						Value: nil,
					}:
					default:
						continue
					}
				} else {
					// For new entries, we need to get the current state
					key := []byte(change.player)
					data, err := db.leveldb.Get(key, nil)
					if err == nil {
						select {
						case ch <- &pb.SyncDatabaseData{
							Key:   key,
							Value: data,
						}:
						default:
							continue
						}
					}
				}
			}
		}
		db.mu.RUnlock()
	}()

	return ch
}

func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true
	return db.leveldb.Close()
}

func (db *DB) NewIterator() iterator.Iterator {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil
	}

	return db.leveldb.NewIterator(util.BytesPrefix(nil), nil)
}
