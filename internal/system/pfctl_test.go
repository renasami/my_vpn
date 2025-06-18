package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPfctlManager(t *testing.T) {
	t.Run("should create new pfctl manager with default config", func(t *testing.T) {
		manager := NewPfctlManager()
		
		assert.NotNil(t, manager)
		assert.Equal(t, "/etc/pf.conf", manager.configPath)
		assert.Equal(t, "/tmp/pf_vpn.conf", manager.vpnConfigPath)
	})

	t.Run("should create manager with custom config", func(t *testing.T) {
		configPath := "/tmp/pf.conf"
		vpnConfigPath := "/tmp/vpn.conf"
		
		manager := NewPfctlManagerWithConfig(configPath, vpnConfigPath)
		
		assert.NotNil(t, manager)
		assert.Equal(t, configPath, manager.configPath)
		assert.Equal(t, vpnConfigPath, manager.vpnConfigPath)
	})
}

func TestPfctlManager_GenerateConfig(t *testing.T) {
	manager := NewPfctlManager()
	
	t.Run("should generate VPN pfctl config", func(t *testing.T) {
		config := &VPNConfig{
			Interface:      "wg0",
			VPNNetwork:     "10.0.0.0/24",
			ExternalInterface: "en0",
		}
		
		pfConfig := manager.GenerateConfig(config)
		
		assert.Contains(t, pfConfig, "# WireGuard VPN NAT Rules")
		assert.Contains(t, pfConfig, "nat on en0 from 10.0.0.0/24 to any")
		assert.Contains(t, pfConfig, "pass in on wg0")
		assert.Contains(t, pfConfig, "pass out on en0")
	})

	t.Run("should include custom ports if specified", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
			ListenPort:        51820,
			AllowedPorts:      []int{80, 443, 22},
		}
		
		pfConfig := manager.GenerateConfig(config)
		
		assert.Contains(t, pfConfig, "pass in on en0 proto udp to port 51820")
		assert.Contains(t, pfConfig, "pass out proto tcp to port { 80 443 22 }")
	})
}

func TestPfctlManager_WriteConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pfctl_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	vpnConfigPath := filepath.Join(tempDir, "vpn.conf")
	manager := NewPfctlManagerWithConfig("/etc/pf.conf", vpnConfigPath)
	
	t.Run("should write VPN config file", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "192.168.100.0/24",
			ExternalInterface: "en0",
		}
		
		err := manager.WriteConfig(config)
		require.NoError(t, err)
		
		assert.FileExists(t, vpnConfigPath)
		
		content, err := os.ReadFile(vpnConfigPath)
		require.NoError(t, err)
		
		configStr := string(content)
		assert.Contains(t, configStr, "192.168.100.0/24")
		assert.Contains(t, configStr, "wg0")
		assert.Contains(t, configStr, "en0")
	})

	t.Run("should create directory if not exists", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "new_dir")
		newConfigPath := filepath.Join(newDir, "vpn.conf")
		manager := NewPfctlManagerWithConfig("/etc/pf.conf", newConfigPath)
		
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
		}
		
		err := manager.WriteConfig(config)
		require.NoError(t, err)
		
		assert.DirExists(t, newDir)
		assert.FileExists(t, newConfigPath)
	})
}

func TestPfctlManager_EnableRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "pfctl_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	vpnConfigPath := filepath.Join(tempDir, "vpn.conf")
	manager := NewPfctlManagerWithConfig("/etc/pf.conf", vpnConfigPath)
	
	t.Run("should handle enable rules", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
		}
		
		// Write config first
		err := manager.WriteConfig(config)
		require.NoError(t, err)
		
		// Enable rules (will fail without root privileges)
		err = manager.EnableRules()
		// We expect this to fail in test environment without sudo
		assert.Error(t, err)
		// Error could be from loading rules or enabling pfctl
		assert.True(t, strings.Contains(err.Error(), "failed to load pfctl rules") ||
			strings.Contains(err.Error(), "failed to enable pfctl rules"))
	})
}

func TestPfctlManager_DisableRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	manager := NewPfctlManager()
	
	t.Run("should handle disable rules", func(t *testing.T) {
		// Disable rules (will fail without root privileges)
		err := manager.DisableRules()
		// We expect this to fail in test environment without sudo
		assert.Error(t, err)
	})
}

func TestPfctlManager_IsEnabled(t *testing.T) {
	manager := NewPfctlManager()
	
	t.Run("should check if pfctl is enabled", func(t *testing.T) {
		enabled, err := manager.IsEnabled()
		// Should handle permission errors gracefully
		if err != nil {
			// If error occurs, it should be a meaningful error message
			assert.Contains(t, err.Error(), "pfctl status")
		} else {
			// pfctl is typically disabled by default on macOS
			assert.False(t, enabled)
		}
	})
}

func TestPfctlManager_GetStatus(t *testing.T) {
	manager := NewPfctlManager()
	
	t.Run("should get pfctl status", func(t *testing.T) {
		status, err := manager.GetStatus()
		// Should handle permission errors gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "pfctl status")
		} else {
			assert.NotNil(t, status)
			assert.Contains(t, []string{"enabled", "disabled"}, status.State)
			assert.GreaterOrEqual(t, status.RuleCount, 0)
		}
	})
}

func TestPfctlManager_GetActiveRules(t *testing.T) {
	manager := NewPfctlManager()
	
	t.Run("should get active rules", func(t *testing.T) {
		rules, err := manager.GetActiveRules()
		// Should handle permission errors gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "pfctl rules")
		} else {
			// Should not error even if no rules are active
			assert.NotNil(t, rules)
		}
	})
}

func TestVPNConfig_Validate(t *testing.T) {
	t.Run("should validate valid config", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
			ListenPort:        51820,
		}
		
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("should fail with empty interface", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
		}
		
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interface name is required")
	})

	t.Run("should fail with invalid network", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "invalid-network",
			ExternalInterface: "en0",
		}
		
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid VPN network CIDR")
	})

	t.Run("should fail with invalid port", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
			ListenPort:        70000,
		}
		
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listen port must be between 1 and 65535")
	})

	t.Run("should fail with invalid allowed ports", func(t *testing.T) {
		config := &VPNConfig{
			Interface:         "wg0",
			VPNNetwork:        "10.0.0.0/24",
			ExternalInterface: "en0",
			AllowedPorts:      []int{80, 70000, 443},
		}
		
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid allowed port")
	})
}

func TestPfctlManager_BackupRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pfctl_backup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	backupPath := filepath.Join(tempDir, "pf_backup.conf")
	manager := NewPfctlManager()
	
	t.Run("should create backup", func(t *testing.T) {
		err := manager.CreateBackup(backupPath)
		require.NoError(t, err)
		
		assert.FileExists(t, backupPath)
	})

	t.Run("should restore from backup", func(t *testing.T) {
		// Create a dummy backup file
		backupContent := "# Test backup\npass all\n"
		err := os.WriteFile(backupPath, []byte(backupContent), 0644)
		require.NoError(t, err)
		
		err = manager.RestoreFromBackup(backupPath)
		// Will fail without root privileges, but should handle gracefully
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to restore pfctl configuration")
	})
}