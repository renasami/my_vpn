// Package api provides REST API endpoints for VPN client and server management.
// It implements HTTP handlers for creating, managing, and monitoring VPN clients,
// as well as server configuration and control operations using the Gin web framework.
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-vpn/internal/database"
	"my-vpn/internal/network"
	"my-vpn/internal/utils"
	"my-vpn/internal/wireguard"
)

// ClientAPI provides REST API endpoints for VPN client management.
// It handles client creation, configuration, and lifecycle management operations,
// integrating with the database, IP pool, and WireGuard server components.
type ClientAPI struct {
	db       *database.Database         // Database interface for client data persistence
	ipPool   *network.IPPool            // IP address pool for client IP allocation
	wgServer *wireguard.WireGuardServer // WireGuard server instance for peer management
}

// Request/Response structures
type CreateClientRequest struct {
	Name string `json:"name" binding:"required,min=1"`
}

type CreateClientResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	IPAddress string `json:"ip_address"`
	Enabled   bool   `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateClientRequest struct {
	Name    string `json:"name,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type ClientResponse struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	PublicKey     string     `json:"public_key"`
	IPAddress     string     `json:"ip_address"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastHandshake *time.Time `json:"last_handshake,omitempty"`
	BytesReceived uint64     `json:"bytes_received"`
	BytesSent     uint64     `json:"bytes_sent"`
}

type GetClientsResponse struct {
	Clients []ClientResponse `json:"clients"`
	Total   int              `json:"total"`
}

type ClientConfigResponse struct {
	Config string `json:"config"`
}

type ClientQRCodeResponse struct {
	QRCode string `json:"qr_code"`
	Format string `json:"format"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// NewClientAPI creates a new client API instance
func NewClientAPI(db *database.Database, ipPool *network.IPPool, wgServer *wireguard.WireGuardServer) *ClientAPI {
	return &ClientAPI{
		db:       db,
		ipPool:   ipPool,
		wgServer: wgServer,
	}
}

// RegisterRoutes registers the client API routes
func (api *ClientAPI) RegisterRoutes(router *gin.Engine) {
	apiGroup := router.Group("/api")
	{
		clients := apiGroup.Group("/clients")
		{
			clients.POST("", api.CreateClient)
			clients.GET("", api.GetClients)
			clients.GET("/:id", api.GetClient)
			clients.PUT("/:id", api.UpdateClient)
			clients.DELETE("/:id", api.DeleteClient)
			clients.GET("/:id/config", api.GetClientConfig)
			clients.GET("/:id/qrcode", api.GetClientQRCode)
		}
	}
}

// CreateClient creates a new client
func (api *ClientAPI) CreateClient(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Generate key pair for client
	keyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate client keys"})
		return
	}

	// Allocate IP address
	clientIP, err := api.ipPool.AllocateIP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to allocate IP address"})
		return
	}

	// Create client in database
	client := &database.Client{
		Name:       req.Name,
		PublicKey:  keyPair.PublicKey,
		PrivateKey: keyPair.PrivateKey,
		IPAddress:  clientIP,
		Enabled:    true,
	}

	if err := api.db.CreateClient(client); err != nil {
		// Release the allocated IP if database creation fails
		api.ipPool.ReleaseIP(clientIP)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create client"})
		return
	}

	// Add peer to WireGuard configuration
	peer := &wireguard.Peer{
		PublicKey:  keyPair.PublicKey,
		AllowedIPs: []string{clientIP + "/32"},
	}

	if err := api.wgServer.AddPeer(peer); err != nil {
		// Note: We continue even if adding peer fails as it might be due to WireGuard not being available
		// The peer will be added when the server is started
	}

	response := CreateClientResponse{
		ID:        client.ID,
		Name:      client.Name,
		PublicKey: client.PublicKey,
		IPAddress: client.IPAddress,
		Enabled:   client.Enabled,
		CreatedAt: client.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// GetClients returns all clients
func (api *ClientAPI) GetClients(c *gin.Context) {
	clients, err := api.db.ListClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get clients"})
		return
	}

	response := GetClientsResponse{
		Clients: make([]ClientResponse, len(clients)),
		Total:   len(clients),
	}

	for i, client := range clients {
		response.Clients[i] = ClientResponse{
			ID:            client.ID,
			Name:          client.Name,
			PublicKey:     client.PublicKey,
			IPAddress:     client.IPAddress,
			Enabled:       client.Enabled,
			CreatedAt:     client.CreatedAt,
			UpdatedAt:     client.UpdatedAt,
			LastHandshake: client.LastHandshake,
			BytesReceived: client.BytesReceived,
			BytesSent:     client.BytesSent,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetClient returns a specific client by ID
func (api *ClientAPI) GetClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid client ID"})
		return
	}

	client, err := api.db.GetClient(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get client"})
		return
	}

	response := ClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		PublicKey:     client.PublicKey,
		IPAddress:     client.IPAddress,
		Enabled:       client.Enabled,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
		LastHandshake: client.LastHandshake,
		BytesReceived: client.BytesReceived,
		BytesSent:     client.BytesSent,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateClient updates an existing client
func (api *ClientAPI) UpdateClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid client ID"})
		return
	}

	var req UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	client, err := api.db.GetClient(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get client"})
		return
	}

	// Update fields if provided
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.Enabled != nil {
		client.Enabled = *req.Enabled
	}

	if err := api.db.UpdateClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update client"})
		return
	}

	response := ClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		PublicKey:     client.PublicKey,
		IPAddress:     client.IPAddress,
		Enabled:       client.Enabled,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
		LastHandshake: client.LastHandshake,
		BytesReceived: client.BytesReceived,
		BytesSent:     client.BytesSent,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteClient deletes a client
