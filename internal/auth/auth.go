// Package auth provides authentication and authorization functionality for the VPN server.
// It implements JWT-based authentication, user management, and session handling
// with support for password hashing and middleware integration.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthManager handles authentication operations including JWT token management
// and password hashing. It provides a secure authentication system for the VPN server.
type AuthManager struct {
	jwtSecret   string        // Secret key for JWT token signing and verification
	tokenExpiry time.Duration // Duration for which tokens remain valid
}

// Claims represents the JWT claims structure for authenticated users.
// It contains user identification and authorization information embedded in tokens.
type Claims struct {
	UserID   uint   `json:"user_id"`  // Unique identifier for the user
	Username string `json:"username"` // Username for display and identification
	jwt.RegisteredClaims
}

// NewAuthManager creates a new authentication manager with default settings.
// The default token expiry is set to 24 hours for security balance between
// usability and protection against token theft.
// Returns a pointer to the newly created AuthManager.
func NewAuthManager(jwtSecret string) *AuthManager {
	return &AuthManager{
		jwtSecret:   jwtSecret,
		tokenExpiry: 24 * time.Hour,
	}
}

// NewAuthManagerWithConfig creates a new authentication manager with custom settings.
// This allows specifying a custom token expiry duration for different security requirements.
// Returns a pointer to the newly created AuthManager.
func NewAuthManagerWithConfig(jwtSecret string, tokenExpiry time.Duration) *AuthManager {
	return &AuthManager{
		jwtSecret:   jwtSecret,
		tokenExpiry: tokenExpiry,
	}
}

// HashPassword creates a bcrypt hash of the provided password.
// It uses bcrypt's default cost factor for security while maintaining reasonable performance.
// The salt is automatically generated and included in the hash.
// Returns the hashed password or an error if hashing fails.
func (am *AuthManager) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword compares a plain text password with a bcrypt hash.
// It uses constant-time comparison to prevent timing attacks.
// Returns true if the password matches the hash, false otherwise.
func (am *AuthManager) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a new JWT token for the specified user.
// The token includes user identification claims and is signed with the manager's secret.
// The token will expire after the configured duration.
// Returns the signed JWT token string or an error if generation fails.
func (am *AuthManager) GenerateToken(userID uint, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "vpn-server",
			Subject:   fmt.Sprintf("user-%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(am.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT token string.
// It verifies the token signature, expiration, and other standard claims.
// Returns the parsed claims if the token is valid, or an error if validation fails.
func (am *AuthManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// RefreshToken generates a new token for a user based on a valid existing token.
// This allows extending user sessions without requiring re-authentication.
// The old token should be discarded after successful refresh.
// Returns a new JWT token string or an error if the original token is invalid.
func (am *AuthManager) RefreshToken(tokenString string) (string, error) {
	claims, err := am.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Generate new token with the same user information
	return am.GenerateToken(claims.UserID, claims.Username)
}

// Valid implements the jwt.Claims interface to validate custom claims.
// It checks if the token has expired and performs other claim validations.
// Returns an error if the claims are invalid or expired.
func (c *Claims) Valid() error {
	// Check if token has expired
	if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("token has expired")
	}
	return nil
}

// GenerateSecureSecret creates a cryptographically secure random secret for JWT signing.
// It generates 32 bytes (256 bits) of random data and encodes it as base64.
// This provides sufficient entropy for secure token signing.
// Returns a base64-encoded secret string or an error if random generation fails.
func GenerateSecureSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure secret: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}