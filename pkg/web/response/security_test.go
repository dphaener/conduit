package response

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderer_File_PathTraversalPrevention(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	forbiddenDir := filepath.Join(tmpDir, "forbidden")

	// Create directories
	os.MkdirAll(allowedDir, 0755)
	os.MkdirAll(forbiddenDir, 0755)

	// Create test files
	allowedFile := filepath.Join(allowedDir, "safe.txt")
	os.WriteFile(allowedFile, []byte("safe content"), 0644)

	forbiddenFile := filepath.Join(forbiddenDir, "secret.txt")
	os.WriteFile(forbiddenFile, []byte("secret content"), 0644)

	// Create renderer with allowed directory
	config := &RendererConfig{
		AllowedDirs: []string{allowedDir},
	}
	renderer := NewRendererWithConfig(config)

	tests := []struct {
		name        string
		filePath    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid file in allowed directory",
			filePath: allowedFile,
			wantErr:  false,
		},
		{
			name:        "path traversal with ../",
			filePath:    filepath.Join(allowedDir, "..", "forbidden", "secret.txt"),
			wantErr:     true,
			errContains: "not in allowed directories",
		},
		{
			name:        "absolute path outside allowed directory",
			filePath:    forbiddenFile,
			wantErr:     true,
			errContains: "not in allowed directories",
		},
		{
			name:        "non-existent file",
			filePath:    filepath.Join(allowedDir, "nonexistent.txt"),
			wantErr:     true,
			errContains: "invalid file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/download", nil)

			err := renderer.File(w, req, tt.filePath, "download.txt")

			if (err != nil) != tt.wantErr {
				t.Errorf("File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}

func TestRenderer_File_NoAllowedDirs(t *testing.T) {
	// Renderer without allowed directories should reject all file access
	renderer := NewRenderer()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download", nil)

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	err := renderer.File(w, req, tmpFile, "test.txt")

	if err == nil {
		t.Error("expected error when no allowed directories configured")
	}

	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should mention not configured, got: %v", err)
	}
}

func TestRenderer_File_SymlinkProtection(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	forbiddenDir := filepath.Join(tmpDir, "forbidden")

	os.MkdirAll(allowedDir, 0755)
	os.MkdirAll(forbiddenDir, 0755)

	// Create a file in forbidden directory
	forbiddenFile := filepath.Join(forbiddenDir, "secret.txt")
	os.WriteFile(forbiddenFile, []byte("secret"), 0644)

	// Create a symlink in allowed directory pointing to forbidden file
	symlinkPath := filepath.Join(allowedDir, "link.txt")
	err := os.Symlink(forbiddenFile, symlinkPath)
	if err != nil {
		t.Skip("symlink creation not supported on this system")
	}

	// Create renderer with only allowed directory
	config := &RendererConfig{
		AllowedDirs: []string{allowedDir},
	}
	renderer := NewRendererWithConfig(config)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download", nil)

	// Attempt to access the symlink should fail because it resolves outside allowed dir
	err = renderer.File(w, req, symlinkPath, "download.txt")

	if err == nil {
		t.Error("expected error when accessing symlink to forbidden directory")
	}

	if !strings.Contains(err.Error(), "not in allowed directories") {
		t.Errorf("error should mention not in allowed directories, got: %v", err)
	}
}

func TestStreamFile_PathTraversalPrevention(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	forbiddenDir := filepath.Join(tmpDir, "forbidden")

	os.MkdirAll(allowedDir, 0755)
	os.MkdirAll(forbiddenDir, 0755)

	// Create test files
	allowedFile := filepath.Join(allowedDir, "video.mp4")
	os.WriteFile(allowedFile, []byte("video content"), 0644)

	forbiddenFile := filepath.Join(forbiddenDir, "secret.mp4")
	os.WriteFile(forbiddenFile, []byte("secret video"), 0644)

	tests := []struct {
		name        string
		filePath    string
		allowedDirs []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid file in allowed directory",
			filePath:    allowedFile,
			allowedDirs: []string{allowedDir},
			wantErr:     false,
		},
		{
			name:        "path traversal attempt",
			filePath:    filepath.Join(allowedDir, "..", "forbidden", "secret.mp4"),
			allowedDirs: []string{allowedDir},
			wantErr:     true,
			errContains: "not in allowed directories",
		},
		{
			name:        "no allowed directories",
			filePath:    allowedFile,
			allowedDirs: []string{},
			wantErr:     true,
			errContains: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/stream", nil)

			err := StreamFile(w, req, tt.filePath, "video.mp4", "video/mp4", tt.allowedDirs)

			if (err != nil) != tt.wantErr {
				t.Errorf("StreamFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}

func TestRenderer_SetAllowedDirs(t *testing.T) {
	renderer := NewRenderer()

	// Initially no allowed dirs
	if len(renderer.allowedDirs) != 0 {
		t.Error("new renderer should have no allowed dirs")
	}

	// Set allowed dirs
	dirs := []string{"/tmp/uploads", "/var/data"}
	renderer.SetAllowedDirs(dirs)

	if len(renderer.allowedDirs) != 2 {
		t.Errorf("expected 2 allowed dirs, got %d", len(renderer.allowedDirs))
	}
}

func TestNewRendererWithConfig(t *testing.T) {
	config := &RendererConfig{
		PrettyPrint: true,
		AllowedDirs: []string{"/tmp/test"},
	}

	renderer := NewRendererWithConfig(config)

	if !renderer.prettyPrint {
		t.Error("pretty print should be enabled")
	}

	if len(renderer.allowedDirs) != 1 {
		t.Errorf("expected 1 allowed dir, got %d", len(renderer.allowedDirs))
	}
}
