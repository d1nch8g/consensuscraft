package database

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

// Item represents a Minecraft item in the inventory
type Item struct {
	TypeID         string                 `json:"typeId,omitempty"`
	Amount         int                    `json:"amount,omitempty"`
	NameTag        string                 `json:"nameTag,omitempty"`
	Lore           []string               `json:"lore,omitempty"`
	Enchantments   []map[string]any       `json:"enchantments,omitempty"`
	Durability     map[string]any         `json:"durability,omitempty"`
	ShulkerContents []any                 `json:"shulker_contents,omitempty"`
	// Store any other fields as raw JSON
	Extra          map[string]any         `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling for Item
func (i *Item) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if v, ok := raw["typeId"].(string); ok {
		i.TypeID = v
		delete(raw, "typeId")
	}
	if v, ok := raw["amount"].(float64); ok {
		i.Amount = int(v)
		delete(raw, "amount")
	}
	if v, ok := raw["nameTag"].(string); ok {
		i.NameTag = v
		delete(raw, "nameTag")
	}
	if v, ok := raw["lore"].([]any); ok {
		i.Lore = make([]string, len(v))
		for idx, loreItem := range v {
			if s, ok := loreItem.(string); ok {
				i.Lore[idx] = s
			}
		}
		delete(raw, "lore")
	}
	if v, ok := raw["enchantments"].([]any); ok {
		i.Enchantments = make([]map[string]any, len(v))
		for idx, enchItem := range v {
			if m, ok := enchItem.(map[string]any); ok {
				i.Enchantments[idx] = m
			}
		}
		delete(raw, "enchantments")
	}
	if v, ok := raw["durability"].(map[string]any); ok {
		i.Durability = v
		delete(raw, "durability")
	}
	if v, ok := raw["shulker_contents"].([]any); ok {
		i.ShulkerContents = v
		delete(raw, "shulker_contents")
	}

	// Store remaining fields in Extra
	if len(raw) > 0 {
		i.Extra = raw
	}

	return nil
}

// MarshalJSON implements custom marshaling for Item
func (i *Item) MarshalJSON() ([]byte, error) {
	// Create a map with all fields
	result := make(map[string]any)

	// Add extra fields first
	for k, v := range i.Extra {
		result[k] = v
	}

	// Add known fields (these will override any conflicting extra fields)
	if i.TypeID != "" {
		result["typeId"] = i.TypeID
	}
	if i.Amount != 0 {
		result["amount"] = i.Amount
	}
	if i.NameTag != "" {
		result["nameTag"] = i.NameTag
	}
	if len(i.Lore) > 0 {
		result["lore"] = i.Lore
	}
	if len(i.Enchantments) > 0 {
		result["enchantments"] = i.Enchantments
	}
	if len(i.Durability) > 0 {
		result["durability"] = i.Durability
	}
	if len(i.ShulkerContents) > 0 {
		result["shulker_contents"] = i.ShulkerContents
	}

	return json.Marshal(result)
}

// hasOriginFromServer checks if an item originates from a specific server
func (i *Item) hasOriginFromServer(server string) bool {
	if len(i.Lore) == 0 {
		return false
	}

	// Check for origin lore pattern: "Origin: server timestamp"
	originPattern := regexp.MustCompile(`^Origin:\s*(.+?)\s+\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	for _, lore := range i.Lore {
		if matches := originPattern.FindStringSubmatch(lore); len(matches) > 1 {
			originServer := strings.TrimSpace(matches[1])
			if originServer == server {
				return true
			}
		}
	}
	return false
}

