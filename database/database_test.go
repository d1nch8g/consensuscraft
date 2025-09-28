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
	for i := range 10 {
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

func TestDB_DeleteWithItemOriginCleaning(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Create inventory with mixed items from different servers
	inventoryWithMixedItems := `[
		{
			"typeId": "minecraft:diamond_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:bread",
			"amount": 64,
			"lore": ["Origin: server2"]
		},
		{
			"typeId": "minecraft:iron_ingot",
			"amount": 32,
			"lore": ["Origin: server1"]
		},
		null,
		{
			"typeId": "minecraft:shulker_box",
			"amount": 1,
			"lore": ["Origin: server2"],
			"shulkerContents": [
				{
					"typeId": "minecraft:diamond",
					"amount": 10,
					"lore": ["Origin: server1"]
				},
				{
					"typeId": "minecraft:gold_ingot",
					"amount": 20,
					"lore": ["Origin: server2"]
				}
			]
		}
	]`

	// Put this inventory from server2 (but it contains items from server1)
	err = db.Put("player1", []byte(inventoryWithMixedItems), "server2")
	require.NoError(t, err)

	// Delete server1 - should clean items with server1 origin from all inventories
	err = db.Delete("server1", false)
	require.NoError(t, err)

	// Get the cleaned inventory
	cleanedInv, err := db.Get("player1")
	require.NoError(t, err)

	// Verify that server1 items were removed
	cleanedStr := string(cleanedInv)
	assert.NotContains(t, cleanedStr, "Origin: server1")
	assert.Contains(t, cleanedStr, "Origin: server2")

	// Should still contain server2 items
	assert.Contains(t, cleanedStr, "minecraft:bread")
	assert.Contains(t, cleanedStr, "minecraft:gold_ingot")

	// Should not contain server1 items
	assert.NotContains(t, cleanedStr, "minecraft:diamond_sword")
	assert.NotContains(t, cleanedStr, "minecraft:iron_ingot")
	assert.NotContains(t, cleanedStr, "minecraft:diamond")
}

func TestDB_CrossServerItemValidation(t *testing.T) {
	// This test demonstrates how validation would catch servers producing items from other servers
	// Note: The actual validation happens at the application level using the validator
	validator := NewItemValidator()

	t.Run("server producing items with wrong origin", func(t *testing.T) {
		// Server1 tries to produce an item with server2's origin
		maliciousInventory := `[
			{
				"typeId": "minecraft:diamond_sword",
				"amount": 1,
				"lore": ["Origin: server2"]
			}
		]`

		// Validate this inventory as if it came from server1
		errors := validator.ValidateInventory([]byte(maliciousInventory), "server1", "player1")

		// Should detect wrong origin
		assert.Len(t, errors, 1)
		assert.Equal(t, "wrong_origin", errors[0].ErrorType)
		assert.Contains(t, errors[0].Message, "server2")
		assert.Contains(t, errors[0].Message, "server1")
	})

	t.Run("server producing items without origin", func(t *testing.T) {
		// Server tries to produce items without proper origin
		maliciousInventory := `[
			{
				"typeId": "minecraft:diamond_sword",
				"amount": 1
			}
		]`

		// Validate this inventory
		errors := validator.ValidateInventory([]byte(maliciousInventory), "server1", "player1")

		// Should detect missing origin
		assert.Len(t, errors, 1)
		assert.Equal(t, "missing_origin", errors[0].ErrorType)
	})

	t.Run("valid server producing own items", func(t *testing.T) {
		// Server1 produces items with correct origin
		validInventory := `[
			{
				"typeId": "minecraft:diamond_sword",
				"amount": 1,
				"lore": ["Origin: server1"]
			},
			{
				"typeId": "minecraft:bread",
				"amount": 64,
				"lore": ["Origin: server1"]
			}
		]`

		// Validate this inventory
		errors := validator.ValidateInventory([]byte(validInventory), "server1", "player1")

		// Should pass validation
		assert.Len(t, errors, 0)
	})
}

func TestDB_ServerDeletionWithNestedItems(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Create complex inventory with nested shulker boxes containing items from different servers
	complexInventory := `[
		{
			"typeId": "minecraft:shulker_box",
			"amount": 1,
			"lore": ["Origin: server1"],
			"shulkerContents": [
				{
					"typeId": "minecraft:diamond",
					"amount": 10,
					"lore": ["Origin: server1"]
				},
				{
					"typeId": "minecraft:gold_ingot",
					"amount": 20,
					"lore": ["Origin: server2"]
				},
				{
					"typeId": "minecraft:shulker_box",
					"amount": 1,
					"lore": ["Origin: server2"],
					"shulkerContents": [
						{
							"typeId": "minecraft:iron_ingot",
							"amount": 30,
							"lore": ["Origin: server1"]
						},
						{
							"typeId": "minecraft:coal",
							"amount": 40,
							"lore": ["Origin: server3"]
						}
					]
				}
			]
		},
		{
			"typeId": "minecraft:bread",
			"amount": 64,
			"lore": ["Origin: server2"]
		}
	]`

	// Store this inventory from server2 (so it won't be deleted when we delete server1)
	err = db.Put("player1", []byte(complexInventory), "server2")
	require.NoError(t, err)

	// Delete server1 - should remove all items with server1 origin, including nested ones
	err = db.Delete("server1", false)
	require.NoError(t, err)

	// Get the cleaned inventory
	cleanedInv, err := db.Get("player1")
	require.NoError(t, err)

	cleanedStr := string(cleanedInv)

	// Should not contain any server1 items
	assert.NotContains(t, cleanedStr, "Origin: server1")

	// Should still contain server2 and server3 items
	assert.Contains(t, cleanedStr, "Origin: server2")
	assert.Contains(t, cleanedStr, "Origin: server3")

	// Verify specific items
	assert.Contains(t, cleanedStr, "minecraft:bread")      // server2 item
	assert.Contains(t, cleanedStr, "minecraft:coal")       // server3 item in nested shulker
	assert.Contains(t, cleanedStr, "minecraft:gold_ingot") // server2 item

	// Should not contain server1 items
	assert.NotContains(t, cleanedStr, "minecraft:diamond")    // server1 item
	assert.NotContains(t, cleanedStr, "minecraft:iron_ingot") // server1 item in nested shulker
}

func TestDB_MultiplePlayerServerDeletion(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Create inventories for multiple players with items from server1
	player1Inv := `[
		{
			"typeId": "minecraft:diamond_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:bread",
			"amount": 64,
			"lore": ["Origin: server2"]
		}
	]`

	player2Inv := `[
		{
			"typeId": "minecraft:iron_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		}
	]`

	player3Inv := `[
		{
			"typeId": "minecraft:gold_ingot",
			"amount": 32,
			"lore": ["Origin: server2"]
		}
	]`

	// Store inventories
	err = db.Put("player1", []byte(player1Inv), "server2")
	require.NoError(t, err)

	err = db.Put("player2", []byte(player2Inv), "server1")
	require.NoError(t, err)

	err = db.Put("player3", []byte(player3Inv), "server2")
	require.NoError(t, err)

	// Delete server1 - should clean all inventories
	err = db.Delete("server1", false)
	require.NoError(t, err)

	// Check player1 - should have server1 items removed but keep server2 items
	player1Result, err := db.Get("player1")
	require.NoError(t, err)
	player1Str := string(player1Result)
	assert.NotContains(t, player1Str, "Origin: server1")
	assert.Contains(t, player1Str, "Origin: server2")
	assert.Contains(t, player1Str, "minecraft:bread")
	assert.NotContains(t, player1Str, "minecraft:diamond_sword")

	// Check player2 - should be deleted entirely (only had server1 items)
	_, err = db.Get("player2")
	assert.Equal(t, ErrPlayerNotFound, err)

	// Check player3 - should be unchanged (no server1 items)
	player3Result, err := db.Get("player3")
	require.NoError(t, err)
	player3Str := string(player3Result)
	assert.Contains(t, player3Str, "Origin: server2")
	assert.Contains(t, player3Str, "minecraft:gold_ingot")
}

func TestDB_ServerDeletionPreservesOtherServers(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Add inventories from multiple servers over time
	server1Inv := `[{"typeId": "minecraft:diamond", "amount": 10, "lore": ["Origin: server1"]}]`
	server2Inv := `[{"typeId": "minecraft:iron_ingot", "amount": 20, "lore": ["Origin: server2"]}]`
	server3Inv := `[{"typeId": "minecraft:gold_ingot", "amount": 30, "lore": ["Origin: server3"]}]`

	err = db.Put(player, []byte(server1Inv), "server1")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte(server2Inv), "server2")
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = db.Put(player, []byte(server3Inv), "server3")
	require.NoError(t, err)

	// Verify we have all 3 entries
	entries, err := db.GetPlayerInventories(player)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Delete server2
	err = db.Delete("server2", false)
	require.NoError(t, err)

	// Should still have server1 and server3 entries
	entries, err = db.GetPlayerInventories(player)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify correct servers remain
	servers := make(map[string]bool)
	for _, entry := range entries {
		servers[entry.Server] = true
	}
	assert.True(t, servers["server1"])
	assert.True(t, servers["server3"])
	assert.False(t, servers["server2"])

	// Latest inventory should be from server3
	latestInv, err := db.Get(player)
	require.NoError(t, err)
	assert.Contains(t, string(latestInv), "Origin: server3")
	assert.Contains(t, string(latestInv), "minecraft:gold_ingot")
}

