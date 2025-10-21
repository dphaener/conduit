package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

func TestLoggingMiddleware(t *testing.T) {
	var logEntry LogEntry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logEntry = entry
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Add request ID to context
	ctx := webcontext.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Check log entry
	if logEntry.RequestID != "test-request-id" {
		t.Errorf("Expected request ID 'test-request-id', got %s", logEntry.RequestID)
	}
	if logEntry.Method != http.MethodGet {
		t.Errorf("Expected method GET, got %s", logEntry.Method)
	}
	if logEntry.Path != "/test" {
		t.Errorf("Expected path /test, got %s", logEntry.Path)
	}
	if logEntry.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, logEntry.StatusCode)
	}
	if logEntry.BytesWritten != 13 { // "test response"
		t.Errorf("Expected 13 bytes written, got %d", logEntry.BytesWritten)
	}
	if logEntry.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestLoggingWithCustomStatusCode(t *testing.T) {
	var logEntry LogEntry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logEntry = entry
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if logEntry.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, logEntry.StatusCode)
	}
}

func TestLoggingSkipPaths(t *testing.T) {
	logCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logCalled = true
		},
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	// Request to skipped path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if logCalled {
		t.Error("Logger should not be called for skipped path")
	}

	// Request to non-skipped path
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !logCalled {
		t.Error("Logger should be called for non-skipped path")
	}
}

func TestLoggingDefaultLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use default logger (should not panic)
	middleware := Logging()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := webcontext.SetRequestID(req.Context(), "test-id")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoggingDuration(t *testing.T) {
	var logEntry LogEntry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logEntry = entry
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if logEntry.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", logEntry.Duration)
	}
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rw.statusCode)
	}

	// Test multiple WriteHeader calls (should only write once)
	rw.WriteHeader(http.StatusInternalServerError)
	if rw.statusCode != http.StatusCreated {
		t.Error("WriteHeader should only write once")
	}
}

func TestResponseWriterWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test Write
	data := []byte("test data")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	if rw.bytesWritten != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), rw.bytesWritten)
	}

	// Check that WriteHeader was called implicitly
	if !rw.wroteHeader {
		t.Error("Expected wroteHeader to be true after Write")
	}
}

func TestResponseWriterMultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Multiple writes
	rw.Write([]byte("hello "))
	rw.Write([]byte("world"))

	if rw.bytesWritten != 11 {
		t.Errorf("Expected 11 bytes written, got %d", rw.bytesWritten)
	}
}

func TestLoggingUserAgent(t *testing.T) {
	var logEntry LogEntry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logEntry = entry
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if logEntry.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent 'TestAgent/1.0', got %s", logEntry.UserAgent)
	}
}

func TestLoggingRemoteAddr(t *testing.T) {
	var logEntry LogEntry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			logEntry = entry
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if logEntry.RemoteAddr != "192.168.1.1:12345" {
		t.Errorf("Expected RemoteAddr '192.168.1.1:12345', got %s", logEntry.RemoteAddr)
	}
}
