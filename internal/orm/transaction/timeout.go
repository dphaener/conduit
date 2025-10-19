package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WithTimeout executes a transaction with a timeout
// If the transaction doesn't complete within the specified duration, it will be rolled back
func (m *Manager) WithTimeout(ctx context.Context, timeout time.Duration, fn func(tx *sql.Tx) error) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute transaction with timeout context
	err := m.WithTransaction(timeoutCtx, fn)
	if err != nil {
		// Check if the error was due to timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: transaction exceeded %v", ErrTransactionTimeout, timeout)
		}
		return err
	}

	return nil
}

// WithTimeoutIsolation executes a transaction with timeout and custom isolation level
func (m *Manager) WithTimeoutIsolation(
	ctx context.Context,
	timeout time.Duration,
	level IsolationLevel,
	fn func(tx *sql.Tx) error,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := m.WithTransactionIsolation(timeoutCtx, level, fn)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: transaction exceeded %v", ErrTransactionTimeout, timeout)
		}
		return err
	}

	return nil
}

// WithTimeoutRetry executes a transaction with timeout and automatic retry on deadlock
func (m *Manager) WithTimeoutRetry(
	ctx context.Context,
	timeout time.Duration,
	config *RetryConfig,
	fn func(tx *sql.Tx) error,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := m.WithRetryConfig(timeoutCtx, config, fn)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: transaction exceeded %v", ErrTransactionTimeout, timeout)
		}
		return err
	}

	return nil
}

// BeginWithTimeout starts a new transaction with a timeout
// The transaction must be committed or rolled back within the specified duration
func (m *Manager) BeginWithTimeout(ctx context.Context, timeout time.Duration) (*Transaction, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	tx, err := m.Begin(timeoutCtx)
	if err != nil {
		cancel()
		return nil, err
	}

	// Store cancel function in transaction so it's called on commit/rollback
	tx.cancelFunc = cancel

	return tx, nil
}

// BeginWithDeadline starts a new transaction that must complete before the deadline
func (m *Manager) BeginWithDeadline(ctx context.Context, deadline time.Time) (*Transaction, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)

	tx, err := m.Begin(deadlineCtx)
	if err != nil {
		cancel()
		return nil, err
	}

	// Store cancel function in transaction so it's called on commit/rollback
	tx.cancelFunc = cancel

	return tx, nil
}