func TestDB_VirtualInventoryScenario_ItemsDisappearToVirtualStorage(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player := "testplayer"

	// Scenario 1: Player produces items with server1 origin
	initialInventory := `[
		{
			"typeId": "minecraft:diamond_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:diamond",
			"amount": 64,
			"lore": ["Origin: server1"]
		}
	]`

	// Store initial inventory from server1
	err = db.Put(player, []byte(initialInventory), "server1")
	require.NoError(t, err)

	// Scenario: Player updates ender chest on server2, items from server1 disappear
	// This simulates the player moving items from their ender chest to another location
	// Items with server1 origin should be moved to server2's virtual inventory
	updatedInventory := `[
		{
			"typeId": "minecraft:bread",
			"amount": 32,
			"lore": ["Origin: server2"]
		}
	]`

	// Store updated inventory from server2 (items from server1 are gone)
	err = db.Put(player, []byte(updatedInventory), "server2")
	require.NoError(t, err)

	// Get the current inventory - should only contain server2 items
	currentInv, err := db.Get(player)
	require.NoError(t, err)
	currentStr := string(currentInv)

	// Should contain server2 items
	assert.Contains(t, currentStr, "Origin: server2")
	assert.Contains(t, currentStr, "minecraft:bread")

	// Should not contain server1 items (they disappeared)
	assert.NotContains(t, currentStr, "Origin: server1")
	assert.NotContains(t, currentStr, "minecraft:diamond_sword")
	assert.NotContains(t, currentStr, "minecraft:diamond")

	// In a real implementation, the disappeared server1 items would be tracked
	// in server2's virtual inventory for potential future validation
	// This test demonstrates the scenario where items disappear from ender chest
}

