package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/awnumar/memguard"
)

type KeyManager struct {
	encryptionKey *memguard.LockedBuffer
}

func NewKeyManager() *KeyManager {
	return &KeyManager{}
}

func (km *KeyManager) GenerateKey() error {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	km.encryptionKey = memguard.NewBufferFromBytes(key)
	
	// Clear the temporary key from memory
	for i := range key {
		key[i] = 0
	}

	return nil
}

func (km *KeyManager) GetKey() *memguard.LockedBuffer {
	return km.encryptionKey
}

func (km *KeyManager) SetKey(keyData []byte) error {
	if km.encryptionKey != nil {
		km.encryptionKey.Destroy()
	}
	km.encryptionKey = memguard.NewBufferFromBytes(keyData)
	return nil
}

func (km *KeyManager) Destroy() {
	if km.encryptionKey != nil {
		km.encryptionKey.Destroy()
	}
}

func (km *KeyManager) ExportKey() ([]byte, error) {
	if km.encryptionKey == nil {
		return nil, fmt.Errorf("no key available")
	}
	
	// Create a copy for export
	keyData := make([]byte, len(km.encryptionKey.Bytes()))
	copy(keyData, km.encryptionKey.Bytes())
	return keyData, nil
}
