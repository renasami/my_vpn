package wireguard

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WireGuardServer manages a WireGuard VPN server instance.
// It provides methods for starting, stopping, and configuring the WireGuard server,
// as well as managing peer connections and server status monitoring.
type WireGuardServer struct {
	configDir     string // Directory where WireGuard configuration files are stored
	interfaceName string // Name of the WireGuard network interface (e.g., "wg0")
}

// ServerStatus represents the current operational status of the WireGuard server.
// It provides information about the server state, connected peers, and any error conditions.
type ServerStatus struct {
	State        string    `json:"state"`                    // Current state: "running", "stopped", or "error"
	Interface    string    `json:"interface"`                // WireGuard interface name
	LastUpdated  time.Time `json:"last_updated"`             // Timestamp of the last status check
	PeerCount    int       `json:"peer_count"`               // Number of connected peers
	ErrorMessage string    `json:"error_message,omitempty"` // Error description if state is "error"
}

// Peer represents a WireGuard peer configuration for server management.
// It contains the essential information needed to add or manage a peer connection.
type Peer struct {
	PublicKey     string   `json:"public_key"`                      // Base64-encoded peer public key
	AllowedIPs    []string `json:"allowed_ips"`                     // IP addresses/ranges allowed for this peer
	Endpoint      string   `json:"endpoint,omitempty"`              // Peer's endpoint address (optional)
	PersistentKA  int      `json:"persistent_keepalive,omitempty"`  // Keepalive interval in seconds (optional)
}

// NewWireGuardServer creates a new WireGuard server with default configuration.
// The server is configured to use the standard WireGuard configuration directory
// (/usr/local/etc/wireguard) and the default interface name (wg0).
// Returns a pointer to the newly created WireGuardServer instance.
func NewWireGuardServer() *WireGuardServer {
	return &WireGuardServer{
		configDir:     "/usr/local/etc/wireguard",
		interfaceName: "wg0",
	}
}

// NewWireGuardServerWithConfig creates a new WireGuard server with custom configuration.
// This allows specifying a custom configuration directory and interface name,
// which is useful for testing or non-standard deployments.
// Returns a pointer to the newly created WireGuardServer instance.
func NewWireGuardServerWithConfig(configDir, interfaceName string) *WireGuardServer {
	return &WireGuardServer{
		configDir:     configDir,
		interfaceName: interfaceName,
	}
}

// WriteConfig writes the server configuration to a WireGuard configuration file.
// It creates the configuration directory if it doesn't exist and writes the
// configuration with appropriate file permissions (0600) for security.
// Returns an error if directory creation or file writing fails.
func (wg *WireGuardServer) WriteConfig(config *ServerConfig) error {
	// Ensure config directory exists
	if err := os.MkdirAll(wg.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(wg.configDir, wg.interfaceName+".conf")
	
	// Generate config content
	configContent := config.GenerateConfigFile()
	
	// Write config file with appropriate permissions
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Start starts the WireGuard server
func (wg *WireGuardServer) Start() error {
	configPath := filepath.Join(wg.configDir, wg.interfaceName+".conf")
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Use wg-quick to start the interface
	cmd := exec.Command("wg-quick", "up", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start WireGuard interface: %w, output: %s", err, string(output))
	}

	return nil
}

// Stop stops the WireGuard server
func (wg *WireGuardServer) Stop() error {
	configPath := filepath.Join(wg.configDir, wg.interfaceName+".conf")
	
	// Use wg-quick to stop the interface
	cmd := exec.Command("wg-quick", "down", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because interface is not running
		if strings.Contains(string(output), "is not a WireGuard interface") ||
		   strings.Contains(string(output), "No such device") {
			// Interface is not running, which is fine
			return nil
		}
		return fmt.Errorf("failed to stop WireGuard interface: %w, output: %s", err, string(output))
	}

	return nil
}

// Status returns the current status of the WireGuard server
func (wg *WireGuardServer) Status() (*ServerStatus, error) {
	status := &ServerStatus{
		Interface:   wg.interfaceName,
		LastUpdated: time.Now(),
		State:       "stopped",
	}

	// Check if interface exists
	cmd := exec.Command("wg", "show", wg.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such device") {
			status.State = "stopped"
			return status, nil
		}
		status.State = "error"
		status.ErrorMessage = fmt.Sprintf("failed to get interface status: %v", err)
		return status, nil
	}

	// Interface exists and is running
	status.State = "running"
	
	// Count peers
	lines := strings.Split(string(output), "\n")
	peerCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "peer:") {
			peerCount++
		}
	}
	status.PeerCount = peerCount

	return status, nil
}

// Restart restarts the WireGuard server
func (wg *WireGuardServer) Restart() error {
	// Stop first (ignore error if not running)
	_ = wg.Stop()
	
	// Wait a moment before starting
	time.Sleep(100 * time.Millisecond)
	
	// Start
	return wg.Start()
}

// AddPeer adds a peer to the WireGuard configuration
func (wg *WireGuardServer) AddPeer(peer *Peer) error {
	configPath := filepath.Join(wg.configDir, wg.interfaceName+".conf")
	
	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Generate peer configuration
	peerConfig := fmt.Sprintf("\n[Peer]\nPublicKey = %s\nAllowedIPs = %s\n",
		peer.PublicKey,
		strings.Join(peer.AllowedIPs, ", "))
	
	if peer.Endpoint != "" {
		peerConfig += fmt.Sprintf("Endpoint = %s\n", peer.Endpoint)
	}
	
	if peer.PersistentKA > 0 {
		peerConfig += fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKA)
	}

	// Append peer configuration
	newContent := string(content) + peerConfig
	
	// Write updated config
	if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	return nil
}

