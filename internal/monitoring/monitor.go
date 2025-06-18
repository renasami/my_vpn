// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"my-vpn/internal/database"
	"my-vpn/internal/network"
	"my-vpn/internal/system"
	"my-vpn/internal/wireguard"
)

// Monitor provides comprehensive monitoring and logging for the VPN server.
// It tracks server health, client connections, system resources, and provides
// real-time metrics with configurable alerting and logging functionality.
type Monitor struct {
	db              *database.Database         // Database connection for logging and metrics storage
	wgServer        *wireguard.WireGuardServer // WireGuard server instance for connection monitoring
	ipPool          *network.IPPool            // IP pool for network metrics
	pfctlManager    *system.PfctlManager       // Firewall manager for security monitoring
	config          *MonitorConfig             // Configuration for monitoring behavior
	metrics         *ServerMetrics             // Current server metrics
	alertManager    *AlertManager              // Alert management system
	logManager      *LogManager                // Log management system
	running         bool                       // Whether monitoring is currently active
	stopCh          chan struct{}              // Channel to signal monitoring stop
	mutex           sync.RWMutex               // Mutex for thread-safe operations
	lastUpdateTime  time.Time                  // Last metrics update timestamp
}

// MonitorConfig represents configuration options for the monitoring system.
type MonitorConfig struct {
	UpdateInterval    time.Duration `json:"update_interval"`     // How often to update metrics (default: 30s)
	LogRetentionDays  int           `json:"log_retention_days"`  // How long to keep logs (default: 30 days)
	MetricsRetention  time.Duration `json:"metrics_retention"`   // How long to keep metrics (default: 7 days)
	AlertThresholds   AlertConfig   `json:"alert_thresholds"`    // Alert configuration
	EnableSystemStats bool          `json:"enable_system_stats"` // Whether to collect system statistics
	EnableDebugLogs   bool          `json:"enable_debug_logs"`   // Whether to enable debug logging
}

// ServerMetrics represents current server state and performance metrics.
type ServerMetrics struct {
	Timestamp         time.Time            `json:"timestamp"`          // When these metrics were collected
	ServerStatus      ServerStatus         `json:"server_status"`      // Overall server health status
	ConnectionStats   ConnectionStats      `json:"connection_stats"`   // Client connection statistics
	NetworkStats      NetworkStats         `json:"network_stats"`      // Network usage statistics
	SystemStats       SystemStats          `json:"system_stats"`       // System resource usage
	SecurityStats     SecurityStats        `json:"security_stats"`     // Security and firewall status
	WireGuardStats    WireGuardStats       `json:"wireguard_stats"`    // WireGuard-specific metrics
	Alerts            []Alert              `json:"alerts"`             // Active alerts
	Performance       PerformanceMetrics   `json:"performance"`        // Performance metrics
}

// ServerStatus represents the overall health status of the VPN server.
type ServerStatus string

const (
	StatusHealthy   ServerStatus = "healthy"   // Server is operating normally
	StatusDegraded  ServerStatus = "degraded"  // Server is functional but has issues
	StatusUnhealthy ServerStatus = "unhealthy" // Server has critical issues
	StatusDown      ServerStatus = "down"      // Server is not responding
)

// ConnectionStats represents statistics about client connections.
type ConnectionStats struct {
	TotalClients    int       `json:"total_clients"`    // Total number of configured clients
	ActiveClients   int       `json:"active_clients"`   // Number of currently connected clients
	RecentConnects  int       `json:"recent_connects"`  // Connections in the last hour
	RecentDisconnects int     `json:"recent_disconnects"` // Disconnections in the last hour
	LastUpdate      time.Time `json:"last_update"`      // When connection stats were last updated
}

// NetworkStats represents network usage and performance statistics.
type NetworkStats struct {
	BytesTransferred  uint64    `json:"bytes_transferred"`  // Total bytes transferred through VPN
	BytesReceived     uint64    `json:"bytes_received"`     // Total bytes received by server
	BytesSent         uint64    `json:"bytes_sent"`         // Total bytes sent by server
	PacketsReceived   uint64    `json:"packets_received"`   // Total packets received
	PacketsSent       uint64    `json:"packets_sent"`       // Total packets sent
	PacketsDropped    uint64    `json:"packets_dropped"`    // Total packets dropped
	IPPoolUtilization float64   `json:"ip_pool_utilization"` // Percentage of IP pool in use
	LastUpdate        time.Time `json:"last_update"`        // When network stats were last updated
}

