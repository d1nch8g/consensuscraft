package security

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNewKeyManagerGenerated(t *testing.T) {
	km, err := NewKeyManagerGenerated()
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}
	defer km.Close()

	// Test that we can read a key
	keyReader := km.GetKeyReader()
	key := make([]byte, 32)
	n, err := io.ReadFull(keyReader, key)
	if err != nil {
		t.Fatalf("Failed to read key: %v", err)
	}
	if n != 32 {
		t.Fatalf("Expected 32 bytes, got %d", n)
	}

	// Verify key is not all zeros
	allZeros := bytes.Repeat([]byte{0}, 32)
	if bytes.Equal(key, allZeros) {
		t.Fatal("Generated key is all zeros")
	}
}

func TestNewKeyManagerFromReader(t *testing.T) {
	// Create a test key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}

	km, err := NewKeyManagerFromReader(bytes.NewReader(testKey))
	if err != nil {
		t.Fatalf("Failed to create key manager from reader: %v", err)
	}
	defer km.Close()

	// Read back the key
	keyReader := km.GetKeyReader()
	readKey := make([]byte, 32)
	n, err := io.ReadFull(keyReader, readKey)
	if err != nil {
		t.Fatalf("Failed to read key: %v", err)
	}
	if n != 32 {
		t.Fatalf("Expected 32 bytes, got %d", n)
	}

	// Verify keys match
	if !bytes.Equal(testKey, readKey) {
		t.Fatal("Read key doesn't match original key")
	}
}

func TestKeyReaderStreaming(t *testing.T) {
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}

	km, err := NewKeyManagerFromReader(bytes.NewReader(testKey))
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}
	defer km.Close()

	// Test streaming reads
	keyReader := km.GetKeyReader()

	// Read first 10 bytes
	part1 := make([]byte, 10)
	n, err := keyReader.Read(part1)
	if err != nil {
		t.Fatalf("Failed to read part 1: %v", err)
	}
	if n != 10 {
		t.Fatalf("Expected 10 bytes, got %d", n)
	}

	// Read next 15 bytes
	part2 := make([]byte, 15)
	n, err = keyReader.Read(part2)
	if err != nil {
		t.Fatalf("Failed to read part 2: %v", err)
	}
	if n != 15 {
		t.Fatalf("Expected 15 bytes, got %d", n)
	}

	// Read remaining 7 bytes
	part3 := make([]byte, 10) // Request more than available
	n, err = keyReader.Read(part3)
	if err != nil {
		t.Fatalf("Failed to read part 3: %v", err)
	}
	if n != 7 {
		t.Fatalf("Expected 7 bytes, got %d", n)
	}

	// Next read should return EOF
	_, err = keyReader.Read(make([]byte, 1))
	if err != io.EOF {
		t.Fatalf("Expected EOF, got %v", err)
	}

	// Verify combined data matches
	combined := append(part1, part2...)
	combined = append(combined, part3[:7]...)
	if !bytes.Equal(testKey, combined) {
		t.Fatal("Combined streamed data doesn't match original key")
	}
}

func TestGenerateRandomPath(t *testing.T) {
	path1 := GenerateRandomPath("test")
	path2 := GenerateRandomPath("test")

	// Verify format
	if !strings.HasPrefix(path1, "/test-") {
		t.Fatalf("Path doesn't have expected format: %s", path1)
	}

	// Verify paths are different
	if path1 == path2 {
		t.Fatal("Generated paths are identical")
	}

	// Verify length (should be /test-16hexchars = 22 chars)
	expectedLen := len("/test-") + 16
	if len(path1) != expectedLen {
		t.Fatalf("Expected path length %d, got %d: %s", expectedLen, len(path1), path1)
	}
}

func TestKeyManagerClose(t *testing.T) {
	km, err := NewKeyManagerGenerated()
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Get a key reader before closing
	keyReader := km.GetKeyReader()

	// Read some data to verify it works
	firstRead := make([]byte, 16)
	_, err = keyReader.Read(firstRead)
	if err != nil {
		t.Fatalf("Failed initial read: %v", err)
	}

	// Close the key manager
	km.Close()

	// Note: After Close(), the behavior is undefined as the buffer is destroyed
	// We can't reliably test this without potentially causing panics
}
