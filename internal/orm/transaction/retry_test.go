package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestIsDeadlockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "PostgreSQL deadlock code",
			err:      errors.New("pq: deadlock detected (SQLSTATE 40P01)"),
			expected: true,
		},
		{
			name:     "deadlock detected message",
			err:      errors.New("deadlock detected while waiting for resource"),
			expected: true,
		},
		{
			name:     "deadlock found message",
			err:      errors.New("ERROR: deadlock found when trying to get lock"),
			expected: true,
		},
		{
			name:     "lock wait timeout",
			err:      errors.New("lock wait timeout exceeded; try restarting transaction"),
			expected: true,
		},
		{
			name:     "serialization failure",
			err:      errors.New("could not serialize access due to concurrent update"),
			expected: true,
		},
		{
			name:     "non-deadlock error",
			err:      errors.New("some other database error"),
			expected: false,
		},
		{
			name:     "constraint violation",
			err:      errors.New("unique constraint violation"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDeadlockError(tt.err)
			if got != tt.expected {
				t.Errorf("isDeadlockError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsSerializationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "PostgreSQL serialization code",
			err:      errors.New("pq: could not serialize access (SQLSTATE 40001)"),
			expected: true,
		},
		{
			name:     "serialization message",
			err:      errors.New("could not serialize access due to concurrent update"),
			expected: true,
		},
		{
			name:     "non-serialization error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSerializationError(tt.err)
			if got != tt.expected {
				t.Errorf("isSerializationError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "deadlock error",
			err:      errors.New("deadlock detected"),
			expected: true,
		},
		{
			name:     "serialization error",
			err:      errors.New("could not serialize access (SQLSTATE 40001)"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("constraint violation"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected MaxRetries %d, got %d", DefaultMaxRetries, config.MaxRetries)
	}

	if config.BaseBackoff != DefaultBaseBackoff {
		t.Errorf("expected BaseBackoff %v, got %v", DefaultBaseBackoff, config.BaseBackoff)
	}
}

func TestManager_WithRetry_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	executed := 0
	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		executed++
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "retry_test", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithRetry failed: %v", err)
	}

	if executed != 1 {
		t.Errorf("expected function to execute once, executed %d times", executed)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "retry_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithRetry_NonDeadlockError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	testErr := errors.New("non-deadlock error")
	executed := 0

	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		executed++
		return testErr
	})

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	// Should only execute once for non-deadlock errors
	if executed != 1 {
		t.Errorf("expected function to execute once, executed %d times", executed)
	}
}

func TestManager_WithRetry_DeadlockRetry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	deadlockErr := errors.New("deadlock detected (SQLSTATE 40P01)")
	executed := 0
	maxAttempts := 2

	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		executed++
		if executed < maxAttempts {
			return deadlockErr
		}
		// Succeed on second attempt
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "retry_deadlock", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithRetry failed: %v", err)
	}

	if executed != maxAttempts {
		t.Errorf("expected function to execute %d times, executed %d times", maxAttempts, executed)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "retry_deadlock").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithRetry_ExceedMaxRetries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}

	if !errors.Is(err, ErrDeadlock) {
		t.Errorf("expected ErrDeadlock, got %v", err)
	}

	if executed != DefaultMaxRetries {
		t.Errorf("expected function to execute %d times, executed %d times", DefaultMaxRetries, executed)
	}
}

func TestManager_WithRetryConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := &RetryConfig{
		MaxRetries:  5,
		BaseBackoff: 10 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithRetryConfig(ctx, config, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}

	if executed != config.MaxRetries {
		t.Errorf("expected function to execute %d times, executed %d times", config.MaxRetries, executed)
	}
}

func TestManager_WithRetryConfig_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx, cancel := context.WithCancel(context.Background())

	config := &RetryConfig{
		MaxRetries:  10,
		BaseBackoff: 100 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	// Cancel context after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := mgr.WithRetryConfig(ctx, config, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should have been cancelled before reaching max retries
	if executed >= config.MaxRetries {
		t.Errorf("expected fewer than %d executions, got %d", config.MaxRetries, executed)
	}
}

func TestManager_WithRetryConfig_ContextCancelledBeforeRetry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	config := &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 10 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithRetryConfig(ctx, config, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should check context before first attempt and fail immediately
	if executed != 0 {
		t.Errorf("expected 0 executions with cancelled context, got %d", executed)
	}

	// Error should mention context cancellation
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestManager_WithRetryIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := DefaultRetryConfig()
	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithRetryIsolation(ctx, Serializable, config, func(tx *sql.Tx) error {
		executed++
		if executed < 2 {
			return deadlockErr
		}
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "retry_isolation", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithRetryIsolation failed: %v", err)
	}

	if executed != 2 {
		t.Errorf("expected 2 executions, got %d", executed)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "retry_isolation").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithRetryIsolation_ContextCancelledBeforeRetry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	config := &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 10 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithRetryIsolation(ctx, Serializable, config, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should check context before first attempt and fail immediately
	if executed != 0 {
		t.Errorf("expected 0 executions with cancelled context, got %d", executed)
	}

	// Error should mention context cancellation
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestManager_WithRetry_BackoffTiming(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 50 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	startTime := time.Now()
	executed := 0

	err := mgr.WithRetryConfig(ctx, config, func(tx *sql.Tx) error {
		executed++
		return deadlockErr
	})

	duration := time.Since(startTime)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Expected backoff: 50ms (attempt 1), 100ms (attempt 2) = 150ms minimum
	// With some tolerance for execution time
	minExpected := 150 * time.Millisecond
	maxExpected := 500 * time.Millisecond

	if duration < minExpected {
		t.Errorf("expected duration >= %v, got %v", minExpected, duration)
	}

	if duration > maxExpected {
		t.Errorf("expected duration <= %v, got %v (might indicate performance issue)", maxExpected, duration)
	}

	if executed != config.MaxRetries {
		t.Errorf("expected %d executions, got %d", config.MaxRetries, executed)
	}
}
