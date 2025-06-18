// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertManager(t *testing.T) {
	t.Run("should create alert manager with default configuration", func(t *testing.T) {
		am := NewAlertManager()

		assert.NotNil(t, am)
		assert.NotNil(t, am.alerts)
		assert.NotNil(t, am.config)
		assert.Equal(t, 80.0, am.config.CPUThreshold)
		assert.Equal(t, 85.0, am.config.MemoryThreshold)
		assert.Equal(t, 90.0, am.config.DiskThreshold)
		assert.True(t, am.config.EnableAlerts)
	})
}

func TestNewAlertManagerWithConfig(t *testing.T) {
	t.Run("should create alert manager with custom configuration", func(t *testing.T) {
		config := AlertConfig{
			CPUThreshold:    70.0,
			MemoryThreshold: 80.0,
			DiskThreshold:   85.0,
			EnableAlerts:    false,
		}

		am := NewAlertManagerWithConfig(config)

		assert.NotNil(t, am)
		assert.Equal(t, 70.0, am.config.CPUThreshold)
		assert.Equal(t, 80.0, am.config.MemoryThreshold)
		assert.Equal(t, 85.0, am.config.DiskThreshold)
		assert.False(t, am.config.EnableAlerts)
	})
}

func TestAlertManager_EvaluateMetrics(t *testing.T) {
	t.Run("should not create alerts when alerts are disabled", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = false

		metrics := &ServerMetrics{
			SystemStats: SystemStats{
				CPUUsage:    90.0, // Above threshold
				MemoryUsage: 90.0, // Above threshold
			},
		}

		am.EvaluateMetrics(metrics)

		alerts := am.GetActiveAlerts()
		assert.Empty(t, alerts)
	})

	t.Run("should create system alerts when thresholds are exceeded", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = true

		metrics := &ServerMetrics{
			SystemStats: SystemStats{
				CPUUsage:    90.0, // Above 80% threshold
				MemoryUsage: 90.0, // Above 85% threshold
				DiskUsage:   95.0, // Above 90% threshold
			},
			SecurityStats: SecurityStats{
				FirewallEnabled: true, // Prevent firewall alert
			},
		}

		am.EvaluateMetrics(metrics)

		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 3) // CPU, Memory, and Disk alerts

		// Check CPU alert
		cpuAlert := findAlertByID(alerts, "system_cpu_high")
		assert.NotNil(t, cpuAlert)
		assert.Equal(t, AlertTypeSystem, cpuAlert.Type)
		assert.Equal(t, SeverityHigh, cpuAlert.Severity)
		assert.Equal(t, AlertStatusActive, cpuAlert.Status)

		// Check Memory alert
		memAlert := findAlertByID(alerts, "system_memory_high")
		assert.NotNil(t, memAlert)
		assert.Equal(t, AlertTypeSystem, memAlert.Type)
		assert.Equal(t, SeverityHigh, memAlert.Severity)

		// Check Disk alert
		diskAlert := findAlertByID(alerts, "system_disk_high")
		assert.NotNil(t, diskAlert)
		assert.Equal(t, AlertTypeSystem, diskAlert.Type)
		assert.Equal(t, SeverityCritical, diskAlert.Severity)
	})

	t.Run("should resolve alerts when conditions return to normal", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = true
		
		// First create alerts
		metrics := &ServerMetrics{
			SystemStats: SystemStats{
				CPUUsage: 90.0, // Above threshold
			},
			SecurityStats: SecurityStats{
				FirewallEnabled: true, // Prevent firewall alert
			},
		}
		am.EvaluateMetrics(metrics)

		// Verify alert was created
		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 1)

		// Now fix the condition
		metrics.SystemStats.CPUUsage = 50.0 // Below threshold
		am.EvaluateMetrics(metrics)

		// Verify alert was resolved
		alerts = am.GetActiveAlerts()
		assert.Empty(t, alerts)
	})

	t.Run("should create security alerts for firewall issues", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = true
		
		metrics := &ServerMetrics{
			SecurityStats: SecurityStats{
				FirewallEnabled: false,
				FailedLogins:    15, // Above threshold
			},
		}

		am.EvaluateMetrics(metrics)

		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 2) // Firewall and failed login alerts

		// Check firewall alert
		fwAlert := findAlertByID(alerts, "security_firewall_disabled")
		assert.NotNil(t, fwAlert)
		assert.Equal(t, AlertTypeSecurity, fwAlert.Type)
		assert.Equal(t, SeverityCritical, fwAlert.Severity)

		// Check failed login alert
		loginAlert := findAlertByID(alerts, "security_failed_logins")
		assert.NotNil(t, loginAlert)
		assert.Equal(t, AlertTypeSecurity, loginAlert.Type)
		assert.Equal(t, SeverityMedium, loginAlert.Severity)
	})

	t.Run("should create network alerts for high IP pool utilization", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = true
		
		metrics := &ServerMetrics{
			NetworkStats: NetworkStats{
				IPPoolUtilization: 92.0, // Above 90% threshold
			},
			SecurityStats: SecurityStats{
				FirewallEnabled: true, // Prevent firewall alert
			},
		}

		am.EvaluateMetrics(metrics)

		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 1)

		alert := alerts[0]
		assert.Equal(t, "network_ip_pool_high", alert.ID)
		assert.Equal(t, AlertTypeNetwork, alert.Type)
		assert.Equal(t, SeverityMedium, alert.Severity)
	})

	t.Run("should increase severity for very high IP pool utilization", func(t *testing.T) {
		am := NewAlertManager()
		am.config.EnableAlerts = true
		
		metrics := &ServerMetrics{
			NetworkStats: NetworkStats{
				IPPoolUtilization: 96.0, // Above 95% threshold
			},
			SecurityStats: SecurityStats{
				FirewallEnabled: true, // Prevent firewall alert
			},
		}

		am.EvaluateMetrics(metrics)

		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 1)

		alert := alerts[0]
		assert.Equal(t, SeverityHigh, alert.Severity) // Should be high, not medium
	})
}

