package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-vpn/internal/database"
	"my-vpn/internal/network"
	"my-vpn/internal/wireguard"
)

type ServerAPI struct {
	db       *database.Database
	ipPool   *network.IPPool
	wgServer *wireguard.WireGuardServer
}

// Request/Response structures
type ServerStatusResponse struct {
	State        string    `json:"state"`
	Interface    string    `json:"interface"`
	LastUpdated  time.Time `json:"last_updated"`
	PeerCount    int       `json:"peer_count"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type ServerControlResponse struct {
	Message string `json:"message"`
}

type ServerConfigResponse struct {
	Network          string    `json:"network"`
	ServerIP         string    `json:"server_ip"`
	Interface        string    `json:"interface"`
	ListenPort       int       `json:"listen_port"`
	DNS              []string  `json:"dns"`
	PublicKey        string    `json:"public_key"`
	PrivateKey       string    `json:"private_key,omitempty"`
	NetworkAddress   string    `json:"network_address"`
	BroadcastAddress string    `json:"broadcast_address"`
	TotalHosts       int       `json:"total_hosts"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpdateServerConfigRequest struct {
	ListenPort int      `json:"listen_port,omitempty"`
	DNS        []string `json:"dns,omitempty"`
}

type InitializeServerRequest struct {
	Network    string   `json:"network" binding:"required"`
	ListenPort int      `json:"listen_port" binding:"required,min=1,max=65535"`
	DNS        []string `json:"dns,omitempty"`
}

type ServerLogsResponse struct {
	Logs  []LogEntry `json:"logs"`
	Total int        `json:"total"`
}

type LogEntry struct {
	ID        uint      `json:"id"`
	ClientID  uint      `json:"client_id"`
	Client    string    `json:"client"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	IPAddress string    `json:"ip_address"`
}

// NewServerAPI creates a new server API instance
func NewServerAPI(db *database.Database, ipPool *network.IPPool, wgServer *wireguard.WireGuardServer) *ServerAPI {
	return &ServerAPI{
		db:       db,
		ipPool:   ipPool,
		wgServer: wgServer,
	}
}

// RegisterRoutes registers the server API routes
func (api *ServerAPI) RegisterRoutes(router *gin.Engine) {
	apiGroup := router.Group("/api")
	{
		server := apiGroup.Group("/server")
		{
			server.GET("/status", api.GetStatus)
			server.POST("/start", api.StartServer)
			server.POST("/stop", api.StopServer)
			server.POST("/restart", api.RestartServer)
			server.GET("/config", api.GetConfig)
			server.PUT("/config", api.UpdateConfig)
			server.POST("/initialize", api.InitializeServer)
			server.GET("/logs", api.GetLogs)
		}
	}
}

// GetStatus returns the current server status
func (api *ServerAPI) GetStatus(c *gin.Context) {
	status, err := api.wgServer.Status()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get server status"})
		return
	}

	response := ServerStatusResponse{
		State:        status.State,
		Interface:    status.Interface,
		LastUpdated:  status.LastUpdated,
		PeerCount:    status.PeerCount,
		ErrorMessage: status.ErrorMessage,
	}

	c.JSON(http.StatusOK, response)
}

// StartServer starts the WireGuard server
func (api *ServerAPI) StartServer(c *gin.Context) {
	// Check if server config exists
	serverConfig, err := api.getOrCreateServerConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get server configuration"})
		return
	}

	// Generate WireGuard config and write to file
	wgConfig := api.convertToWireGuardConfig(serverConfig)
	if err := api.wgServer.WriteConfig(wgConfig); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to write server configuration"})
		return
	}

	// Start the server
	if err := api.wgServer.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start server"})
		return
	}

	response := ServerControlResponse{
		Message: "Server started successfully",
	}

	c.JSON(http.StatusOK, response)
}

// StopServer stops the WireGuard server
func (api *ServerAPI) StopServer(c *gin.Context) {
	if err := api.wgServer.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to stop server"})
		return
	}

	response := ServerControlResponse{
		Message: "Server stopped successfully",
	}

	c.JSON(http.StatusOK, response)
}

// RestartServer restarts the WireGuard server
func (api *ServerAPI) RestartServer(c *gin.Context) {
	if err := api.wgServer.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to restart server"})
		return
	}

	response := ServerControlResponse{
		Message: "Server restarted successfully",
	}

	c.JSON(http.StatusOK, response)
}

// GetConfig returns the current server configuration
func (api *ServerAPI) GetConfig(c *gin.Context) {
	serverConfig, err := api.getOrCreateServerConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get server configuration"})
		return
	}

	networkInfo := api.ipPool.GetNetworkInfo()

	// Parse DNS
	var dns []string
	if serverConfig.DNS != "" {
		dns = strings.Split(serverConfig.DNS, ",")
		for i := range dns {
			dns[i] = strings.TrimSpace(dns[i])
		}
	}

	response := ServerConfigResponse{
		Network:          networkInfo.Network,
		ServerIP:         networkInfo.ServerIP,
		Interface:        serverConfig.Interface,
		ListenPort:       serverConfig.ListenPort,
		DNS:              dns,
		PublicKey:        serverConfig.PublicKey,
		PrivateKey:       serverConfig.PrivateKey,
		NetworkAddress:   networkInfo.NetworkAddress,
		BroadcastAddress: networkInfo.BroadcastAddress,
		TotalHosts:       networkInfo.TotalHosts,
		CreatedAt:        serverConfig.CreatedAt,
		UpdatedAt:        serverConfig.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfig updates the server configuration
func (api *ServerAPI) UpdateConfig(c *gin.Context) {
	var req UpdateServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Validate listen port
	if req.ListenPort != 0 && (req.ListenPort < 1 || req.ListenPort > 65535) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Listen port must be between 1 and 65535"})
		return
	}

	serverConfig, err := api.getOrCreateServerConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get server configuration"})
		return
	}

	// Update fields if provided
	if req.ListenPort != 0 {
		serverConfig.ListenPort = req.ListenPort
	}
	if req.DNS != nil {
		serverConfig.DNS = strings.Join(req.DNS, ",")
	}

	if err := api.db.UpdateServerConfig(serverConfig); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update server configuration"})
		return
	}

	// Return updated config
	api.GetConfig(c)
}

// InitializeServer initializes the server with a new configuration
func (api *ServerAPI) InitializeServer(c *gin.Context) {
	var req InitializeServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Validate network
	newIPPool, err := network.NewIPPool(req.Network)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network CIDR"})
		return
	}

	// Generate server keys
	keyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate server keys"})
		return
	}

	// Set default DNS if not provided
	dns := req.DNS
	if len(dns) == 0 {
		dns = []string{"8.8.8.8", "8.8.4.4"}
	}

	// Create server config
	serverConfig := &database.ServerConfig{
		PrivateKey: keyPair.PrivateKey,
		PublicKey:  keyPair.PublicKey,
		ListenPort: req.ListenPort,
		Network:    req.Network,
		Interface:  "wg0",
		DNS:        strings.Join(dns, ","),
	}

	if err := api.db.CreateServerConfig(serverConfig); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save server configuration"})
		return
	}

	// Update the IP pool
	api.ipPool = newIPPool

	// Return the new config
	api.GetConfig(c)
}

// GetLogs returns server connection logs
func (api *ServerAPI) GetLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	logs, err := api.db.GetConnectionLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get logs"})
		return
	}

	response := ServerLogsResponse{
		Logs:  make([]LogEntry, len(logs)),
		Total: len(logs),
	}

	for i, log := range logs {
		response.Logs[i] = LogEntry{
			ID:        log.ID,
			ClientID:  log.ClientID,
			Client:    log.Client.Name,
			Action:    log.Action,
			Timestamp: log.Timestamp,
			IPAddress: log.IPAddress,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to get or create server config
func (api *ServerAPI) getOrCreateServerConfig() (*database.ServerConfig, error) {
	serverConfig, err := api.db.GetServerConfig()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default server config
			keyPair, err := wireguard.GenerateKeyPair()
			if err != nil {
				return nil, err
			}

			networkInfo := api.ipPool.GetNetworkInfo()
			serverConfig = &database.ServerConfig{
				PrivateKey: keyPair.PrivateKey,
				PublicKey:  keyPair.PublicKey,
				ListenPort: 51820,
				Network:    networkInfo.Network,
				Interface:  "wg0",
				DNS:        "8.8.8.8,8.8.4.4",
			}

			if err := api.db.CreateServerConfig(serverConfig); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return serverConfig, nil
}

// Helper function to convert database config to WireGuard config
func (api *ServerAPI) convertToWireGuardConfig(dbConfig *database.ServerConfig) *wireguard.ServerConfig {
	networkInfo := api.ipPool.GetNetworkInfo()
	
	// Parse DNS
	var dns []string
	if dbConfig.DNS != "" {
		dns = strings.Split(dbConfig.DNS, ",")
		for i := range dns {
			dns[i] = strings.TrimSpace(dns[i])
		}
	}

	return &wireguard.ServerConfig{
		PrivateKey: dbConfig.PrivateKey,
		PublicKey:  dbConfig.PublicKey,
		Address:    fmt.Sprintf("%s/24", networkInfo.ServerIP),
		ListenPort: dbConfig.ListenPort,
		DNS:        dns,
		PostUp: []string{
			"iptables -A FORWARD -i " + dbConfig.Interface + " -j ACCEPT",
			"iptables -t nat -A POSTROUTING -o en0 -j MASQUERADE",
		},
		PostDown: []string{
			"iptables -D FORWARD -i " + dbConfig.Interface + " -j ACCEPT",
			"iptables -t nat -D POSTROUTING -o en0 -j MASQUERADE",
		},
		Interface: dbConfig.Interface,
	}
}