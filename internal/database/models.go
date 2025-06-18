// Package database provides data models and database access layer for the VPN server.
// It defines the database schema using GORM for ORM functionality and includes
// models for clients, server configuration, and connection logging.
package database

import (
	"time"
)

// User represents an authenticated user in the VPN server system.
// It stores user credentials and authentication information for accessing
// the VPN management interface and API endpoints.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`                    // Unique identifier for the user
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`    // Unique username for login
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`       // User's email address (unique)
	Password  string    `gorm:"not null" json:"-"`                       // Hashed password (excluded from JSON)
	Role      string    `gorm:"default:user" json:"role"`                // User role: "admin" or "user"
	Active    bool      `gorm:"default:true" json:"active"`              // Whether the user account is active
	CreatedAt time.Time `json:"created_at"`                              // Account creation timestamp
	UpdatedAt time.Time `json:"updated_at"`                              // Last update timestamp
	LastLogin *time.Time `json:"last_login,omitempty"`                   // Last login timestamp
}

// Client represents a VPN client in the database.
// It stores all necessary information for a WireGuard client including
// cryptographic keys, network configuration, and connection statistics.
type Client struct {
	ID            uint       `gorm:"primaryKey" json:"id"`                       // Unique identifier for the client
	Name          string     `gorm:"not null" json:"name"`                       // Human-readable name for the client
	PublicKey     string     `gorm:"uniqueIndex;not null" json:"public_key"`     // WireGuard public key (unique)
	PrivateKey    string     `gorm:"not null" json:"private_key"`                // WireGuard private key
	IPAddress     string     `gorm:"uniqueIndex;not null" json:"ip_address"`     // Assigned IP address (unique)
	Enabled       bool       `gorm:"default:true" json:"enabled"`                // Whether the client is active
	CreatedAt     time.Time  `json:"created_at"`                                 // Creation timestamp
	UpdatedAt     time.Time  `json:"updated_at"`                                 // Last update timestamp
	LastHandshake *time.Time `json:"last_handshake,omitempty"`                   // Last WireGuard handshake time
	BytesReceived uint64     `gorm:"default:0" json:"bytes_received"`            // Total bytes received by client
	BytesSent     uint64     `gorm:"default:0" json:"bytes_sent"`                // Total bytes sent by client
}

// ServerConfig represents the WireGuard server configuration in the database.
// It stores the server's cryptographic keys, network settings, and interface configuration.
type ServerConfig struct {
	ID         uint      `gorm:"primaryKey" json:"id"`           // Unique identifier for the configuration
	PrivateKey string    `gorm:"not null" json:"private_key"`    // WireGuard server private key
	PublicKey  string    `gorm:"not null" json:"public_key"`     // WireGuard server public key
	ListenPort int       `gorm:"not null" json:"listen_port"`    // UDP port for WireGuard to listen on
	Network    string    `gorm:"not null" json:"network"`        // VPN network CIDR (e.g., "10.0.0.0/24")
	Interface  string    `gorm:"default:wg0" json:"interface"`   // WireGuard interface name
	DNS        string    `gorm:"type:text" json:"dns"`           // DNS servers for clients (comma-separated)
	CreatedAt  time.Time `json:"created_at"`                     // Creation timestamp
	UpdatedAt  time.Time `json:"updated_at"`                     // Last update timestamp
}

// ConnectionLog represents a client connection event in the database.
// It tracks when clients connect and disconnect for auditing and monitoring purposes.
type ConnectionLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`           // Unique identifier for the log entry
	ClientID  uint      `gorm:"not null" json:"client_id"`      // Foreign key reference to Client
	Client    Client    `gorm:"foreignKey:ClientID" json:"client"` // Associated client record
	Action    string    `gorm:"not null" json:"action"`         // Action type: "connect" or "disconnect"
	Timestamp time.Time `gorm:"autoCreateTime" json:"timestamp"` // When the action occurred
	IPAddress string    `json:"ip_address"`                     // Client's remote IP address
}

// TableName returns the database table name for User model.
// This implements the GORM Tabler interface to specify custom table names.
func (User) TableName() string {
	return "users"
}

// TableName returns the database table name for Client model.
// This implements the GORM Tabler interface to specify custom table names.
func (Client) TableName() string {
	return "clients"
}

// TableName returns the database table name for ServerConfig model.
// This implements the GORM Tabler interface to specify custom table names.
func (ServerConfig) TableName() string {
	return "server_configs"
}

// TableName returns the database table name for ConnectionLog model.
// This implements the GORM Tabler interface to specify custom table names.
func (ConnectionLog) TableName() string {
	return "connection_logs"
}