func TestDB_VirtualInventoryScenario_ItemsReappearFromOtherPlayer(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	player1 := "player1"
	player2 := "player2"

	// Scenario: Player1 had items from server1 that disappeared (moved to virtual storage)
	// Now player2 appears with those same items from server1

	// Player1's current inventory (server1 items disappeared)
	player1Inventory := `[
		{
			"typeId": "minecraft:bread",
			"amount": 32,
			"lore": ["Origin: server2"]
		}
	]`

	err = db.Put(player1, []byte(player1Inventory), "server2")
	require.NoError(t, err)

	// Player2 appears with the "missing" server1 items
	player2Inventory := `[
		{
			"typeId": "minecraft:diamond_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:diamond",
			"amount": 64,
			"lore": ["Origin: server1"]
		}
	]`

	err = db.Put(player2, []byte(player2Inventory), "server1")
	require.NoError(t, err)

	// Verify both players have their respective inventories
	player1Inv, err := db.Get(player1)
	require.NoError(t, err)
	player1Str := string(player1Inv)
	assert.Contains(t, player1Str, "Origin: server2")
	assert.NotContains(t, player1Str, "Origin: server1")

	player2Inv, err := db.Get(player2)
	require.NoError(t, err)
	player2Str := string(player2Inv)
	assert.Contains(t, player2Str, "Origin: server1")
	assert.NotContains(t, player2Str, "Origin: server2")

	// In a real implementation, when player2 appears with server1 items,
	// those items would be removed from server2's virtual inventory
	// This prevents the same items from being "spent" multiple times
}

