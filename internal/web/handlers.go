package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"my-vpn/internal/auth"
	"my-vpn/internal/monitoring"
)

// loginPage serves the login page.
func (s *Server) loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "VPN Server - Login",
	})
}

// handleLogin processes login form submission.
func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `form:"username" json:"username" binding:"required"`
		Password string `form:"password" json:"password" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"title": "VPN Server - Login",
			"error": "Please provide username and password",
		})
		return
	}

	// Authenticate user
	user, err := s.db.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "VPN Server - Login",
			"error": "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	token, err := s.authManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"title": "VPN Server - Login",
			"error": "Failed to generate authentication token",
		})
		return
	}

	// Set token as cookie and redirect to dashboard
	c.SetCookie("auth_token", token, 3600*24, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

// registerPage serves the registration page.
func (s *Server) registerPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title": "VPN Server - Register",
	})
}

// handleRegister processes registration form submission.
func (s *Server) handleRegister(c *gin.Context) {
	var req struct {
		Username string `form:"username" json:"username" binding:"required"`
		Email    string `form:"email" json:"email" binding:"required,email"`
		Password string `form:"password" json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title": "VPN Server - Register",
			"error": "Please provide valid registration information",
		})
		return
	}

	// Create user
	user, err := s.db.CreateUserWithCredentials(req.Username, req.Email, req.Password)
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title": "VPN Server - Register",
			"error": "Username or email already exists",
		})
		return
	}

	// Generate JWT token
	token, err := s.authManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"title": "VPN Server - Register",
			"error": "Registration successful, but failed to log in automatically",
		})
		return
	}

	// Set token as cookie and redirect to dashboard
	c.SetCookie("auth_token", token, 3600*24, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

// dashboard serves the main dashboard page.
func (s *Server) dashboard(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	userClaims := user.(*auth.Claims)

	// Get server metrics
	metrics := s.monitor.GetMetrics()
	
	// Get server status
	serverStatus := s.monitor.GetServerStatus()

	// Get recent clients
	clients, _ := s.db.ListClients()
	
	// Get active alerts
	alerts := s.monitor.GetMetrics().Alerts

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":        "VPN Server Dashboard",
		"user":         userClaims.Username,
		"serverStatus": serverStatus,
		"metrics":      metrics,
		"clients":      clients,
		"alerts":       alerts,
		"clientCount":  len(clients),
		"activeClients": metrics.ConnectionStats.ActiveClients,
	})
}

// clientsPage serves the clients management page.
func (s *Server) clientsPage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	userClaims := user.(*auth.Claims)

	// Get all clients
	clients, err := s.db.ListClients()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": "Failed to load clients",
		})
		return
	}

	// Get IP pool information
	ipInfo := s.ipPool.GetNetworkInfo()

	c.HTML(http.StatusOK, "clients.html", gin.H{
		"title":   "Client Management",
		"user":    userClaims.Username,
		"clients": clients,
		"ipInfo":  ipInfo,
	})
}

// monitoringPage serves the monitoring page.
func (s *Server) monitoringPage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	userClaims := user.(*auth.Claims)

	// Get metrics
	metrics := s.monitor.GetMetrics()

	c.HTML(http.StatusOK, "monitoring.html", gin.H{
		"title":   "Server Monitoring",
		"user":    userClaims.Username,
		"metrics": metrics,
	})
}

// settingsPage serves the settings page.
func (s *Server) settingsPage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	userClaims := user.(*auth.Claims)

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"title": "Settings",
		"user":  userClaims.Username,
	})
}

// API handlers for AJAX requests

// getMetrics returns current server metrics as JSON.
func (s *Server) getMetrics(c *gin.Context) {
	metrics := s.monitor.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// getAlerts returns current alerts as JSON.
func (s *Server) getAlerts(c *gin.Context) {
	metrics := s.monitor.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"alerts": metrics.Alerts,
	})
}

// getLogs returns recent logs as JSON.
func (s *Server) getLogs(c *gin.Context) {
	// Get query parameters
	countStr := c.DefaultQuery("count", "100")
	levelStr := c.DefaultQuery("level", "")

	_, err := strconv.Atoi(countStr)
	if err != nil {
		// Default to 100 if invalid
	}

	var logs []monitoring.LogEntry
	
	if levelStr != "" {
		// Parse log level
		switch levelStr {
		case "debug":
			_ = monitoring.LogLevelDebug
		case "info":
			_ = monitoring.LogLevelInfo
		case "warn":
			_ = monitoring.LogLevelWarn
		case "error":
			_ = monitoring.LogLevelError
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log level"})
			return
		}
		
		// Get logs by level (this would need to be implemented in LogManager)
		logs = []monitoring.LogEntry{} // Placeholder
	} else {
		// Get recent logs (this would need to be implemented in LogManager)
		logs = []monitoring.LogEntry{} // Placeholder
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}