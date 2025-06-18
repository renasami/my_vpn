// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// AlertManager manages alerts and notifications for the VPN server monitoring system.
// It evaluates alert conditions, maintains alert states, and provides notification
// capabilities for various alert types and severity levels.
type AlertManager struct {
	alerts       map[string]*Alert // Active alerts indexed by alert ID
	config       AlertConfig       // Alert configuration and thresholds
	mutex        sync.RWMutex      // Mutex for thread-safe operations
	lastEvalTime time.Time         // Last time alerts were evaluated
}

// AlertConfig represents configuration for alert thresholds and notification settings.
type AlertConfig struct {
	CPUThreshold       float64       `json:"cpu_threshold"`        // CPU usage threshold (percentage)
	MemoryThreshold    float64       `json:"memory_threshold"`     // Memory usage threshold (percentage)
	DiskThreshold      float64       `json:"disk_threshold"`       // Disk usage threshold (percentage)
	ConnectionThreshold int          `json:"connection_threshold"` // Max number of concurrent connections
	ResponseTimeThreshold time.Duration `json:"response_time_threshold"` // Max acceptable response time
	ErrorRateThreshold float64       `json:"error_rate_threshold"` // Max acceptable error rate (percentage)
	EnableAlerts       bool          `json:"enable_alerts"`        // Whether alerts are enabled
	AlertCooldown      time.Duration `json:"alert_cooldown"`       // Minimum time between identical alerts
	NotificationChannels []string    `json:"notification_channels"` // Enabled notification channels
}

// Alert represents an active alert in the system.
type Alert struct {
	ID          string    `json:"id"`          // Unique identifier for the alert
	Type        AlertType `json:"type"`        // Type/category of the alert
	Severity    Severity  `json:"severity"`    // Severity level of the alert
	Title       string    `json:"title"`       // Human-readable alert title
	Description string    `json:"description"` // Detailed alert description
	CreatedAt   time.Time `json:"created_at"`  // When the alert was first triggered
	UpdatedAt   time.Time `json:"updated_at"`  // When the alert was last updated
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"` // When the alert was resolved (if resolved)
	Status      AlertStatus `json:"status"`    // Current status of the alert
	Metadata    map[string]interface{} `json:"metadata"` // Additional alert metadata
	Count       int       `json:"count"`       // Number of times this alert has been triggered
}

// AlertType represents the type/category of an alert.
type AlertType string

const (
	AlertTypeSystem      AlertType = "system"      // System resource alerts
	AlertTypeNetwork     AlertType = "network"     // Network-related alerts
	AlertTypeSecurity    AlertType = "security"    // Security-related alerts
	AlertTypeConnection  AlertType = "connection"  // Client connection alerts
	AlertTypePerformance AlertType = "performance" // Performance-related alerts
	AlertTypeApplication AlertType = "application" // Application-specific alerts
)

// Severity represents the severity level of an alert.
type Severity string

const (
	SeverityLow      Severity = "low"      // Low severity - informational
	SeverityMedium   Severity = "medium"   // Medium severity - requires attention
	SeverityHigh     Severity = "high"     // High severity - requires immediate attention
	SeverityCritical Severity = "critical" // Critical severity - system at risk
)

// AlertStatus represents the current status of an alert.
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"    // Alert is currently active
	AlertStatusResolved  AlertStatus = "resolved"  // Alert has been resolved
	AlertStatusSuppressed AlertStatus = "suppressed" // Alert is temporarily suppressed
)

// NewAlertManager creates a new alert manager with default configuration.
// It initializes the alert storage and sets up default alert thresholds
// appropriate for most VPN server deployments.
// Returns a pointer to the newly created AlertManager.
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts: make(map[string]*Alert),
		config: AlertConfig{
			CPUThreshold:          80.0,
			MemoryThreshold:       85.0,
			DiskThreshold:         90.0,
			ConnectionThreshold:   1000,
			ResponseTimeThreshold: 5 * time.Second,
			ErrorRateThreshold:    5.0,
			EnableAlerts:          true,
			AlertCooldown:         5 * time.Minute,
			NotificationChannels:  []string{"log"},
		},
		lastEvalTime: time.Now(),
	}
}

// NewAlertManagerWithConfig creates a new alert manager with custom configuration.
// This allows fine-tuning of alert thresholds and notification settings
// for specific deployment requirements and operational needs.
// Returns a pointer to the newly created AlertManager.
func NewAlertManagerWithConfig(config AlertConfig) *AlertManager {
	manager := NewAlertManager()
	manager.config = config
	return manager
}

// EvaluateMetrics evaluates the provided metrics against alert thresholds.
// It checks all configured thresholds and creates or updates alerts as needed.
// This method should be called periodically with current system metrics.
func (am *AlertManager) EvaluateMetrics(metrics *ServerMetrics) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if !am.config.EnableAlerts {
		return
	}

	now := time.Now()
	am.lastEvalTime = now

	// Evaluate system resource alerts
	am.evaluateSystemAlerts(metrics.SystemStats, now)
	
	// Evaluate network alerts
	am.evaluateNetworkAlerts(metrics.NetworkStats, now)
	
	// Evaluate security alerts
	am.evaluateSecurityAlerts(metrics.SecurityStats, now)
	
	// Evaluate connection alerts
	am.evaluateConnectionAlerts(metrics.ConnectionStats, now)
	
	// Evaluate performance alerts
	am.evaluatePerformanceAlerts(metrics.Performance, now)

	// Clean up resolved alerts
	am.cleanupResolvedAlerts(now)
}