func (api *ClientAPI) DeleteClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid client ID"})
		return
	}

	client, err := api.db.GetClient(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get client"})
		return
	}

	// Remove peer from WireGuard configuration
	if err := api.wgServer.RemovePeer(client.PublicKey); err != nil {
		// Note: We continue even if removing peer fails as it might be due to WireGuard not being available
	}

	// Release IP address
	if err := api.ipPool.ReleaseIP(client.IPAddress); err != nil {
		// Log error but continue with deletion
	}

	// Delete client from database
	if err := api.db.DeleteClient(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete client"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetClientConfig returns the WireGuard configuration for a client
func (api *ClientAPI) GetClientConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid client ID"})
		return
	}

	client, err := api.db.GetClient(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get client"})
		return
	}

	// Get server configuration to generate client config
	serverIP := api.ipPool.GetServerIP()
	serverConfig := &wireguard.ServerConfig{
		PublicKey: "dummy-server-public-key", // This should come from actual server config
		Address:   serverIP + "/24",
		ListenPort: 51820,
	}

	// Create client configuration
	clientConfig := &wireguard.ClientConfig{
		PrivateKey:      client.PrivateKey,
		PublicKey:       client.PublicKey,
		Address:         client.IPAddress + "/32",
		DNS:             []string{"8.8.8.8", "8.8.4.4"},
		ServerPublicKey: serverConfig.PublicKey,
		ServerEndpoint:  fmt.Sprintf("your-server-ip:%d", serverConfig.ListenPort),
		AllowedIPs:      []string{"0.0.0.0/0"},
	}

	configString := clientConfig.GenerateConfigFile()

	response := ClientConfigResponse{
		Config: configString,
	}

	c.JSON(http.StatusOK, response)
}

// GetClientQRCode returns a QR code for the WireGuard configuration of a client
func (api *ClientAPI) GetClientQRCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid client ID"})
		return
	}

	// Get query parameters for QR code options
	format := c.DefaultQuery("format", "base64") // base64, png, terminal
	sizeStr := c.DefaultQuery("size", "256")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		size = 256
	}

	// Validate format early
	if format != "base64" && format != "png" && format != "terminal" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Unsupported format. Use 'png', 'base64', or 'terminal'",
		})
		return
	}

	client, err := api.db.GetClient(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get client"})
		return
	}

	// Get server configuration to generate client config
	serverIP := api.ipPool.GetServerIP()
	serverConfig := &wireguard.ServerConfig{
		PublicKey: "dummy-server-public-key", // This should come from actual server config
		Address:   serverIP + "/24",
		ListenPort: 51820,
	}

	// Create client configuration
	clientConfig := &wireguard.ClientConfig{
		PrivateKey:      client.PrivateKey,
		PublicKey:       client.PublicKey,
		Address:         client.IPAddress + "/32",
		DNS:             []string{"8.8.8.8", "8.8.4.4"},
		ServerPublicKey: serverConfig.PublicKey,
		ServerEndpoint:  fmt.Sprintf("your-server-ip:%d", serverConfig.ListenPort),
		AllowedIPs:      []string{"0.0.0.0/0"},
	}

	configString := clientConfig.GenerateConfigFile()

	// Generate QR code options
	qrOptions := utils.QRCodeOptions{
		Size:          size,
		RecoveryLevel: utils.GetDefaultQRCodeOptions().RecoveryLevel,
		Format:        format,
	}

	// Generate QR code
	qrCodeData, err := utils.GenerateWireGuardConfigQR(configString, qrOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to generate QR code: %v", err),
		})
		return
	}

	// Handle different response formats
	switch format {
	case "png":
		// Return PNG data directly
		pngData := qrCodeData.([]byte)
		c.Header("Content-Type", "image/png")
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=client-%d-config.png", id))
		c.Data(http.StatusOK, "image/png", pngData)
	case "base64", "terminal":
		// Return JSON response
		qrString := qrCodeData.(string)
		response := ClientQRCodeResponse{
			QRCode: qrString,
			Format: format,
		}
		c.JSON(http.StatusOK, response)
	}
}