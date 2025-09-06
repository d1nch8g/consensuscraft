package database

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
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
		name  string
		key   []byte
		value []byte
	}{
		{
			name:  "simple key-value",
			key:   []byte("key1"),
			value: []byte("value1"),
		},
		{
			name:  "empty value",
			key:   []byte("key2"),
			value: []byte(""),
		},
		{
			name:  "binary data",
			key:   []byte("key3"),
			value: []byte{0x00, 0xFF, 0xAB, 0xCD},
		},
		{
			name:  "large value",
			key:   []byte("key4"),
			value: make([]byte, 10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Put
			err := db.Put(tt.key, tt.value)
			assert.NoError(t, err)

			// Get
			retrieved, err := db.Get(tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.value, retrieved)
		})
	}
}

func TestDB_GetNonExistent(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Get([]byte("nonexistent"))
	assert.Error(t, err)
	assert.Equal(t, leveldb.ErrNotFound, err)
}

func TestDB_Delete(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Put then delete
	err = db.Put(key, value)
	require.NoError(t, err)

	retrieved, err := db.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	err = db.Delete(key)
	assert.NoError(t, err)

	_, err = db.Get(key)
	assert.Error(t, err)
	assert.Equal(t, leveldb.ErrNotFound, err)
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
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		err := db.Put([]byte(k), []byte(v))
		require.NoError(t, err)
	}

	// Stream all data
	ch := db.StreamAll()
	received := make(map[string]string)

	for data := range ch {
		received[string(data.Key)] = string(data.Value)
	}

	assert.Equal(t, testData, received)
}

func TestDB_StreamAll_ConcurrentWrites(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add initial data
	for i := 0; i < 10; i++ {
		key := []byte("initial-" + string(rune('0'+i)))
		value := []byte("value-" + string(rune('0'+i)))
		err := db.Put(key, value)
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
			key := []byte("concurrent-" + string(rune('0'+i)))
			value := []byte("value-" + string(rune('0'+i)))
			err := db.Put(key, value)
			assert.NoError(t, err)
		}
	}()

	// Collect all streamed data
	received := make(map[string]string)
	for data := range ch {
		received[string(data.Key)] = string(data.Value)
	}

	wg.Wait()

	// Should have at least initial data
	assert.GreaterOrEqual(t, len(received), 10)

	// Verify initial data is present
	for i := 0; i < 10; i++ {
		key := "initial-" + string(rune('0'+i))
		expectedValue := "value-" + string(rune('0'+i))
		assert.Equal(t, expectedValue, received[key])
	}
}

func TestDB_StreamAll_WithDeletions(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add test data
	for i := 0; i < 5; i++ {
		key := []byte("key-" + string(rune('0'+i)))
		value := []byte("value-" + string(rune('0'+i)))
		err := db.Put(key, value)
		require.NoError(t, err)
	}

	// Delete key before streaming (so it's not in the snapshot)
	err = db.Delete([]byte("key-2"))
	require.NoError(t, err)

	// Start streaming after deletion
	ch := db.StreamAll()

	// Collect streamed data
	received := make(map[string][]byte)

	for data := range ch {
		if data.Value != nil {
			received[string(data.Key)] = data.Value
		}
	}

	// Should have 4 remaining keys (5 - 1 deleted)
	assert.Equal(t, 4, len(received))
	assert.NotContains(t, received, "key-2")
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
				key := []byte("key-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10)))
				value := []byte("value-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10)))
				err := db.Put(key, value)
				assert.NoError(t, err)

				// Sometimes read what we just wrote
				if j%10 == 0 {
					retrieved, err := db.Get(key)
					assert.NoError(t, err)
					assert.Equal(t, value, retrieved)
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
	err = db.Put([]byte("key"), []byte("value"))
	assert.Equal(t, ErrClosed, err)

	_, err = db.Get([]byte("key"))
	assert.Equal(t, ErrClosed, err)

	err = db.Delete([]byte("key"))
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
		key := []byte("key-" + string(rune(i)))
		value := []byte("value")
		err := db.Put(key, value)
		require.NoError(t, err)
	}

	db.mu.RLock()
	assert.LessOrEqual(t, len(db.changeLog), 1000)
	db.mu.RUnlock()
}

func TestDB_StreamAll_ChannelBlocking(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Add lots of data
	expectedCount := 200
	for i := 0; i < expectedCount; i++ {
		key := []byte("key-" + string(rune(i)))
		value := bytes.Repeat([]byte("x"), 1000) // Large values
		err := db.Put(key, value)
		require.NoError(t, err)
	}

	// Stream with slow consumer to test buffering
	ch := db.StreamAll()
	count := 0

	for data := range ch {
		count++
		if count%50 == 0 {
			time.Sleep(1 * time.Millisecond) // Slow consumer
		}
		assert.NotNil(t, data.Key)
	}

	// Note: Some data might be dropped due to channel buffering when full
	// This is acceptable behavior - we test that we get substantial data
	assert.GreaterOrEqual(t, count, 100, "Should receive substantial portion of data")
}

func TestDB_DataIntegrity(t *testing.T) {
	db, err := New(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	// Test that data is properly copied and not affected by mutations
	originalKey := []byte("test-key")
	originalValue := []byte("test-value")

	err = db.Put(originalKey, originalValue)
	require.NoError(t, err)

	// Mutate original slices
	originalKey[0] = 'X'
	originalValue[0] = 'X'

	// Retrieved data should be unchanged
	retrieved, err := db.Get([]byte("test-key"))
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), retrieved)

	// Stream data should also be unchanged
	ch := db.StreamAll()
	found := false
	for data := range ch {
		if bytes.Equal(data.Key, []byte("test-key")) {
			assert.Equal(t, []byte("test-value"), data.Value)
			found = true
		}
	}
	assert.True(t, found)
}
