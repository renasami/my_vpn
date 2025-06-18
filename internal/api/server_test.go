package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"my-vpn/internal/database"
	"my-vpn/internal/network"
	"my-vpn/internal/wireguard"
)

func setupTestServerAPI(t *testing.T) (*ServerAPI, *gin.Engine, func()) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&database.Client{}, &database.ServerConfig{}, &database.ConnectionLog{})
	require.NoError(t, err)

	database := &database.Database{DB: db}

	// Create IP pool
	ipPool, err := network.NewIPPool("10.0.0.0/24")
	require.NoError(t, err)

	// Create WireGuard server
	wgServer := wireguard.NewWireGuardServerWithConfig("/tmp", "wg0")

	// Create server API
	serverAPI := NewServerAPI(database, ipPool, wgServer)

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	serverAPI.RegisterRoutes(router)

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS clients")
		db.Exec("DROP TABLE IF EXISTS server_configs")
		db.Exec("DROP TABLE IF EXISTS connection_logs")
	}

	return serverAPI, router, cleanup
}

func TestServerAPI_GetStatus(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should return server status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/server/status", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerStatusResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, []string{"running", "stopped", "error"}, response.State)
		assert.Equal(t, "wg0", response.Interface)
		assert.GreaterOrEqual(t, response.PeerCount, 0)
	})
}

func TestServerAPI_StartServer(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should attempt to start server", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/server/start", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		// Should return 500 because WireGuard is not actually installed
		// but the API should handle the error gracefully
		assert.Equal(t, http.StatusInternalServerError, resp.Code)

		var response ErrorResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response.Error, "Failed to start server")
	})
}

func TestServerAPI_StopServer(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should attempt to stop server", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/server/stop", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		// In test environment without WireGuard, stop operation may fail
		// but the API should handle it gracefully
		if resp.Code == http.StatusOK {
			var response ServerControlResponse
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "Server stopped successfully", response.Message)
		} else {
			// If WireGuard is not available, expect error response
			assert.Equal(t, http.StatusInternalServerError, resp.Code)
			var response ErrorResponse
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response.Error, "Failed to stop server")
		}
	})
}

func TestServerAPI_RestartServer(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should attempt to restart server", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/server/restart", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		// Should return 500 because WireGuard is not actually installed
		assert.Equal(t, http.StatusInternalServerError, resp.Code)

		var response ErrorResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response.Error, "Failed to restart server")
	})
}

func TestServerAPI_GetConfig(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should return server config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/server/config", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerConfigResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "10.0.0.0/24", response.Network)
		assert.Equal(t, "10.0.0.1", response.ServerIP)
		assert.Equal(t, "wg0", response.Interface)
		assert.Equal(t, 51820, response.ListenPort)
		assert.NotEmpty(t, response.PublicKey)
	})
}

func TestServerAPI_UpdateConfig(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should update server config", func(t *testing.T) {
		updateReq := UpdateServerConfigRequest{
			ListenPort: 51821,
			DNS:        []string{"1.1.1.1", "1.0.0.1"},
		}

		body, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/server/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerConfigResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 51821, response.ListenPort)
		assert.Equal(t, []string{"1.1.1.1", "1.0.0.1"}, response.DNS)
	})

	t.Run("should validate listen port range", func(t *testing.T) {
		updateReq := UpdateServerConfigRequest{
			ListenPort: 70000, // Invalid port
		}

		body, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/server/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestServerAPI_InitializeServer(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should initialize server with default config", func(t *testing.T) {
		initReq := InitializeServerRequest{
			Network:    "192.168.100.0/24",
			ListenPort: 51820,
		}

		body, err := json.Marshal(initReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/server/initialize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerConfigResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "192.168.100.0/24", response.Network)
		assert.Equal(t, "192.168.100.1", response.ServerIP)
		assert.Equal(t, 51820, response.ListenPort)
		assert.NotEmpty(t, response.PublicKey)
		assert.NotEmpty(t, response.PrivateKey)
	})

	t.Run("should fail with invalid network", func(t *testing.T) {
		initReq := InitializeServerRequest{
			Network:    "invalid-network",
			ListenPort: 51820,
		}

		body, err := json.Marshal(initReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/server/initialize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestServerAPI_GetLogs(t *testing.T) {
	_, router, cleanup := setupTestServerAPI(t)
	defer cleanup()

	t.Run("should return empty logs initially", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/server/logs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerLogsResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Logs)
		assert.Equal(t, 0, response.Total)
	})

	t.Run("should respect limit parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/server/logs?limit=50", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ServerLogsResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Logs)
	})
}