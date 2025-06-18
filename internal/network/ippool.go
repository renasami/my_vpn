// Package network provides IP address pool management for VPN clients.
// It handles allocation, deallocation, and tracking of IP addresses within
// a specified CIDR network range, ensuring no conflicts and proper resource management.
package network

import (
	"fmt"
	"net"
	"sort"
	"sync"
)

// IPPool manages a pool of IP addresses for VPN client allocation.
// It provides thread-safe operations for allocating and releasing IP addresses
// within a specified network range, while reserving the first usable IP for the server.
type IPPool struct {
	mu               sync.RWMutex    // Protects concurrent access to the pool
	network          string          // Original CIDR notation (e.g., "10.0.0.0/24")
	ipNet            *net.IPNet      // Parsed network information
	serverIP         string          // Reserved IP address for the VPN server
	allocated        map[string]bool // Tracks which IP addresses are currently allocated
	networkAddress   string          // Network address (e.g., "10.0.0.0")
	broadcastAddress string          // Broadcast address (e.g., "10.0.0.255")
	totalHosts       int             // Total number of usable host addresses
}

// NetworkInfo provides detailed information about the network configuration.
// It includes network topology details and addressing scheme information.
type NetworkInfo struct {
	Network          string `json:"network"`           // CIDR notation of the network
	ServerIP         string `json:"server_ip"`         // IP address reserved for the server
	NetworkAddress   string `json:"network_address"`   // Network address
	BroadcastAddress string `json:"broadcast_address"` // Broadcast address
	TotalHosts       int    `json:"total_hosts"`       // Total number of usable host addresses
}

// NewIPPool creates a new IP pool from the given CIDR notation.
// It validates the network range, calculates available addresses, and reserves
// the first usable IP address for the VPN server. The network must be at least /29
// to provide sufficient addresses for meaningful VPN usage.
// Returns an IPPool instance or an error if the CIDR is invalid or too small.
func NewIPPool(cidr string) (*IPPool, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	// Check if it's IPv4
	if ipNet.IP.To4() == nil {
		return nil, fmt.Errorf("IPv6 not supported")
	}

	// Calculate network size
	ones, bits := ipNet.Mask.Size()
	if bits-ones < 3 {
		return nil, fmt.Errorf("network too small, need at least /29")
	}

	// Calculate total hosts (excluding network and broadcast)
	totalHosts := (1 << (bits - ones)) - 2

	// Get network and broadcast addresses
	networkAddr := ipNet.IP.Mask(ipNet.Mask)
	broadcastAddr := make(net.IP, len(networkAddr))
	copy(broadcastAddr, networkAddr)
	
	// Calculate broadcast address
	for i := 0; i < len(broadcastAddr); i++ {
		broadcastAddr[i] |= ^ipNet.Mask[i]
	}

	// Server IP is typically the first usable IP (network + 1)
	serverIP := incrementIP(networkAddr, 1)

	pool := &IPPool{
		network:          cidr,
		ipNet:            ipNet,
		serverIP:         serverIP.String(),
		allocated:        make(map[string]bool),
		networkAddress:   networkAddr.String(),
		broadcastAddress: broadcastAddr.String(),
		totalHosts:       totalHosts,
	}

	// Mark server IP as allocated
	pool.allocated[pool.serverIP] = true

	return pool, nil
}

// AllocateIP allocates the next available IP address from the pool.
// It performs a sequential search starting from the second usable IP address
// (since the first is reserved for the server) and returns the first available address.
// This method is thread-safe and will not allocate network, broadcast, or server addresses.
// Returns the allocated IP address as a string or an error if no addresses are available.
func (p *IPPool) AllocateIP() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Start from the second IP (server IP is first)
	currentIP := incrementIP(p.ipNet.IP.Mask(p.ipNet.Mask), 2)
	broadcastIP := net.ParseIP(p.broadcastAddress)

	for !currentIP.Equal(broadcastIP) {
		ipStr := currentIP.String()
		if !p.allocated[ipStr] {
			p.allocated[ipStr] = true
			return ipStr, nil
		}
		currentIP = incrementIP(currentIP, 1)
	}

	return "", fmt.Errorf("no available IP addresses in pool")
}

