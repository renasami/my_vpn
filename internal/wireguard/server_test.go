package wireguard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWireGuardServer(t *testing.T) {
	t.Run("should create new server with default config", func(t *testing.T) {
		server := NewWireGuardServer()
		
		assert.NotNil(t, server)
		assert.Equal(t, "/usr/local/etc/wireguard", server.configDir)
		assert.Equal(t, "wg0", server.interfaceName)
	})

	t.Run("should create server with custom config", func(t *testing.T) {
		configDir := "/tmp/wireguard"
		interfaceName := "wg1"
		
		server := NewWireGuardServerWithConfig(configDir, interfaceName)
		
		assert.NotNil(t, server)
		assert.Equal(t, configDir, server.configDir)
		assert.Equal(t, interfaceName, server.interfaceName)
	})
}

func TestWireGuardServer_WriteConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg0")
	
	t.Run("should write config file successfully", func(t *testing.T) {
		config := &ServerConfig{
			PrivateKey: "test-private-key",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			Interface:  "wg0",
		}
		
		err := server.WriteConfig(config)
		require.NoError(t, err)
		
		configPath := filepath.Join(tempDir, "wg0.conf")
		assert.FileExists(t, configPath)
		
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		
		configStr := string(content)
		assert.Contains(t, configStr, "PrivateKey = test-private-key")
		assert.Contains(t, configStr, "Address = 10.0.0.1/24")
		assert.Contains(t, configStr, "ListenPort = 51820")
	})

	t.Run("should create config directory if not exists", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "new_dir")
		server := NewWireGuardServerWithConfig(nonExistentDir, "wg0")
		
		config := &ServerConfig{
			PrivateKey: "test-private-key",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			Interface:  "wg0",
		}
		
		err := server.WriteConfig(config)
		require.NoError(t, err)
		
		assert.DirExists(t, nonExistentDir)
	})
}

func TestWireGuardServer_Start(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg_test")
	
	t.Run("should fail to start without config", func(t *testing.T) {
		err := server.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config file not found")
	})

	t.Run("should fail to start with invalid config", func(t *testing.T) {
		// Write invalid config
		configPath := filepath.Join(tempDir, "wg_test.conf")
		err := os.WriteFile(configPath, []byte("invalid config"), 0600)
		require.NoError(t, err)
		
		err = server.Start()
		assert.Error(t, err)
	})
}

func TestWireGuardServer_Stop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg_test")
	
	t.Run("should handle stop when not running", func(t *testing.T) {
		err := server.Stop()
		// Should not error when stopping non-running interface
		assert.NoError(t, err)
	})
}

func TestWireGuardServer_Status(t *testing.T) {
	server := NewWireGuardServer()
	
	t.Run("should return server status", func(t *testing.T) {
		status, err := server.Status()
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Contains(t, []string{"running", "stopped", "error"}, status.State)
	})
}

func TestWireGuardServer_Restart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg_test")
	
	t.Run("should handle restart", func(t *testing.T) {
		err := server.Restart()
		// Should handle restart gracefully even if not running
		assert.NoError(t, err)
	})
}

func TestWireGuardServer_AddPeer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg0")
	
	t.Run("should add peer to config", func(t *testing.T) {
		// First create a basic config
		baseConfig := &ServerConfig{
			PrivateKey: "test-private-key",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			Interface:  "wg0",
		}
		err := server.WriteConfig(baseConfig)
		require.NoError(t, err)
		
		peer := &Peer{
			PublicKey:  "peer-public-key",
			AllowedIPs: []string{"10.0.0.2/32"},
		}
		
		err = server.AddPeer(peer)
		require.NoError(t, err)
		
		configPath := filepath.Join(tempDir, "wg0.conf")
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		
		configStr := string(content)
		assert.Contains(t, configStr, "[Peer]")
		assert.Contains(t, configStr, "PublicKey = peer-public-key")
		assert.Contains(t, configStr, "AllowedIPs = 10.0.0.2/32")
	})
}

func TestWireGuardServer_RemovePeer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wireguard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server := NewWireGuardServerWithConfig(tempDir, "wg0")
	
	t.Run("should remove peer from config", func(t *testing.T) {
		// Create config with peer
		configContent := `[Interface]
PrivateKey = test-private-key
Address = 10.0.0.1/24
ListenPort = 51820

[Peer]
PublicKey = peer-to-remove
AllowedIPs = 10.0.0.2/32

[Peer]
PublicKey = peer-to-keep
AllowedIPs = 10.0.0.3/32
`
		configPath := filepath.Join(tempDir, "wg0.conf")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)
		
		err = server.RemovePeer("peer-to-remove")
		require.NoError(t, err)
		
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		
		configStr := string(content)
		assert.NotContains(t, configStr, "peer-to-remove")
		assert.Contains(t, configStr, "peer-to-keep")
	})
}