// GetActiveAlerts returns all currently active alerts.
// This provides a thread-safe way to retrieve active alerts for display
// in dashboards, APIs, or notification systems.
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var activeAlerts []Alert
	for _, alert := range am.alerts {
		if alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, *alert)
		}
	}

	return activeAlerts
}

// GetAllAlerts returns all alerts (active and resolved) within the specified time range.
// This is useful for historical analysis and alert reporting.
func (am *AlertManager) GetAllAlerts(since time.Time) []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var alerts []Alert
	for _, alert := range am.alerts {
		if alert.CreatedAt.After(since) {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// ResolveAlert manually resolves an active alert by ID.
// This allows operators to acknowledge and resolve alerts that may require
// manual intervention or have been addressed outside the monitoring system.
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return &AlertError{Message: "alert not found", AlertID: alertID}
	}

	if alert.Status == AlertStatusResolved {
		return &AlertError{Message: "alert already resolved", AlertID: alertID}
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	return nil
}

// SuppressAlert temporarily suppresses an alert to prevent notifications.
// This is useful for maintenance periods or when alerts are expected
// and should not trigger notifications.
func (am *AlertManager) SuppressAlert(alertID string, duration time.Duration) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return &AlertError{Message: "alert not found", AlertID: alertID}
	}

	alert.Status = AlertStatusSuppressed
	alert.UpdatedAt = time.Now()
	
	// Set metadata for suppression duration
	if alert.Metadata == nil {
		alert.Metadata = make(map[string]interface{})
	}
	alert.Metadata["suppressed_until"] = time.Now().Add(duration)

	return nil
}

// evaluateSystemAlerts checks system resource metrics against thresholds.
func (am *AlertManager) evaluateSystemAlerts(stats SystemStats, now time.Time) {
	// CPU usage alert
	if stats.CPUUsage > am.config.CPUThreshold {
		am.createOrUpdateAlert("system_cpu_high", AlertTypeSystem, SeverityHigh,
			"High CPU Usage",
			fmt.Sprintf("CPU usage is %.1f%%, exceeding threshold of %.1f%%", stats.CPUUsage, am.config.CPUThreshold),
			now, map[string]interface{}{
				"cpu_usage": stats.CPUUsage,
				"threshold": am.config.CPUThreshold,
			})
	} else {
		am.resolveAlert("system_cpu_high", now)
	}

	// Memory usage alert
	if stats.MemoryUsage > am.config.MemoryThreshold {
		am.createOrUpdateAlert("system_memory_high", AlertTypeSystem, SeverityHigh,
			"High Memory Usage",
			fmt.Sprintf("Memory usage is %.1f%%, exceeding threshold of %.1f%%", stats.MemoryUsage, am.config.MemoryThreshold),
			now, map[string]interface{}{
				"memory_usage": stats.MemoryUsage,
				"threshold":    am.config.MemoryThreshold,
			})
	} else {
		am.resolveAlert("system_memory_high", now)
	}

	// Disk usage alert
	if stats.DiskUsage > am.config.DiskThreshold {
		am.createOrUpdateAlert("system_disk_high", AlertTypeSystem, SeverityCritical,
			"High Disk Usage",
			fmt.Sprintf("Disk usage is %.1f%%, exceeding threshold of %.1f%%", stats.DiskUsage, am.config.DiskThreshold),
			now, map[string]interface{}{
				"disk_usage": stats.DiskUsage,
				"threshold":  am.config.DiskThreshold,
			})
	} else {
		am.resolveAlert("system_disk_high", now)
	}
}

// evaluateNetworkAlerts checks network metrics against thresholds.
func (am *AlertManager) evaluateNetworkAlerts(stats NetworkStats, now time.Time) {
	// IP pool utilization alert
	if stats.IPPoolUtilization > 90.0 {
		severity := SeverityMedium
		if stats.IPPoolUtilization > 95.0 {
			severity = SeverityHigh
		}
		
		am.createOrUpdateAlert("network_ip_pool_high", AlertTypeNetwork, severity,
			"High IP Pool Utilization",
			fmt.Sprintf("IP pool utilization is %.1f%%, nearing capacity", stats.IPPoolUtilization),
			now, map[string]interface{}{
				"utilization": stats.IPPoolUtilization,
			})
	} else {
		am.resolveAlert("network_ip_pool_high", now)
	}
}

