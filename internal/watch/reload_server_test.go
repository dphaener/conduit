package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestReloadServer_NewReloadServer(t *testing.T) {
	rs := NewReloadServer()
	if rs == nil {
		t.Fatal("Expected reload server to be created")
	}

	if rs.connections == nil {
		t.Error("Expected connections map to be initialized")
	}

	if rs.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}

	defer rs.Close()
}

func TestReloadServer_HandleWebSocket(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect as client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	if rs.ConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", rs.ConnectionCount())
	}
}

func TestReloadServer_NotifyBuilding(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for connection
	time.Sleep(50 * time.Millisecond)

	// Send notification
	files := []string{"test.cdt", "post.cdt"}
	rs.NotifyBuilding(files)

	// Read message
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var msg ReloadMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if msg.Type != "building" {
		t.Errorf("Expected type 'building', got %q", msg.Type)
	}

	if len(msg.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(msg.Files))
	}
}

func TestReloadServer_NotifySuccess(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	duration := 150 * time.Millisecond
	rs.NotifySuccess(duration)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var msg ReloadMessage
	json.Unmarshal(message, &msg)

	if msg.Type != "success" {
		t.Errorf("Expected type 'success', got %q", msg.Type)
	}

	if msg.Duration != 150.0 {
		t.Errorf("Expected duration 150ms, got %.0f", msg.Duration)
	}
}

func TestReloadServer_NotifyReload(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	rs.NotifyReload("ui")

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var msg ReloadMessage
	json.Unmarshal(message, &msg)

	if msg.Type != "reload" {
		t.Errorf("Expected type 'reload', got %q", msg.Type)
	}

	if msg.Scope != "ui" {
		t.Errorf("Expected scope 'ui', got %q", msg.Scope)
	}
}

func TestReloadServer_NotifyError(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	errorInfo := &ErrorInfo{
		Message: "Syntax error",
		File:    "test.cdt",
		Line:    10,
		Column:  5,
		Code:    "PARSE001",
		Phase:   "parser",
	}
	rs.NotifyError(errorInfo)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var msg ReloadMessage
	json.Unmarshal(message, &msg)

	if msg.Type != "error" {
		t.Errorf("Expected type 'error', got %q", msg.Type)
	}

	if msg.Error == nil {
		t.Fatal("Expected error info to be set")
	}

	if msg.Error.Message != "Syntax error" {
		t.Errorf("Expected message 'Syntax error', got %q", msg.Error.Message)
	}

	if msg.Error.Line != 10 {
		t.Errorf("Expected line 10, got %d", msg.Error.Line)
	}
}

func TestReloadServer_MultipleConnections(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect multiple clients
	conns := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	if rs.ConnectionCount() != 3 {
		t.Errorf("Expected 3 connections, got %d", rs.ConnectionCount())
	}

	// Broadcast message
	rs.NotifyReload("backend")

	// All clients should receive the message
	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Client %d failed to read message: %v", i, err)
			continue
		}

		var msg ReloadMessage
		json.Unmarshal(message, &msg)

		if msg.Type != "reload" {
			t.Errorf("Client %d: Expected type 'reload', got %q", i, msg.Type)
		}
	}
}

func TestReloadServer_ConnectionCount(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	if rs.ConnectionCount() != 0 {
		t.Errorf("Expected 0 connections initially, got %d", rs.ConnectionCount())
	}

	server := httptest.NewServer(http.HandlerFunc(rs.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	time.Sleep(50 * time.Millisecond)

	if rs.ConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", rs.ConnectionCount())
	}

	conn.Close()
	time.Sleep(100 * time.Millisecond)

	if rs.ConnectionCount() != 0 {
		t.Errorf("Expected 0 connections after close, got %d", rs.ConnectionCount())
	}
}

func TestReloadServer_OriginCheck(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{"no origin", "", true},
		{"localhost http", "http://localhost:3000", true},
		{"localhost https", "https://localhost:3000", true},
		{"127.0.0.1 http", "http://127.0.0.1:3000", true},
		{"127.0.0.1 https", "https://127.0.0.1:3000", true},
		{"external origin", "http://evil.com", false},
		{"external https", "https://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: http.Header{},
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := rs.upgrader.CheckOrigin(req)
			if result != tt.expected {
				t.Errorf("Origin %q: expected %v, got %v", tt.origin, tt.expected, result)
			}
		})
	}
}

func TestReloadServer_CloseStopsGoroutine(t *testing.T) {
	rs := NewReloadServer()

	// Close should signal the done channel
	rs.Close()

	// Give time for goroutine to exit
	time.Sleep(100 * time.Millisecond)

	// Verify connections are cleared
	if rs.ConnectionCount() != 0 {
		t.Errorf("Expected 0 connections after close, got %d", rs.ConnectionCount())
	}
}

func BenchmarkReloadServer_NotifyReload(b *testing.B) {
	rs := NewReloadServer()
	defer rs.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.NotifyReload("backend")
	}
}
