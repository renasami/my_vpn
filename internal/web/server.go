// Package web provides HTTP server and web UI functionality for the VPN server management interface.
// It implements REST API endpoints, serves static files, and provides a web-based dashboard
// for monitoring and managing the VPN server, clients, and system status.
package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"my-vpn/internal/api"
	"my-vpn/internal/auth"
	"my-vpn/internal/database"
	"my-vpn/internal/monitoring"
	"my-vpn/internal/network"
	"my-vpn/internal/system"
	"my-vpn/internal/wireguard"
)

// Server represents the HTTP server for the VPN management interface.
// It provides both REST API endpoints and serves the web UI dashboard.
type Server struct {
	router       *gin.Engine                // Gin HTTP router
	server       *http.Server               // HTTP server instance
	config       *ServerConfig              // Server configuration
	db           *database.Database         // Database connection
	wgServer     *wireguard.WireGuardServer // WireGuard server instance
	ipPool       *network.IPPool            // IP pool manager
	pfctlManager *system.PfctlManager       // Firewall manager
	monitor      *monitoring.Monitor        // Monitoring system
	authManager  *auth.AuthManager          // Authentication manager
}

// ServerConfig represents configuration options for the web server.
type ServerConfig struct {
	Host         string        `json:"host"`          // Server host address (default: "localhost")
	Port         int           `json:"port"`          // Server port (default: 8080)
	ReadTimeout  time.Duration `json:"read_timeout"`  // HTTP read timeout
	WriteTimeout time.Duration `json:"write_timeout"` // HTTP write timeout
	EnableTLS    bool          `json:"enable_tls"`    // Whether to enable HTTPS
	CertFile     string        `json:"cert_file"`     // TLS certificate file path
	KeyFile      string        `json:"key_file"`      // TLS private key file path
	StaticDir    string        `json:"static_dir"`    // Static files directory
	TemplateDir  string        `json:"template_dir"`  // Template files directory
	Debug        bool          `json:"debug"`         // Enable debug mode
}

// NewServer creates a new web server with default configuration.
// It initializes the HTTP server, sets up routes, and configures middleware
// for authentication, logging, and CORS. Returns a Server instance.
func NewServer(db *database.Database, wgServer *wireguard.WireGuardServer, ipPool *network.IPPool, pfctlManager *system.PfctlManager, monitor *monitoring.Monitor) *Server {
	config := &ServerConfig{
		Host:         "localhost",
		Port:         8080,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		EnableTLS:    false,
		StaticDir:    "web/static",
		TemplateDir:  "web/templates",
		Debug:        false,
	}

	return NewServerWithConfig(db, wgServer, ipPool, pfctlManager, monitor, config)
}

// NewServerWithConfig creates a new web server with custom configuration.
// This allows fine-tuning of server behavior for specific deployment requirements.
// Returns a Server instance with the specified configuration.
func NewServerWithConfig(db *database.Database, wgServer *wireguard.WireGuardServer, ipPool *network.IPPool, pfctlManager *system.PfctlManager, monitor *monitoring.Monitor, config *ServerConfig) *Server {
	// Set Gin mode based on debug setting
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create authentication manager with a default secret (should be from config in production)
	authManager := auth.NewAuthManager("default-secret-key-change-in-production")

	server := &Server{
		router:       gin.New(),
		config:       config,
		db:           db,
		wgServer:     wgServer,
		ipPool:       ipPool,
		pfctlManager: pfctlManager,
		monitor:      monitor,
		authManager:  authManager,
	}

	server.setupRoutes()
	server.setupHTTPServer()

	return server
}

// Start starts the HTTP server.
// It begins listening for HTTP requests on the configured host and port.
// This method is non-blocking and returns immediately after starting the server.
func (s *Server) Start() error {
	if s.config.EnableTLS {
		return s.server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
// It waits for existing connections to complete before stopping.
// This method blocks until the server has shut down completely.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// GetAddress returns the full server address including protocol, host, and port.
// This is useful for constructing URLs and displaying server information.
func (s *Server) GetAddress() string {
	protocol := "http"
	if s.config.EnableTLS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, s.config.Host, s.config.Port)
}

// setupRoutes configures all HTTP routes and middleware for the server.
// It sets up API endpoints, static file serving, and web UI routes.
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(s.corsMiddleware())

	// Load HTML templates
	s.router.LoadHTMLGlob(s.config.TemplateDir + "/*")

	// Serve static files
	s.router.Static("/static", s.config.StaticDir)

	// Authentication middleware
	authMiddleware := auth.NewAuthMiddleware(s.authManager)

	// Public routes (no authentication required)
	public := s.router.Group("/")
	{
		// Serve login page
		public.GET("/login", s.loginPage)
		public.POST("/login", s.handleLogin)
		public.GET("/register", s.registerPage)
		public.POST("/register", s.handleRegister)
	}

	// API routes
	apiV1 := s.router.Group("/api/v1")
	{
		// Public API endpoints
		authAPI := api.NewAuthAPI(s.db, s.authManager)
		apiV1.POST("/auth/login", authAPI.Login)
		apiV1.POST("/auth/register", authAPI.Register)

		// Protected API endpoints
		protected := apiV1.Group("/")
		protected.Use(authMiddleware.RequireAuth())
		{
			// Authentication endpoints
			protected.POST("/auth/refresh", authAPI.RefreshToken)
			protected.GET("/auth/profile", authAPI.GetProfile)
			protected.POST("/auth/change-password", authAPI.ChangePassword)

			// Server management endpoints
			serverAPI := api.NewServerAPI(s.db, s.ipPool, s.wgServer)
			protected.GET("/server/status", serverAPI.GetStatus)
			protected.POST("/server/start", serverAPI.StartServer)
			protected.POST("/server/stop", serverAPI.StopServer)
			protected.POST("/server/restart", serverAPI.RestartServer)

			// Client management endpoints
			clientAPI := api.NewClientAPI(s.db, s.ipPool, s.wgServer)
			protected.GET("/clients", clientAPI.GetClients)
			protected.POST("/clients", clientAPI.CreateClient)
			protected.GET("/clients/:id", clientAPI.GetClient)
			protected.PUT("/clients/:id", clientAPI.UpdateClient)
			protected.DELETE("/clients/:id", clientAPI.DeleteClient)
			protected.GET("/clients/:id/config", clientAPI.GetClientConfig)
			protected.GET("/clients/:id/qr", clientAPI.GetClientQRCode)

			// Monitoring endpoints
			protected.GET("/monitoring/metrics", s.getMetrics)
			protected.GET("/monitoring/alerts", s.getAlerts)
			protected.GET("/monitoring/logs", s.getLogs)
		}
	}

	// Protected web UI routes
	webUI := s.router.Group("/")
	webUI.Use(authMiddleware.RequireAuth())
	{
		webUI.GET("/", s.dashboard)
		webUI.GET("/dashboard", s.dashboard)
		webUI.GET("/clients", s.clientsPage)
		webUI.GET("/monitoring", s.monitoringPage)
		webUI.GET("/settings", s.settingsPage)
	}
}

// setupHTTPServer configures the HTTP server with timeouts and other settings.
func (s *Server) setupHTTPServer() {
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	s.server = &http.Server{
		Addr:         address,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}
}

// corsMiddleware sets up CORS headers for cross-origin requests.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}