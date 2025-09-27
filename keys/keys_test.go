package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Clean up test keys directory
	defer cleanupTestKeys(t)

	t.Run("creates new key manager with valid web address", func(t *testing.T) {
		km, err := New("example.com")
		require.NoError(t, err)
		assert.NotNil(t, km)
		assert.Equal(t, "example.com", km.webAddress)
		assert.NotNil(t, km.privateKey)
		assert.NotNil(t, km.publicKey)

		// Check that key files were created
		assert.FileExists(t, filepath.Join("keys", "example.com.private.key"))
		assert.FileExists(t, filepath.Join("keys", "example.com.public.key"))
	})

	t.Run("loads existing keys if they exist", func(t *testing.T) {
		// First create a key manager
		km1, err := New("test.com")
		require.NoError(t, err)
		
		originalPubKey, err := km1.Public()
		require.NoError(t, err)

		// Create another key manager with the same web address
		km2, err := New("test.com")
		require.NoError(t, err)

		// Should load the same keys
		loadedPubKey, err := km2.Public()
		require.NoError(t, err)
		assert.Equal(t, originalPubKey, loadedPubKey)
	})

	t.Run("sanitizes web address for filename", func(t *testing.T) {
		km, err := New("https://example.com:8080/path?query=1")
		require.NoError(t, err)
		assert.NotNil(t, km)

		// Check that sanitized filename was used
		assert.FileExists(t, filepath.Join("keys", "https___example.com_8080_path_query_1.private.key"))
		assert.FileExists(t, filepath.Join("keys", "https___example.com_8080_path_query_1.public.key"))
	})

	t.Run("returns error for empty web address", func(t *testing.T) {
		km, err := New("")
		assert.Error(t, err)
		assert.Nil(t, km)
		assert.Contains(t, err.Error(), "web address cannot be empty")
	})
}

func TestSign(t *testing.T) {
	defer cleanupTestKeys(t)

	km, err := New("test.com")
	require.NoError(t, err)

	t.Run("signs message successfully", func(t *testing.T) {
		player := "testplayer"
		inventory := []byte("inventory_data")

		signature, err := km.Sign(player, inventory)
		require.NoError(t, err)
		assert.NotNil(t, signature)
		assert.Equal(t, ed25519.SignatureSize, len(signature))
	})

	t.Run("returns error for empty player name", func(t *testing.T) {
		inventory := []byte("inventory_data")

		signature, err := km.Sign("", inventory)
		assert.Error(t, err)
		assert.Nil(t, signature)
		assert.Contains(t, err.Error(), "player name cannot be empty")
	})

	t.Run("handles nil inventory", func(t *testing.T) {
		player := "testplayer"

		signature, err := km.Sign(player, nil)
		require.NoError(t, err)
		assert.NotNil(t, signature)
	})

	t.Run("returns error when private key not initialized", func(t *testing.T) {
		km := &KeyManager{}
		signature, err := km.Sign("player", []byte("data"))
		assert.Error(t, err)
		assert.Nil(t, signature)
		assert.Contains(t, err.Error(), "private key not initialized")
	})
}

