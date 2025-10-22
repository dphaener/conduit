package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 1024, config.ReadBufferSize)
	assert.Equal(t, 1024, config.WriteBufferSize)
	assert.NotNil(t, config.CheckOrigin)
	assert.NotNil(t, config.TokenExtractor)
	assert.False(t, config.EnableCompression)
}

func TestNewUpgrader(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	config := DefaultConfig()

	upgrader := NewUpgrader(config, hub)

	assert.NotNil(t, upgrader)
	assert.Equal(t, config, upgrader.config)
	assert.Equal(t, hub, upgrader.hub)
	assert.NotNil(t, upgrader.upgrader)
}

func TestUpgraderWithNilConfig(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	upgrader := NewUpgrader(nil, hub)

	assert.NotNil(t, upgrader)
	assert.NotNil(t, upgrader.config)
}

func TestNewServer(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	server := NewServer(ctx, config)

	assert.NotNil(t, server)
	assert.NotNil(t, server.Hub)
	assert.NotNil(t, server.Upgrader)
	assert.Equal(t, config, server.Config)
}

func TestServerStartShutdown(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)

	server.Start()

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, server.Hub.ClientCount())

	server.Shutdown()

	time.Sleep(50 * time.Millisecond)
}

func TestUpgraderServeHTTP(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	// Create test HTTP server
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Give server time to register client
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, server.Hub.ClientCount())
}

func TestUpgraderWithAuthentication(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)

	// Set auth handler
	server.Hub.SetAuthHandler(func(ctx context.Context, token string) (string, error) {
		if token == "valid-token" {
			return "user-123", nil
		}
		return "", assert.AnError
	})

	server.Start()
	defer server.Shutdown()

	// Create test HTTP server
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "?token=valid-token"

	// Connect with valid token
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, server.Hub.ClientCount())

	// Check client has user ID
	clients := server.Hub.GetClients()
	assert.Equal(t, 1, len(clients))
	assert.Equal(t, "user-123", clients[0].UserID)
}

func TestUpgraderInvalidAuthToken(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)

	// Set auth handler that rejects invalid tokens
	server.Hub.SetAuthHandler(func(ctx context.Context, token string) (string, error) {
		if token == "valid-token" {
			return "user-123", nil
		}
		return "", assert.AnError
	})

	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "?token=invalid-token"

	// Connect with invalid token - should fail
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	if err == nil {
		ws.Close()
		// Connection was established but should be closed immediately
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, 0, server.Hub.ClientCount())
	} else {
		// Connection was rejected
		assert.Error(t, err)
		if resp != nil {
			assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode)
		}
	}
}

func TestTokenExtractorFromQueryParam(t *testing.T) {
	config := DefaultConfig()

	req := httptest.NewRequest("GET", "/?token=test-token", nil)

	token := config.TokenExtractor(req)

	assert.Equal(t, "test-token", token)
}

func TestTokenExtractorFromHeader(t *testing.T) {
	config := DefaultConfig()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	token := config.TokenExtractor(req)

	assert.Equal(t, "Bearer test-token", token)
}

func TestTokenExtractorPriority(t *testing.T) {
	config := DefaultConfig()

	req := httptest.NewRequest("GET", "/?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")

	token := config.TokenExtractor(req)

	// Query parameter should take priority
	assert.Equal(t, "query-token", token)
}

func TestCheckOriginDefault(t *testing.T) {
	config := DefaultConfig()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")

	// Default should allow all origins
	allowed := config.CheckOrigin(req)

	assert.True(t, allowed)
}

func TestServerHandlerRegistration(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)

	handler := server.Handler()

	assert.NotNil(t, handler)
}

func TestMultipleClientsConnection(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients
	clients := make([]*websocket.Conn, 5)
	for i := 0; i < 5; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = ws
		defer ws.Close()
	}

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 5, server.Hub.ClientCount())
}

func TestClientDisconnection(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx, nil)
	server.Start()
	defer server.Shutdown()

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, server.Hub.ClientCount())

	// Close connection
	ws.Close()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, server.Hub.ClientCount())
}