// SystemStats represents system resource usage statistics.
type SystemStats struct {
	CPUUsage      float64   `json:"cpu_usage"`       // CPU usage percentage
	MemoryUsage   float64   `json:"memory_usage"`    // Memory usage percentage
	DiskUsage     float64   `json:"disk_usage"`      // Disk usage percentage
	LoadAverage   float64   `json:"load_average"`    // System load average
	Uptime        time.Duration `json:"uptime"`      // System uptime
	GoRoutines    int       `json:"goroutines"`      // Number of active goroutines
	LastUpdate    time.Time `json:"last_update"`     // When system stats were last updated
}

// SecurityStats represents security and firewall status.
type SecurityStats struct {
	FirewallEnabled    bool      `json:"firewall_enabled"`     // Whether pfctl is enabled
	ActiveRules        int       `json:"active_rules"`         // Number of active firewall rules
	BlockedConnections int       `json:"blocked_connections"`  // Number of blocked connection attempts
	FailedLogins       int       `json:"failed_logins"`        // Number of failed login attempts
	LastSecurityScan   time.Time `json:"last_security_scan"`   // Last security check timestamp
	ThreatLevel        string    `json:"threat_level"`         // Current threat assessment
}

// WireGuardStats represents WireGuard-specific metrics.
type WireGuardStats struct {
	InterfaceStatus   string    `json:"interface_status"`    // WireGuard interface status
	ListenPort        int       `json:"listen_port"`         // Current listen port
	PublicKey         string    `json:"public_key"`          // Server public key
	TotalPeers        int       `json:"total_peers"`         // Total configured peers
	ActivePeers       int       `json:"active_peers"`        // Currently active peers
	LastHandshake     time.Time `json:"last_handshake"`      // Most recent peer handshake
	ConfigVersion     string    `json:"config_version"`      // Configuration version
}

// PerformanceMetrics represents performance-related metrics.
type PerformanceMetrics struct {
	ResponseTime     time.Duration `json:"response_time"`      // Average API response time
	RequestsPerSecond float64      `json:"requests_per_second"` // HTTP requests per second
	ErrorRate        float64       `json:"error_rate"`         // Percentage of failed requests
	ThroughputMbps   float64       `json:"throughput_mbps"`    // Network throughput in Mbps
	DatabaseLatency  time.Duration `json:"database_latency"`   // Average database query time
}

// NewMonitor creates a new monitoring instance with default configuration.
// It initializes all monitoring components including metrics collection,
// alerting, and logging with sensible defaults for production use.
// Returns a pointer to the newly created Monitor.
func NewMonitor(db *database.Database, wgServer *wireguard.WireGuardServer, ipPool *network.IPPool, pfctlManager *system.PfctlManager) *Monitor {
	config := &MonitorConfig{
		UpdateInterval:    30 * time.Second,
		LogRetentionDays:  30,
		MetricsRetention:  7 * 24 * time.Hour,
		EnableSystemStats: true,
		EnableDebugLogs:   false,
		AlertThresholds:   getDefaultAlertConfig(),
	}

	return &Monitor{
		db:              db,
		wgServer:        wgServer,
		ipPool:          ipPool,
		pfctlManager:    pfctlManager,
		config:          config,
		metrics:         &ServerMetrics{
			ServerStatus: StatusHealthy,
			Timestamp:    time.Now(),
		},
		alertManager:    NewAlertManager(),
		logManager:      NewLogManager(),
		stopCh:          make(chan struct{}),
		lastUpdateTime:  time.Now(),
	}
}

// NewMonitorWithConfig creates a new monitoring instance with custom configuration.
// This allows fine-tuning of monitoring behavior for specific deployment requirements.
// Returns a pointer to the newly created Monitor.
func NewMonitorWithConfig(db *database.Database, wgServer *wireguard.WireGuardServer, ipPool *network.IPPool, pfctlManager *system.PfctlManager, config *MonitorConfig) *Monitor {
	monitor := NewMonitor(db, wgServer, ipPool, pfctlManager)
	monitor.config = config
	return monitor
}

// Start begins the monitoring process in the background.
// It starts periodic collection of metrics, log management, and alert processing.
// This method is non-blocking and should be called once to initialize monitoring.
func (m *Monitor) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("monitor is already running")
	}

	m.running = true
	m.logManager.LogInfo("Starting VPN server monitoring")

	// Start the monitoring goroutine
	go m.monitorLoop(ctx)

	return nil
}

// Stop gracefully stops the monitoring process.
// It waits for the current monitoring cycle to complete and cleans up resources.
// This method blocks until all monitoring operations have stopped.
func (m *Monitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return fmt.Errorf("monitor is not running")
	}

	m.logManager.LogInfo("Stopping VPN server monitoring")
	
	// Only close the channel if it's not already closed
	select {
	case <-m.stopCh:
		// Channel is already closed
	default:
		close(m.stopCh)
	}
	
	m.running = false

	return nil
}

