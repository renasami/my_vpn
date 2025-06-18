package database

import (
	"fmt"
	"time"
	
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps a GORM database instance and provides high-level operations
// for VPN server data management. It encapsulates all database interactions
// for clients, server configuration, and connection logging.
type Database struct {
	*gorm.DB
}

// New creates a new Database instance and establishes a connection to SQLite.
// It automatically runs database migrations for all defined models.
// The dbPath parameter specifies the path to the SQLite database file.
// Returns a Database instance or an error if connection or migration fails.
func New(dbPath string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&User{}, &Client{}, &ServerConfig{}, &ConnectionLog{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Database{DB: db}, nil
}

// CreateClient inserts a new client record into the database.
// The client parameter must have all required fields populated.
// Returns an error if the creation fails due to validation or database constraints.
func (db *Database) CreateClient(client *Client) error {
	return db.Create(client).Error
}

// GetClient retrieves a client by their unique ID.
// Returns the client record and an error if the client is not found or query fails.
func (db *Database) GetClient(id uint) (*Client, error) {
	var client Client
	err := db.First(&client, id).Error
	return &client, err
}

// GetClientByPublicKey retrieves a client by their WireGuard public key.
// This is useful for looking up clients during WireGuard handshake validation.
// Returns the client record and an error if the client is not found or query fails.
func (db *Database) GetClientByPublicKey(publicKey string) (*Client, error) {
	var client Client
	err := db.Where("public_key = ?", publicKey).First(&client).Error
	return &client, err
}

// ListClients retrieves all client records from the database.
// Returns a slice of all clients and an error if the query fails.
func (db *Database) ListClients() ([]Client, error) {
	var clients []Client
	err := db.Find(&clients).Error
	return clients, err
}

// UpdateClient updates an existing client record in the database.
// The client parameter must have the ID field set to identify the record to update.
// Returns an error if the update fails.
func (db *Database) UpdateClient(client *Client) error {
	return db.Save(client).Error
}

// DeleteClient removes a client record from the database by ID.
// This operation is permanent and cannot be undone.
// Returns an error if the deletion fails or the client doesn't exist.
func (db *Database) DeleteClient(id uint) error {
	return db.Delete(&Client{}, id).Error
}

// CreateServerConfig inserts a new server configuration record.
// This is typically called once during server initialization.
// Returns an error if the creation fails due to validation or database constraints.
func (db *Database) CreateServerConfig(config *ServerConfig) error {
	return db.Create(config).Error
}

// GetServerConfig retrieves the server configuration record.
// There should typically be only one server configuration in the database.
// Returns the server configuration and an error if not found or query fails.
func (db *Database) GetServerConfig() (*ServerConfig, error) {
	var config ServerConfig
	err := db.First(&config).Error
	return &config, err
}

// UpdateServerConfig updates the existing server configuration record.
// The config parameter must have the ID field set to identify the record to update.
// Returns an error if the update fails.
func (db *Database) UpdateServerConfig(config *ServerConfig) error {
	return db.Save(config).Error
}

// LogConnection records a client connection event in the database.
// This is used for auditing and monitoring client connections and disconnections.
// The action parameter should be either "connect" or "disconnect".
// Returns an error if the logging fails.
func (db *Database) LogConnection(clientID uint, action, ipAddress string) error {
	log := &ConnectionLog{
		ClientID:  clientID,
		Action:    action,
		IPAddress: ipAddress,
	}
	return db.Create(log).Error
}

// GetConnectionLogs retrieves the most recent connection log entries.
// The logs are returned in descending order by timestamp (most recent first).
// The limit parameter controls the maximum number of records to return.
// Returns a slice of connection logs with preloaded client information and an error if query fails.
func (db *Database) GetConnectionLogs(limit int) ([]ConnectionLog, error) {
	var logs []ConnectionLog
	err := db.Preload("Client").Order("timestamp desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// CreateUser inserts a new user record into the database.
// The user parameter must have all required fields populated including hashed password.
// Returns an error if the creation fails due to validation or database constraints.
func (db *Database) CreateUser(user *User) error {
	return db.Create(user).Error
}

// GetUser retrieves a user by their unique ID.
// Returns the user record and an error if the user is not found or query fails.
func (db *Database) GetUser(id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}

// GetUserByUsername retrieves a user by their username.
// This is used for authentication during login.
// Returns the user record and an error if the user is not found or query fails.
func (db *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	return &user, err
}

// GetUserByEmail retrieves a user by their email address.
// This is useful for password reset and duplicate email validation.
// Returns the user record and an error if the user is not found or query fails.
func (db *Database) GetUserByEmail(email string) (*User, error) {
	var user User
	err := db.Where("email = ?", email).First(&user).Error
	return &user, err
}

// ListUsers retrieves all user records from the database.
// Returns a slice of all users and an error if the query fails.
func (db *Database) ListUsers() ([]User, error) {
	var users []User
	err := db.Find(&users).Error
	return users, err
}

// UpdateUser updates an existing user record in the database.
// The user parameter must have the ID field set to identify the record to update.
// Returns an error if the update fails.
func (db *Database) UpdateUser(user *User) error {
	return db.Save(user).Error
}

// UpdateUserLastLogin updates the last login timestamp for a user.
// This is called after successful authentication.
// Returns an error if the update fails.
func (db *Database) UpdateUserLastLogin(userID uint) error {
	now := time.Now()
	return db.Model(&User{}).Where("id = ?", userID).Update("last_login", &now).Error
}

// DeactivateUser sets a user's active status to false.
// This is a soft delete that preserves the user record but prevents login.
// Returns an error if the update fails.
func (db *Database) DeactivateUser(id uint) error {
	return db.Model(&User{}).Where("id = ?", id).Update("active", false).Error
}

// ActivateUser sets a user's active status to true.
// This re-enables a previously deactivated user account.
// Returns an error if the update fails.
func (db *Database) ActivateUser(id uint) error {
	return db.Model(&User{}).Where("id = ?", id).Update("active", true).Error
}

// DeleteUser removes a user record from the database by ID.
// This operation is permanent and cannot be undone.
// Returns an error if the deletion fails or the user doesn't exist.
func (db *Database) DeleteUser(id uint) error {
	return db.Delete(&User{}, id).Error
}

// AuthenticateUser validates user credentials and returns the user if successful.
// It checks the provided username and password against the database records.
// Returns the authenticated user and an error if authentication fails.
func (db *Database) AuthenticateUser(username, password string) (*User, error) {
	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Check if user is active
	if !user.Active {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Verify password using bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	db.UpdateUserLastLogin(user.ID)

	return user, nil
}

// CreateUserWithCredentials creates a new user with username, email, and password.
// It hashes the password before storing it in the database.
// Returns the created user and an error if creation fails.
func (db *Database) CreateUserWithCredentials(username, email, password string) (*User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		Role:      "user",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}