// AllocateSpecificIP allocates a specific IP address if it's available.
// This method allows manual assignment of IP addresses for specific clients.
// It validates that the IP is within the network range, not reserved, and not already allocated.
// Returns an error if the IP address is invalid, outside the network range,
// reserved for special use, or already allocated to another client.
func (p *IPPool) AllocateSpecificIP(ip string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// Check if IP is in network range
	if !p.ipNet.Contains(parsedIP) {
		return fmt.Errorf("IP address not in network range: %s", ip)
	}

	// Check if it's the network address
	if ip == p.networkAddress {
		return fmt.Errorf("cannot allocate network address: %s", ip)
	}

	// Check if it's the broadcast address
	if ip == p.broadcastAddress {
		return fmt.Errorf("cannot allocate broadcast address: %s", ip)
	}

	// Check if it's the server IP
	if ip == p.serverIP {
		return fmt.Errorf("IP address reserved for server: %s", ip)
	}

	// Check if already allocated
	if p.allocated[ip] {
		return fmt.Errorf("IP address already allocated: %s", ip)
	}

	p.allocated[ip] = true
	return nil
}

// ReleaseIP releases a previously allocated IP address back to the pool.
// The released address becomes available for future allocation to other clients.
// This method validates that the IP is within the network range and currently allocated.
// The server IP address cannot be released as it's permanently reserved.
// Returns an error if the IP is invalid, not in the network, not allocated, or is the server IP.
func (p *IPPool) ReleaseIP(ip string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// Check if IP is in network range
	if !p.ipNet.Contains(parsedIP) {
		return fmt.Errorf("IP address not in network range: %s", ip)
	}

	// Check if it's allocated
	if !p.allocated[ip] {
		return fmt.Errorf("IP address not allocated: %s", ip)
	}

	// Don't allow releasing server IP
	if ip == p.serverIP {
		return fmt.Errorf("cannot release server IP: %s", ip)
	}

	delete(p.allocated, ip)
	return nil
}

// IsAllocated checks if an IP address is currently allocated.
// This method provides a thread-safe way to query the allocation status
// of any IP address, including the server IP which is always considered allocated.
// Returns true if the IP address is allocated, false otherwise.
func (p *IPPool) IsAllocated(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.allocated[ip]
}

// GetServerIP returns the IP address reserved for the VPN server.
// This address is automatically reserved during pool creation and cannot be allocated to clients.
// Returns the server IP address as a string.
func (p *IPPool) GetServerIP() string {
	return p.serverIP
}

// GetAllocatedIPs returns a sorted list of IP addresses currently allocated to clients.
// The server IP address is excluded from this list as it's a special reserved address.
// This method is thread-safe and returns a new slice that can be safely modified.
// Returns a slice of IP address strings sorted in ascending order.
func (p *IPPool) GetAllocatedIPs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var ips []string
	for ip := range p.allocated {
		if ip != p.serverIP {
			ips = append(ips, ip)
		}
	}

	sort.Strings(ips)
	return ips
}

// GetAvailableCount returns the number of IP addresses available for allocation.
// This count excludes the server IP, network address, and broadcast address,
// as well as any currently allocated client addresses.
// This method is thread-safe and provides real-time availability information.
// Returns the count of available IP addresses as an integer.
func (p *IPPool) GetAvailableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Total hosts minus allocated IPs (excluding server IP which is always allocated)
	allocatedCount := len(p.allocated) - 1 // -1 for server IP
	return p.totalHosts - 1 - allocatedCount // -1 for server IP
}

// GetNetworkInfo returns comprehensive information about the network configuration.
// This includes the network topology, addressing scheme, and capacity information.
// The returned information is useful for monitoring, configuration, and troubleshooting.
// This method is thread-safe and returns a copy of the network information.
// Returns a NetworkInfo structure containing detailed network configuration.
func (p *IPPool) GetNetworkInfo() NetworkInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return NetworkInfo{
		Network:          p.network,
		ServerIP:         p.serverIP,
		NetworkAddress:   p.networkAddress,
		BroadcastAddress: p.broadcastAddress,
		TotalHosts:       p.totalHosts,
	}
}

// GetTotalIPs returns the total number of usable IP addresses in the pool.
// This includes both allocated and available addresses, excluding network
// and broadcast addresses which are not usable for host assignment.
// This method is thread-safe and provides capacity information for monitoring.
// Returns the total number of usable host addresses in the network.
func (p *IPPool) GetTotalIPs() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.totalHosts
}

// GetAllocatedCount returns the number of currently allocated IP addresses.
// This count includes the server IP and all client IPs that have been assigned.
// This method is thread-safe and provides utilization information for monitoring.
// Returns the current number of allocated IP addresses.
func (p *IPPool) GetAllocatedCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return len(p.allocated)
}

// incrementIP increments an IP address by the given amount.
// This is a helper function that performs arithmetic on IP addresses,
// properly handling byte overflow across octets. It's used internally
// for sequential IP address allocation and network calculations.
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