func TestVerify(t *testing.T) {
	defer cleanupTestKeys(t)

	km, err := New("test.com")
	require.NoError(t, err)

	t.Run("verifies valid signature", func(t *testing.T) {
		player := "testplayer"
		inventory := []byte("inventory_data")

		signature, err := km.Sign(player, inventory)
		require.NoError(t, err)

		err = km.Verify(player, inventory, signature)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid signature", func(t *testing.T) {
		player := "testplayer"
		inventory := []byte("inventory_data")

		// Create a fake signature
		fakeSignature := make([]byte, ed25519.SignatureSize)
		rand.Read(fakeSignature)

		err := km.Verify(player, inventory, fakeSignature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})

	t.Run("rejects signature with wrong data", func(t *testing.T) {
		player := "testplayer"
		inventory := []byte("inventory_data")

		signature, err := km.Sign(player, inventory)
		require.NoError(t, err)

		// Try to verify with different data
		err = km.Verify(player, []byte("different_data"), signature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})

	t.Run("rejects signature with wrong player", func(t *testing.T) {
		player := "testplayer"
		inventory := []byte("inventory_data")

		signature, err := km.Sign(player, inventory)
		require.NoError(t, err)

		// Try to verify with different player
		err = km.Verify("different_player", inventory, signature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})

	t.Run("returns error for empty player name", func(t *testing.T) {
		signature := make([]byte, ed25519.SignatureSize)
		err := km.Verify("", []byte("data"), signature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player name cannot be empty")
	})

	t.Run("returns error for invalid signature size", func(t *testing.T) {
		invalidSignature := []byte("too_short")
		err := km.Verify("player", []byte("data"), invalidSignature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature size")
	})

	t.Run("returns error when public key not initialized", func(t *testing.T) {
		km := &KeyManager{}
		signature := make([]byte, ed25519.SignatureSize)
		err := km.Verify("player", []byte("data"), signature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key not initialized")
	})
}

func TestPublic(t *testing.T) {
	defer cleanupTestKeys(t)

	km, err := New("test.com")
	require.NoError(t, err)

	t.Run("returns public key", func(t *testing.T) {
		pubKey, err := km.Public()
		require.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.Equal(t, ed25519.PublicKeySize, len(pubKey))
	})

	t.Run("returns error when public key not initialized", func(t *testing.T) {
		km := &KeyManager{}
		pubKey, err := km.Public()
		assert.Error(t, err)
		assert.Nil(t, pubKey)
		assert.Contains(t, err.Error(), "public key not initialized")
	})
}

func TestSave(t *testing.T) {
	defer cleanupTestKeys(t)

	km, err := New("test.com")
	require.NoError(t, err)

	t.Run("saves public key for new server", func(t *testing.T) {
		// Generate a test public key
		_, testPrivKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		testPubKey := testPrivKey.Public().(ed25519.PublicKey)

		err = km.Save("newserver.com", testPubKey)
		assert.NoError(t, err)

		// Check that the file was created
		assert.FileExists(t, filepath.Join("keys", "newserver.com.public.key"))

		// Verify the content
		savedKey, err := os.ReadFile(filepath.Join("keys", "newserver.com.public.key"))
		require.NoError(t, err)
		assert.Equal(t, []byte(testPubKey), savedKey)
	})

	t.Run("returns error for empty web address", func(t *testing.T) {
		testPubKey := make([]byte, ed25519.PublicKeySize)
		err := km.Save("", testPubKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "web address cannot be empty")
	})

	t.Run("returns error for invalid public key size", func(t *testing.T) {
		invalidPubKey := []byte("too_short")
		err := km.Save("server.com", invalidPubKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid public key size")
	})

	t.Run("returns error if key already exists", func(t *testing.T) {
		// Generate a test public key
		_, testPrivKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		testPubKey := testPrivKey.Public().(ed25519.PublicKey)

		// Save the key first time
		err = km.Save("existing.com", testPubKey)
		require.NoError(t, err)

		// Try to save again
		err = km.Save("existing.com", testPubKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("sanitizes web address for filename", func(t *testing.T) {
		_, testPrivKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		testPubKey := testPrivKey.Public().(ed25519.PublicKey)

		err = km.Save("https://server.com:8080/path", testPubKey)
		assert.NoError(t, err)

		// Check that sanitized filename was used
		assert.FileExists(t, filepath.Join("keys", "https___server.com_8080_path.public.key"))
	})
}

func TestSanitizeWebAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"https://example.com", "https___example.com"},
		{"example.com:8080", "example.com_8080"},
		{"https://example.com:8080/path?query=1", "https___example.com_8080_path_query_1"},
		{"server\\path", "server_path"},
		{"server*name", "server_name"},
		{"server<name>", "server_name_"},
		{"server|name", "server_name"},
		{"server\"name", "server_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeWebAddress(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeyPersistence(t *testing.T) {
	defer cleanupTestKeys(t)

	t.Run("keys persist across instances", func(t *testing.T) {
		// Create first instance
		km1, err := New("persistence.com")
		require.NoError(t, err)

		// Sign something
		player := "testplayer"
		inventory := []byte("test_inventory")
		signature, err := km1.Sign(player, inventory)
		require.NoError(t, err)

		// Create second instance with same web address
		km2, err := New("persistence.com")
		require.NoError(t, err)

		// Should be able to verify signature from first instance
		err = km2.Verify(player, inventory, signature)
		assert.NoError(t, err)

		// Public keys should be identical
		pub1, err := km1.Public()
		require.NoError(t, err)
		pub2, err := km2.Public()
		require.NoError(t, err)
		assert.Equal(t, pub1, pub2)
	})
}

func TestCrossVerification(t *testing.T) {
	defer cleanupTestKeys(t)

	t.Run("different key managers cannot verify each other's signatures", func(t *testing.T) {
		km1, err := New("server1.com")
		require.NoError(t, err)

		km2, err := New("server2.com")
		require.NoError(t, err)

		player := "testplayer"
		inventory := []byte("test_inventory")

		// Sign with km1
		signature, err := km1.Sign(player, inventory)
		require.NoError(t, err)

		// Try to verify with km2 - should fail
		err = km2.Verify(player, inventory, signature)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})
}

// Helper function to clean up test keys
func cleanupTestKeys(t *testing.T) {
	// Remove all test key files
	matches, err := filepath.Glob(filepath.Join("keys", "*.key"))
	if err != nil {
		t.Logf("Warning: failed to glob key files: %v", err)
		return
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			t.Logf("Warning: failed to remove %s: %v", match, err)
		}
	}
}
