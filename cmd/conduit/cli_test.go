package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	testBinary     string
	testBinaryOnce sync.Once
	testBinaryErr  error
)

// buildTestBinary builds the conduit binary once for all tests
func buildTestBinary() (string, error) {
	testBinaryOnce.Do(func() {
		tmpBinary := filepath.Join(os.TempDir(), "conduit-test")
		cmd := exec.Command("go", "build", "-o", tmpBinary, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			testBinaryErr = err
			testBinary = string(out)
			return
		}
		testBinary = tmpBinary
	})

	if testBinaryErr != nil {
		return "", testBinaryErr
	}
	return testBinary, nil
}

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	cmd := exec.Command(binary, "version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("version command failed: %v\nOutput: %s", err, output)
	}

	// Check output contains expected strings
	outputStr := string(output)
	expected := []string{
		"Conduit version:",
		"Git commit:",
		"Build date:",
		"Go version:",
	}

	for _, exp := range expected {
		if !contains(outputStr, exp) {
			t.Errorf("version output missing expected string: %q\nGot: %s", exp, outputStr)
		}
	}
}

// TestNewCommand tests the new command
func TestNewCommand(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Run conduit new
	projectName := "test-project"
	cmd := exec.Command(binary, "new", projectName)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("new command failed: %v\nOutput: %s", err, output)
	}

	// Check project directory was created
	projectPath := filepath.Join(tmpDir, projectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory was not created: %s", projectPath)
	}

	// Check required directories exist
	requiredDirs := []string{
		"app",
		"migrations",
		"build",
	}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("required directory not created: %s", dir)
		}
	}

	// Check required files exist
	requiredFiles := []string{
		"app/main.cdt",
		".gitignore",
		"conduit.yaml",
		"README.md",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(projectPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("required file not created: %s", file)
		}
	}

	// Check main.cdt contains resource definition
	mainCdt, err := os.ReadFile(filepath.Join(projectPath, "app/main.cdt"))
	if err != nil {
		t.Fatalf("failed to read main.cdt: %v", err)
	}

	if !contains(string(mainCdt), "resource Post") {
		t.Errorf("main.cdt does not contain expected resource definition")
	}
}

// TestNewCommandExistingDirectory tests error handling for existing directory
func TestNewCommandExistingDirectory(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()
	projectName := "existing"

	// Create directory first
	os.Mkdir(filepath.Join(tmpDir, projectName), 0755)

	// Try to create project with same name
	cmd := exec.Command(binary, "new", projectName)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("new command should fail for existing directory")
	}

	if !contains(string(output), "already exists") {
		t.Errorf("error message should mention directory exists, got: %s", output)
	}
}

// TestNewCommandPathTraversal tests path traversal protection
func TestNewCommandPathTraversal(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()

	testCases := []struct {
		name          string
		projectName   string
		expectedError string
	}{
		{
			name:          "double dots",
			projectName:   "../malware",
			expectedError: "cannot contain '..'",
		},
		{
			name:          "multiple double dots",
			projectName:   "../../etc/malware",
			expectedError: "cannot contain '..'",
		},
		{
			name:          "dots only",
			projectName:   "...",
			expectedError: "cannot contain '..'",
		},
		{
			name:          "forward slash",
			projectName:   "foo/bar",
			expectedError: "cannot contain path separators",
		},
		{
			name:          "backslash",
			projectName:   "foo\\bar",
			expectedError: "cannot contain path separators",
		},
		{
			name:          "starts with dot",
			projectName:   ".hidden",
			expectedError: "cannot start with '.'",
		},
		{
			name:          "empty name",
			projectName:   "",
			expectedError: "cannot be empty",
		},
		{
			name:          "whitespace only",
			projectName:   "   ",
			expectedError: "cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, "new", tc.projectName)
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()

			// Should fail
			if err == nil {
				t.Errorf("new command should fail for project name: %q", tc.projectName)
			}

			if !contains(string(output), tc.expectedError) {
				t.Errorf("error message should contain %q, got: %s", tc.expectedError, output)
			}
		})
	}
}

// TestBuildCommandNoApp tests error handling when app/ directory is missing
func TestBuildCommandNoApp(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()

	cmd := exec.Command(binary, "build")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("build command should fail when app/ directory is missing")
	}

	if !contains(string(output), "app/ directory not found") {
		t.Errorf("error message should mention missing app/, got: %s", output)
	}
}

// TestBuildCommandNoCdtFiles tests error handling when no .cdt files exist
func TestBuildCommandNoCdtFiles(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()

	// Create empty app directory
	os.Mkdir(filepath.Join(tmpDir, "app"), 0755)

	cmd := exec.Command(binary, "build")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("build command should fail when no .cdt files exist")
	}

	if !contains(string(output), "no .cdt files found") {
		t.Errorf("error message should mention no .cdt files, got: %s", output)
	}
}

// TestBuildCommandWithSyntaxError tests compilation with syntax errors
func TestBuildCommandWithSyntaxError(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()

	// Create app directory with invalid .cdt file
	os.Mkdir(filepath.Join(tmpDir, "app"), 0755)
	invalidCode := `resource Post {
		title: string!
		# Missing closing brace
	`
	os.WriteFile(filepath.Join(tmpDir, "app/main.cdt"), []byte(invalidCode), 0644)

	cmd := exec.Command(binary, "build")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("build command should fail with syntax error")
	}

	outputStr := string(output)
	if !contains(outputStr, "failed") {
		t.Errorf("error output should indicate compilation failure, got: %s", outputStr)
	}
}

// TestBuildCommandJSONOutput tests --json flag
func TestBuildCommandJSONOutput(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	tmpDir := t.TempDir()

	// Create app directory with invalid .cdt file
	os.Mkdir(filepath.Join(tmpDir, "app"), 0755)
	invalidCode := `resource Post {
		title: string!
	`
	os.WriteFile(filepath.Join(tmpDir, "app/main.cdt"), []byte(invalidCode), 0644)

	cmd := exec.Command(binary, "build", "--json")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail with JSON output
	if err == nil {
		t.Error("build command should fail with syntax error")
	}

	outputStr := string(output)
	if !contains(outputStr, `"success"`) && !contains(outputStr, `"errors"`) {
		t.Errorf("JSON output should contain success and errors fields, got: %s", outputStr)
	}
}

// TestMigrateCommandNoDatabaseURL tests error handling for missing DATABASE_URL
func TestMigrateCommandNoDatabaseURL(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	// Unset DATABASE_URL
	oldURL := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", oldURL)

	cmd := exec.Command(binary, "migrate", "up")
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("migrate up should fail when DATABASE_URL is not set")
	}

	if !contains(string(output), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %s", output)
	}
}

// TestMigrateStatusNoDatabaseURL tests migrate status without DATABASE_URL
func TestMigrateStatusNoDatabaseURL(t *testing.T) {
	binary, err := buildTestBinary()
	if err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	// Unset DATABASE_URL
	oldURL := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", oldURL)

	cmd := exec.Command(binary, "migrate", "status")
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("migrate status should fail when DATABASE_URL is not set")
	}

	if !contains(string(output), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %s", output)
	}
}

// Helper functions

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
