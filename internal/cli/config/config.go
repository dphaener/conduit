package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the Conduit configuration
type Config struct {
	ProjectName string         `mapstructure:"project_name"`
	Database    DatabaseConfig `mapstructure:"database"`
	Server      ServerConfig   `mapstructure:"server"`
	Build       BuildConfig    `mapstructure:"build"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port      int    `mapstructure:"port"`
	Host      string `mapstructure:"host"`
	APIPrefix string `mapstructure:"api_prefix"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Output       string `mapstructure:"output"`
	GeneratedDir string `mapstructure:"generated_dir"`
}

// Load loads the configuration from conduit.yml or conduit.yaml
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 3000)
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.api_prefix", "")
	v.SetDefault("build.output", "build/app")
	v.SetDefault("build.generated_dir", "build/generated")

	// Set config name and paths
	v.SetConfigName("conduit")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// Enable environment variable support
	v.AutomaticEnv()

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found - use defaults
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDatabaseURL returns the database URL from config or environment
func GetDatabaseURL() string {
	// First check environment variable
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// Then check config file
	cfg, err := Load()
	if err != nil {
		return ""
	}

	return cfg.Database.URL
}

// InProject checks if the current directory is a Conduit project
func InProject() bool {
	// Check if app directory exists
	if _, err := os.Stat("app"); err != nil {
		return false
	}

	// Check if conduit.yml or conduit.yaml exists
	if _, err := os.Stat("conduit.yml"); err == nil {
		return true
	}
	if _, err := os.Stat("conduit.yaml"); err == nil {
		return true
	}

	return false
}

// GetProjectRoot tries to find the project root by looking for conduit.yml
func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check for conduit.yml or conduit.yaml
		if _, err := os.Stat(filepath.Join(dir, "conduit.yml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "conduit.yaml")); err == nil {
			return dir, nil
		}

		// Check for app directory as fallback
		if _, err := os.Stat(filepath.Join(dir, "app")); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", fmt.Errorf("not in a Conduit project (no conduit.yml found)")
		}
		dir = parent
	}
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate API prefix format
	if cfg.Server.APIPrefix != "" {
		if !strings.HasPrefix(cfg.Server.APIPrefix, "/") {
			return fmt.Errorf("server.api_prefix must start with '/', got: %s", cfg.Server.APIPrefix)
		}
		if strings.HasSuffix(cfg.Server.APIPrefix, "/") {
			return fmt.Errorf("server.api_prefix must not end with '/', got: %s", cfg.Server.APIPrefix)
		}
	}
	return nil
}
