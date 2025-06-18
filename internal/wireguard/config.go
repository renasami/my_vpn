package wireguard

import (
	"fmt"
	"net"
	"strings"
)

// ServerConfig represents the WireGuard server configuration parameters.
// It contains all necessary settings to generate a complete WireGuard server
// configuration file, including cryptographic keys, network settings, and
// system integration commands for routing and firewall management.
type ServerConfig struct {
	PrivateKey string   // Base64-encoded server private key
	PublicKey  string   // Base64-encoded server public key
	Address    string   // Server IP address with CIDR notation (e.g., "10.0.0.1/24")
	ListenPort int      // UDP port for WireGuard to listen on
	DNS        []string // DNS servers to provide to clients
	PostUp     []string // Commands to execute when the interface comes up
	PostDown   []string // Commands to execute when the interface goes down
	Interface  string   // Name of the WireGuard interface (e.g., "wg0")
}

// ClientConfig represents the WireGuard client configuration parameters.
// It contains all settings needed to generate a complete WireGuard client
// configuration file that can connect to the VPN server.
type ClientConfig struct {
	PrivateKey      string   // Base64-encoded client private key
	PublicKey       string   // Base64-encoded client public key
	Address         string   // Client IP address with CIDR notation (e.g., "10.0.0.2/32")
	DNS             []string // DNS servers for the client to use
	ServerPublicKey string   // Base64-encoded server public key for authentication
	ServerEndpoint  string   // Server endpoint in "host:port" format
	AllowedIPs      []string // IP ranges that should be routed through the VPN
}

// NewServerConfig creates a new server configuration with generated cryptographic keys.
// It automatically generates a secure key pair and configures the server to use
// the first usable IP address in the specified network. The configuration includes
// default DNS servers and system integration commands for macOS.
// Returns a ServerConfig pointer or an error if key generation or network parsing fails.
func NewServerConfig(listenPort int, networkCIDR string) (*ServerConfig, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate server keys: %w", err)
	}

	_, ipNet, err := net.ParseCIDR(networkCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}

	serverIP := incrementIP(ipNet.IP, 1)

	return &ServerConfig{
		PrivateKey: keyPair.PrivateKey,
		PublicKey:  keyPair.PublicKey,
		Address:    fmt.Sprintf("%s/%d", serverIP.String(), getCIDRBits(ipNet)),
		ListenPort: listenPort,
		DNS:        []string{"8.8.8.8", "8.8.4.4"},
		PostUp: []string{
			fmt.Sprintf("iptables -A FORWARD -i wg0 -j ACCEPT"),
			fmt.Sprintf("iptables -t nat -A POSTROUTING -o en0 -j MASQUERADE"),
		},
		PostDown: []string{
			fmt.Sprintf("iptables -D FORWARD -i wg0 -j ACCEPT"),
			fmt.Sprintf("iptables -t nat -D POSTROUTING -o en0 -j MASQUERADE"),
		},
		Interface: "wg0",
	}, nil
}

// GenerateConfigFile creates a WireGuard configuration file content for the server.
// It generates the [Interface] section with all server settings but does not include
// any [Peer] sections. Peer configurations should be added separately using AddPeer.
// Returns the configuration file content as a string in WireGuard's INI-like format.
func (sc *ServerConfig) GenerateConfigFile() string {
	var config strings.Builder
	
	config.WriteString("[Interface]\n")
	config.WriteString(fmt.Sprintf("PrivateKey = %s\n", sc.PrivateKey))
	config.WriteString(fmt.Sprintf("Address = %s\n", sc.Address))
	config.WriteString(fmt.Sprintf("ListenPort = %d\n", sc.ListenPort))
	
	for _, cmd := range sc.PostUp {
		config.WriteString(fmt.Sprintf("PostUp = %s\n", cmd))
	}
	
	for _, cmd := range sc.PostDown {
		config.WriteString(fmt.Sprintf("PostDown = %s\n", cmd))
	}
	
	return config.String()
}

// AddPeer generates a [Peer] section configuration for a client.
// This method creates the configuration text that can be appended to the server
// configuration file to allow a specific client to connect. The client is allowed
// to use only their assigned IP address (/32 network).
// Returns the peer configuration section as a string.
func (sc *ServerConfig) AddPeer(clientPublicKey, clientIP string) string {
	return fmt.Sprintf("\n[Peer]\nPublicKey = %s\nAllowedIPs = %s/32\n", clientPublicKey, clientIP)
}

// NewClientConfig creates a new client configuration with generated cryptographic keys.
// It automatically generates a secure key pair for the client and configures it
// to connect to the specified server. The client is configured to route all traffic
// through the VPN by default (AllowedIPs = "0.0.0.0/0").
// Returns a ClientConfig pointer or an error if key generation fails.
func NewClientConfig(serverConfig *ServerConfig, clientIP, serverEndpoint string) (*ClientConfig, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client keys: %w", err)
	}

	return &ClientConfig{
		PrivateKey:      keyPair.PrivateKey,
		PublicKey:       keyPair.PublicKey,
		Address:         fmt.Sprintf("%s/32", clientIP),
		DNS:             serverConfig.DNS,
		ServerPublicKey: serverConfig.PublicKey,
		ServerEndpoint:  serverEndpoint,
		AllowedIPs:      []string{"0.0.0.0/0"},
	}, nil
}

// GenerateConfigFile creates a WireGuard configuration file content for the client.
// It generates a complete client configuration including the [Interface] section
// with client settings and a [Peer] section for connecting to the server.
// The configuration includes persistent keepalive to maintain the connection.
// Returns the configuration file content as a string in WireGuard's INI-like format.
func (cc *ClientConfig) GenerateConfigFile() string {
	var config strings.Builder
	
	config.WriteString("[Interface]\n")
	config.WriteString(fmt.Sprintf("PrivateKey = %s\n", cc.PrivateKey))
	config.WriteString(fmt.Sprintf("Address = %s\n", cc.Address))
	config.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(cc.DNS, ", ")))
	
	config.WriteString("\n[Peer]\n")
	config.WriteString(fmt.Sprintf("PublicKey = %s\n", cc.ServerPublicKey))
	config.WriteString(fmt.Sprintf("Endpoint = %s\n", cc.ServerEndpoint))
	config.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(cc.AllowedIPs, ", ")))
	config.WriteString("PersistentKeepalive = 25\n")
	
	return config.String()
}

// incrementIP increments an IP address by the given amount.
// This helper function performs arithmetic on IP addresses, properly handling
// byte overflow across octets. It's used for calculating server IP addresses
// within a network range.
// Returns a new IP address that is 'inc' positions higher than the input.
func incrementIP(ip net.IP, inc int) net.IP {
	result := make(net.IP, len(ip))
	copy(result, ip)
	
	for i := len(result) - 1; i >= 0 && inc > 0; i-- {
		val := int(result[i]) + inc
		result[i] = byte(val & 0xFF)
		inc = val >> 8
	}
	
	return result
}

// getCIDRBits extracts the number of network bits from a subnet mask.
// This helper function is used to convert subnet masks to CIDR notation
// for use in WireGuard configuration files.
// Returns the number of network bits (e.g., 24 for a /24 network).
func getCIDRBits(ipNet *net.IPNet) int {
	ones, _ := ipNet.Mask.Size()
	return ones
}