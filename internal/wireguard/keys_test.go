package wireguard

import (
	"encoding/base64"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Run("should generate valid key pair", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		require.NoError(t, err)
		require.NotNil(t, keyPair)
		
		assert.NotEmpty(t, keyPair.PrivateKey)
		assert.NotEmpty(t, keyPair.PublicKey)
		assert.NotEqual(t, keyPair.PrivateKey, keyPair.PublicKey)
	})

	t.Run("should generate valid base64 encoded keys", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		require.NoError(t, err)
		
		_, err = base64.StdEncoding.DecodeString(keyPair.PrivateKey)
		assert.NoError(t, err, "Private key should be valid base64")
		
		_, err = base64.StdEncoding.DecodeString(keyPair.PublicKey)
		assert.NoError(t, err, "Public key should be valid base64")
	})

	t.Run("should generate 32-byte keys", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		require.NoError(t, err)
		
		privateBytes, err := base64.StdEncoding.DecodeString(keyPair.PrivateKey)
		require.NoError(t, err)
		assert.Len(t, privateBytes, 32, "Private key should be 32 bytes")
		
		publicBytes, err := base64.StdEncoding.DecodeString(keyPair.PublicKey)
		require.NoError(t, err)
		assert.Len(t, publicBytes, 32, "Public key should be 32 bytes")
	})

	t.Run("should generate unique key pairs", func(t *testing.T) {
		keyPair1, err := GenerateKeyPair()
		require.NoError(t, err)
		
		keyPair2, err := GenerateKeyPair()
		require.NoError(t, err)
		
		assert.NotEqual(t, keyPair1.PrivateKey, keyPair2.PrivateKey)
		assert.NotEqual(t, keyPair1.PublicKey, keyPair2.PublicKey)
	})
}

func TestKeyPair_PrivateKeyBytes(t *testing.T) {
	t.Run("should return correct private key bytes", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		require.NoError(t, err)
		
		bytes, err := keyPair.PrivateKeyBytes()
		require.NoError(t, err)
		
		expectedBytes, err := base64.StdEncoding.DecodeString(keyPair.PrivateKey)
		require.NoError(t, err)
		
		assert.Equal(t, expectedBytes, bytes[:])
	})

	t.Run("should handle invalid base64", func(t *testing.T) {
		keyPair := &KeyPair{PrivateKey: "invalid-base64!@#"}
		
		_, err := keyPair.PrivateKeyBytes()
		assert.Error(t, err)
	})
}

func TestKeyPair_PublicKeyBytes(t *testing.T) {
	t.Run("should return correct public key bytes", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		require.NoError(t, err)
		
		bytes, err := keyPair.PublicKeyBytes()
		require.NoError(t, err)
		
		expectedBytes, err := base64.StdEncoding.DecodeString(keyPair.PublicKey)
		require.NoError(t, err)
		
		assert.Equal(t, expectedBytes, bytes[:])
	})

	t.Run("should handle invalid base64", func(t *testing.T) {
		keyPair := &KeyPair{PublicKey: "invalid-base64!@#"}
		
		_, err := keyPair.PublicKeyBytes()
		assert.Error(t, err)
	})
}