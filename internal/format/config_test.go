package format

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".conduit-format.yml")

	// Create custom config
	config := &Config{
		IndentSize:  4,
		AlignFields: false,
	}

	// Save config
	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loaded.IndentSize != 4 {
		t.Errorf("Expected indent size 4, got %d", loaded.IndentSize)
	}
	if loaded.AlignFields {
		t.Errorf("Expected align fields false")
	}
}

func TestConfigLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".conduit-format.yml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:\n  - bad"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid yaml: %v", err)
	}

	// Should return error
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("Expected error loading invalid YAML")
	}
}

func TestConfigPartialSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".conduit-format.yml")

	// Write partial config (only some fields)
	yamlContent := `format:
  indent_size: 3
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write yaml: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify custom values
	if loaded.IndentSize != 3 {
		t.Errorf("Expected indent size 3, got %d", loaded.IndentSize)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.IndentSize != 2 {
		t.Errorf("Default indent size should be 2, got %d", config.IndentSize)
	}
	if !config.AlignFields {
		t.Errorf("Default align fields should be true")
	}
}

func TestConfigLoadWithZeroValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".conduit-format.yml")

	// Write config with zero values to test defaults
	yamlContent := `format:
  indent_size: 0
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write yaml: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults are applied for zero values
	if loaded.IndentSize != 2 {
		t.Errorf("Expected default indent size 2 for zero value, got %d", loaded.IndentSize)
	}
}

func TestConfigSaveError(t *testing.T) {
	// Try to save to a non-existent directory
	err := SaveConfig("/nonexistent/directory/.conduit-format.yml", DefaultConfig())
	if err == nil {
		t.Errorf("SaveConfig should return error for invalid path")
	}
}
