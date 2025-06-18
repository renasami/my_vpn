// Package server provides HTTP server functionality for the VPN management interface.
// It handles basic web requests and provides health check endpoints for monitoring.
package server

import (
	"fmt"
	"net/http"
)

// Server represents an HTTP server instance for the VPN management interface.
// It encapsulates the server configuration and provides methods for starting
// and handling HTTP requests.
type Server struct {
	port string // The port on which the server listens (e.g., ":8080")
}

// New creates a new Server instance with default configuration.
// The server is configured to listen on port 8080 by default.
// Returns a pointer to the newly created Server.
func New() *Server {
	return &Server{
		port: ":8080",
	}
}

// Start initializes and starts the HTTP server.
// It registers route handlers for the root path and health check endpoint,
// then begins listening for incoming connections on the configured port.
// Returns an error if the server fails to start or bind to the port.
func (s *Server) Start() error {
	http.HandleFunc("/", s.indexHandler)
	http.HandleFunc("/health", s.healthHandler)
	
	fmt.Printf("Server starting on port %s\n", s.port)
	return http.ListenAndServe(s.port, nil)
}

// indexHandler handles requests to the root path ("/").
// It returns an HTML page showing the VPN server management interface
// with basic status information.
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>VPN Server Management</h1><p>Status: Running</p>")
}

// healthHandler handles health check requests ("/health").
// It returns a JSON response indicating the server status,
// which can be used for monitoring and load balancer health checks.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "ok"}`)
}