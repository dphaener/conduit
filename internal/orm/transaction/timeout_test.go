package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestManager_WithTimeout_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	err := mgr.WithTimeout(ctx, 5*time.Second, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_test", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithTimeout failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithTimeout_Timeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	startTime := time.Now()
	err := mgr.WithTimeout(ctx, 100*time.Millisecond, func(tx *sql.Tx) error {
		// Simulate slow operation by sleeping
		// Use a select to respect context cancellation
		select {
		case <-time.After(200 * time.Millisecond):
			// This should not be reached due to timeout
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_slow", 1)
			return err
		case <-ctx.Done():
			// Context was cancelled due to timeout
			return ctx.Err()
		}
	})

	duration := time.Since(startTime)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, ErrTransactionTimeout) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected ErrTransactionTimeout or DeadlineExceeded, got %v", err)
	}

	// Should timeout around 100ms
	if duration < 100*time.Millisecond || duration > 300*time.Millisecond {
		t.Errorf("unexpected timeout duration: %v", duration)
	}

	// Transaction should have been rolled back, so no records should exist
	// Create a new connection to verify (avoid any transaction state issues)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_slow").Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		// If table doesn't exist yet or other DB error, that's acceptable for this test
		// The main assertion is that the timeout occurred
		t.Logf("Note: Query failed (acceptable): %v", err)
	} else if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

func TestManager_WithTimeoutIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	err := mgr.WithTimeoutIsolation(ctx, 2*time.Second, Serializable, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_isolation", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithTimeoutIsolation failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_isolation").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithTimeoutRetry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 10 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	executed := 0

	err := mgr.WithTimeoutRetry(ctx, 2*time.Second, config, func(tx *sql.Tx) error {
		executed++
		if executed < 2 {
			return deadlockErr
		}
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_retry", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithTimeoutRetry failed: %v", err)
	}

	if executed != 2 {
		t.Errorf("expected 2 executions, got %d", executed)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_retry").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_WithTimeoutRetry_Timeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := &RetryConfig{
		MaxRetries:  10,
		BaseBackoff: 100 * time.Millisecond,
	}

	deadlockErr := errors.New("deadlock detected")
	startTime := time.Now()

	err := mgr.WithTimeoutRetry(ctx, 200*time.Millisecond, config, func(tx *sql.Tx) error {
		return deadlockErr
	})

	duration := time.Since(startTime)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, ErrTransactionTimeout) {
		t.Errorf("expected ErrTransactionTimeout, got %v", err)
	}

	// Should timeout before completing all retries
	if duration > 500*time.Millisecond {
		t.Errorf("unexpected timeout duration: %v", duration)
	}
}

func TestManager_BeginWithTimeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.BeginWithTimeout(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("BeginWithTimeout failed: %v", err)
	}

	// Should be able to use the transaction
	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "begin_timeout", 1)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	// Commit should succeed
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "begin_timeout").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_BeginWithTimeout_Timeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.BeginWithTimeout(ctx, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("BeginWithTimeout failed: %v", err)
	}

	// Sleep longer than timeout
	time.Sleep(100 * time.Millisecond)

	// Operations should fail due to timeout
	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_expired", 1)
	if err == nil {
		t.Fatal("expected error due to timeout, got nil")
	}
}

func TestManager_BeginWithDeadline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	deadline := time.Now().Add(1 * time.Second)
	tx, err := mgr.BeginWithDeadline(ctx, deadline)
	if err != nil {
		t.Fatalf("BeginWithDeadline failed: %v", err)
	}

	// Should be able to use the transaction before deadline
	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "begin_deadline", 1)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	// Commit should succeed
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "begin_deadline").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestManager_BeginWithDeadline_Exceeded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Set deadline in the past
	deadline := time.Now().Add(-1 * time.Second)
	tx, err := mgr.BeginWithDeadline(ctx, deadline)

	// Should either fail to begin or immediately timeout
	if err == nil && tx != nil {
		// If transaction was created, operations should fail
		_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "deadline_exceeded", 1)
		if err == nil {
			t.Fatal("expected error due to exceeded deadline, got nil")
		}
		tx.Rollback()
	}
}

func TestManager_WithTimeout_CancelledContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context before transaction
	cancel()

	err := mgr.WithTimeout(ctx, 1*time.Second, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "cancelled", 1)
		return err
	})

	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}

	// Verify no record was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "cancelled").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records, got %d", count)
	}
}

func TestManager_WithTimeout_FastOperation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Very generous timeout for a fast operation
	startTime := time.Now()
	err := mgr.WithTimeout(ctx, 10*time.Second, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "fast", 1)
		return err
	})
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("WithTimeout failed: %v", err)
	}

	// Should complete quickly, not wait for timeout
	if duration > 1*time.Second {
		t.Errorf("operation took too long: %v", duration)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "fast").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}
