package commands

import (
	"os"
	"testing"
)

func TestNewRunCommand(t *testing.T) {
	cmd := NewRunCommand()

	if cmd.Use != "run" {
		t.Errorf("expected Use to be 'run', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check flags are registered
	if cmd.Flags().Lookup("port") == nil {
		t.Error("expected --port flag to be registered")
	}

	if cmd.Flags().Lookup("hot-reload") == nil {
		t.Error("expected --hot-reload flag to be registered")
	}

	if cmd.Flags().Lookup("build") == nil {
		t.Error("expected --build flag to be registered")
	}
}

func TestRunRun_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create app directory with a simple .cdt file to avoid build issues
	if err := os.MkdirAll("app", 0755); err != nil {
		t.Fatalf("failed to create app directory: %v", err)
	}

	// Set --no-build flag to skip build step
	cmd := NewRunCommand()
	cmd.Flags().Set("build", "false")

	err := runRun(cmd, []string{})

	if err == nil {
		t.Error("expected error when binary not found, got nil")
	}
	if err != nil && !containsStr(err.Error(), "not found") && !containsStr(err.Error(), "build may have failed") {
		t.Errorf("expected error about binary not found, got: %v", err)
	}
}

func TestRunRun_PortFlag(t *testing.T) {
	// This test verifies the port flag is correctly parsed
	cmd := NewRunCommand()

	// Set port flag
	cmd.Flags().Set("port", "8080")

	// Verify the flag value was set
	portFlag := cmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("port flag not found")
	}

	if portFlag.Value.String() != "8080" {
		t.Errorf("expected port to be 8080, got %s", portFlag.Value.String())
	}
}

// Helper function
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubStr(s, substr)))
}

func findSubStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
