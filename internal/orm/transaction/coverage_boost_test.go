package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestCoverage_WithTransactionPanic tests panic recovery in WithTransaction
func TestCoverage_WithTransactionPanic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to be recovered and re-thrown")
		}
	}()

	_ = mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "panic_test", 1)
		if err != nil {
			return err
		}
		panic("test panic in WithTransaction")
	})
}

// TestCoverage_WithTransactionIsolationPanic tests panic recovery with isolation level
func TestCoverage_WithTransactionIsolationPanic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to be recovered and re-thrown")
		}
	}()

	_ = mgr.WithTransactionIsolation(ctx, Serializable, func(tx *sql.Tx) error {
		panic("test panic in WithTransactionIsolation")
	})
}

// TestCoverage_BeginWithTimeoutCommit tests BeginWithTimeout with commit
func TestCoverage_BeginWithTimeoutCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.BeginWithTimeout(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("BeginWithTimeout failed: %v", err)
	}

	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_commit", 1)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Commit should call cancel function
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_commit").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// TestCoverage_BeginWithTimeoutRollback tests BeginWithTimeout with rollback
func TestCoverage_BeginWithTimeoutRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.BeginWithTimeout(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("BeginWithTimeout failed: %v", err)
	}

	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_rollback", 1)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Rollback should call cancel function
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify record doesn't exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_rollback").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

// TestCoverage_BeginWithDeadlineCommit tests BeginWithDeadline with commit
func TestCoverage_BeginWithDeadlineCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	deadline := time.Now().Add(1 * time.Second)
	tx, err := mgr.BeginWithDeadline(ctx, deadline)
	if err != nil {
		t.Fatalf("BeginWithDeadline failed: %v", err)
	}

	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "deadline_commit", 1)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Commit should call cancel function
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "deadline_commit").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// TestCoverage_BeginWithDeadlineRollback tests BeginWithDeadline with rollback
func TestCoverage_BeginWithDeadlineRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	deadline := time.Now().Add(1 * time.Second)
	tx, err := mgr.BeginWithDeadline(ctx, deadline)
	if err != nil {
		t.Fatalf("BeginWithDeadline failed: %v", err)
	}

	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "deadline_rollback", 1)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Rollback should call cancel function
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify record doesn't exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "deadline_rollback").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

// TestCoverage_WithTimeoutRollbackError tests WithTimeout with rollback error
func TestCoverage_WithTimeoutRollbackError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	testErr := errors.New("test error")

	err := mgr.WithTimeout(ctx, 1*time.Second, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "timeout_err", 1)
		if err != nil {
			return err
		}
		return testErr
	})

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	// Verify record was rolled back
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "timeout_err").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

// TestCoverage_WithTimeoutIsolationError tests WithTimeoutIsolation with error
func TestCoverage_WithTimeoutIsolationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	testErr := errors.New("test error")

	err := mgr.WithTimeoutIsolation(ctx, 1*time.Second, RepeatableRead, func(tx *sql.Tx) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}
}

// TestCoverage_WithTimeoutRetryError tests WithTimeoutRetry with error
func TestCoverage_WithTimeoutRetryError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := DefaultRetryConfig()
	testErr := errors.New("test error")

	err := mgr.WithTimeoutRetry(ctx, 1*time.Second, config, func(tx *sql.Tx) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}
}

// TestCoverage_ToSQLOptionsDefault tests ToSQLOptions default case
func TestCoverage_ToSQLOptionsDefault(t *testing.T) {
	level := IsolationLevel(999) // Invalid level
	opts := level.ToSQLOptions()

	if opts == nil {
		t.Fatal("expected non-nil TxOptions")
	}

	// Should default to ReadCommitted
	if opts.Isolation != sql.LevelReadCommitted {
		t.Errorf("expected LevelReadCommitted for invalid level, got %v", opts.Isolation)
	}
}

// TestCoverage_IsolationLevelStringDefault tests IsolationLevel.String default case
func TestCoverage_IsolationLevelStringDefault(t *testing.T) {
	level := IsolationLevel(999) // Invalid level
	str := level.String()

	// Should default to "READ COMMITTED"
	if str != "READ COMMITTED" {
		t.Errorf("expected 'READ COMMITTED' for invalid level, got '%s'", str)
	}
}

// TestCoverage_BeginWithTimeoutError tests BeginWithTimeout with immediate error
func TestCoverage_BeginWithTimeoutError(t *testing.T) {
	// Create a database that will fail on Begin
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.Close() // Close it immediately to cause error

	mgr := NewManager(db)
	ctx := context.Background()

	_, err = mgr.BeginWithTimeout(ctx, 1*time.Second)
	if err == nil {
		t.Fatal("expected error when beginning transaction on closed database")
	}
}

// TestCoverage_BeginWithDeadlineError tests BeginWithDeadline with immediate error
func TestCoverage_BeginWithDeadlineError(t *testing.T) {
	// Create a database that will fail on Begin
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.Close() // Close it immediately to cause error

	mgr := NewManager(db)
	ctx := context.Background()

	deadline := time.Now().Add(1 * time.Second)
	_, err = mgr.BeginWithDeadline(ctx, deadline)
	if err == nil {
		t.Fatal("expected error when beginning transaction on closed database")
	}
}
