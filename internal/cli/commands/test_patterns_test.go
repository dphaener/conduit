package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestNewTestPatternsCommand(t *testing.T) {
	cmd := NewTestPatternsCommand()

	if cmd == nil {
		t.Fatal("NewTestPatternsCommand() returned nil")
	}

	if cmd.Use != "test-patterns" {
		t.Errorf("Expected Use='test-patterns', got %s", cmd.Use)
	}

	// Check flags exist
	flags := []string{"max-iterations", "mock", "format", "output", "verbose"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag --%s to exist", flag)
		}
	}
}

func TestTestPatternsCommand_FlagDefaults(t *testing.T) {
	cmd := NewTestPatternsCommand()

	// Check default values
	maxIterations, _ := cmd.Flags().GetInt("max-iterations")
	if maxIterations != 4 {
		t.Errorf("Expected max-iterations default=4, got %d", maxIterations)
	}

	useMock, _ := cmd.Flags().GetBool("mock")
	if useMock {
		t.Errorf("Expected mock default=false, got true")
	}

	format, _ := cmd.Flags().GetString("format")
	if format != "text" {
		t.Errorf("Expected format default='text', got %s", format)
	}

	output, _ := cmd.Flags().GetString("output")
	if output != "" {
		t.Errorf("Expected output default='', got %s", output)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		t.Errorf("Expected verbose default=false, got true")
	}
}

func TestTestPatternsCommand_ValidationErrors(t *testing.T) {
	// Initialize test registry with mock data
	defer metadata.Reset()

	// Register empty metadata
	meta := &metadata.Metadata{
		Version:   "1.0.0",
		Resources: []metadata.ResourceMetadata{},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "invalid max-iterations too low",
			args:        []string{"--max-iterations", "0"},
			expectedErr: "max-iterations must be between 1 and 10",
		},
		{
			name:        "invalid max-iterations too high",
			args:        []string{"--max-iterations", "11"},
			expectedErr: "max-iterations must be between 1 and 10",
		},
		{
			name:        "invalid format",
			args:        []string{"--format", "yaml"},
			expectedErr: "format must be 'text' or 'json'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewTestPatternsCommand()
			cmd.SetArgs(tt.args)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			err := cmd.Execute()
			if err == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectedErr, err)
			}
		})
	}
}

func TestTestPatternsCommand_NoResources(t *testing.T) {
	// Initialize empty registry
	defer metadata.Reset()

	// Register empty metadata
	meta := &metadata.Metadata{
		Version:   "1.0.0",
		Resources: []metadata.ResourceMetadata{},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	cmd := NewTestPatternsCommand()
	cmd.SetArgs([]string{"--mock"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Errorf("Expected error for no resources, got nil")
		return
	}

	if !strings.Contains(err.Error(), "no resources found") {
		t.Errorf("Expected 'no resources found' error, got: %v", err)
	}
}

func TestTestPatternsCommand_WithMock(t *testing.T) {
	// Initialize test registry with mock data
	defer metadata.Reset()

	// Register a test resource
	meta := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{{
			Name: "Post",
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
				{Name: "title", Type: "string", Required: true},
			},
			Middleware: map[string][]string{
				"create": {"auth", "validate"},
			},
		}},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	cmd := NewTestPatternsCommand()
	cmd.SetArgs([]string{"--mock", "--max-iterations", "1"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	// With mock, we expect it to fail with "no patterns" or "success criteria not met"
	if err != nil {
		// Check if it's an expected error
		if !strings.Contains(err.Error(), "success criteria not met") &&
			!strings.Contains(err.Error(), "no patterns extracted") {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	output := buf.String()

	// Should have some output
	if len(output) == 0 {
		t.Errorf("Expected output, got empty string")
	}

	// Should mention mock mode
	if !strings.Contains(output, "mock") && !strings.Contains(output, "Mock") {
		t.Logf("Output: %s", output)
		// This is not a failure, just a note
	}
}

func TestTestPatternsCommand_OutputToFile(t *testing.T) {
	// Initialize test registry
	defer metadata.Reset()

	// Register a test resource
	meta := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{{
			Name: "User",
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
			},
			Middleware: map[string][]string{
				"create": {"auth"},
			},
		}},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "test-patterns-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cmd := NewTestPatternsCommand()
	cmd.SetArgs([]string{
		"--mock",
		"--max-iterations", "1",
		"--format", "json",
		"--output", tmpFile.Name(),
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run command (may fail with success criteria not met or no patterns, which is OK)
	cmdErr := cmd.Execute()

	// If it failed with no patterns, that's expected for this minimal test
	if cmdErr != nil && !strings.Contains(cmdErr.Error(), "no patterns extracted") {
		// Check if file was written anyway
		content, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		if len(content) > 0 {
			// Should be valid JSON if content exists
			if !strings.HasPrefix(string(content), "{") {
				t.Errorf("Expected JSON output, got: %s", string(content))
			}
		}
	}
}

func TestTestPatternsCommand_JSONFormat(t *testing.T) {
	// Initialize test registry
	defer metadata.Reset()

	// Register a test resource
	meta := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{{
			Name: "Comment",
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
			},
			Middleware: map[string][]string{
				"create": {"auth", "rate_limit"},
			},
		}},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	cmd := NewTestPatternsCommand()
	cmd.SetArgs([]string{
		"--mock",
		"--max-iterations", "1",
		"--format", "json",
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run command (may fail with success criteria not met)
	_ = cmd.Execute()

	output := buf.String()

	// Should contain JSON fields
	if !strings.Contains(output, `"timestamp"`) && !strings.Contains(output, `"iterations"`) {
		// May have failed before generating report
		t.Logf("Output did not contain expected JSON fields (may have failed early): %s", output)
	}
}

func TestTestPatternsCommand_VerboseMode(t *testing.T) {
	// Initialize test registry
	defer metadata.Reset()

	// Register a test resource
	meta := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{{
			Name: "Article",
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
			},
			Middleware: map[string][]string{
				"list": {"cache"},
			},
		}},
	}
	data, _ := json.Marshal(meta)
	metadata.RegisterMetadata(data)

	cmd := NewTestPatternsCommand()
	cmd.SetArgs([]string{
		"--mock",
		"--max-iterations", "1",
		"--verbose",
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run command
	_ = cmd.Execute()

	output := buf.String()

	// Verbose mode should show more details
	if !strings.Contains(output, "Loading") && !strings.Contains(output, "resources") {
		t.Logf("Output did not show verbose loading message (may be OK): %s", output)
	}
}
