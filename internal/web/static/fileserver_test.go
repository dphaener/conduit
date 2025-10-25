package static

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestFileServer_BasicFileServing(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test file
	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != testContent {
		t.Errorf("Expected body %q, got %q", testContent, body)
	}
}

func TestFileServer_DirectoryTraversal_BasicDotDot(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	tests := []string{
		"/../etc/passwd",
		"/../../etc/passwd",
		"/../../../etc/passwd",
		"/test/../../../etc/passwd",
	}

	for _, urlPath := range tests {
		t.Run(urlPath, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, urlPath, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Should NOT return 200 (success) - should be rejected or not found
			// Our security check prevents escaping the root, so 404 or 403 or 400 are all acceptable
			if rec.Code == http.StatusOK {
				t.Errorf("Should not return 200 OK for directory traversal attempt %q, got %d",
					urlPath, rec.Code)
			}
		})
	}
}

func TestFileServer_DirectoryTraversal_URLEncoded(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	tests := []string{
		"/%2e%2e/etc/passwd",           // URL-encoded ..
		"/%2e%2e%2f%2e%2e/etc/passwd",  // URL-encoded ../..
		"/test/%2e%2e/etc/passwd",      // URL-encoded .. in path
		"/%252e%252e/etc/passwd",       // Double URL-encoded ..
	}

	for _, encodedPath := range tests {
		t.Run(encodedPath, func(t *testing.T) {
			// URL decode happens before our handler sees it
			decodedPath, err := url.QueryUnescape(encodedPath)
			if err != nil {
				decodedPath = encodedPath
			}

			req := httptest.NewRequest(http.MethodGet, decodedPath, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
				t.Errorf("Expected status %d, %d or %d for path %q, got %d",
					http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound, encodedPath, rec.Code)
			}
		})
	}
}

func TestFileServer_DirectoryTraversal_AbsolutePathEscape(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file outside the root
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	// Try to access the file outside the root
	req := httptest.NewRequest(http.MethodGet, outsideFile, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should be forbidden or not found, not OK
	if rec.Code == http.StatusOK {
		t.Error("Should not be able to access files outside root directory")
	}
}

func TestFileServer_ETagGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.EnableETag = true
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("Expected ETag header to be set")
	}

	// Verify ETag format (should be weak ETag with size and mtime)
	if len(etag) < 5 || etag[:3] != `"W/` {
		t.Errorf("Expected weak ETag format, got %q", etag)
	}
}

func TestFileServer_IfNoneMatch(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.EnableETag = true
	handler := FileServer(config)

	// First request to get the ETag
	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("Expected ETag header")
	}

	// Second request with If-None-Match
	req = httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("Expected status %d, got %d", http.StatusNotModified, rec.Code)
	}
}

func TestFileServer_IfModifiedSince(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	// First request to get Last-Modified
	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	lastModified := rec.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Fatal("Expected Last-Modified header")
	}

	// Second request with If-Modified-Since
	req = httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	req.Header.Set("If-Modified-Since", lastModified)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("Expected status %d, got %d", http.StatusNotModified, rec.Code)
	}
}

func TestFileServer_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestFileServer_IndexFileHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory with index.html
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexContent := "<html><body>Index</body></html>"
	indexFile := filepath.Join(subDir, "index.html")
	if err := os.WriteFile(indexFile, []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.IndexFile = "index.html"
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/subdir/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != indexContent {
		t.Errorf("Expected body %q, got %q", indexContent, body)
	}
}

func TestFileServer_DirectoryWithoutIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory without index.html
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.IndexFile = "index.html"
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/subdir/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestFileServer_MethodNotAllowed(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test.txt", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for method %s, got %d",
					http.StatusMethodNotAllowed, method, rec.Code)
			}
		})
	}
}

func TestFileServer_HeadMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodHead, "/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// HEAD should not return body
	body := rec.Body.String()
	if body != "" {
		t.Errorf("Expected empty body for HEAD request, got %q", body)
	}

	// But should return Content-Type and other headers
	if rec.Header().Get("Content-Type") == "" {
		t.Error("Expected Content-Type header")
	}
}

func TestFileServer_CustomNotFoundHandler(t *testing.T) {
	tmpDir := t.TempDir()

	customNotFoundCalled := false
	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customNotFoundCalled = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Custom 404"))
	})
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !customNotFoundCalled {
		t.Error("Expected custom not found handler to be called")
	}

	body := rec.Body.String()
	if body != "Custom 404" {
		t.Errorf("Expected custom 404 message, got %q", body)
	}
}

func TestFileServer_ContentTypeDetection(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		filename    string
		content     string
		contentType string
	}{
		{"test.html", "<html></html>", "text/html; charset=utf-8"},
		{"test.css", "body {}", "text/css; charset=utf-8"},
		{"test.js", "console.log();", "application/javascript; charset=utf-8"},
		{"test.json", "{}", "application/json; charset=utf-8"},
		{"test.png", "fake-png-data", "image/png"},
		{"test.jpg", "fake-jpg-data", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			config := DefaultFileServerConfig(tmpDir)
			config.Prefix = ""
			handler := FileServer(config)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.filename, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			contentType := rec.Header().Get("Content-Type")
			if contentType != tt.contentType {
				t.Errorf("Expected Content-Type %q, got %q", tt.contentType, contentType)
			}
		})
	}
}

func TestFileServer_CacheHeaders(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.MaxAge = 3600
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cacheControl := rec.Header().Get("Cache-Control")
	expected := "public, max-age=3600"
	if cacheControl != expected {
		t.Errorf("Expected Cache-Control %q, got %q", expected, cacheControl)
	}
}

func TestFileServer_PrefixStripping(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = "/static"
	handler := FileServer(config)

	req := httptest.NewRequest(http.MethodGet, "/static/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != testContent {
		t.Errorf("Expected body %q, got %q", testContent, body)
	}
}

func TestNewFileServer(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	handler := NewFileServer(tmpDir, "/static")

	req := httptest.NewRequest(http.MethodGet, "/static/test.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != testContent {
		t.Errorf("Expected body %q, got %q", testContent, body)
	}
}

func BenchmarkFileServer_SmallFile(b *testing.B) {
	tmpDir := b.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		b.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	handler := FileServer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkFileServer_WithETag(b *testing.B) {
	tmpDir := b.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		b.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.EnableETag = true
	handler := FileServer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkFileServer_Cached(b *testing.B) {
	tmpDir := b.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		b.Fatal(err)
	}

	config := DefaultFileServerConfig(tmpDir)
	config.Prefix = ""
	config.EnableETag = true
	handler := FileServer(config)

	// Get ETag first
	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	etag := rec.Header().Get("ETag")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
		req.Header.Set("If-None-Match", etag)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Consume body to simulate real usage
		io.Copy(io.Discard, rec.Body)
	}
}
