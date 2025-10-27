// Package llm provides an automated test harness for validating LLM code generation
// against extracted patterns. It supports multiple LLM providers and validates that
// LLMs can correctly use introspection patterns to generate code.
package llm

import (
	"fmt"
	"os"
	"time"
)

// ProviderType represents the type of LLM provider.
type ProviderType string

const (
	// ProviderClaude represents Anthropic's Claude API.
	ProviderClaude ProviderType = "claude"

	// ProviderOpenAI represents OpenAI's API.
	ProviderOpenAI ProviderType = "openai"
)

// Config holds configuration for the LLM test harness.
type Config struct {
	// Providers is the list of LLM providers to test against.
	Providers []ProviderConfig

	// MaxConcurrentRequests limits parallel LLM API calls.
	MaxConcurrentRequests int

	// DefaultTimeout is the default timeout for LLM API calls.
	DefaultTimeout time.Duration

	// RateLimitDelay is the minimum delay between API calls to the same provider.
	RateLimitDelay time.Duration
}

// ProviderConfig holds configuration for a single LLM provider.
type ProviderConfig struct {
	// Type is the provider type (claude, openai).
	Type ProviderType

	// Model is the specific model to use (e.g., "claude-opus-4", "gpt-4").
	Model string

	// APIKey is the authentication key for the provider.
	APIKey string

	// Timeout is the timeout for API calls to this provider.
	Timeout time.Duration

	// MaxRetries is the number of times to retry failed requests.
	MaxRetries int

	// Enabled determines if this provider should be tested.
	Enabled bool
}

// NewDefaultConfig creates a Config with sensible defaults.
// API keys are loaded from environment variables.
func NewDefaultConfig() *Config {
	return &Config{
		Providers:             defaultProviders(),
		MaxConcurrentRequests: 3,
		DefaultTimeout:        60 * time.Second,
		RateLimitDelay:        2 * time.Second,
	}
}

// defaultProviders creates the default list of LLM providers.
// Only providers with valid API keys are enabled.
func defaultProviders() []ProviderConfig {
	providers := []ProviderConfig{
		{
			Type:       ProviderClaude,
			Model:      "claude-opus-4-20250514",
			APIKey:     os.Getenv("ANTHROPIC_API_KEY"),
			Timeout:    60 * time.Second,
			MaxRetries: 3,
			Enabled:    os.Getenv("ANTHROPIC_API_KEY") != "",
		},
		{
			Type:       ProviderOpenAI,
			Model:      "gpt-4",
			APIKey:     os.Getenv("OPENAI_API_KEY"),
			Timeout:    60 * time.Second,
			MaxRetries: 3,
			Enabled:    os.Getenv("OPENAI_API_KEY") != "",
		},
		{
			Type:       ProviderOpenAI,
			Model:      "gpt-3.5-turbo",
			APIKey:     os.Getenv("OPENAI_API_KEY"),
			Timeout:    60 * time.Second,
			MaxRetries: 3,
			Enabled:    os.Getenv("OPENAI_API_KEY") != "",
		},
	}

	return providers
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("MaxConcurrentRequests must be positive")
	}

	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("DefaultTimeout must be positive")
	}

	if c.RateLimitDelay < 0 {
		return fmt.Errorf("RateLimitDelay must be non-negative")
	}

	// Validate each provider
	enabledCount := 0
	for i, provider := range c.Providers {
		if !provider.Enabled {
			continue
		}

		enabledCount++

		if err := provider.Validate(); err != nil {
			return fmt.Errorf("provider %d: %w", i, err)
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("at least one provider must be enabled")
	}

	return nil
}

// EnabledProviders returns only the enabled providers.
func (c *Config) EnabledProviders() []ProviderConfig {
	var enabled []ProviderConfig
	for _, p := range c.Providers {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// Validate checks that a provider configuration is valid.
func (p *ProviderConfig) Validate() error {
	if p.Type != ProviderClaude && p.Type != ProviderOpenAI {
		return fmt.Errorf("invalid provider type: %s", p.Type)
	}

	if p.Model == "" {
		return fmt.Errorf("model must be specified")
	}

	if p.APIKey == "" {
		return fmt.Errorf("API key must be provided")
	}

	if p.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if p.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries must be non-negative")
	}

	return nil
}

// String returns a human-readable identifier for the provider.
func (p *ProviderConfig) String() string {
	return fmt.Sprintf("%s:%s", p.Type, p.Model)
}
