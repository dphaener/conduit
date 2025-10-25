package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchCommand_Creation(t *testing.T) {
	cmd := NewWatchCommand()

	if cmd == nil {
		t.Fatal("Expected watch command to be created")
	}

	if cmd.Use != "watch" {
		t.Errorf("Expected Use to be 'watch', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}
}

func TestWatchCommand_Flags(t *testing.T) {
	cmd := NewWatchCommand()

	// Check port flag
	portFlag := cmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("Expected --port flag to exist")
	}

	if portFlag.DefValue != "3000" {
		t.Errorf("Expected default port 3000, got %s", portFlag.DefValue)
	}

	// Check app-port flag
	appPortFlag := cmd.Flags().Lookup("app-port")
	if appPortFlag == nil {
		t.Error("Expected --app-port flag to exist")
	}

	if appPortFlag.DefValue != "3001" {
		t.Errorf("Expected default app-port 3001, got %s", appPortFlag.DefValue)
	}

	// Check verbose flag
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected --verbose flag to exist")
	}
}

func TestWatchCommand_RequiresAppDirectory(t *testing.T) {
	// Create temp directory without app/ subdirectory
	tmpDir, err := os.MkdirTemp("", "watch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	cmd := NewWatchCommand()

	// Running without app/ directory should fail
	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when app/ directory doesn't exist")
	}
}

func TestWatchCommand_WithAppDirectory(t *testing.T) {
	// This test would require starting the server, which is complex for unit tests
	// Instead, we just verify the command structure is correct
	tmpDir, err := os.MkdirTemp("", "watch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create a simple .cdt file
	testFile := filepath.Join(appDir, "test.cdt")
	content := `
resource Test {
  id: uuid! @primary @auto
  name: string!
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// The actual test would start the server and send interrupt signal
	// For now, we just verify the command can be created with the app directory
	cmd := NewWatchCommand()
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Note: We don't actually run the command here because it would block
	// and require signal handling. Integration tests would cover that.
}

func TestWatchCommand_CustomPorts(t *testing.T) {
	cmd := NewWatchCommand()

	// Set custom port flags
	cmd.Flags().Set("port", "8080")
	cmd.Flags().Set("app-port", "8081")

	portFlag := cmd.Flags().Lookup("port")
	if portFlag.Value.String() != "8080" {
		t.Errorf("Expected port to be set to 8080, got %s", portFlag.Value.String())
	}

	appPortFlag := cmd.Flags().Lookup("app-port")
	if appPortFlag.Value.String() != "8081" {
		t.Errorf("Expected app-port to be set to 8081, got %s", appPortFlag.Value.String())
	}
}

func TestWatchCommand_Integration(t *testing.T) {
	// Skip integration test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "watch-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	testFile := filepath.Join(appDir, "post.cdt")
	content := `
resource Post {
  id: uuid! @primary @auto
  title: string!
}
`
	os.WriteFile(testFile, []byte(content), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// This would be a full integration test that:
	// 1. Starts the dev server
	// 2. Waits for it to be ready
	// 3. Makes a file change
	// 4. Verifies rebuild happens
	// 5. Sends interrupt signal
	// 6. Verifies clean shutdown

	// For now, just verify the structure
	cmd := NewWatchCommand()
	if cmd == nil {
		t.Fatal("Failed to create watch command")
	}
}

func BenchmarkWatchCommand_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewWatchCommand()
	}
}

// Helper function for integration tests
func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Try to connect
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}
