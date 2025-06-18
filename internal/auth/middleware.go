// Package auth provides authentication and authorization functionality for the VPN server.
// It implements JWT-based authentication, user management, and session handling
// with support for password hashing and middleware integration.
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides HTTP middleware for JWT authentication.
// It validates JWT tokens in request headers and provides user context
// for authenticated routes in the VPN server application.
type AuthMiddleware struct {
	authManager *AuthManager // Authentication manager for token validation
}

// ErrorResponse represents an authentication error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewAuthMiddleware creates a new authentication middleware instance.
// It requires an AuthManager to handle token validation and user authentication.
// Returns a pointer to the newly created AuthMiddleware.
func NewAuthMiddleware(authManager *AuthManager) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: authManager,
	}
}

// RequireAuth is a middleware function that requires authentication for protected routes.
// It extracts the Authorization header, validates the JWT token, and sets user context.
// If authentication fails, it returns a 401 Unauthorized response.
// On success, it adds the user claims to the Gin context for use in handlers.
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authorization header must start with 'Bearer '",
			})
			c.Abort()
			return
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "JWT token is required",
			})
			c.Abort()
			return
		}

		// Validate the token
		claims, err := am.authManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("claims", claims)

		// Continue to the next middleware/handler
		c.Next()
	}
}

// OptionalAuth is a middleware function that provides optional authentication.
// It extracts and validates the JWT token if present, but doesn't require it.
// This is useful for routes that provide different functionality for authenticated users
// but remain accessible to anonymous users.
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No authorization header, continue without authentication
			c.Next()
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			// Empty token, continue without authentication
			c.Next()
			return
		}

		// Validate the token
		claims, err := am.authManager.ValidateToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			c.Next()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("claims", claims)

		// Continue to the next middleware/handler
		c.Next()
	}
}

// GetUserID extracts the user ID from the Gin context.
// This should be called after RequireAuth middleware has run.
// Returns the user ID and a boolean indicating if it was found.
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	
	id, ok := userID.(uint)
	return id, ok
}

// GetUsername extracts the username from the Gin context.
// This should be called after RequireAuth middleware has run.
// Returns the username and a boolean indicating if it was found.
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	
	name, ok := username.(string)
	return name, ok
}

// GetClaims extracts the JWT claims from the Gin context.
// This should be called after RequireAuth middleware has run.
// Returns the claims and a boolean indicating if they were found.
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	
	claimsObj, ok := claims.(*Claims)
	return claimsObj, ok
}

// IsAuthenticated checks if the current request is authenticated.
// Returns true if the user is authenticated, false otherwise.
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}