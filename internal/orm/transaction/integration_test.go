package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestIntegration_CompleteWorkflow tests a complete transaction workflow
func TestIntegration_CompleteWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Test full workflow with multiple operations
	err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Insert multiple records
		for i := 1; i <= 3; i++ {
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)",
				"workflow_test", i*10)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify all records were inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "workflow_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}
}

// TestIntegration_PanicRecovery tests transaction rollback on panic
func TestIntegration_PanicRecovery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Function should panic and transaction should be rolled back
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to be recovered")
			}
		}()

		_ = mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "panic_test", 1)
			if err != nil {
				return err
			}
			panic("test panic")
		})
	}()

	// Verify record was rolled back
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "panic_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

// TestIntegration_MultipleTransactions tests multiple sequential transactions
func TestIntegration_MultipleTransactions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Run 5 sequential transactions
	numTx := 5
	for i := 0; i < numTx; i++ {
		err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)",
				"multi_test", i)
			return err
		})
		if err != nil {
			t.Errorf("Transaction %d failed: %v", i, err)
		}
	}

	// Verify all records were inserted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "multi_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != numTx {
		t.Errorf("expected %d records, got %d", numTx, count)
	}
}

// TestIntegration_RetryWithSuccess tests retry mechanism with eventual success
func TestIntegration_RetryWithSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	attempts := 0
	deadlockErr := errors.New("deadlock detected (SQLSTATE 40P01)")

	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		attempts++
		if attempts < 2 {
			return deadlockErr
		}
		// Succeed on second attempt
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "retry_success", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithRetry failed: %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "retry_success").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// TestIntegration_IsolationLevels tests different isolation levels
func TestIntegration_IsolationLevels(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	levels := []IsolationLevel{
		ReadCommitted,
		RepeatableRead,
		Serializable,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			err := mgr.WithTransactionIsolation(ctx, level, func(tx *sql.Tx) error {
				_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)",
					"isolation_test", int(level))
				return err
			})

			if err != nil {
				t.Errorf("Transaction with %s failed: %v", level.String(), err)
			}
		})
	}

	// Verify all records were inserted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "isolation_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != len(levels) {
		t.Errorf("expected %d records, got %d", len(levels), count)
	}
}

// TestEdgeCase_DoubleRollback tests calling rollback twice
func TestEdgeCase_DoubleRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// First rollback should succeed
	if err := tx.Rollback(); err != nil {
		t.Fatalf("First rollback failed: %v", err)
	}

	// Second rollback should be a no-op (no error)
	if err := tx.Rollback(); err != nil {
		t.Errorf("Second rollback should not error, got: %v", err)
	}

	if !tx.IsRolledBack() {
		t.Error("expected IsRolledBack to be true")
	}
}

// TestEdgeCase_RollbackAfterCommit tests rollback after commit
func TestEdgeCase_RollbackAfterCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Rollback after commit should fail
	err = tx.Rollback()
	if err == nil {
		t.Error("expected error on rollback after commit")
	}
}

// TestEdgeCase_EmptyTransaction tests transaction with no operations
func TestEdgeCase_EmptyTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Empty transaction should commit successfully
	err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Do nothing
		return nil
	})

	if err != nil {
		t.Errorf("Empty transaction failed: %v", err)
	}
}

// TestEdgeCase_TransactionWithoutContextValue tests transaction without stored context value
func TestEdgeCase_TransactionWithoutContextValue(t *testing.T) {
	ctx := context.Background()

	// Should return nil and false
	tx, ok := FromContext(ctx)
	if ok {
		t.Error("expected ok to be false for context without transaction")
	}
	if tx != nil {
		t.Error("expected nil transaction")
	}
}

// TestEdgeCase_ContextPropagationChain tests context propagation through multiple layers
func TestEdgeCase_ContextPropagationChain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Chain contexts
	ctx1 := tx.Context()
	ctx2 := context.WithValue(ctx1, "key", "value")
	ctx3 := context.WithValue(ctx2, "key2", "value2")

	// Should be able to retrieve transaction from any context in the chain
	retrievedTx, ok := FromContext(ctx3)
	if !ok {
		t.Fatal("expected to find transaction in chained context")
	}

	if retrievedTx != tx {
		t.Error("retrieved transaction is not the same as original")
	}

	// Should also retrieve custom values
	if val := ctx3.Value("key"); val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
	if val := ctx3.Value("key2"); val != "value2" {
		t.Errorf("expected 'value2', got %v", val)
	}
}

// TestIntegration_WithTransactionIsolation tests WithTransactionIsolation with error
func TestIntegration_WithTransactionIsolationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	testErr := errors.New("test error")

	err := mgr.WithTransactionIsolation(ctx, Serializable, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "iso_error", 1)
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
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "iso_error").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records (rolled back), got %d", count)
	}
}

// TestEdgeCase_ToSQLOptionsAllLevels tests ToSQLOptions for all isolation levels
func TestEdgeCase_ToSQLOptionsAllLevels(t *testing.T) {
	levels := []IsolationLevel{
		ReadUncommitted,
		ReadCommitted,
		RepeatableRead,
		Serializable,
		IsolationLevel(999), // Unknown level (should default to ReadCommitted)
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			opts := level.ToSQLOptions()
			if opts == nil {
				t.Error("expected non-nil TxOptions")
			}
		})
	}
}