// RemovePeer removes a peer from the WireGuard configuration
func (wg *WireGuardServer) RemovePeer(publicKey string) error {
	configPath := filepath.Join(wg.configDir, wg.interfaceName+".conf")
	
	// Read existing config
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()

	var newLines []string
	scanner := bufio.NewScanner(file)
	
	skipPeerSection := false
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check if this is the start of a peer section
		if strings.TrimSpace(line) == "[Peer]" {
			skipPeerSection = false
			// Look ahead to see if this is the peer we want to remove
			tempLines := []string{line}
			
			// Read the peer section
			for scanner.Scan() {
				nextLine := scanner.Text()
				tempLines = append(tempLines, nextLine)
				
				if strings.HasPrefix(strings.TrimSpace(nextLine), "PublicKey = ") {
					if strings.Contains(nextLine, publicKey) {
						skipPeerSection = true
						break
					}
				}
				
				// If we hit another section or empty line, break
				if strings.HasPrefix(strings.TrimSpace(nextLine), "[") ||
				   strings.TrimSpace(nextLine) == "" {
					scanner = bufio.NewScanner(strings.NewReader(nextLine + "\n"))
					break
				}
			}
			
			// If this is not the peer to remove, add the lines
			if !skipPeerSection {
				newLines = append(newLines, tempLines...)
			}
			continue
		}
		
		// If we're not skipping this peer section, add the line
		if !skipPeerSection {
			newLines = append(newLines, line)
		}
		
		// Check if we've reached the end of the peer section we're skipping
		if skipPeerSection && (strings.HasPrefix(strings.TrimSpace(line), "[") || strings.TrimSpace(line) == "") {
			skipPeerSection = false
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Write the updated config
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (wg *WireGuardServer) GetConfigPath() string {
	return filepath.Join(wg.configDir, wg.interfaceName+".conf")
}

// IsRunning checks if the WireGuard interface is currently running
func (wg *WireGuardServer) IsRunning() bool {
	status, err := wg.Status()
	if err != nil {
		return false
	}
	return status.State == "running"
}

// GetConfig returns the current WireGuard server configuration.
// This parses the configuration file and returns the server settings including
// private key, listen port, and network configuration. This method is useful
// for monitoring and displaying current server configuration.
// Returns ServerConfig struct or an error if configuration cannot be read.
func (wg *WireGuardServer) GetConfig() (*ServerConfig, error) {
	configPath := wg.GetConfigPath()
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not exist: %s", configPath)
	}
	
	// Read configuration file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	// Parse configuration
	config := &ServerConfig{}
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			switch key {
			case "PrivateKey":
				config.PrivateKey = value
			case "ListenPort":
				if port, err := strconv.Atoi(value); err == nil {
					config.ListenPort = port
				}
			case "Address":
				config.Address = value
			}
		}
	}
	
	// Generate public key from private key if available
	if config.PrivateKey != "" {
		if pubKey, err := wg.generatePublicKey(config.PrivateKey); err == nil {
			config.PublicKey = pubKey
		}
	}
	
	return config, nil
}

// GetPeers returns a list of currently configured peers.
// This parses the WireGuard configuration and returns peer information
// including public keys, allowed IPs, and endpoint information.
// Returns a slice of Peer structs or an error if peers cannot be retrieved.
func (wg *WireGuardServer) GetPeers() ([]Peer, error) {
	configPath := wg.GetConfigPath()
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []Peer{}, nil // Return empty slice if no config exists
	}
	
	// Read configuration file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	// Parse peers from configuration
	var peers []Peer
	lines := strings.Split(string(content), "\n")
	var currentPeer *Peer
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check for [Peer] section
		if line == "[Peer]" {
			// Save previous peer if exists
			if currentPeer != nil && currentPeer.PublicKey != "" {
				peers = append(peers, *currentPeer)
			}
			// Start new peer
			currentPeer = &Peer{}
			continue
		}
		
		// Parse peer properties
		if currentPeer != nil && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			switch key {
			case "PublicKey":
				currentPeer.PublicKey = value
			case "AllowedIPs":
				// Parse comma-separated allowed IPs
				allowedIPs := strings.Split(value, ",")
				for i, ip := range allowedIPs {
					allowedIPs[i] = strings.TrimSpace(ip)
				}
				currentPeer.AllowedIPs = allowedIPs
			case "Endpoint":
				currentPeer.Endpoint = value
			case "PersistentKeepalive":
				if keepalive, err := strconv.Atoi(value); err == nil {
					currentPeer.PersistentKA = keepalive
				}
			}
		}
	}
	
	// Don't forget to add the last peer
	if currentPeer != nil && currentPeer.PublicKey != "" {
		peers = append(peers, *currentPeer)
	}
	
	return peers, nil
}

// generatePublicKey generates a public key from a private key using wg command.
// This is a helper method for deriving public keys when only private keys are available.
func (wg *WireGuardServer) generatePublicKey(privateKey string) (string, error) {
	cmd := exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(privateKey)
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}
	
	return strings.TrimSpace(string(output)), nil
}