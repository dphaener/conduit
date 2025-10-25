package commands

import (
	"os"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	testCases := []struct {
		name        string
		projectName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid name",
			projectName: "my-project",
			expectError: false,
		},
		{
			name:        "valid name with underscores",
			projectName: "my_project",
			expectError: false,
		},
		{
			name:        "valid name alphanumeric",
			projectName: "myproject123",
			expectError: false,
		},
		{
			name:        "empty string",
			projectName: "",
			expectError: true,
			errorMsg:    "must be 1-100 characters",
		},
		{
			name:        "whitespace only",
			projectName: "   ",
			expectError: true,
			errorMsg:    "must be 1-100 characters",
		},
		{
			name:        "too long",
			projectName: "a" + string(make([]byte, 100)),
			expectError: true,
			errorMsg:    "must be 1-100 characters",
		},
		{
			name:        "contains slash",
			projectName: "my/project",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
		{
			name:        "contains backslash",
			projectName: "my\\project",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
		{
			name:        "contains dot",
			projectName: "my.project",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
		{
			name:        "path traversal attempt",
			projectName: "../malicious",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
		{
			name:        "absolute path",
			projectName: "/usr/bin/malware",
			expectError: true,
			errorMsg:    "cannot be an absolute path",
		},
		{
			name:        "starts with dot",
			projectName: ".hidden",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
		{
			name:        "contains special chars",
			projectName: "my@project!",
			expectError: true,
			errorMsg:    "can only contain letters, numbers, dashes, and underscores",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProjectName(tc.projectName)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for project name %q, got nil", tc.projectName)
				} else if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for project name %q, got %v", tc.projectName, err)
				}
			}
		})
	}
}

func TestNewNewCommand(t *testing.T) {
	cmd := NewNewCommand()

	if cmd.Use != "new [project-name]" {
		t.Errorf("expected Use to be 'new [project-name]', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check flags are registered
	if cmd.Flags().Lookup("interactive") == nil {
		t.Error("expected --interactive flag to be registered")
	}

	if cmd.Flags().Lookup("database") == nil {
		t.Error("expected --database flag to be registered")
	}

	if cmd.Flags().Lookup("port") == nil {
		t.Error("expected --port flag to be registered")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRunNew_DirectoryAlreadyExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a subdirectory that will conflict
	existingDir := tmpDir + "/existing-project"
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Try to create project with same name
	cmd := NewNewCommand()
	err := runNew(cmd, []string{"existing-project"})

	if err == nil {
		t.Error("expected error when directory already exists, got nil")
	}
	if !contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestRunNew_InvalidProjectName(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	testCases := []struct {
		name        string
		projectName string
	}{
		{"empty name", ""},
		{"with slash", "my/project"},
		{"with dots", "my.project"},
		{"absolute path", "/tmp/project"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewNewCommand()
			err := runNew(cmd, []string{tc.projectName})

			if err == nil {
				t.Errorf("expected error for project name %q, got nil", tc.projectName)
			}
		})
	}
}

func TestRunNew_ValidProjectCreation(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create a valid project
	cmd := NewNewCommand()
	err := runNew(cmd, []string{"test-project"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{
		"test-project",
		"test-project/app",
		"test-project/migrations",
		"test-project/build",
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", dir)
		}
	}

	// Verify files
	expectedFiles := []string{
		"test-project/app/main.cdt",
		"test-project/.gitignore",
		"test-project/conduit.yaml",
		"test-project/README.md",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", file)
		}
	}
}