func TestDB_CrossServerProductionValidation(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Scenario 3: Server tries to produce items with wrong origin - should be caught by validation
	// This test demonstrates that the database itself doesn't prevent this,
	// but the validation layer should catch it

	player := "testplayer"

	// Server2 tries to produce items with server1's origin (malicious behavior)
	maliciousInventory := `[
		{
			"typeId": "minecraft:netherite_sword",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:netherite_ingot",
			"amount": 16,
			"lore": ["Origin: server1"]
		}
	]`

	// The database Put operation itself will succeed (it just stores data)
	// But in a real system, this should be caught by the validation layer
	err = db.Put(player, []byte(maliciousInventory), "server2")
	require.NoError(t, err, "Database Put should succeed - validation happens at application level")

	// Verify the data was stored
	storedInv, err := db.Get(player)
	require.NoError(t, err)
	storedStr := string(storedInv)

	// The malicious items are stored, but in a real system they would be invalid
	assert.Contains(t, storedStr, "Origin: server1")
	assert.Contains(t, storedStr, "minecraft:netherite_sword")

	// This demonstrates why the validation layer is crucial - it prevents
	// servers from producing items with wrong origins
}

func TestDB_ComplexCrossServerScenario(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Complex scenario combining all the elements:
	// 1. Items disappear from one server
	// 2. Same items appear on another player
	// 3. Validation prevents wrong origin production

	player1 := "player1"
	player2 := "player2"

	// Phase 1: Player1 has items from server1
	phase1Inventory := `[
		{
			"typeId": "minecraft:diamond_pickaxe",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:diamond",
			"amount": 32,
			"lore": ["Origin: server1"]
		}
	]`

	err = db.Put(player1, []byte(phase1Inventory), "server1")
	require.NoError(t, err)

	// Phase 2: Player1 updates on server2, items disappear
	phase2Inventory := `[
		{
			"typeId": "minecraft:stone",
			"amount": 64,
			"lore": ["Origin: server2"]
		}
	]`

	err = db.Put(player1, []byte(phase2Inventory), "server2")
	require.NoError(t, err)

	// Phase 3: Player2 appears with the "missing" server1 items
	player2Inventory := `[
		{
			"typeId": "minecraft:diamond_pickaxe",
			"amount": 1,
			"lore": ["Origin: server1"]
		},
		{
			"typeId": "minecraft:diamond",
			"amount": 32,
			"lore": ["Origin: server1"]
		}
	]`

	err = db.Put(player2, []byte(player2Inventory), "server1")
	require.NoError(t, err)

	// Verify final state
	player1Final, err := db.Get(player1)
	require.NoError(t, err)
	player1Str := string(player1Final)
	assert.Contains(t, player1Str, "Origin: server2")
	assert.NotContains(t, player1Str, "Origin: server1")

	player2Final, err := db.Get(player2)
	require.NoError(t, err)
	player2Str := string(player2Final)
	assert.Contains(t, player2Str, "Origin: server1")
	assert.NotContains(t, player2Str, "Origin: server2")

	// This complex scenario demonstrates the need for:
	// 1. Virtual inventory tracking when items disappear
	// 2. Cross-server item validation
	// 3. Prevention of item duplication across servers
}

func TestDB_ServerVirtualInventoryTracking(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// This test demonstrates the concept of virtual server inventory tracking
	// In a real implementation, the database would track which items are available
	// to each server from other servers

	player := "testplayer"

	// Server1 produces items
	server1Inventory := `[
		{
			"typeId": "minecraft:emerald",
			"amount": 16,
			"lore": ["Origin: server1"]
		}
	]`

	err = db.Put(player, []byte(server1Inventory), "server1")
	require.NoError(t, err)

	// Player moves to server2, items disappear from ender chest
	// In a real system, this would trigger virtual inventory tracking
	server2Inventory := `[]` // Empty inventory - items disappeared

	err = db.Put(player, []byte(server2Inventory), "server2")
	require.NoError(t, err)

	// The disappeared server1 items would be tracked in server2's virtual inventory
	// This allows server2 to "spend" those items when they appear elsewhere

	// Later, if the same items appear on another player from server1,
	// they would be validated against the virtual inventory to prevent duplication

	// This test demonstrates the database's role in storing the state,
	// while the validation layer handles the actual virtual inventory logic
}
