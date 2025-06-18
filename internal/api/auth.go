// Package api provides REST API endpoints for VPN client and server management.
// It implements HTTP handlers for creating, managing, and monitoring VPN clients,
// as well as server configuration and control operations using the Gin web framework.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-vpn/internal/auth"
	"my-vpn/internal/database"
)

// AuthAPI provides REST API endpoints for user authentication and management.
// It handles user registration, login, token refresh, and user profile operations,
// integrating with the authentication manager and database components.
type AuthAPI struct {
	db          *database.Database // Database interface for user data persistence
	authManager *auth.AuthManager  // Authentication manager for token and password operations
}

// Request/Response structures for authentication
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type UpdateProfileRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}

// NewAuthAPI creates a new authentication API instance.
// It requires a database instance for user data persistence and an authentication manager
// for token and password operations.
// Returns a pointer to the newly created AuthAPI.
func NewAuthAPI(db *database.Database, authManager *auth.AuthManager) *AuthAPI {
	return &AuthAPI{
		db:          db,
		authManager: authManager,
	}
}

// RegisterRoutes registers the authentication API routes.
// It sets up all endpoints for user registration, login, token management, and profile operations.
func (api *AuthAPI) RegisterRoutes(router *gin.Engine, middleware *auth.AuthMiddleware) {
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/register", api.Register)
		authGroup.POST("/login", api.Login)
		authGroup.POST("/refresh", api.RefreshToken)
		
		// Protected routes requiring authentication
		protected := authGroup.Group("")
		protected.Use(middleware.RequireAuth())
		{
			protected.GET("/profile", api.GetProfile)
			protected.PUT("/profile", api.UpdateProfile)
			protected.POST("/change-password", api.ChangePassword)
			protected.POST("/logout", api.Logout)
		}
	}
}

// Register handles user registration requests.
// It validates the registration data, checks for existing users, hashes the password,
// and creates a new user account in the database.
func (api *AuthAPI) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Check if username already exists
	_, err := api.db.GetUserByUsername(req.Username)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "Username already exists"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check username"})
		return
	}

	// Check if email already exists
	_, err = api.db.GetUserByEmail(req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "Email already exists"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check email"})
		return
	}

	// Hash password
	hashedPassword, err := api.authManager.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to hash password"})
		return
	}

	// Create user
	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
		Active:   true,
	}

	if err := api.db.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	// Generate token
	token, err := api.authManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Update last login
	api.db.UpdateUserLastLogin(user.ID)

	response := AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Should match token expiry
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Active:    user.Active,
			CreatedAt: user.CreatedAt,
			LastLogin: user.LastLogin,
		},
	}

	c.JSON(http.StatusCreated, response)
}

// Login handles user login requests.
// It validates credentials, checks if the user is active, and generates a JWT token
// for authenticated access to protected endpoints.
func (api *AuthAPI) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get user by username
	user, err := api.db.GetUserByUsername(req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user"})
		return
	}

	// Check if user is active
	if !user.Active {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Account is deactivated"})
		return
	}

	// Verify password
	if !api.authManager.VerifyPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Generate token
	token, err := api.authManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Update last login
	api.db.UpdateUserLastLogin(user.ID)

	response := AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Should match token expiry
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Active:    user.Active,
			CreatedAt: user.CreatedAt,
			LastLogin: user.LastLogin,
		},
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken handles token refresh requests.
// It validates the existing token and generates a new one with extended expiry time.
func (api *AuthAPI) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Refresh token
	newToken, err := api.authManager.RefreshToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid or expired token"})
		return
	}

	// Validate new token to get user info
	claims, err := api.authManager.ValidateToken(newToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to validate new token"})
		return
	}

	// Get user details
	user, err := api.db.GetUser(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user"})
		return
	}

	response := AuthResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Should match token expiry
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Active:    user.Active,
			CreatedAt: user.CreatedAt,
			LastLogin: user.LastLogin,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile returns the current user's profile information.
// This endpoint requires authentication and returns the user's details.
func (api *AuthAPI) GetProfile(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	user, err := api.db.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user profile"})
		return
	}

	userInfo := UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		LastLogin: user.LastLogin,
	}

	c.JSON(http.StatusOK, userInfo)
}

// UpdateProfile handles user profile update requests.
// It allows users to update their email address and other profile information.
func (api *AuthAPI) UpdateProfile(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := api.db.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user"})
		return
	}

	// Update email if provided
	if req.Email != "" && req.Email != user.Email {
		// Check if email already exists
		_, err := api.db.GetUserByEmail(req.Email)
		if err == nil {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "Email already exists"})
			return
		}
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check email"})
			return
		}
		user.Email = req.Email
	}

	if err := api.db.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update profile"})
		return
	}

	userInfo := UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		LastLogin: user.LastLogin,
	}

	c.JSON(http.StatusOK, userInfo)
}

// ChangePassword handles password change requests.
// It validates the current password and updates it with a new hashed password.
func (api *AuthAPI) ChangePassword(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := api.db.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user"})
		return
	}

	// Verify current password
	if !api.authManager.VerifyPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := api.authManager.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to hash new password"})
		return
	}

	// Update password
	user.Password = hashedPassword
	if err := api.db.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// Logout handles user logout requests.
// Currently, this is a placeholder as JWT tokens are stateless.
// In a production system, you might want to implement token blacklisting.
func (api *AuthAPI) Logout(c *gin.Context) {
	// For JWT tokens, logout is typically handled client-side by discarding the token
	// In a production system, you might want to implement token blacklisting
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}