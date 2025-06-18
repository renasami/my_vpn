// Package api provides REST API endpoints for VPN client and server management.
// It implements HTTP handlers for creating, managing, and monitoring VPN clients,
// as well as server configuration and control operations using the Gin web framework.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"my-vpn/internal/auth"
	"my-vpn/internal/database"
)

func setupAuthTest(t *testing.T) (*database.Database, *auth.AuthManager, *AuthAPI, *gin.Engine) {
	// Create temporary database
	db, err := database.New(":memory:")
	require.NoError(t, err)

	// Create auth manager
	authManager := auth.NewAuthManager("test-secret")

	// Create API
	api := NewAuthAPI(db, authManager)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create middleware
	middleware := auth.NewAuthMiddleware(authManager)
	
	// Register routes
	api.RegisterRoutes(router, middleware)

	return db, authManager, api, router
}

func TestAuthAPI_Register(t *testing.T) {
	db, _, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	t.Run("should register new user successfully", func(t *testing.T) {
		reqBody := RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "testpassword123",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response AuthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "testuser", response.User.Username)
		assert.Equal(t, "test@example.com", response.User.Email)
		assert.Equal(t, "user", response.User.Role)
		assert.True(t, response.User.Active)
		
		// Verify user is in database
		user, err := db.GetUserByUsername("testuser")
		require.NoError(t, err)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("should reject registration with existing username", func(t *testing.T) {
		// First registration
		reqBody := RegisterRequest{
			Username: "existinguser",
			Email:    "first@example.com",
			Password: "testpassword123",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Second registration with same username
		reqBody.Email = "second@example.com"
		body, _ = json.Marshal(reqBody)
		req, _ = http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Username already exists", response.Error)
	})

	t.Run("should reject registration with existing email", func(t *testing.T) {
		// First registration
		reqBody := RegisterRequest{
			Username: "user1",
			Email:    "duplicate@example.com",
			Password: "testpassword123",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Second registration with same email
		reqBody.Username = "user2"
		body, _ = json.Marshal(reqBody)
		req, _ = http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Email already exists", response.Error)
	})

	t.Run("should reject registration with invalid data", func(t *testing.T) {
		testCases := []struct {
			name string
			body RegisterRequest
		}{
			{
				name: "empty username",
				body: RegisterRequest{Username: "", Email: "test@example.com", Password: "testpassword123"},
			},
			{
				name: "short username",
				body: RegisterRequest{Username: "ab", Email: "test@example.com", Password: "testpassword123"},
			},
			{
				name: "invalid email",
				body: RegisterRequest{Username: "testuser", Email: "invalid-email", Password: "testpassword123"},
			},
			{
				name: "short password",
				body: RegisterRequest{Username: "testuser", Email: "test@example.com", Password: "short"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.body)
				req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})
}

func TestAuthAPI_Login(t *testing.T) {
	db, authManager, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	// Create test user
	hashedPassword, _ := authManager.HashPassword("testpassword123")
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}
	require.NoError(t, db.CreateUser(user))

	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "testuser",
			Password: "testpassword123",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AuthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "testuser", response.User.Username)
		assert.Equal(t, "test@example.com", response.User.Email)
		assert.Equal(t, "user", response.User.Role)
		assert.True(t, response.User.Active)
	})

	t.Run("should reject login with invalid username", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "nonexistent",
			Password: "testpassword123",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Invalid credentials", response.Error)
	})

	t.Run("should reject login with invalid password", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "testuser",
			Password: "wrongpassword",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Invalid credentials", response.Error)
	})

	t.Run("should reject login for inactive user", func(t *testing.T) {
		// Create inactive user
		hashedPassword, _ := authManager.HashPassword("testpassword123")
		inactiveUser := &database.User{
			Username: "inactive",
			Email:    "inactive@example.com",
			Password: hashedPassword,
			Role:     "user",
			Active:   true, // Create as active first
		}
		require.NoError(t, db.CreateUser(inactiveUser))
		
		// Then deactivate the user
		require.NoError(t, db.DeactivateUser(inactiveUser.ID))

		reqBody := LoginRequest{
			Username: "inactive",
			Password: "testpassword123",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Account is deactivated", response.Error)
	})
}

func TestAuthAPI_RefreshToken(t *testing.T) {
	db, authManager, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	// Create test user
	hashedPassword, _ := authManager.HashPassword("testpassword123")
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}
	require.NoError(t, db.CreateUser(user))

	t.Run("should refresh token successfully", func(t *testing.T) {
		// Generate initial token
		token, err := authManager.GenerateToken(user.ID, user.Username)
		require.NoError(t, err)

		reqBody := RefreshTokenRequest{
			Token: token,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AuthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "testuser", response.User.Username)
		assert.Equal(t, "test@example.com", response.User.Email)
	})

	t.Run("should reject invalid token", func(t *testing.T) {
		reqBody := RefreshTokenRequest{
			Token: "invalid.jwt.token",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Invalid or expired token", response.Error)
	})
}

func TestAuthAPI_GetProfile(t *testing.T) {
	db, authManager, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	// Create test user
	hashedPassword, _ := authManager.HashPassword("testpassword123")
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}
	require.NoError(t, db.CreateUser(user))

	t.Run("should get profile successfully", func(t *testing.T) {
		// Generate token
		token, err := authManager.GenerateToken(user.ID, user.Username)
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/api/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response UserInfo
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "testuser", response.Username)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Equal(t, "user", response.Role)
		assert.True(t, response.Active)
	})

	t.Run("should reject request without authorization", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/profile", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthAPI_UpdateProfile(t *testing.T) {
	db, authManager, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	// Create test user
	hashedPassword, _ := authManager.HashPassword("testpassword123")
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}
	require.NoError(t, db.CreateUser(user))

	t.Run("should update profile successfully", func(t *testing.T) {
		// Generate token
		token, err := authManager.GenerateToken(user.ID, user.Username)
		require.NoError(t, err)

		reqBody := UpdateProfileRequest{
			Email: "updated@example.com",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response UserInfo
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "updated@example.com", response.Email)
		
		// Verify in database
		updatedUser, err := db.GetUser(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", updatedUser.Email)
	})
}

func TestAuthAPI_ChangePassword(t *testing.T) {
	db, authManager, _, router := setupAuthTest(t)
	defer os.Remove(":memory:")

	// Create test user
	hashedPassword, _ := authManager.HashPassword("testpassword123")
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}
	require.NoError(t, db.CreateUser(user))

	t.Run("should change password successfully", func(t *testing.T) {
		// Generate token
		token, err := authManager.GenerateToken(user.ID, user.Username)
		require.NoError(t, err)

		reqBody := ChangePasswordRequest{
			CurrentPassword: "testpassword123",
			NewPassword:     "newpassword456",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify new password works
		updatedUser, err := db.GetUser(user.ID)
		require.NoError(t, err)
		assert.True(t, authManager.VerifyPassword("newpassword456", updatedUser.Password))
		assert.False(t, authManager.VerifyPassword("testpassword123", updatedUser.Password))
	})

	t.Run("should reject with wrong current password", func(t *testing.T) {
		// Generate token
		token, err := authManager.GenerateToken(user.ID, user.Username)
		require.NoError(t, err)

		reqBody := ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword456",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Current password is incorrect", response.Error)
	})
}