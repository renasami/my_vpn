// Package auth provides authentication and authorization functionality for the VPN server.
// It implements JWT-based authentication, user management, and session handling
// with support for password hashing and middleware integration.
package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManager(t *testing.T) {
	t.Run("should create auth manager with default settings", func(t *testing.T) {
		manager := NewAuthManager("test-secret")
		
		assert.NotNil(t, manager)
		assert.Equal(t, "test-secret", manager.jwtSecret)
		assert.Equal(t, 24*time.Hour, manager.tokenExpiry)
	})

	t.Run("should create auth manager with custom settings", func(t *testing.T) {
		expiry := 2 * time.Hour
		manager := NewAuthManagerWithConfig("custom-secret", expiry)
		
		assert.NotNil(t, manager)
		assert.Equal(t, "custom-secret", manager.jwtSecret)
		assert.Equal(t, expiry, manager.tokenExpiry)
	})
}

func TestAuthManager_HashPassword(t *testing.T) {
	manager := NewAuthManager("test-secret")
	
	t.Run("should hash password successfully", func(t *testing.T) {
		password := "testpassword123"
		hash, err := manager.HashPassword(password)
		
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		assert.Greater(t, len(hash), 20) // bcrypt hashes are typically longer
	})

	t.Run("should generate different hashes for same password", func(t *testing.T) {
		password := "testpassword123"
		hash1, err := manager.HashPassword(password)
		require.NoError(t, err)
		
		hash2, err := manager.HashPassword(password)
		require.NoError(t, err)
		
		assert.NotEqual(t, hash1, hash2) // bcrypt includes salt
	})

	t.Run("should handle empty password", func(t *testing.T) {
		hash, err := manager.HashPassword("")
		
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestAuthManager_VerifyPassword(t *testing.T) {
	manager := NewAuthManager("test-secret")
	
	t.Run("should verify correct password", func(t *testing.T) {
		password := "testpassword123"
		hash, err := manager.HashPassword(password)
		require.NoError(t, err)
		
		valid := manager.VerifyPassword(password, hash)
		assert.True(t, valid)
	})

	t.Run("should reject incorrect password", func(t *testing.T) {
		password := "testpassword123"
		wrongPassword := "wrongpassword"
		hash, err := manager.HashPassword(password)
		require.NoError(t, err)
		
		valid := manager.VerifyPassword(wrongPassword, hash)
		assert.False(t, valid)
	})

	t.Run("should handle invalid hash", func(t *testing.T) {
		password := "testpassword123"
		invalidHash := "invalid-hash"
		
		valid := manager.VerifyPassword(password, invalidHash)
		assert.False(t, valid)
	})
}

func TestAuthManager_GenerateToken(t *testing.T) {
	manager := NewAuthManager("test-secret")
	
	t.Run("should generate valid JWT token", func(t *testing.T) {
		userID := uint(123)
		username := "testuser"
		
		token, err := manager.GenerateToken(userID, username)
		
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Contains(t, token, ".") // JWT has dots separating sections
	})

	t.Run("should generate different tokens for different users", func(t *testing.T) {
		token1, err := manager.GenerateToken(1, "user1")
		require.NoError(t, err)
		
		token2, err := manager.GenerateToken(2, "user2")
		require.NoError(t, err)
		
		assert.NotEqual(t, token1, token2)
	})
}

func TestAuthManager_ValidateToken(t *testing.T) {
	manager := NewAuthManager("test-secret")
	
	t.Run("should validate valid token", func(t *testing.T) {
		userID := uint(123)
		username := "testuser"
		
		token, err := manager.GenerateToken(userID, username)
		require.NoError(t, err)
		
		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
	})

	t.Run("should reject invalid token", func(t *testing.T) {
		invalidToken := "invalid.jwt.token"
		
		_, err := manager.ValidateToken(invalidToken)
		assert.Error(t, err)
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		wrongManager := NewAuthManager("wrong-secret")
		rightManager := NewAuthManager("right-secret")
		
		token, err := wrongManager.GenerateToken(123, "testuser")
		require.NoError(t, err)
		
		_, err = rightManager.ValidateToken(token)
		assert.Error(t, err)
	})

	t.Run("should reject expired token", func(t *testing.T) {
		// Create manager with very short expiry
		shortManager := NewAuthManagerWithConfig("test-secret", 1*time.Millisecond)
		
		token, err := shortManager.GenerateToken(123, "testuser")
		require.NoError(t, err)
		
		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)
		
		_, err = shortManager.ValidateToken(token)
		assert.Error(t, err)
	})
}

func TestAuthManager_RefreshToken(t *testing.T) {
	manager := NewAuthManager("test-secret")
	
	t.Run("should refresh valid token", func(t *testing.T) {
		userID := uint(123)
		username := "testuser"
		
		originalToken, err := manager.GenerateToken(userID, username)
		require.NoError(t, err)
		
		// Wait to ensure different timestamps
		time.Sleep(1 * time.Second)
		
		newToken, err := manager.RefreshToken(originalToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		
		// The important test is that the new token is valid and has correct claims
		claims, err := manager.ValidateToken(newToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		
		// Also verify that both tokens are valid (for grace period)
		originalClaims, err := manager.ValidateToken(originalToken)
		require.NoError(t, err)
		assert.Equal(t, userID, originalClaims.UserID)
	})

	t.Run("should reject invalid token for refresh", func(t *testing.T) {
		invalidToken := "invalid.jwt.token"
		
		_, err := manager.RefreshToken(invalidToken)
		assert.Error(t, err)
	})
}

func TestClaims_Valid(t *testing.T) {
	t.Run("should validate non-expired claims", func(t *testing.T) {
		claims := &Claims{
			UserID:   123,
			Username: "testuser",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		
		err := claims.Valid()
		assert.NoError(t, err)
	})

	t.Run("should reject expired claims", func(t *testing.T) {
		claims := &Claims{
			UserID:   123,
			Username: "testuser",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		
		err := claims.Valid()
		assert.Error(t, err)
	})
}

func TestGenerateSecureSecret(t *testing.T) {
	t.Run("should generate secure secret", func(t *testing.T) {
		secret, err := GenerateSecureSecret()
		
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.GreaterOrEqual(t, len(secret), 32) // Should be at least 256 bits
	})

	t.Run("should generate different secrets", func(t *testing.T) {
		secret1, err := GenerateSecureSecret()
		require.NoError(t, err)
		
		secret2, err := GenerateSecureSecret()
		require.NoError(t, err)
		
		assert.NotEqual(t, secret1, secret2)
	})
}