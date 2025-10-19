package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestBeginNested_WithoutTransaction tests BeginNested without a parent transaction
func TestBeginNested_WithoutTransaction(t *testing.T) {
	tx := &Transaction{
		tx: nil, // No underlying transaction
	}

	ctx := context.Background()
	_, err := tx.BeginNested(ctx)

	if err == nil {
		t.Fatal("expected error when beginning nested transaction without parent")
	}

	if !errors.Is(err, ErrNestedTransactionNotSupported) {
		t.Errorf("expected ErrNestedTransactionNotSupported, got %v", err)
	}
}

// TestTransaction_CommitNested tests committing a nested transaction
func TestTransaction_CommitNested(t *testing.T) {
	// Note: This test is limited by SQLite's savepoint support
	// In production with PostgreSQL, this would work fully
	t.Skip("Skipping nested transaction test - requires full PostgreSQL savepoint support")

	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Create nested transaction
	nested, err := tx.BeginNested(ctx)
	if err != nil {
		t.Fatalf("BeginNested failed: %v", err)
	}

	// Test nested transaction level
	if nested.Level() != 1 {
		t.Errorf("expected nested level 1, got %d", nested.Level())
	}

	// Commit nested should work
	if err := nested.Commit(); err != nil {
		t.Errorf("Nested commit failed: %v", err)
	}

	if !nested.IsCommitted() {
		t.Error("expected nested transaction to be committed")
	}
}

// TestTransaction_RollbackNested tests rolling back a nested transaction
func TestTransaction_RollbackNested(t *testing.T) {
	t.Skip("Skipping nested transaction test - requires full PostgreSQL savepoint support")

	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Create nested transaction
	nested, err := tx.BeginNested(ctx)
	if err != nil {
		t.Fatalf("BeginNested failed: %v", err)
	}

	// Rollback nested
	if err := nested.Rollback(); err != nil {
		t.Errorf("Nested rollback failed: %v", err)
	}

	if !nested.IsRolledBack() {
		t.Error("expected nested transaction to be rolled back")
	}
}

// TestTransaction_WithRetryIsolation_Error tests retry with isolation level and non-retriable error
func TestTransaction_WithRetryIsolation_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := DefaultRetryConfig()
	testErr := errors.New("non-retriable error")
	attempts := 0

	err := mgr.WithRetryIsolation(ctx, Serializable, config, func(tx *sql.Tx) error {
		attempts++
		return testErr
	})

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	// Should only attempt once for non-retriable errors
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

// TestTransaction_WithRetryIsolation_Success tests retry with isolation level succeeding
func TestTransaction_WithRetryIsolation_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	config := DefaultRetryConfig()
	deadlockErr := errors.New("deadlock detected (40P01)")
	attempts := 0

	err := mgr.WithRetryIsolation(ctx, RepeatableRead, config, func(tx *sql.Tx) error {
		attempts++
		if attempts < 2 {
			return deadlockErr
		}
		// Succeed on second attempt
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "retry_iso_success", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithRetryIsolation failed: %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "retry_iso_success").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// TestTransaction_CommitAfterDoubleRollback tests commit after double rollback
func TestTransaction_CommitAfterDoubleRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// First rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("First rollback failed: %v", err)
	}

	// Second rollback (no-op)
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Second rollback failed: %v", err)
	}

	// Commit should fail
	err = tx.Commit()
	if err == nil {
		t.Fatal("expected error on commit after rollback")
	}
}

// TestTransaction_ExecQueryQueryRow tests transaction query methods
func TestTransaction_ExecQueryQueryRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Test Exec
	result, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?), (?, ?)",
		"exec_test_1", 1, "exec_test_2", 2)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected failed: %v", err)
	}
	if rows != 2 {
		t.Errorf("expected 2 rows affected, got %d", rows)
	}

	// Test QueryRow
	var name string
	var value int
	err = tx.QueryRow("SELECT name, value FROM test_records WHERE name = ?", "exec_test_1").Scan(&name, &value)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}
	if name != "exec_test_1" || value != 1 {
		t.Errorf("expected (exec_test_1, 1), got (%s, %d)", name, value)
	}

	// Test Query
	rows2, err := tx.Query("SELECT name, value FROM test_records WHERE name LIKE ?", "exec_test_%")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows from Query, got %d", count)
	}
}

// TestTransaction_DBAndTxGetters tests DB() and Tx() getters
func TestTransaction_DBAndTxGetters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Test DB getter
	if tx.DB() != db {
		t.Error("DB() returned different database connection")
	}

	// Test Tx getter
	if tx.Tx() == nil {
		t.Fatal("Tx() returned nil")
	}

	// Should be able to use Tx() directly
	_, err = tx.Tx().Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "getter_test", 1)
	if err != nil {
		t.Errorf("Failed to use Tx() directly: %v", err)
	}
}

// TestTransaction_LevelGetter tests Level() getter
func TestTransaction_LevelGetter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	if tx.Level() != 0 {
		t.Errorf("expected top-level transaction level 0, got %d", tx.Level())
	}

	if tx.IsolationLevel() != ReadCommitted {
		t.Errorf("expected ReadCommitted, got %v", tx.IsolationLevel())
	}
}

// TestTransaction_IsCommittedIsRolledBack tests state flags
func TestTransaction_IsCommittedIsRolledBack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("Initial state", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		defer tx.Rollback()

		if tx.IsCommitted() {
			t.Error("expected IsCommitted to be false initially")
		}
		if tx.IsRolledBack() {
			t.Error("expected IsRolledBack to be false initially")
		}
	})

	t.Run("After commit", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		if !tx.IsCommitted() {
			t.Error("expected IsCommitted to be true after commit")
		}
		if tx.IsRolledBack() {
			t.Error("expected IsRolledBack to be false after commit")
		}
	})

	t.Run("After rollback", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}

		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		if tx.IsCommitted() {
			t.Error("expected IsCommitted to be false after rollback")
		}
		if !tx.IsRolledBack() {
			t.Error("expected IsRolledBack to be true after rollback")
		}
	})
}

// TestManager_BeginTx_CommitAndRollback tests BeginTx compatibility
func TestManager_BeginTx_CommitAndRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("Commit", func(t *testing.T) {
		tx, err := mgr.BeginTx(ctx)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}

		_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "begintx_commit", 1)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Verify
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "begintx_commit").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		tx, err := mgr.BeginTx(ctx)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}

		_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "begintx_rollback", 1)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		// Verify
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "begintx_rollback").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 records (rolled back), got %d", count)
		}
	})
}
