package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test loading with no config file (should use defaults)
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading defaults, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to be non-nil")
	}

	// Check defaults
	if cfg.Server.Port != 3000 {
		t.Errorf("expected default port 3000, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("expected default host 'localhost', got %s", cfg.Server.Host)
	}

	if cfg.Build.Output != "build/app" {
		t.Errorf("expected default output 'build/app', got %s", cfg.Build.Output)
	}

	if cfg.Build.GeneratedDir != "build/generated" {
		t.Errorf("expected default generated dir 'build/generated', got %s", cfg.Build.GeneratedDir)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	// Create temporary directory with config file
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Write config file
	configContent := `
project_name: test-project
server:
  port: 8080
  host: 0.0.0.0
build:
  output: dist/app
  generated_dir: dist/generated
database:
  url: postgresql://localhost/testdb
`
	os.WriteFile("conduit.yml", []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if cfg.ProjectName != "test-project" {
		t.Errorf("expected project name 'test-project', got %s", cfg.ProjectName)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %s", cfg.Server.Host)
	}

	if cfg.Build.Output != "dist/app" {
		t.Errorf("expected output 'dist/app', got %s", cfg.Build.Output)
	}

	if cfg.Database.URL != "postgresql://localhost/testdb" {
		t.Errorf("expected database URL, got %s", cfg.Database.URL)
	}
}

func TestGetDatabaseURL(t *testing.T) {
	// Test with environment variable
	os.Setenv("DATABASE_URL", "postgresql://env/testdb")
	defer os.Unsetenv("DATABASE_URL")

	url := GetDatabaseURL()
	if url != "postgresql://env/testdb" {
		t.Errorf("expected DATABASE_URL from environment, got %s", url)
	}
}

func TestGetDatabaseURLFromConfig(t *testing.T) {
	// Create temporary directory with config file
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Ensure no environment variable
	os.Unsetenv("DATABASE_URL")

	// Write config file
	configContent := `
database:
  url: postgresql://config/testdb
`
	os.WriteFile("conduit.yml", []byte(configContent), 0644)

	url := GetDatabaseURL()
	if url != "postgresql://config/testdb" {
		t.Errorf("expected DATABASE_URL from config, got %s", url)
	}
}

func TestInProject(t *testing.T) {
	// Test in non-project directory
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	if InProject() {
		t.Error("expected InProject to return false in non-project directory")
	}

	// Create app directory
	os.Mkdir("app", 0755)
	os.WriteFile("conduit.yml", []byte(""), 0644)

	if !InProject() {
		t.Error("expected InProject to return true in project directory")
	}
}

func TestGetProjectRoot(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	// Create project root with conduit.yml
	os.WriteFile(filepath.Join(tmpDir, "conduit.yml"), []byte(""), 0644)

	// Create nested subdirectory
	subDir := filepath.Join(tmpDir, "src", "deep", "nested")
	os.MkdirAll(subDir, 0755)
	os.Chdir(subDir)

	root, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("expected to find project root, got error: %v", err)
	}

	// On macOS, /tmp is symlinked to /private/tmp, so resolve both paths
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolvedTmpDir, _ := filepath.EvalSymlinks(tmpDir)

	if resolvedRoot != resolvedTmpDir {
		t.Errorf("expected project root to be %s, got %s", resolvedTmpDir, resolvedRoot)
	}
}

func TestGetProjectRootNotInProject(t *testing.T) {
	// Create temporary directory with no project markers
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err := GetProjectRoot()
	if err == nil {
		t.Error("expected error when not in a project, got nil")
	}
}