func TestAlertManager_ResolveAlert(t *testing.T) {
	am := NewAlertManager()

	t.Run("should resolve active alert", func(t *testing.T) {
		// Create an alert first
		now := time.Now()
		am.createOrUpdateAlert("test_alert", AlertTypeSystem, SeverityMedium, "Test Alert", "Test description", now, nil)

		// Verify alert is active
		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 1)

		// Resolve the alert
		err := am.ResolveAlert("test_alert")
		assert.NoError(t, err)

		// Verify alert is resolved
		alerts = am.GetActiveAlerts()
		assert.Empty(t, alerts)

		// Check that the alert still exists but is resolved
		allAlerts := am.GetAllAlerts(time.Now().Add(-time.Hour))
		assert.Len(t, allAlerts, 1)
		assert.Equal(t, AlertStatusResolved, allAlerts[0].Status)
		assert.NotNil(t, allAlerts[0].ResolvedAt)
	})

	t.Run("should return error for non-existent alert", func(t *testing.T) {
		err := am.ResolveAlert("non_existent_alert")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert not found")
	})

	t.Run("should return error for already resolved alert", func(t *testing.T) {
		// Create and resolve an alert
		now := time.Now()
		am.createOrUpdateAlert("test_alert_2", AlertTypeSystem, SeverityMedium, "Test Alert", "Test description", now, nil)
		err := am.ResolveAlert("test_alert_2")
		require.NoError(t, err)

		// Try to resolve again
		err = am.ResolveAlert("test_alert_2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already resolved")
	})
}

