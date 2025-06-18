// Package main provides the entry point for the VPN Server application.
// This server manages WireGuard VPN connections, client management, and web interface
// for macOS systems using pfctl for firewall management.
package main

import (
	"log"
	"my-vpn/internal/server"
)

// main initializes and starts the VPN server.
// It creates a new server instance and starts it, handling any startup errors
// by logging them and terminating the application.
func main() {
	log.Println("Starting VPN Server...")
	
	srv := server.New()
	if err := srv.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}