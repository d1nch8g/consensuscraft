package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyManager handles cryptographic operations for ConsensusCraft
type KeyManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	webAddress string
}

// New initializes a new KeyManager instance with keys stored in keys/{webaddress}.private.key and keys/{webaddress}.public.key
// If keys don't exist, they will be generated and saved
func New(webAddress string) (*KeyManager, error) {
	if webAddress == "" {
		return nil, fmt.Errorf("web address cannot be empty")
	}

	// Sanitize web address for filename
	sanitized := sanitizeWebAddress(webAddress)

	privateKeyPath := filepath.Join("keys", sanitized+".private.key")
	publicKeyPath := filepath.Join("keys", sanitized+".public.key")

	km := &KeyManager{
		webAddress: webAddress,
	}

	// Try to load existing keys
	if err := km.loadKeys(privateKeyPath, publicKeyPath); err != nil {
		// If loading fails, generate new keys
		if err := km.generateKeys(); err != nil {
			return nil, fmt.Errorf("failed to generate keys: %w", err)
		}

		// Save the newly generated keys
		if err := km.saveKeys(privateKeyPath, publicKeyPath); err != nil {
			return nil, fmt.Errorf("failed to save keys: %w", err)
		}
	}

	return km, nil
}

// Sign signs a message with player name and inventory bytes, returning the signature
func (k *KeyManager) Sign(player string, inventory []byte) ([]byte, error) {
	if player == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	if k.privateKey == nil {
		return nil, fmt.Errorf("private key not initialized")
	}

	// Create message to sign: player name + inventory data
	message := append([]byte(player), inventory...)

	// Sign the message
	signature := ed25519.Sign(k.privateKey, message)

	return signature, nil
}

// Verify verifies a signature for the provided player name, inventory data, and signature
func (k *KeyManager) Verify(player string, inventory []byte, signature []byte) error {
	if player == "" {
		return fmt.Errorf("player name cannot be empty")
	}

	if k.publicKey == nil {
		return fmt.Errorf("public key not initialized")
	}

	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size: expected %d, got %d", ed25519.SignatureSize, len(signature))
	}

	// Recreate the message that was signed
	message := append([]byte(player), inventory...)

	// Verify the signature
	if !ed25519.Verify(k.publicKey, message, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// Public returns the public key bytes
func (k *KeyManager) Public() ([]byte, error) {
	if k.publicKey == nil {
		return nil, fmt.Errorf("public key not initialized")
	}

	return k.publicKey, nil
}

// Save saves a public key for a server if there is no existing key for that server
func (k *KeyManager) Save(webAddress string, pubkey []byte) error {
	if webAddress == "" {
		return fmt.Errorf("web address cannot be empty")
	}

	if len(pubkey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(pubkey))
	}

	// Sanitize web address for filename
	sanitized := sanitizeWebAddress(webAddress)
	publicKeyPath := filepath.Join("keys", sanitized+".public.key")

	// Check if key already exists
	if _, err := os.Stat(publicKeyPath); err == nil {
		return fmt.Errorf("public key for %s already exists", webAddress)
	}

	// Ensure keys directory exists
	if err := os.MkdirAll("keys", 0755); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Save the public key
	if err := os.WriteFile(publicKeyPath, pubkey, 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// generateKeys generates a new Ed25519 key pair
func (k *KeyManager) generateKeys() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	k.privateKey = privateKey
	k.publicKey = publicKey

	return nil
}

// loadKeys loads existing keys from files
func (k *KeyManager) loadKeys(privateKeyPath, publicKeyPath string) error {
	// Load private key
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	if len(privateKeyData) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKeyData))
	}

	// Load public key
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	if len(publicKeyData) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyData))
	}

	k.privateKey = ed25519.PrivateKey(privateKeyData)
	k.publicKey = ed25519.PublicKey(publicKeyData)

	return nil
}

// saveKeys saves keys to files
func (k *KeyManager) saveKeys(privateKeyPath, publicKeyPath string) error {
	// Ensure keys directory exists
	if err := os.MkdirAll("keys", 0755); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Save private key with restricted permissions
	if err := os.WriteFile(privateKeyPath, k.privateKey, 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key
	if err := os.WriteFile(publicKeyPath, k.publicKey, 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// sanitizeWebAddress sanitizes a web address to be safe for use as a filename
func sanitizeWebAddress(webAddress string) string {
	// Replace unsafe characters with underscores
	sanitized := strings.ReplaceAll(webAddress, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, "?", "_")
	sanitized = strings.ReplaceAll(sanitized, "=", "_")
	sanitized = strings.ReplaceAll(sanitized, "*", "_")
	sanitized = strings.ReplaceAll(sanitized, "<", "_")
	sanitized = strings.ReplaceAll(sanitized, ">", "_")
	sanitized = strings.ReplaceAll(sanitized, "|", "_")
	sanitized = strings.ReplaceAll(sanitized, "\"", "_")
	
	return sanitized
}
