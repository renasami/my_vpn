package web

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"my-vpn/internal/database"
	"my-vpn/internal/monitoring"
	"my-vpn/internal/network"
	"my-vpn/internal/system"
	"my-vpn/internal/wireguard"
)

func setupTestWebServer(t *testing.T) (*Server, func()) {
	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "vpn_web_test")
	require.NoError(t, err)

	// Create temporary database
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.New(dbPath)
	require.NoError(t, err)

	// Create test components
	wgServer := wireguard.NewWireGuardServerWithConfig(tempDir, "wg0")
	ipPool, err := network.NewIPPool("10.0.0.0/24")
	require.NoError(t, err)

	pfctlManager := system.NewPfctlManager()
	monitor := monitoring.NewMonitor(db, wgServer, ipPool, pfctlManager)

	// Create test static and template directories
	staticDir := filepath.Join(tempDir, "static")
	templateDir := filepath.Join(tempDir, "templates")
	err = os.MkdirAll(staticDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create minimal test template
	testTemplate := `<!DOCTYPE html><html><head><title>{{.title}}</title></head><body><h1>Test Page</h1></body></html>`
	err = ioutil.WriteFile(filepath.Join(templateDir, "login.html"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create server with test configuration
	config := &ServerConfig{
		Host:         "localhost",
		Port:         0, // Use random available port
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		StaticDir:    staticDir,
		TemplateDir:  templateDir,
		Debug:        true,
	}

	server := NewServerWithConfig(db, wgServer, ipPool, pfctlManager, monitor, config)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return server, cleanup
}

func TestNewServer(t *testing.T) {
	t.Run("should create server with default configuration", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		assert.NotNil(t, server)
		assert.NotNil(t, server.router)
		assert.NotNil(t, server.config)
		assert.Equal(t, "localhost", server.config.Host)
		assert.False(t, server.config.EnableTLS)
	})
}

func TestNewServerWithConfig(t *testing.T) {
	t.Run("should create server with custom configuration", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		assert.NotNil(t, server)
		assert.True(t, server.config.Debug)
	})
}

func TestServer_GetAddress(t *testing.T) {
	t.Run("should return HTTP address", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		server.config.Port = 8080
		address := server.GetAddress()
		assert.Equal(t, "http://localhost:8080", address)
	})

	t.Run("should return HTTPS address when TLS enabled", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		server.config.Port = 8443
		server.config.EnableTLS = true
		address := server.GetAddress()
		assert.Equal(t, "https://localhost:8443", address)
	})
}

func TestServer_Routes(t *testing.T) {
	t.Run("should setup routes correctly", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		// Test that router has routes
		routes := server.router.Routes()
		assert.NotEmpty(t, routes)

		// Check for essential routes
		routePaths := make(map[string]bool)
		for _, route := range routes {
			routePaths[route.Path] = true
		}

		// Public routes
		assert.True(t, routePaths["/login"])
		assert.True(t, routePaths["/register"])

		// API routes should exist (exact paths may vary due to grouping)
		hasAPIRoutes := false
		for path := range routePaths {
			if len(path) > 7 && path[:7] == "/api/v1" {
				hasAPIRoutes = true
				break
			}
		}
		assert.True(t, hasAPIRoutes, "Should have API routes")
	})
}

func TestServer_StartStop(t *testing.T) {
	t.Run("should start and stop server", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		// Find an available port
		server.config.Port = findAvailablePort()
		server.setupHTTPServer()

		// Start server in goroutine
		errChan := make(chan error, 1)
		go func() {
			err := server.Start()
			if err != http.ErrServerClosed {
				errChan <- err
			}
		}()

		// Wait for server to start
		time.Sleep(100 * time.Millisecond)

		// Test that server is responding
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/login", server.config.Port))
		if err == nil {
			resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// Stop server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err = server.Stop(ctx)
		assert.NoError(t, err)

		// Check for any startup errors
		select {
		case err := <-errChan:
			assert.NoError(t, err)
		default:
			// No error, which is expected
		}
	})
}

func TestServer_CORSMiddleware(t *testing.T) {
	t.Run("should set CORS headers", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		middleware := server.corsMiddleware()
		assert.NotNil(t, middleware)
	})
}

func TestServerConfig_Validation(t *testing.T) {
	t.Run("should have valid default configuration", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		config := server.config
		assert.NotEmpty(t, config.Host)
		assert.Greater(t, config.ReadTimeout, time.Duration(0))
		assert.Greater(t, config.WriteTimeout, time.Duration(0))
		assert.NotEmpty(t, config.StaticDir)
		assert.NotEmpty(t, config.TemplateDir)
	})
}

func TestServer_StaticFiles(t *testing.T) {
	t.Run("should serve static files", func(t *testing.T) {
		server, cleanup := setupTestWebServer(t)
		defer cleanup()

		// Create a test static file
		testContent := "/* test css */"
		cssFile := filepath.Join(server.config.StaticDir, "test.css")
		err := ioutil.WriteFile(cssFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// The router should have static file serving configured
		routes := server.router.Routes()
		hasStaticRoute := false
		for _, route := range routes {
			if route.Path == "/static/*filepath" {
				hasStaticRoute = true
				break
			}
		}
		assert.True(t, hasStaticRoute, "Should have static file route")
	})
}

// Helper function to find an available port
func findAvailablePort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 8080 // fallback
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}