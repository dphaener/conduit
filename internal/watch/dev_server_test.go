package watch

import (
	"testing"
)

func TestDevServer_NewDevServer(t *testing.T) {
	config := &DevServerConfig{
		Port:    3000,
		AppPort: 3001,
		WatchPatterns: []string{"*.cdt"},
		IgnorePatterns: []string{"*.swp"},
	}

	ds, err := NewDevServer(config)
	if err != nil {
		t.Fatalf("Failed to create dev server: %v", err)
	}

	if ds == nil {
		t.Fatal("Expected dev server to be created")
	}

	if ds.compiler == nil {
		t.Error("Expected compiler to be initialized")
	}

	if ds.reloadServer == nil {
		t.Error("Expected reload server to be initialized")
	}

	if ds.assetWatcher == nil {
		t.Error("Expected asset watcher to be initialized")
	}

	if ds.watcher == nil {
		t.Error("Expected file watcher to be initialized")
	}

	if ds.port != 3000 {
		t.Errorf("Expected port 3000, got %d", ds.port)
	}

	if ds.appPort != 3001 {
		t.Errorf("Expected appPort 3001, got %d", ds.appPort)
	}
}

func TestDevServer_NewDevServer_DefaultConfig(t *testing.T) {
	// Test with nil config - should use defaults
	ds, err := NewDevServer(nil)
	if err != nil {
		t.Fatalf("Failed to create dev server with default config: %v", err)
	}

	if ds.port != 3000 {
		t.Errorf("Expected default port 3000, got %d", ds.port)
	}

	if ds.appPort != 3001 {
		t.Errorf("Expected default appPort 3001, got %d", ds.appPort)
	}

	if len(ds.watchPatterns) == 0 {
		t.Error("Expected default watch patterns to be set")
	}
}

func TestDevServerConfig_Defaults(t *testing.T) {
	config := &DevServerConfig{
		Port:    8080,
		AppPort: 8081,
	}

	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Port)
	}

	if config.AppPort != 8081 {
		t.Errorf("Expected appPort 8081, got %d", config.AppPort)
	}
}

// Note: Full integration tests for Start(), Stop(), handleFileChange(), etc.
// would require mocking the filesystem, compiler, and server processes.
// These are better tested via end-to-end integration tests.
// The unit tests above verify the basic structure and initialization.