// cleanShulkerContents removes items from shulker contents that originate from a specific server
func (i *Item) cleanShulkerContents(server string) bool {
	if len(i.ShulkerContents) == 0 {
		return false
	}

	var cleaned []any
	modified := false

	for _, content := range i.ShulkerContents {
		if content == nil {
			cleaned = append(cleaned, nil)
			continue
		}

		// Try to parse as Item
		contentBytes, err := json.Marshal(content)
		if err != nil {
			cleaned = append(cleaned, content)
			continue
		}

		var item Item
		if err := json.Unmarshal(contentBytes, &item); err != nil {
			cleaned = append(cleaned, content)
			continue
		}

		// Check if this item should be removed
		if item.hasOriginFromServer(server) {
			// Remove this item (don't add to cleaned)
			modified = true
			continue
		}

		// Recursively clean nested shulker contents
		if item.cleanShulkerContents(server) {
			modified = true
		}

		cleaned = append(cleaned, item)
	}

	if modified {
		i.ShulkerContents = cleaned
	}

	return modified
}

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

	// Try to unmarshal as PlayerInventories (new format)
	var playerInv PlayerInventories
	if err := json.Unmarshal(data, &playerInv); err != nil {
		// If that fails, check if it's old format (raw JSON array)
		// Try to unmarshal as raw array to validate it's valid JSON
		var rawArray []any
		if arrayErr := json.Unmarshal(data, &rawArray); arrayErr != nil {
			// Neither format worked, return the original error
			return nil, err
		}

		// It's old format, return the raw data directly
		return data, nil
	}

	if len(playerInv.Entries) == 0 {
		return nil, ErrPlayerNotFound
	}

	// Entries are already sorted by timestamp (newest first)
	return playerInv.Entries[0].Inventory, nil
}

// Delete removes all items originating from a specific server from all player inventories
// This includes items in shulker boxes and nested containers
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

		var newEntries []InventoryEntry
		var serverTimestamp time.Time
		modified := false

		// Find the latest timestamp from the server to be deleted
		for _, entry := range playerInv.Entries {
			if entry.Server == server {
				if entry.Timestamp.After(serverTimestamp) {
					serverTimestamp = entry.Timestamp
				}
			}
		}

		// Process each entry
		for _, entry := range playerInv.Entries {
			if entry.Server == server {
				// Remove all entries from this server
				modified = true
				continue
			}

			if force && !serverTimestamp.IsZero() && entry.Timestamp.After(serverTimestamp) {
				// Remove entries that came after the server's latest entry
				modified = true
				continue
			}

			// Parse and clean the inventory contents
			cleanedEntry := entry
			if cleanedInventory, inventoryModified := db.cleanInventoryContents(entry.Inventory, server); inventoryModified {
				cleanedEntry.Inventory = cleanedInventory
				modified = true
			}

			newEntries = append(newEntries, cleanedEntry)
		}

		// Only update if something changed
		if modified {
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

// cleanInventoryContents removes items originating from a specific server from an inventory
func (db *DB) cleanInventoryContents(inventoryData []byte, server string) ([]byte, bool) {
	// Try to parse as inventory array
	var inventory []any
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		// If parsing fails, return original data unchanged
		return inventoryData, false
	}

	var cleanedInventory []any
	modified := false

	for _, slot := range inventory {
		if slot == nil {
			cleanedInventory = append(cleanedInventory, nil)
			continue
		}

		// Try to parse as Item
		slotBytes, err := json.Marshal(slot)
		if err != nil {
			cleanedInventory = append(cleanedInventory, slot)
			continue
		}

		var item Item
		if err := json.Unmarshal(slotBytes, &item); err != nil {
			cleanedInventory = append(cleanedInventory, slot)
			continue
		}

		// Check if this item should be removed
		if item.hasOriginFromServer(server) {
			// Remove this item (replace with null)
			cleanedInventory = append(cleanedInventory, nil)
			modified = true
			continue
		}

		// Clean shulker contents recursively
		if item.cleanShulkerContents(server) {
			modified = true
		}

		cleanedInventory = append(cleanedInventory, item)
	}

	if !modified {
		return inventoryData, false
	}

	// Marshal the cleaned inventory
	cleanedData, err := json.Marshal(cleanedInventory)
	if err != nil {
		// If marshaling fails, return original data
		return inventoryData, false
	}

	return cleanedData, true
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
