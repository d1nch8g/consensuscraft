package database

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_New(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		shouldError bool
	}{
		{
			name:        "valid temp directory",
			path:        t.TempDir(),
			shouldError: false,
		},
		{
			name:        "invalid path",
			path:        "/invalid/path/that/does/not/exist",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.path)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					assert.NoError(t, db.Close())
				}
			}
		})
	}
}

func TestDB_PutGet(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name      string
		player    string
		inventory []byte
		server    string
	}{
		{
			name:      "simple inventory",
			player:    "player1",
			inventory: []byte("inventory1"),
			server:    "server1",
		},
		{
			name:      "empty inventory",
			player:    "player2",
			inventory: []byte(""),
			server:    "server1",
		},
		{
			name:      "binary data",
			player:    "player3",
			inventory: []byte{0x00, 0xFF, 0xAB, 0xCD},
			server:    "server2",
		},
		{
			name:      "large inventory",
			player:    "player4",
			inventory: make([]byte, 10000),
			server:    "server1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Put
			err := db.Put(tt.player, tt.inventory, tt.server)
			assert.NoError(t, err)

			// Get
			retrieved, err := db.Get(tt.player)
			assert.NoError(t, err)
			assert.Equal(t, tt.inventory, retrieved)
		})
	}
}

func TestDB_GetNonExistent(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Get("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestDB_MultipleServers(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Add inventory from server1
	inventory1 := []byte("inventory1")
	err = db.Put(player, inventory1, "server1")
	require.NoError(t, err)

	// Add inventory from server2 (should be newer)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	inventory2 := []byte("inventory2")
	err = db.Put(player, inventory2, "server2")
	require.NoError(t, err)

	// Get should return the latest (inventory2)
	retrieved, err := db.Get(player)
	require.NoError(t, err)
	assert.Equal(t, inventory2, retrieved)

	// Check all inventories
	entries, err := db.GetPlayerInventories(player)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Should be sorted by timestamp (newest first)
	assert.Equal(t, inventory2, entries[0].Inventory)
	assert.Equal(t, "server2", entries[0].Server)
	assert.Equal(t, inventory1, entries[1].Inventory)
	assert.Equal(t, "server1", entries[1].Server)
}

func TestDB_Delete(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player1 := "player1"
	player2 := "player2"

	// Add inventories from different servers
	err = db.Put(player1, []byte("inv1"), "server1")
	require.NoError(t, err)

	err = db.Put(player1, []byte("inv2"), "server2")
	require.NoError(t, err)

	err = db.Put(player2, []byte("inv3"), "server1")
	require.NoError(t, err)

	// Delete all entries from server1
	err = db.Delete("server1", false)
	assert.NoError(t, err)

	// Player1 should still exist with server2's inventory
	retrieved, err := db.Get(player1)
	assert.NoError(t, err)
	assert.Equal(t, []byte("inv2"), retrieved)

	// Player2 should not exist anymore
	_, err = db.Get(player2)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestDB_DeleteWithForce(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Add inventory from server1
	err = db.Put(player, []byte("inv1"), "server1")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	// Add inventory from server2 (after server1)
	err = db.Put(player, []byte("inv2"), "server2")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	// Add inventory from server3 (after server1)
	err = db.Put(player, []byte("inv3"), "server3")
	require.NoError(t, err)

	// Delete server1 with force=true (should also remove server2 and server3 entries that came after)
	err = db.Delete("server1", true)
	assert.NoError(t, err)

	// Player should not exist anymore
	_, err = db.Get(player)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestDB_StreamAll_Empty(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	ch := db.StreamAll()
	count := 0

	for range ch {
		count++
	}

	assert.Equal(t, 0, count)
}

func TestDB_StreamAll_WithData(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add test data
	testData := map[string][]byte{
		"player1": []byte("inventory1"),
		"player2": []byte("inventory2"),
		"player3": []byte("inventory3"),
	}

	for player, inventory := range testData {
		err := db.Put(player, inventory, "server1")
		require.NoError(t, err)
	}

	// Stream all data
	ch := db.StreamAll()
	received := make(map[string][]byte)

	for data := range ch {
		// Since we're storing JSON, we need to parse it to get the latest inventory
		player := string(data.Key)
		if data.Value != nil {
			// Get the latest inventory for this player
			latestInv, err := db.Get(player)
			if err == nil {
				received[player] = latestInv
			}
		}
	}

	assert.Equal(t, len(testData), len(received))
	for player, expectedInv := range testData {
		assert.Equal(t, expectedInv, received[player])
	}
}

func TestDB_ConcurrentAccess(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	const numGoroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writers
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				player := "player-" + string(rune('0'+id))
				inventory := []byte("inventory-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10)))
				server := "server-" + string(rune('0'+id))

				err := db.Put(player, inventory, server)
				assert.NoError(t, err)

				// Sometimes read what we just wrote
				if j%10 == 0 {
					retrieved, err := db.Get(player)
					assert.NoError(t, err)
					assert.NotNil(t, retrieved)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state by streaming all data
	ch := db.StreamAll()
	count := 0
	for range ch {
		count++
	}

	// Should have data (exact count depends on overwrites)
	assert.Greater(t, count, 0)
}

func TestDB_ClosedOperations(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	// Operations on closed DB should fail
	err = db.Put("player", []byte("inventory"), "server")
	assert.Equal(t, ErrClosed, err)

	_, err = db.Get("player")
	assert.Equal(t, ErrClosed, err)

	err = db.Delete("server", false)
	assert.Equal(t, ErrClosed, err)

	// Closing again should not error
	err = db.Close()
	assert.NoError(t, err)
}

func TestDB_ChangeLogBounding(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add more than 1000 entries to test log bounding
	for i := 0; i < 1500; i++ {
		player := "player-" + string(rune(i%100)) // Reuse players to avoid too many unique keys
		inventory := []byte("inventory")
		server := "server1"
		err := db.Put(player, inventory, server)
		require.NoError(t, err)
	}

	db.mu.RLock()
	assert.LessOrEqual(t, len(db.changeLog), 1000)
	db.mu.RUnlock()
}

func TestDB_StreamAll_ConcurrentWrites(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add initial data
	for i := 0; i < 10; i++ {
		player := "initial-player-" + string(rune('0'+i))
		inventory := []byte("inventory-" + string(rune('0'+i)))
		err := db.Put(player, inventory, "server1")
		require.NoError(t, err)
	}

	// Start streaming
	ch := db.StreamAll()

	// Concurrently add more data while streaming
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Let stream start
		for i := 0; i < 5; i++ {
			player := "concurrent-player-" + string(rune('0'+i))
			inventory := []byte("inventory-" + string(rune('0'+i)))
			err := db.Put(player, inventory, "server1")
			assert.NoError(t, err)
		}
	}()

	// Collect all streamed data
	receivedCount := 0
	for range ch {
		receivedCount++
	}

	wg.Wait()

	// Should have at least initial data
	assert.GreaterOrEqual(t, receivedCount, 10)
}

func TestDB_DataIntegrity(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Test that data is properly copied and not affected by mutations
	originalPlayer := "test-player"
	originalInventory := []byte("test-inventory")
	originalServer := "test-server"

	err = db.Put(originalPlayer, originalInventory, originalServer)
	require.NoError(t, err)

	// Mutate original slices
	originalInventory[0] = 'X'

	// Retrieved data should be unchanged
	retrieved, err := db.Get("test-player")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-inventory"), retrieved)

	// Check inventory entries
	entries, err := db.GetPlayerInventories("test-player")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, []byte("test-inventory"), entries[0].Inventory)
	assert.Equal(t, "test-server", entries[0].Server)
}

func TestDB_GetPlayerInventories(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Add multiple inventories
	err = db.Put(player, []byte("inv1"), "server1")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte("inv2"), "server2")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte("inv3"), "server1")
	require.NoError(t, err)

	// Get all inventories
	entries, err := db.GetPlayerInventories(player)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Should be sorted by timestamp (newest first)
	assert.Equal(t, []byte("inv3"), entries[0].Inventory)
	assert.Equal(t, "server1", entries[0].Server)
	assert.Equal(t, []byte("inv2"), entries[1].Inventory)
	assert.Equal(t, "server2", entries[1].Server)
	assert.Equal(t, []byte("inv1"), entries[2].Inventory)
	assert.Equal(t, "server1", entries[2].Server)
}

func TestDB_DeleteComplexScenario(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Timeline: server1 -> server2 -> server1 -> server3
	err = db.Put(player, []byte("inv1"), "server1")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte("inv2"), "server2")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte("inv3"), "server1")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte("inv4"), "server3")
	require.NoError(t, err)

	// Delete server1 with force=true
	// This should remove all server1 entries and everything after the latest server1 entry
	err = db.Delete("server1", true)
	require.NoError(t, err)

	// Only server2's entry should remain (it came before server1's latest entry)
	entries, err := db.GetPlayerInventories(player)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, []byte("inv2"), entries[0].Inventory)
	assert.Equal(t, "server2", entries[0].Server)
}