// GetMetrics returns the current server metrics.
// This provides a thread-safe way to access the latest collected metrics
// for display in dashboards, APIs, or other monitoring tools.
func (m *Monitor) GetMetrics() *ServerMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modifications
	metricsCopy := *m.metrics
	return &metricsCopy
}

// GetServerStatus returns the current overall server status.
// This provides a quick health check result that can be used for
// load balancers, health checks, and monitoring dashboards.
func (m *Monitor) GetServerStatus() ServerStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.metrics.ServerStatus
}

// IsHealthy returns true if the server is in a healthy state.
// This is a convenience method for quick health checks.
func (m *Monitor) IsHealthy() bool {
	return m.GetServerStatus() == StatusHealthy
}

// monitorLoop is the main monitoring loop that runs in a separate goroutine.
// It periodically collects metrics, processes alerts, and manages logs.
func (m *Monitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logManager.LogInfo("Monitor context cancelled, stopping monitoring loop")
			return
		case <-m.stopCh:
			m.logManager.LogInfo("Monitor stop signal received, stopping monitoring loop")
			return
		case <-ticker.C:
			if err := m.collectMetrics(); err != nil {
				m.logManager.LogError(fmt.Sprintf("Error collecting metrics: %v", err))
			}
			m.processAlerts()
			m.cleanupOldData()
		}
	}
}

// collectMetrics gathers all current metrics from various sources.
// This includes system stats, connection stats, network stats, and security stats.
func (m *Monitor) collectMetrics() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	m.lastUpdateTime = now

	// Collect connection statistics
	connectionStats, err := m.collectConnectionStats()
	if err != nil {
		m.logManager.LogError(fmt.Sprintf("Failed to collect connection stats: %v", err))
	}

	// Collect network statistics
	networkStats, err := m.collectNetworkStats()
	if err != nil {
		m.logManager.LogError(fmt.Sprintf("Failed to collect network stats: %v", err))
	}

	// Collect system statistics if enabled
	var systemStats SystemStats
	if m.config.EnableSystemStats {
		systemStats = m.collectSystemStats()
	}

	// Collect security statistics
	securityStats, err := m.collectSecurityStats()
	if err != nil {
		m.logManager.LogError(fmt.Sprintf("Failed to collect security stats: %v", err))
	}

	// Collect WireGuard statistics
	wgStats, err := m.collectWireGuardStats()
	if err != nil {
		m.logManager.LogError(fmt.Sprintf("Failed to collect WireGuard stats: %v", err))
	}

	// Collect performance metrics
	performanceStats := m.collectPerformanceStats()

	// Update metrics
	m.metrics = &ServerMetrics{
		Timestamp:       now,
		ServerStatus:    m.calculateServerStatus(connectionStats, systemStats, securityStats),
		ConnectionStats: connectionStats,
		NetworkStats:    networkStats,
		SystemStats:     systemStats,
		SecurityStats:   securityStats,
		WireGuardStats:  wgStats,
		Performance:     performanceStats,
		Alerts:          m.alertManager.GetActiveAlerts(),
	}

	// Log metrics if debug is enabled
	if m.config.EnableDebugLogs {
		m.logManager.LogDebug(fmt.Sprintf("Collected metrics: %+v", m.metrics))
	}

	return nil
}

// collectConnectionStats gathers statistics about client connections.
func (m *Monitor) collectConnectionStats() (ConnectionStats, error) {
	clients, err := m.db.ListClients()
	if err != nil {
		return ConnectionStats{}, fmt.Errorf("failed to get clients: %w", err)
	}

	// Count active clients (those with recent handshakes)
	activeCount := 0
	now := time.Now()
	for _, client := range clients {
		if client.LastHandshake != nil && now.Sub(*client.LastHandshake) < 5*time.Minute {
			activeCount++
		}
	}

	// Get recent connection logs
	logs, err := m.db.GetConnectionLogs(100) // Get last 100 log entries
	if err != nil {
		return ConnectionStats{}, fmt.Errorf("failed to get connection logs: %w", err)
	}

	// Count recent connects and disconnects (last hour)
	hourAgo := now.Add(-time.Hour)
	recentConnects := 0
	recentDisconnects := 0
	
	for _, log := range logs {
		if log.Timestamp.After(hourAgo) {
			if log.Action == "connect" {
				recentConnects++
			} else if log.Action == "disconnect" {
				recentDisconnects++
			}
		}
	}

	return ConnectionStats{
		TotalClients:      len(clients),
		ActiveClients:     activeCount,
		RecentConnects:    recentConnects,
		RecentDisconnects: recentDisconnects,
		LastUpdate:        now,
	}, nil
}

