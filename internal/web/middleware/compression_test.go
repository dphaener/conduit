package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressionMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("test data ", 200))) // > 1KB
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that response is compressed
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip")
	}

	// Check Vary header
	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("Expected Vary: Accept-Encoding")
	}

	// Decompress and verify content
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	expected := strings.Repeat("test data ", 200)
	if string(decompressed) != expected {
		t.Error("Decompressed content does not match original")
	}
}

func TestCompressionNoGzipSupport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test data"))
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not compress
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not set Content-Encoding when client doesn't support gzip")
	}

	if rec.Body.String() != "test data" {
		t.Error("Response should not be compressed")
	}
}

func TestCompressionMinSize(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("small")) // < 1KB
	})

	config := CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 1024,
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not compress small responses
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not compress responses smaller than MinSize")
	}

	if rec.Body.String() != "small" {
		t.Error("Small response should not be compressed")
	}
}

func TestCompressionExcludedContentTypes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte(strings.Repeat("x", 2000))) // > 1KB
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not compress excluded content types
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not compress excluded content types")
	}
}

func TestCompressionExcludedPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("test ", 500))) // > 1KB
	})

	config := CompressionConfig{
		Level:         gzip.DefaultCompression,
		MinSize:       100,
		ExcludedPaths: []string{"/health", "/metrics"},
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	// Test excluded path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not compress excluded paths")
	}

	// Test non-excluded path
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Should compress non-excluded paths")
	}
}

func TestCompressionCustomLevel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("test data ", 200))) // > 1KB
	})

	config := CompressionConfig{
		Level:   gzip.BestCompression,
		MinSize: 100,
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip")
	}

	// Verify it can be decompressed
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	_, err = io.ReadAll(gr)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}
}

func TestGzipResponseWriterWriteHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(strings.Repeat(`{"key":"value"}`, 100))) // > 1KB
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip")
	}
}

func TestCompressionMultipleWrites(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Multiple writes that together exceed MinSize
		for i := 0; i < 10; i++ {
			w.Write([]byte(strings.Repeat("data ", 50)))
		}
	})

	config := CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 100,
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip")
	}

	// Verify decompression
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	expected := strings.Repeat(strings.Repeat("data ", 50), 10)
	if string(decompressed) != expected {
		t.Error("Decompressed content does not match")
	}
}

func TestCompressionVideoContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Write([]byte(strings.Repeat("x", 2000)))
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Video content should not be compressed
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not compress video content")
	}
}

func TestCompressionAudioContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte(strings.Repeat("x", 2000)))
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Audio content should not be compressed
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("Should not compress audio content")
	}
}

func TestCompressionNoContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No content type set
		w.Write([]byte(strings.Repeat("test ", 500))) // > 1KB
	})

	config := CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 100,
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should compress when no content type (default behavior)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Should compress when no content type is set")
	}
}
