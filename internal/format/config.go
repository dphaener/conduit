package format

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents formatting configuration options
type Config struct {
	IndentSize  int  `yaml:"indent_size"`
	AlignFields bool `yaml:"align_fields"`
}

// DefaultConfig returns the default formatting configuration
func DefaultConfig() *Config {
	return &Config{
		IndentSize:  2,
		AlignFields: true,
	}
}

// LoadConfig loads formatting configuration from a file
// If the file doesn't exist, returns the default configuration
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var wrapper struct {
		Format Config `yaml:"format"`
	}

	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	// Set defaults for any missing values
	config := &wrapper.Format
	if config.IndentSize == 0 {
		config.IndentSize = 2
	}

	return config, nil
}

// SaveConfig saves the formatting configuration to a file
func SaveConfig(path string, config *Config) error {
	wrapper := struct {
		Format Config `yaml:"format"`
	}{
		Format: *config,
	}

	data, err := yaml.Marshal(wrapper)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
