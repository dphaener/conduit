package commands

import (
	"os"
	"testing"

	"github.com/conduit-lang/conduit/compiler/errors"
	"github.com/fatih/color"
)

func TestNewBuildCommand(t *testing.T) {
	cmd := NewBuildCommand()

	if cmd.Use != "build" {
		t.Errorf("expected Use to be 'build', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check flags are registered
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected --json flag to be registered")
	}

	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag to be registered")
	}

	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to be registered")
	}
}

func TestOutputErrorsJSON(t *testing.T) {
	errs := []errors.CompilerError{
		{
			Phase:    "lexer",
			Code:     "LEX001",
			Message:  "test error",
			Severity: errors.Error,
			Location: errors.SourceLocation{
				File:   "test.cdt",
				Line:   1,
				Column: 5,
			},
		},
	}

	// This function writes to stdout, so we can't easily test output
	// But we can at least call it to ensure it doesn't panic
	outputErrorsJSON(errs)
}

func TestRunBuild_NoAppDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := NewBuildCommand()
	err := runBuild(cmd, []string{})

	if err == nil {
		t.Error("expected error when app/ directory not found, got nil")
	}
	if err != nil && !containsString(err.Error(), "app/") {
		t.Errorf("expected error about app/ directory, got: %v", err)
	}
}

func TestRunBuild_NoSourceFiles(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create app directory but no .cdt files
	if err := os.MkdirAll("app", 0755); err != nil {
		t.Fatalf("failed to create app directory: %v", err)
	}

	cmd := NewBuildCommand()
	err := runBuild(cmd, []string{})

	if err == nil {
		t.Error("expected error when no .cdt files found, got nil")
	}
	if err != nil && !containsString(err.Error(), "no .cdt files") {
		t.Errorf("expected error about no .cdt files, got: %v", err)
	}
}

func TestOutputErrorsTerminal(t *testing.T) {
	errs := []errors.CompilerError{
		{
			Phase:    "parser",
			Code:     "PARSE001",
			Message:  "unexpected token",
			Severity: errors.Error,
			Location: errors.SourceLocation{
				File:   "test.cdt",
				Line:   5,
				Column: 10,
			},
		},
		{
			Phase:    "type_checker",
			Code:     "TYPE001",
			Message:  "type mismatch",
			Severity: errors.Error,
			Location: errors.SourceLocation{
				File:   "test.cdt",
				Line:   12,
				Column: 3,
			},
		},
	}

	// This function writes to stderr, so we can't easily test output
	// But we can at least call it to ensure it doesn't panic
	errorColor := color.New(color.FgRed, color.Bold)
	outputErrorsTerminal(errs, errorColor)
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
