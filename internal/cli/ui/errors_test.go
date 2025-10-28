package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestFormatError(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		opts     ErrorOptions
		contains []string
	}{
		{
			name: "basic error",
			opts: ErrorOptions{
				Level:   ErrorLevelError,
				Context: "RESOURCE NOT FOUND",
				Problem: "Cannot find resource 'Post'.",
			},
			contains: []string{
				"❌",
				"RESOURCE NOT FOUND",
				"Cannot find resource 'Post'.",
			},
		},
		{
			name: "error with suggestions",
			opts: ErrorOptions{
				Level:       ErrorLevelError,
				Context:     "RESOURCE NOT FOUND",
				Problem:     "Cannot find resource 'Pst'.",
				Suggestions: []string{"Post", "User"},
			},
			contains: []string{
				"Did you mean: Post, User?",
			},
		},
		{
			name: "error with help commands",
			opts: ErrorOptions{
				Level:   ErrorLevelError,
				Context: "BUILD FAILED",
				Problem: "Syntax error in file",
				HelpCommands: []string{
					"Check syntax: conduit format --check",
					"Get help: conduit build --help",
				},
			},
			contains: []string{
				"→ Check syntax: conduit format --check",
				"→ Get help: conduit build --help",
			},
		},
		{
			name: "warning message",
			opts: ErrorOptions{
				Level:   ErrorLevelWarning,
				Problem: "Deprecated feature used",
			},
			contains: []string{
				"⚠️",
				"Deprecated feature used",
			},
		},
		{
			name: "info message",
			opts: ErrorOptions{
				Level:   ErrorLevelInfo,
				Problem: "Migration completed successfully",
			},
			contains: []string{
				"ℹ️",
				"Migration completed successfully",
			},
		},
		{
			name: "error with consequence",
			opts: ErrorOptions{
				Level:       ErrorLevelError,
				Context:     "MIGRATION FAILED",
				Problem:     "Database connection lost",
				Consequence: "Database may be in inconsistent state",
			},
			contains: []string{
				"Database connection lost",
				"Database may be in inconsistent state",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.opts)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatError() output missing expected string:\nExpected to contain: %q\nGot: %q", expected, result)
				}
			}
		})
	}
}

func TestResourceNotFoundError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := ResourceNotFoundError("Pst", []string{"Post", "User"}, true)

	expected := []string{
		"RESOURCE NOT FOUND",
		"Cannot find resource 'Pst'.",
		"Did you mean: Post, User?",
		"See all resources: conduit introspect resources",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("ResourceNotFoundError() missing expected string: %q", exp)
		}
	}
}

func TestPatternNotFoundError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := PatternNotFoundError("AuthPattern", []string{"Authentication", "Authorization"}, true)

	expected := []string{
		"PATTERN NOT FOUND",
		"Cannot find pattern 'AuthPattern'.",
		"Did you mean: Authentication, Authorization?",
		"See all patterns: conduit introspect patterns",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("PatternNotFoundError() missing expected string: %q", exp)
		}
	}
}

func TestBuildError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := BuildError("Syntax error on line 42", []string{"Check parentheses", "Verify semicolons"}, true)

	expected := []string{
		"BUILD FAILED",
		"Syntax error on line 42",
		"Did you mean: Check parentheses, Verify semicolons?",
		"Check syntax: conduit format --check",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("BuildError() missing expected string: %q", exp)
		}
	}
}

func TestMigrationError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := MigrationError(
		"Failed to apply migration 003",
		"Database may be in inconsistent state",
		[]string{"Check database logs"},
		true,
	)

	expected := []string{
		"MIGRATION FAILED",
		"Failed to apply migration 003",
		"Database may be in inconsistent state",
		"Check migration status: conduit migrate status",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("MigrationError() missing expected string: %q", exp)
		}
	}
}

func TestWriteError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	opts := ErrorOptions{
		Level:   ErrorLevelError,
		Context: "TEST ERROR",
		Problem: "This is a test",
	}

	WriteError(&buf, opts)

	output := buf.String()
	if !strings.Contains(output, "TEST ERROR") {
		t.Errorf("WriteError() did not write to buffer correctly")
	}
}

func TestFormatSuccess(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := FormatSuccess("Build completed", true)

	if !strings.Contains(result, "✓") {
		t.Errorf("FormatSuccess() missing checkmark")
	}
	if !strings.Contains(result, "Build completed") {
		t.Errorf("FormatSuccess() missing message")
	}
}

func TestWriteSuccess(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	WriteSuccess(&buf, "Test success", true)

	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Errorf("WriteSuccess() missing checkmark")
	}
	if !strings.Contains(output, "Test success") {
		t.Errorf("WriteSuccess() missing message")
	}
}

func TestWarning(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := Warning("Deprecated feature", []string{"Use new API"}, true)

	expected := []string{
		"⚠️",
		"Deprecated feature",
		"Did you mean: Use new API?",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Warning() missing expected string: %q", exp)
		}
	}
}

func TestInfo(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := Info("Process starting", true)

	expected := []string{
		"ℹ️",
		"Process starting",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Info() missing expected string: %q", exp)
		}
	}
}

func TestConfigError(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := ConfigError("Invalid YAML syntax", []string{"Check indentation"}, true)

	expected := []string{
		"CONFIGURATION ERROR",
		"Invalid YAML syntax",
		"Did you mean: Check indentation?",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("ConfigError() missing expected string: %q", exp)
		}
	}
}
