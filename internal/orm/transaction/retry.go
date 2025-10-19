package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultMaxRetries is the default number of retry attempts for deadlocks
	DefaultMaxRetries = 3
	// DefaultBaseBackoff is the default base backoff duration
	DefaultBaseBackoff = 100 * time.Millisecond
)

// RetryConfig configures retry behavior for transactions
type RetryConfig struct {
	MaxRetries  int
	BaseBackoff time.Duration
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  DefaultMaxRetries,
		BaseBackoff: DefaultBaseBackoff,
	}
}

// WithRetry executes a transaction with automatic retry on deadlock
func (m *Manager) WithRetry(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return m.WithRetryConfig(ctx, DefaultRetryConfig(), fn)
}

// WithRetryConfig executes a transaction with custom retry configuration
func (m *Manager) WithRetryConfig(ctx context.Context, config *RetryConfig, fn func(tx *sql.Tx) error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		// Check if context is already cancelled before starting retry attempt
		if ctx.Err() != nil {
			return fmt.Errorf("transaction cancelled before retry %d: %w", attempt, ctx.Err())
		}

		err := m.WithTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		// Check if this is a deadlock error
		if isDeadlockError(err) {
			lastErr = err

			// Calculate exponential backoff: baseBackoff * 2^attempt
			backoff := config.BaseBackoff * time.Duration(1<<uint(attempt))

			// Check if context is still valid before sleeping
			select {
			case <-ctx.Done():
				return fmt.Errorf("transaction cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Continue to next retry attempt
				continue
			}
		}

		// Non-deadlock error or timeout - fail immediately
		return err
	}

	return fmt.Errorf("%w: transaction failed after %d retries: %v", ErrDeadlock, config.MaxRetries, lastErr)
}

// WithRetryIsolation executes a transaction with retry and custom isolation level
func (m *Manager) WithRetryIsolation(
	ctx context.Context,
	level IsolationLevel,
	config *RetryConfig,
	fn func(tx *sql.Tx) error,
) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		// Check if context is already cancelled before starting retry attempt
		if ctx.Err() != nil {
			return fmt.Errorf("transaction cancelled before retry %d: %w", attempt, ctx.Err())
		}

		err := m.WithTransactionIsolation(ctx, level, fn)
		if err == nil {
			return nil
		}

		if isDeadlockError(err) {
			lastErr = err
			backoff := config.BaseBackoff * time.Duration(1<<uint(attempt))

			select {
			case <-ctx.Done():
				return fmt.Errorf("transaction cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				continue
			}
		}

		return err
	}

	return fmt.Errorf("%w: transaction failed after %d retries: %v", ErrDeadlock, config.MaxRetries, lastErr)
}

// isDeadlockError checks if an error is a deadlock error
// Detects PostgreSQL deadlock error codes and messages
func isDeadlockError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// PostgreSQL deadlock detection error code: 40P01
	if strings.Contains(errStr, "40P01") {
		return true
	}

	// Common deadlock error messages
	deadlockMessages := []string{
		"deadlock detected",
		"deadlock found",
		"lock wait timeout exceeded",
		"could not serialize access",
	}

	for _, msg := range deadlockMessages {
		if strings.Contains(strings.ToLower(errStr), msg) {
			return true
		}
	}

	return false
}

// isSerializationError checks if an error is a serialization failure
// These can also benefit from retry
func isSerializationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// PostgreSQL serialization failure code: 40001
	if strings.Contains(errStr, "40001") {
		return true
	}

	if strings.Contains(strings.ToLower(errStr), "could not serialize access") {
		return true
	}

	return false
}

// IsRetryableError checks if an error is retryable (deadlock or serialization failure)
func IsRetryableError(err error) bool {
	return isDeadlockError(err) || isSerializationError(err)
}