// evaluateSecurityAlerts checks security metrics against thresholds.
func (am *AlertManager) evaluateSecurityAlerts(stats SecurityStats, now time.Time) {
	// Firewall disabled alert
	if !stats.FirewallEnabled {
		am.createOrUpdateAlert("security_firewall_disabled", AlertTypeSecurity, SeverityCritical,
			"Firewall Disabled",
			"The pfctl firewall is disabled, leaving the server vulnerable",
			now, map[string]interface{}{
				"firewall_enabled": stats.FirewallEnabled,
			})
	} else {
		am.resolveAlert("security_firewall_disabled", now)
	}

	// High failed login attempts
	if stats.FailedLogins > 10 {
		am.createOrUpdateAlert("security_failed_logins", AlertTypeSecurity, SeverityMedium,
			"High Failed Login Attempts",
			fmt.Sprintf("Detected %d failed login attempts", stats.FailedLogins),
			now, map[string]interface{}{
				"failed_logins": stats.FailedLogins,
			})
	} else {
		am.resolveAlert("security_failed_logins", now)
	}
}

// evaluateConnectionAlerts checks connection metrics against thresholds.
func (am *AlertManager) evaluateConnectionAlerts(stats ConnectionStats, now time.Time) {
	// High connection count alert
	if stats.ActiveClients > am.config.ConnectionThreshold {
		am.createOrUpdateAlert("connection_high_count", AlertTypeConnection, SeverityMedium,
			"High Active Connection Count",
			fmt.Sprintf("Active connections (%d) exceed threshold (%d)", stats.ActiveClients, am.config.ConnectionThreshold),
			now, map[string]interface{}{
				"active_clients": stats.ActiveClients,
				"threshold":      am.config.ConnectionThreshold,
			})
	} else {
		am.resolveAlert("connection_high_count", now)
	}
}

// evaluatePerformanceAlerts checks performance metrics against thresholds.
func (am *AlertManager) evaluatePerformanceAlerts(stats PerformanceMetrics, now time.Time) {
	// High response time alert
	if stats.ResponseTime > am.config.ResponseTimeThreshold {
		am.createOrUpdateAlert("performance_response_time", AlertTypePerformance, SeverityMedium,
			"High Response Time",
			fmt.Sprintf("Response time (%v) exceeds threshold (%v)", stats.ResponseTime, am.config.ResponseTimeThreshold),
			now, map[string]interface{}{
				"response_time": stats.ResponseTime,
				"threshold":     am.config.ResponseTimeThreshold,
			})
	} else {
		am.resolveAlert("performance_response_time", now)
	}

	// High error rate alert
	if stats.ErrorRate > am.config.ErrorRateThreshold {
		am.createOrUpdateAlert("performance_error_rate", AlertTypePerformance, SeverityHigh,
			"High Error Rate",
			fmt.Sprintf("Error rate (%.1f%%) exceeds threshold (%.1f%%)", stats.ErrorRate, am.config.ErrorRateThreshold),
			now, map[string]interface{}{
				"error_rate": stats.ErrorRate,
				"threshold":  am.config.ErrorRateThreshold,
			})
	} else {
		am.resolveAlert("performance_error_rate", now)
	}
}

// createOrUpdateAlert creates a new alert or updates an existing one.
func (am *AlertManager) createOrUpdateAlert(id string, alertType AlertType, severity Severity, title, description string, now time.Time, metadata map[string]interface{}) {
	alert, exists := am.alerts[id]
	
	if exists {
		// Update existing alert
		alert.UpdatedAt = now
		alert.Count++
		if alert.Metadata == nil {
			alert.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			alert.Metadata[k] = v
		}
	} else {
		// Create new alert
		am.alerts[id] = &Alert{
			ID:          id,
			Type:        alertType,
			Severity:    severity,
			Title:       title,
			Description: description,
			CreatedAt:   now,
			UpdatedAt:   now,
			Status:      AlertStatusActive,
			Metadata:    metadata,
			Count:       1,
		}
	}
}

// resolveAlert resolves an alert if it exists and is active.
func (am *AlertManager) resolveAlert(id string, now time.Time) {
	alert, exists := am.alerts[id]
	if exists && alert.Status == AlertStatusActive {
		alert.Status = AlertStatusResolved
		alert.ResolvedAt = &now
		alert.UpdatedAt = now
	}
}

// cleanupResolvedAlerts removes old resolved alerts to prevent memory leaks.
func (am *AlertManager) cleanupResolvedAlerts(now time.Time) {
	for id, alert := range am.alerts {
		if alert.Status == AlertStatusResolved && alert.ResolvedAt != nil {
			// Remove alerts resolved more than 24 hours ago
			if now.Sub(*alert.ResolvedAt) > 24*time.Hour {
				delete(am.alerts, id)
			}
		}
	}
}

// AlertError represents an error related to alert operations.
type AlertError struct {
	Message string
	AlertID string
}

// Error implements the error interface for AlertError.
func (e *AlertError) Error() string {
	return fmt.Sprintf("alert error [%s]: %s", e.AlertID, e.Message)
}

// GetConfig returns the current alert configuration.
// This provides read-only access to the alert manager configuration.
func (am *AlertManager) GetConfig() AlertConfig {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	return am.config
}

// UpdateConfig updates the alert manager configuration.
// This allows dynamic reconfiguration of alert thresholds and settings
// without restarting the monitoring system.
func (am *AlertManager) UpdateConfig(config AlertConfig) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.config = config
}