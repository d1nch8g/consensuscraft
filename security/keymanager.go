package security

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/awnumar/memguard"
)

// KeyManager manages encryption keys using memguard for secure memory handling
type KeyManager struct {
	buffer *memguard.LockedBuffer
}

// NewKeyManagerFromReader creates a key manager from a network-received key
func NewKeyManagerFromReader(keyReader io.Reader) (*KeyManager, error) {
	// Initialize memguard
	memguard.CatchInterrupt()

	// Create secure buffer directly from reader
	buffer, err := memguard.NewBufferFromReader(keyReader, 32) // 256-bit key
	if err != nil {
		return nil, fmt.Errorf("failed to create secure buffer from reader: %w", err)
	}

	return &KeyManager{buffer: buffer}, nil
}

// NewKeyManagerGenerated generates a new key and stores it securely
func NewKeyManagerGenerated() (*KeyManager, error) {
	// Initialize memguard
	memguard.CatchInterrupt()

	// Generate secure random buffer directly
	buffer := memguard.NewBufferRandom(32) // 256-bit key

	return &KeyManager{buffer: buffer}, nil
}

// GetKeyReader returns an io.Reader that provides streaming access to the key
func (km *KeyManager) GetKeyReader() io.Reader {
	return &keyReader{
		buffer:   km.buffer,
		position: 0,
	}
}

// Close safely destroys the key material
func (km *KeyManager) Close() {
	if km.buffer != nil {
		km.buffer.Destroy()
	}
}

// keyReader provides streaming access to the secure key buffer
type keyReader struct {
	buffer   *memguard.LockedBuffer
	position int
}

func (kr *keyReader) Read(p []byte) (int, error) {
	bufferSize := kr.buffer.Size()
	if kr.position >= bufferSize {
		return 0, io.EOF
	}

	// Calculate how much to read
	remaining := bufferSize - kr.position
	toCopy := min(len(p), remaining)

	// Copy bytes from the secure buffer
	keyBytes := kr.buffer.Bytes()
	copy(p, keyBytes[kr.position:kr.position+toCopy])
	kr.position += toCopy

	return toCopy, nil
}

// GenerateRandomPath generates a cryptographically random path for server placement
func GenerateRandomPath(prefix string) string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	return fmt.Sprintf("/%s-%x", prefix, randomBytes)
}
