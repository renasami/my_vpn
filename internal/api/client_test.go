package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func setupTestAPI(t *testing.T) (*ClientAPI, *gin.Engine, func()) {
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

	// Create client API
	clientAPI := NewClientAPI(database, ipPool, wgServer)

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	clientAPI.RegisterRoutes(router)

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS clients")
		db.Exec("DROP TABLE IF EXISTS server_configs")
		db.Exec("DROP TABLE IF EXISTS connection_logs")
	}

	return clientAPI, router, cleanup
}

func TestClientAPI_CreateClient(t *testing.T) {
	clientAPI, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should create client successfully", func(t *testing.T) {
		createReq := CreateClientRequest{
			Name: "test-client",
		}

		body, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var response CreateClientResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotZero(t, response.ID)
		assert.Equal(t, "test-client", response.Name)
		assert.NotEmpty(t, response.PublicKey)
		assert.NotEmpty(t, response.IPAddress)
		assert.Equal(t, true, response.Enabled)
	})

	t.Run("should fail with empty name", func(t *testing.T) {
		createReq := CreateClientRequest{
			Name: "",
		}

		body, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail when IP pool is exhausted", func(t *testing.T) {
		// Create a small IP pool and exhaust it
		smallPool, err := network.NewIPPool("10.1.0.0/29") // Only 6 hosts available (8 total - network - broadcast = 6, minus server = 5 client IPs)
		require.NoError(t, err)

		db := clientAPI.db.DB
		wgServer := clientAPI.wgServer
		smallAPI := NewClientAPI(&database.Database{DB: db}, smallPool, wgServer)

		router := gin.New()
		gin.SetMode(gin.TestMode)
		smallAPI.RegisterRoutes(router)

		// Allocate all available IPs (5 total in /29)
		for i := 1; i <= 5; i++ {
			createReq := CreateClientRequest{Name: fmt.Sprintf("client%d", i)}
			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusCreated, resp.Code)
		}

		// Try to allocate one more IP (should fail as pool is exhausted)
		createReq := CreateClientRequest{Name: "client-extra"}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestClientAPI_GetClients(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should return empty list initially", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response GetClientsResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Clients)
		assert.Equal(t, 0, response.Total)
	})

	t.Run("should return created clients", func(t *testing.T) {
		// Create two clients
		for i := 1; i <= 2; i++ {
			createReq := CreateClientRequest{Name: fmt.Sprintf("client-%d", i)}
			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusCreated, resp.Code)
		}

		// Get all clients
		req := httptest.NewRequest("GET", "/api/clients", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response GetClientsResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Clients, 2)
		assert.Equal(t, 2, response.Total)
	})
}

func TestClientAPI_GetClient(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should return client by ID", func(t *testing.T) {
		// Create a client first
		createReq := CreateClientRequest{Name: "test-client"}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse CreateClientResponse
		err := json.Unmarshal(resp.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		// Get the client
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d", createResponse.ID), nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, createResponse.ID, response.ID)
		assert.Equal(t, "test-client", response.Name)
	})

	t.Run("should return 404 for non-existent client", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients/999", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients/invalid", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestClientAPI_UpdateClient(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should update client successfully", func(t *testing.T) {
		// Create a client first
		createReq := CreateClientRequest{Name: "original-name"}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse CreateClientResponse
		err := json.Unmarshal(resp.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		// Update the client
		enabled := false
		updateReq := UpdateClientRequest{
			Name:    "updated-name",
			Enabled: &enabled,
		}
		body, _ = json.Marshal(updateReq)
		req = httptest.NewRequest("PUT", fmt.Sprintf("/api/clients/%d", createResponse.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "updated-name", response.Name)
		assert.Equal(t, false, response.Enabled)
	})

	t.Run("should return 404 for non-existent client", func(t *testing.T) {
		updateReq := UpdateClientRequest{Name: "test"}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest("PUT", "/api/clients/999", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestClientAPI_DeleteClient(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should delete client successfully", func(t *testing.T) {
		// Create a client first
		createReq := CreateClientRequest{Name: "to-be-deleted"}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse CreateClientResponse
		err := json.Unmarshal(resp.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		// Delete the client
		req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/clients/%d", createResponse.ID), nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Verify client is deleted
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d", createResponse.ID), nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should return 404 for non-existent client", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/clients/999", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestClientAPI_GetClientConfig(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("should return client config", func(t *testing.T) {
		// Create a client first
		createReq := CreateClientRequest{Name: "config-client"}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse CreateClientResponse
		err := json.Unmarshal(resp.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		// Get client config
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/config", createResponse.ID), nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientConfigResponse
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Config)
		assert.Contains(t, response.Config, "[Interface]")
		assert.Contains(t, response.Config, "[Peer]")
		assert.Contains(t, response.Config, "PrivateKey")
		assert.Contains(t, response.Config, "Address")
	})

	t.Run("should return 404 for non-existent client", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients/999/config", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestClientAPI_GetClientQRCode(t *testing.T) {
	_, router, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create a client first for testing
	createReq := CreateClientRequest{Name: "qr-client"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/clients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Code)

	var createResponse CreateClientResponse
	err := json.Unmarshal(resp.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	t.Run("should return base64 QR code by default", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientQRCodeResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.QRCode)
		assert.Equal(t, "base64", response.Format)
		assert.True(t, strings.HasPrefix(response.QRCode, "data:image/png;base64,"))
	})

	t.Run("should return base64 QR code when format=base64", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=base64", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientQRCodeResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.QRCode)
		assert.Equal(t, "base64", response.Format)
		assert.True(t, strings.HasPrefix(response.QRCode, "data:image/png;base64,"))
	})

	t.Run("should return terminal QR code when format=terminal", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=terminal", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientQRCodeResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.QRCode)
		assert.Equal(t, "terminal", response.Format)
		assert.Greater(t, len(response.QRCode), 50) // Terminal QR should be reasonably sized
	})

	t.Run("should return PNG QR code when format=png", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=png", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "image/png", resp.Header().Get("Content-Type"))
		assert.Contains(t, resp.Header().Get("Content-Disposition"), "client-")
		assert.Contains(t, resp.Header().Get("Content-Disposition"), "-config.png")

		// Check PNG header
		pngData := resp.Body.Bytes()
		assert.Greater(t, len(pngData), 100)
		assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, pngData[:4]) // PNG magic number
	})

	t.Run("should handle custom size parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=base64&size=512", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientQRCodeResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.QRCode)
		assert.Equal(t, "base64", response.Format)
	})

	t.Run("should use default size for invalid size parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=base64&size=invalid", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response ClientQRCodeResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.QRCode)
		assert.Equal(t, "base64", response.Format)
	})

	t.Run("should reject unsupported format", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/clients/%d/qrcode?format=unsupported", createResponse.ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var response ErrorResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response.Error, "Unsupported format")
	})

	t.Run("should return 404 for non-existent client", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients/999/qrcode", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should return 400 for invalid client ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/clients/invalid/qrcode", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}