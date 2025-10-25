package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	// Fast handler that completes before timeout
	fastHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	middleware := Timeout(1 * time.Second)
	wrapped := middleware(fastHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestTimeoutExceeded(t *testing.T) {
	// Slow handler that exceeds timeout
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("OK"))
	})

	middleware := Timeout(50 * time.Millisecond)
	wrapped := middleware(slowHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d", w.Code)
	}
}

func TestTimeoutWithConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	config := TimeoutConfig{
		Timeout:      1 * time.Second,
		ErrorMessage: "Custom timeout message",
		StatusCode:   http.StatusRequestTimeout,
	}

	middleware := TimeoutWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.ErrorMessage != "Request timeout" {
		t.Errorf("Expected error message 'Request timeout', got %s", config.ErrorMessage)
	}

	if config.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d", config.StatusCode)
	}
}

func TestFastTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has timeout
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("Context should have deadline")
		}
		w.Write([]byte("OK"))
	})

	middleware := FastTimeout(1 * time.Second)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
