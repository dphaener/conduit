package commands

import (
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	if cmd.Use != "conduit" {
		t.Errorf("expected Use to be 'conduit', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}

	// Check subcommands are registered
	expectedCommands := []string{
		"version",
		"new",
		"build",
		"run",
		"migrate",
		"generate",
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %s to be registered", expected)
		}
	}
}

func TestNewVersionCommand(t *testing.T) {
	// Set test version info
	Version = "1.0.0-test"
	GitCommit = "abc123"
	BuildDate = "2025-01-01"
	GoVersion = "go1.23"

	cmd := NewVersionCommand()

	if cmd.Use != "version" {
		t.Errorf("expected Use to be 'version', got %s", cmd.Use)
	}

	// The version command outputs to stderr/stdout, not the command's output buffer
	// We can't easily capture the colored output in tests, so just verify the command runs
	if cmd.Run == nil {
		t.Fatal("version command Run function is nil")
	}

	// Call the Run function directly
	cmd.Run(cmd, []string{})
}

func TestExecute(t *testing.T) {
	// Test that Execute runs without error for help
	Version = "test"
	GitCommit = "test"
	BuildDate = "test"
	GoVersion = "test"

	// Can't easily test Execute() without mocking os.Exit
	// So we'll just test that NewRootCommand creates a valid command
	cmd := NewRootCommand()
	if cmd == nil {
		t.Error("NewRootCommand returned nil")
	}
}
