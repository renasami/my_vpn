// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"my-vpn/internal/database"
	"my-vpn/internal/network"
	"my-vpn/internal/system"
	"my-vpn/internal/wireguard"
)

func setupTestMonitor(t *testing.T) (*Monitor, func()) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&database.User{}, &database.Client{}, &database.ServerConfig{}, &database.ConnectionLog{})
	require.NoError(t, err)

	database := &database.Database{DB: db}

	// Create IP pool
	ipPool, err := network.NewIPPool("10.0.0.0/24")
	require.NoError(t, err)

	// Create WireGuard server
	wgServer := wireguard.NewWireGuardServerWithConfig("/tmp", "wg0")

	// Create pfctl manager
	pfctlManager := system.NewPfctlManager()

	// Create monitor
	monitor := NewMonitor(database, wgServer, ipPool, pfctlManager)

	cleanup := func() {
		monitor.Stop()
		db.Exec("DROP TABLE IF EXISTS users")
		db.Exec("DROP TABLE IF EXISTS clients")
		db.Exec("DROP TABLE IF EXISTS server_configs")
		db.Exec("DROP TABLE IF EXISTS connection_logs")
	}

	return monitor, cleanup
}

func TestNewMonitor(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should create monitor with default configuration", func(t *testing.T) {
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.config)
		assert.NotNil(t, monitor.metrics)
		assert.NotNil(t, monitor.alertManager)
		assert.NotNil(t, monitor.logManager)
		assert.Equal(t, 30*time.Second, monitor.config.UpdateInterval)
		assert.Equal(t, 30, monitor.config.LogRetentionDays)
		assert.True(t, monitor.config.EnableSystemStats)
	})
}

func TestNewMonitorWithConfig(t *testing.T) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&database.User{}, &database.Client{}, &database.ServerConfig{}, &database.ConnectionLog{})
	require.NoError(t, err)

	database := &database.Database{DB: db}
	ipPool, _ := network.NewIPPool("10.0.0.0/24")
	wgServer := wireguard.NewWireGuardServerWithConfig("/tmp", "wg0")
	pfctlManager := system.NewPfctlManager()

	t.Run("should create monitor with custom configuration", func(t *testing.T) {
		config := &MonitorConfig{
			UpdateInterval:    10 * time.Second,
			LogRetentionDays:  60,
			EnableSystemStats: false,
			EnableDebugLogs:   true,
		}

		monitor := NewMonitorWithConfig(database, wgServer, ipPool, pfctlManager, config)
		defer monitor.Stop()

		assert.NotNil(t, monitor)
		assert.Equal(t, 10*time.Second, monitor.config.UpdateInterval)
		assert.Equal(t, 60, monitor.config.LogRetentionDays)
		assert.False(t, monitor.config.EnableSystemStats)
		assert.True(t, monitor.config.EnableDebugLogs)
	})
}

func TestMonitor_StartStop(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should start and stop monitor successfully", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start monitor
		err := monitor.Start(ctx)
		assert.NoError(t, err)
		assert.True(t, monitor.running)

		// Stop monitor
		err = monitor.Stop()
		assert.NoError(t, err)
		assert.False(t, monitor.running)
	})

	t.Run("should not start already running monitor", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start monitor
		err := monitor.Start(ctx)
		require.NoError(t, err)

		// Try to start again
		err = monitor.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")

		// Clean up
		monitor.Stop()
	})

	t.Run("should not stop non-running monitor", func(t *testing.T) {
		// Try to stop non-running monitor
		err := monitor.Stop()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})
}

func TestMonitor_GetMetrics(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should return current metrics", func(t *testing.T) {
		metrics := monitor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.IsType(t, &ServerMetrics{}, metrics)
	})

	t.Run("should return copy of metrics", func(t *testing.T) {
		metrics1 := monitor.GetMetrics()
		metrics2 := monitor.GetMetrics()

		// Should be different pointers but same values
		assert.NotSame(t, metrics1, metrics2)
		assert.Equal(t, metrics1.Timestamp, metrics2.Timestamp)
	})
}

func TestMonitor_GetServerStatus(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should return current server status", func(t *testing.T) {
		status := monitor.GetServerStatus()
		assert.NotEmpty(t, status)
		assert.Contains(t, []ServerStatus{StatusHealthy, StatusDegraded, StatusUnhealthy, StatusDown}, status)
	})
}

func TestMonitor_IsHealthy(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should return health status", func(t *testing.T) {
		isHealthy := monitor.IsHealthy()
		assert.IsType(t, true, isHealthy)
	})
}

func TestMonitor_CollectMetrics(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should collect metrics without error", func(t *testing.T) {
		err := monitor.collectMetrics()
		assert.NoError(t, err)

		metrics := monitor.GetMetrics()
		assert.NotZero(t, metrics.Timestamp)
		assert.NotEmpty(t, metrics.ServerStatus)
	})
}