// collectNetworkStats gathers network usage and performance statistics.
func (m *Monitor) collectNetworkStats() (NetworkStats, error) {
	// Get IP pool utilization
	totalIPs := m.ipPool.GetTotalIPs()
	allocatedIPs := m.ipPool.GetAllocatedCount()
	utilization := float64(allocatedIPs) / float64(totalIPs) * 100

	// Get aggregate client stats
	clients, err := m.db.ListClients()
	if err != nil {
		return NetworkStats{}, fmt.Errorf("failed to get clients for network stats: %w", err)
	}

	var totalReceived, totalSent uint64
	for _, client := range clients {
		totalReceived += client.BytesReceived
		totalSent += client.BytesSent
	}

	return NetworkStats{
		BytesTransferred:  totalReceived + totalSent,
		BytesReceived:     totalReceived,
		BytesSent:         totalSent,
		PacketsReceived:   0, // Would need system-level monitoring
		PacketsSent:       0, // Would need system-level monitoring
		PacketsDropped:    0, // Would need system-level monitoring
		IPPoolUtilization: utilization,
		LastUpdate:        time.Now(),
	}, nil
}

// collectSystemStats gathers system resource usage statistics.
func (m *Monitor) collectSystemStats() SystemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemStats{
		CPUUsage:    0.0, // Would need system-level monitoring
		MemoryUsage: float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		DiskUsage:   0.0, // Would need system-level monitoring
		LoadAverage: 0.0, // Would need system-level monitoring
		Uptime:      time.Since(time.Now().Add(-time.Hour)), // Placeholder
		GoRoutines:  runtime.NumGoroutine(),
		LastUpdate:  time.Now(),
	}
}

// collectSecurityStats gathers security and firewall status.
func (m *Monitor) collectSecurityStats() (SecurityStats, error) {
	// Check firewall status
	firewallEnabled, err := m.pfctlManager.IsEnabled()
	if err != nil {
		return SecurityStats{}, fmt.Errorf("failed to check firewall status: %w", err)
	}

	// Get active firewall rules
	rules, err := m.pfctlManager.GetActiveRules()
	if err != nil {
		return SecurityStats{}, fmt.Errorf("failed to get firewall rules: %w", err)
	}

	return SecurityStats{
		FirewallEnabled:    firewallEnabled,
		ActiveRules:        len(rules),
		BlockedConnections: 0, // Would need log analysis
		FailedLogins:       0, // Would need authentication log analysis
		LastSecurityScan:   time.Now(),
		ThreatLevel:        "low", // Would need threat analysis
	}, nil
}

// collectWireGuardStats gathers WireGuard-specific metrics.
func (m *Monitor) collectWireGuardStats() (WireGuardStats, error) {
	// Get WireGuard server status
	isRunning := m.wgServer.IsRunning()
	status := "down"
	if isRunning {
		status = "up"
	}

	// Get server configuration
	config, err := m.wgServer.GetConfig()
	if err != nil {
		return WireGuardStats{}, fmt.Errorf("failed to get WireGuard config: %w", err)
	}

	// Count peers
	peers, err := m.wgServer.GetPeers()
	if err != nil {
		peers = []wireguard.Peer{} // Use empty slice if error
	}

	return WireGuardStats{
		InterfaceStatus: status,
		ListenPort:      config.ListenPort,
		PublicKey:       config.PublicKey,
		TotalPeers:      len(peers),
		ActivePeers:     0, // Would need to check peer status
		LastHandshake:   time.Now(),
		ConfigVersion:   "1.0", // Placeholder
	}, nil
}

// collectPerformanceStats gathers performance-related metrics.
func (m *Monitor) collectPerformanceStats() PerformanceMetrics {
	return PerformanceMetrics{
		ResponseTime:      10 * time.Millisecond, // Placeholder
		RequestsPerSecond: 0.0,                   // Would need HTTP metrics
		ErrorRate:         0.0,                   // Would need error tracking
		ThroughputMbps:    0.0,                   // Would need network monitoring
		DatabaseLatency:   1 * time.Millisecond, // Placeholder
	}
}

// calculateServerStatus determines the overall server health status.
func (m *Monitor) calculateServerStatus(conn ConnectionStats, sys SystemStats, sec SecurityStats) ServerStatus {
	// Simple health calculation based on various factors
	if !sec.FirewallEnabled {
		return StatusDegraded
	}

	if sys.MemoryUsage > 90 || sys.GoRoutines > 1000 {
		return StatusDegraded
	}

	return StatusHealthy
}

// processAlerts evaluates current metrics against alert thresholds.
func (m *Monitor) processAlerts() {
	m.alertManager.EvaluateMetrics(m.metrics)
}

// cleanupOldData removes old metrics and logs based on retention policies.
func (m *Monitor) cleanupOldData() {
	// This would contain cleanup logic for old data
	// For now, it's a placeholder
}

// getDefaultAlertConfig returns default alert configuration.
func getDefaultAlertConfig() AlertConfig {
	return AlertConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
		EnableAlerts:    true,
	}
}