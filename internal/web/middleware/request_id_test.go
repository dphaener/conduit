package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	var requestIDFromContext string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestIDFromContext = GetRequestID(r.Context())
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that request ID was generated
	if requestIDFromContext == "" {
		t.Error("Expected request ID in context, got empty string")
	}

	// Check that request ID is in response header
	responseID := rec.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("Expected X-Request-ID header in response")
	}

	// Check that context and header IDs match
	if requestIDFromContext != responseID {
		t.Errorf("Context ID (%s) does not match header ID (%s)", requestIDFromContext, responseID)
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	var requestIDFromContext string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestIDFromContext = GetRequestID(r.Context())
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that custom request ID was used
	if requestIDFromContext != "custom-request-id" {
		t.Errorf("Expected 'custom-request-id', got %s", requestIDFromContext)
	}

	// Check that request ID is in response header
	responseID := rec.Header().Get("X-Request-ID")
	if responseID != "custom-request-id" {
		t.Errorf("Expected 'custom-request-id' in response header, got %s", responseID)
	}
}

func TestRequestIDCustomConfig(t *testing.T) {
	var requestIDFromContext string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestIDFromContext = GetRequestID(r.Context())
	})

	config := RequestIDConfig{
		HeaderName: "X-Custom-Request-ID",
		Generator: func() string {
			return "custom-generated-id"
		},
	}

	middleware := RequestIDWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that custom generator was used
	if requestIDFromContext != "custom-generated-id" {
		t.Errorf("Expected 'custom-generated-id', got %s", requestIDFromContext)
	}

	// Check that custom header name was used
	responseID := rec.Header().Get("X-Custom-Request-ID")
	if responseID != "custom-generated-id" {
		t.Errorf("Expected 'custom-generated-id' in X-Custom-Request-ID header, got %s", responseID)
	}

	// Check that default header is not set
	defaultHeader := rec.Header().Get("X-Request-ID")
	if defaultHeader != "" {
		t.Errorf("Expected X-Request-ID header to be empty, got %s", defaultHeader)
	}
}

func TestGetRequestIDFromContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("Expected request ID from context, got empty string")
		}
		w.Write([]byte(id))
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	body := rec.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body")
	}
}

func TestGetRequestIDEmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := GetRequestID(req.Context())
	if id != "" {
		t.Errorf("Expected empty string for context without request ID, got %s", id)
	}
}

func TestRequestIDUniquePerRequest(t *testing.T) {
	var ids []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, GetRequestID(r.Context()))
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	// Check that all IDs are unique
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("Duplicate request ID found: %s", id)
		}
		seen[id] = true
	}

	if len(ids) != 10 {
		t.Errorf("Expected 10 unique IDs, got %d", len(ids))
	}
}

func TestDefaultRequestIDGenerator(t *testing.T) {
	id := defaultRequestIDGenerator()
	if id == "" {
		t.Error("Expected non-empty request ID")
	}

	// Check that it's a valid UUID format (simple check)
	if len(id) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(id))
	}
}
