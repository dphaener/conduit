package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDefaultConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	config := DefaultConfig(handler)

	if config.Address != ":8080" {
		t.Errorf("Expected address :8080, got %s", config.Address)
	}

	if config.ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout 15s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout 15s, got %v", config.WriteTimeout)
	}

	if config.IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout 60s, got %v", config.IdleTimeout)
	}

	if !config.EnableHTTP2 {
		t.Error("Expected HTTP/2 to be enabled")
	}
}

func TestNewServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	config := DefaultConfig(handler)
	srv, err := New(config)

	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("Server is nil")
	}

	if srv.config.Address != ":8080" {
		t.Errorf("Expected address :8080, got %s", srv.config.Address)
	}
}

func TestNewServerNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestNewServerNilHandler(t *testing.T) {
	config := DefaultConfig(nil)
	_, err := New(config)
	if err == nil {
		t.Error("Expected error for nil handler")
	}
}

func TestServerWithDatabase(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	config := DefaultConfig(handler)
	config.Database = DefaultDatabaseConfig(db)

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server with database: %v", err)
	}

	if srv == nil {
		t.Fatal("Server is nil")
	}

	// Verify database connection is configured
	stats := db.Stats()
	if stats.MaxOpenConnections != 100 {
		t.Errorf("Expected MaxOpenConnections 100, got %d", stats.MaxOpenConnections)
	}
}

func TestServerShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	config := DefaultConfig(handler)
	config.Address = ":0" // Use random port

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestDefaultDatabaseConfig(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	config := DefaultDatabaseConfig(db)

	if config.MaxOpenConns != 100 {
		t.Errorf("Expected MaxOpenConns 100, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns 10, got %d", config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime 1h, got %v", config.ConnMaxLifetime)
	}

	if config.ConnMaxIdleTime != 10*time.Minute {
		t.Errorf("Expected ConnMaxIdleTime 10m, got %v", config.ConnMaxIdleTime)
	}
}

func TestServerServeHTTP(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("OK"))
	})

	config := DefaultConfig(handler)
	srv, _ := New(config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}
