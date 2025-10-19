package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check CORS headers
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected CORS origin header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for preflight request")
	})

	middleware := CORS()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for preflight, got %d", http.StatusNoContent, rec.Code)
	}

	// Check preflight headers
	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowMethods, "GET") {
		t.Errorf("Expected GET in allowed methods, got %s", allowMethods)
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowHeaders, "Content-Type") {
		t.Errorf("Expected Content-Type in allowed headers, got %s", allowHeaders)
	}

	maxAge := rec.Header().Get("Access-Control-Max-Age")
	if maxAge == "" {
		t.Error("Expected Max-Age header for preflight")
	}
}

func TestCORSCustomConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := CORSConfig{
		AllowedOrigins:   []string{"http://example.com", "http://test.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{"X-Custom-Header"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	middleware := CORSWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check credentials
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected credentials to be allowed")
	}

	// Check exposed headers
	if rec.Header().Get("Access-Control-Expose-Headers") != "X-Custom-Header" {
		t.Errorf("Expected X-Custom-Header in exposed headers, got %s", rec.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := CORSConfig{
		AllowedOrigins: []string{"http://allowed.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	middleware := CORSWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://disallowed.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not set CORS headers for disallowed origin
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set for disallowed origin")
	}
}

func TestCORSWildcardOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	middleware := CORSWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://any-origin.com" {
		t.Error("Wildcard should allow any origin")
	}
}

func TestCORSWildcardSubdomain(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	tests := []struct {
		origin   string
		expected bool
	}{
		{"http://api.example.com", true},
		{"http://www.example.com", true},
		{"http://example.com", false},
		{"http://other.com", false},
	}

	for _, test := range tests {
		allowed := isOriginAllowed(test.origin, config.AllowedOrigins)
		if allowed != test.expected {
			t.Errorf("Origin %s: expected %v, got %v", test.origin, test.expected, allowed)
		}
	}
}

func TestCORSNoOriginHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not set CORS headers when no origin
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set when no origin header")
	}

	// Should still process the request
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCORSMultipleOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := CORSConfig{
		AllowedOrigins: []string{"http://origin1.com", "http://origin2.com", "http://origin3.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	middleware := CORSWithConfig(config)
	wrapped := middleware(handler)

	// Test allowed origins
	for _, origin := range config.AllowedOrigins {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Errorf("Expected origin %s to be allowed", origin)
		}
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{"http://example.com", []string{"http://example.com"}, true},
		{"http://example.com", []string{"http://other.com"}, false},
		{"http://example.com", []string{"*"}, true},
		{"http://api.example.com", []string{"*.example.com"}, true},
		{"http://example.com", []string{"*.example.com"}, false},
		{"http://test.com", []string{"http://example.com", "http://test.com"}, true},
	}

	for _, test := range tests {
		result := isOriginAllowed(test.origin, test.allowedOrigins)
		if result != test.expected {
			t.Errorf("isOriginAllowed(%s, %v): expected %v, got %v",
				test.origin, test.allowedOrigins, test.expected, result)
		}
	}
}