func TestMonitor_CollectConnectionStats(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should collect connection stats", func(t *testing.T) {
		// Create some test clients
		client1 := &database.Client{
			Name:      "test-client-1",
			PublicKey: "key1",
			IPAddress: "10.0.0.2",
			Enabled:   true,
		}
		client2 := &database.Client{
			Name:      "test-client-2",
			PublicKey: "key2",
			IPAddress: "10.0.0.3",
			Enabled:   true,
		}

		err := monitor.db.CreateClient(client1)
		require.NoError(t, err)
		err = monitor.db.CreateClient(client2)
		require.NoError(t, err)

		// Collect connection stats
		stats, err := monitor.collectConnectionStats()
		assert.NoError(t, err)
		assert.Equal(t, 2, stats.TotalClients)
		assert.GreaterOrEqual(t, stats.ActiveClients, 0)
		assert.GreaterOrEqual(t, stats.RecentConnects, 0)
		assert.GreaterOrEqual(t, stats.RecentDisconnects, 0)
		assert.NotZero(t, stats.LastUpdate)
	})
}

func TestMonitor_CollectNetworkStats(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should collect network stats", func(t *testing.T) {
		stats, err := monitor.collectNetworkStats()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, stats.BytesTransferred, uint64(0))
		assert.GreaterOrEqual(t, stats.BytesReceived, uint64(0))
		assert.GreaterOrEqual(t, stats.BytesSent, uint64(0))
		assert.GreaterOrEqual(t, stats.IPPoolUtilization, 0.0)
		assert.NotZero(t, stats.LastUpdate)
	})
}

func TestMonitor_CollectSystemStats(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should collect system stats", func(t *testing.T) {
		stats := monitor.collectSystemStats()
		assert.GreaterOrEqual(t, stats.CPUUsage, 0.0)
		assert.GreaterOrEqual(t, stats.MemoryUsage, 0.0)
		assert.GreaterOrEqual(t, stats.DiskUsage, 0.0)
		assert.GreaterOrEqual(t, stats.LoadAverage, 0.0)
		assert.Greater(t, stats.GoRoutines, 0)
		assert.NotZero(t, stats.LastUpdate)
	})
}

func TestServerStatus_String(t *testing.T) {
	t.Run("should return correct string representations", func(t *testing.T) {
		assert.Equal(t, "healthy", string(StatusHealthy))
		assert.Equal(t, "degraded", string(StatusDegraded))
		assert.Equal(t, "unhealthy", string(StatusUnhealthy))
		assert.Equal(t, "down", string(StatusDown))
	})
}

func TestMonitor_Integration(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should work end-to-end", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start monitoring
		err := monitor.Start(ctx)
		require.NoError(t, err)

		// Wait for at least one metrics collection cycle
		time.Sleep(100 * time.Millisecond)

		// Check that metrics are being collected
		metrics := monitor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.NotZero(t, metrics.Timestamp)

		// Check server status
		status := monitor.GetServerStatus()
		assert.NotEmpty(t, status)

		// Stop monitoring
		err = monitor.Stop()
		assert.NoError(t, err)
	})
}

func TestMonitor_CalculateServerStatus(t *testing.T) {
	monitor, cleanup := setupTestMonitor(t)
	defer cleanup()

	t.Run("should return healthy status for normal conditions", func(t *testing.T) {
		connStats := ConnectionStats{TotalClients: 5, ActiveClients: 3}
		sysStats := SystemStats{MemoryUsage: 50.0, GoRoutines: 100}
		secStats := SecurityStats{FirewallEnabled: true}

		status := monitor.calculateServerStatus(connStats, sysStats, secStats)
		assert.Equal(t, StatusHealthy, status)
	})

	t.Run("should return degraded status for firewall disabled", func(t *testing.T) {
		connStats := ConnectionStats{TotalClients: 5, ActiveClients: 3}
		sysStats := SystemStats{MemoryUsage: 50.0, GoRoutines: 100}
		secStats := SecurityStats{FirewallEnabled: false}

		status := monitor.calculateServerStatus(connStats, sysStats, secStats)
		assert.Equal(t, StatusDegraded, status)
	})

	t.Run("should return degraded status for high memory usage", func(t *testing.T) {
		connStats := ConnectionStats{TotalClients: 5, ActiveClients: 3}
		sysStats := SystemStats{MemoryUsage: 95.0, GoRoutines: 100}
		secStats := SecurityStats{FirewallEnabled: true}

		status := monitor.calculateServerStatus(connStats, sysStats, secStats)
		assert.Equal(t, StatusDegraded, status)
	})

	t.Run("should return degraded status for too many goroutines", func(t *testing.T) {
		connStats := ConnectionStats{TotalClients: 5, ActiveClients: 3}
		sysStats := SystemStats{MemoryUsage: 50.0, GoRoutines: 1500}
		secStats := SecurityStats{FirewallEnabled: true}

		status := monitor.calculateServerStatus(connStats, sysStats, secStats)
		assert.Equal(t, StatusDegraded, status)
	})
}