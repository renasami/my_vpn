// Package auth provides authentication and authorization functionality for the VPN server.
// It implements JWT-based authentication, user management, and session handling
// with support for password hashing and middleware integration.
package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthMiddleware(t *testing.T) {
	t.Run("should create auth middleware", func(t *testing.T) {
		authManager := NewAuthManager("test-secret")
		middleware := NewAuthMiddleware(authManager)
		
		assert.NotNil(t, middleware)
		assert.Equal(t, authManager, middleware.authManager)
	})
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	authManager := NewAuthManager("test-secret")
	middleware := NewAuthMiddleware(authManager)
	
	// Setup test router
	gin.SetMode(gin.TestMode)
	
	t.Run("should allow valid token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequireAuth())
		router.GET("/protected", func(c *gin.Context) {
			userID, exists := GetUserID(c)
			if !exists {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found"})
				return
			}
			
			username, exists := GetUsername(c)
			if !exists {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "username not found"})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"user_id":  userID,
				"username": username,
			})
		})
		
		// Generate valid token
		token, err := authManager.GenerateToken(123, "testuser")
		require.NoError(t, err)
		
		// Create request with valid token
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, float64(123), response["user_id"])
		assert.Equal(t, "testuser", response["username"])
	})
	
	t.Run("should reject request without authorization header", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequireAuth())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		// Create request without authorization header
		req, _ := http.NewRequest("GET", "/protected", nil)
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "Authorization header is required", response.Error)
	})
	
	t.Run("should reject request with invalid authorization header format", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequireAuth())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		// Create request with invalid authorization header
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "Authorization header must start with 'Bearer '", response.Error)
	})
	
	t.Run("should reject request with empty token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequireAuth())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		// Create request with empty token
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "JWT token is required", response.Error)
	})
	
	t.Run("should reject request with invalid token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequireAuth())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		// Create request with invalid token
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt.token")
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "Invalid or expired token", response.Error)
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	authManager := NewAuthManager("test-secret")
	middleware := NewAuthMiddleware(authManager)
	
	// Setup test router
	gin.SetMode(gin.TestMode)
	
	t.Run("should allow request with valid token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			if IsAuthenticated(c) {
				userID, _ := GetUserID(c)
				username, _ := GetUsername(c)
				c.JSON(http.StatusOK, gin.H{
					"authenticated": true,
					"user_id":       userID,
					"username":      username,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": false,
				})
			}
		})
		
		// Generate valid token
		token, err := authManager.GenerateToken(123, "testuser")
		require.NoError(t, err)
		
		// Create request with valid token
		req, _ := http.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, true, response["authenticated"])
		assert.Equal(t, float64(123), response["user_id"])
		assert.Equal(t, "testuser", response["username"])
	})
	
	t.Run("should allow request without authorization header", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			if IsAuthenticated(c) {
				userID, _ := GetUserID(c)
				username, _ := GetUsername(c)
				c.JSON(http.StatusOK, gin.H{
					"authenticated": true,
					"user_id":       userID,
					"username":      username,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": false,
				})
			}
		})
		
		// Create request without authorization header
		req, _ := http.NewRequest("GET", "/optional", nil)
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, false, response["authenticated"])
	})
	
	t.Run("should allow request with invalid token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			if IsAuthenticated(c) {
				userID, _ := GetUserID(c)
				username, _ := GetUsername(c)
				c.JSON(http.StatusOK, gin.H{
					"authenticated": true,
					"user_id":       userID,
					"username":      username,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": false,
				})
			}
		})
		
		// Create request with invalid token
		req, _ := http.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt.token")
		
		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, false, response["authenticated"])
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("should return user ID when present", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", uint(123))
		
		userID, exists := GetUserID(c)
		assert.True(t, exists)
		assert.Equal(t, uint(123), userID)
	})
	
	t.Run("should return false when not present", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		userID, exists := GetUserID(c)
		assert.False(t, exists)
		assert.Equal(t, uint(0), userID)
	})
}

func TestGetUsername(t *testing.T) {
	t.Run("should return username when present", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("username", "testuser")
		
		username, exists := GetUsername(c)
		assert.True(t, exists)
		assert.Equal(t, "testuser", username)
	})
	
	t.Run("should return false when not present", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		username, exists := GetUsername(c)
		assert.False(t, exists)
		assert.Equal(t, "", username)
	})
}

func TestIsAuthenticated(t *testing.T) {
	t.Run("should return true when authenticated", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", uint(123))
		
		assert.True(t, IsAuthenticated(c))
	})
	
	t.Run("should return false when not authenticated", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		assert.False(t, IsAuthenticated(c))
	})
}