func TestAlertManager_SuppressAlert(t *testing.T) {
	am := NewAlertManager()

	t.Run("should suppress active alert", func(t *testing.T) {
		// Create an alert first
		now := time.Now()
		am.createOrUpdateAlert("test_alert", AlertTypeSystem, SeverityMedium, "Test Alert", "Test description", now, nil)

		// Suppress the alert
		err := am.SuppressAlert("test_alert", time.Hour)
		assert.NoError(t, err)

		// Verify alert is suppressed
		alerts := am.GetActiveAlerts()
		assert.Empty(t, alerts) // Suppressed alerts should not appear in active alerts

		// Check that the alert exists but is suppressed
		allAlerts := am.GetAllAlerts(time.Now().Add(-time.Hour))
		assert.Len(t, allAlerts, 1)
		assert.Equal(t, AlertStatusSuppressed, allAlerts[0].Status)
		assert.NotNil(t, allAlerts[0].Metadata["suppressed_until"])
	})

	t.Run("should return error for non-existent alert", func(t *testing.T) {
		err := am.SuppressAlert("non_existent_alert", time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert not found")
	})
}

func TestAlertManager_GetAllAlerts(t *testing.T) {
	am := NewAlertManager()

	t.Run("should return alerts within time range", func(t *testing.T) {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		
		// Create an alert from yesterday
		am.createOrUpdateAlert("old_alert", AlertTypeSystem, SeverityMedium, "Old Alert", "Old description", yesterday, nil)
		
		// Create an alert from now
		am.createOrUpdateAlert("new_alert", AlertTypeSystem, SeverityMedium, "New Alert", "New description", now, nil)

		// Get alerts since 12 hours ago
		since := now.Add(-12 * time.Hour)
		alerts := am.GetAllAlerts(since)

		// Should only return the new alert
		assert.Len(t, alerts, 1)
		assert.Equal(t, "new_alert", alerts[0].ID)
	})
}

func TestAlertLevel_String(t *testing.T) {
	t.Run("should return correct string representations", func(t *testing.T) {
		assert.Equal(t, "low", string(SeverityLow))
		assert.Equal(t, "medium", string(SeverityMedium))
		assert.Equal(t, "high", string(SeverityHigh))
		assert.Equal(t, "critical", string(SeverityCritical))
	})
}

func TestAlertType_String(t *testing.T) {
	t.Run("should return correct string representations", func(t *testing.T) {
		assert.Equal(t, "system", string(AlertTypeSystem))
		assert.Equal(t, "network", string(AlertTypeNetwork))
		assert.Equal(t, "security", string(AlertTypeSecurity))
		assert.Equal(t, "connection", string(AlertTypeConnection))
		assert.Equal(t, "performance", string(AlertTypePerformance))
		assert.Equal(t, "application", string(AlertTypeApplication))
	})
}

func TestAlertStatus_String(t *testing.T) {
	t.Run("should return correct string representations", func(t *testing.T) {
		assert.Equal(t, "active", string(AlertStatusActive))
		assert.Equal(t, "resolved", string(AlertStatusResolved))
		assert.Equal(t, "suppressed", string(AlertStatusSuppressed))
	})
}

func TestAlertManager_UpdateConfig(t *testing.T) {
	am := NewAlertManager()

	t.Run("should update configuration", func(t *testing.T) {
		newConfig := AlertConfig{
			CPUThreshold:    60.0,
			MemoryThreshold: 70.0,
			EnableAlerts:    false,
		}

		am.UpdateConfig(newConfig)

		config := am.GetConfig()
		assert.Equal(t, 60.0, config.CPUThreshold)
		assert.Equal(t, 70.0, config.MemoryThreshold)
		assert.False(t, config.EnableAlerts)
	})
}

func TestAlert_Count(t *testing.T) {
	am := NewAlertManager()

	t.Run("should increment alert count on repeated triggers", func(t *testing.T) {
		now := time.Now()

		// Create alert first time
		am.createOrUpdateAlert("test_alert", AlertTypeSystem, SeverityMedium, "Test Alert", "Test description", now, nil)
		
		alerts := am.GetActiveAlerts()
		assert.Len(t, alerts, 1)
		assert.Equal(t, 1, alerts[0].Count)

		// Trigger same alert again
		am.createOrUpdateAlert("test_alert", AlertTypeSystem, SeverityMedium, "Test Alert", "Test description", now, nil)
		
		alerts = am.GetActiveAlerts()
		assert.Len(t, alerts, 1)
		assert.Equal(t, 2, alerts[0].Count)
	})
}

// Helper function to find an alert by ID in a slice of alerts
func findAlertByID(alerts []Alert, id string) *Alert {
	for _, alert := range alerts {
		if alert.ID == id {
			return &alert
		}
	}
	return nil
}