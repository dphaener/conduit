package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
	_ "github.com/mattn/go-sqlite3"
)

func TestHookExecutorWithTransactions_WrapHookWithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	executor := NewHookExecutorWithTransactions(db)
	ctx := context.Background()

	t.Run("WithTransaction", func(t *testing.T) {
		executed := false
		record := map[string]interface{}{"name": "test"}

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			executed = true

			// Should have transaction in context
			tx, ok := GetTransactionFromContext(ctx)
			if !ok {
				t.Error("expected transaction in context")
			}
			if tx == nil {
				t.Error("expected non-nil transaction")
			}

			// Use the transaction
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "wrapped_hook", 1)
			return err
		}

		wrappedFn := executor.WrapHookWithTransaction(hookFn, true)

		err := wrappedFn(ctx, record)
		if err != nil {
			t.Fatalf("wrapped hook failed: %v", err)
		}

		if !executed {
			t.Error("hook was not executed")
		}

		// Verify record was inserted
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "wrapped_hook").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})

	t.Run("WithoutTransaction", func(t *testing.T) {
		executed := false
		record := map[string]interface{}{"name": "test"}

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			executed = true
			// Should NOT have transaction in context
			_, ok := GetTransactionFromContext(ctx)
			if ok {
				t.Error("expected no transaction in context")
			}
			return nil
		}

		wrappedFn := executor.WrapHookWithTransaction(hookFn, false)

		err := wrappedFn(ctx, record)
		if err != nil {
			t.Fatalf("wrapped hook failed: %v", err)
		}

		if !executed {
			t.Error("hook was not executed")
		}
	})

	t.Run("ExistingTransaction", func(t *testing.T) {
		// Start a transaction first
		mgr := NewManager(db)
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		defer tx.Rollback()

		txCtx := tx.Context()
		executed := false
		record := map[string]interface{}{"name": "test"}

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			executed = true
			// Should use existing transaction
			retrievedTx, ok := GetTransactionFromContext(ctx)
			if !ok {
				t.Error("expected transaction in context")
			}
			if retrievedTx == nil {
				t.Error("expected non-nil transaction")
			}
			return nil
		}

		wrappedFn := executor.WrapHookWithTransaction(hookFn, true)

		err = wrappedFn(txCtx, record)
		if err != nil {
			t.Fatalf("wrapped hook failed: %v", err)
		}

		if !executed {
			t.Error("hook was not executed")
		}
	})
}

func TestHookExecutorWithTransactions_ExecuteHookWithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	executor := NewHookExecutorWithTransactions(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		record := map[string]interface{}{"name": "test"}

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			tx, ok := GetTransactionFromContext(ctx)
			if !ok {
				t.Error("expected transaction in context")
			}

			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "execute_hook", 1)
			return err
		}

		err := executor.ExecuteHookWithTransaction(ctx, schema.BeforeCreate, record, hookFn)
		if err != nil {
			t.Fatalf("ExecuteHookWithTransaction failed: %v", err)
		}

		// Verify record was inserted
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "execute_hook").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})

	t.Run("Error", func(t *testing.T) {
		record := map[string]interface{}{"name": "test"}
		testErr := errors.New("test error")

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			tx, ok := GetTransactionFromContext(ctx)
			if !ok {
				t.Error("expected transaction in context")
			}

			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "execute_error", 1)
			if err != nil {
				return err
			}
			return testErr
		}

		err := executor.ExecuteHookWithTransaction(ctx, schema.AfterCreate, record, hookFn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify record was rolled back
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "execute_error").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 records (rolled back), got %d", count)
		}
	})

	t.Run("ExistingTransaction", func(t *testing.T) {
		mgr := NewManager(db)
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		defer tx.Rollback()

		txCtx := tx.Context()
		record := map[string]interface{}{"name": "test"}

		hookFn := func(ctx context.Context, record map[string]interface{}) error {
			retrievedTx, ok := GetTransactionFromContext(ctx)
			if !ok {
				t.Error("expected transaction in context")
			}

			_, err := retrievedTx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "existing_tx", 1)
			return err
		}

		err = executor.ExecuteHookWithTransaction(txCtx, schema.BeforeUpdate, record, hookFn)
		if err != nil {
			t.Fatalf("ExecuteHookWithTransaction failed: %v", err)
		}

		// Commit parent transaction
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Verify record was inserted
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "existing_tx").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})
}

func TestGetTransactionFromContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("WithTransaction", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		defer tx.Rollback()

		txCtx := tx.Context()

		retrievedTx, ok := GetTransactionFromContext(txCtx)
		if !ok {
			t.Fatal("expected transaction in context")
		}
		if retrievedTx == nil {
			t.Error("expected non-nil transaction")
		}

		// Should be the same transaction
		if retrievedTx != tx.Tx() {
			t.Error("retrieved transaction is different from original")
		}
	})

	t.Run("WithoutTransaction", func(t *testing.T) {
		retrievedTx, ok := GetTransactionFromContext(ctx)
		if ok {
			t.Error("expected no transaction in context")
		}
		if retrievedTx != nil {
			t.Error("expected nil transaction")
		}
	})
}

func TestGetManagerFromContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	mgr := GetManagerFromContext(ctx, db)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// Should be able to use the manager
	err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "manager_test", 1)
		return err
	})

	if err != nil {
		t.Fatalf("WithTransaction failed: %v", err)
	}

	// Verify record was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "manager_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestHookExecutorWithTransactions_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	executor := NewHookExecutorWithTransactions(db)
	ctx := context.Background()

	// Simulate a complete hook lifecycle with @transaction attribute
	record := map[string]interface{}{
		"name":  "integration_test",
		"value": 42,
	}

	// Before create hook with @transaction
	beforeCreateFn := func(ctx context.Context, record map[string]interface{}) error {
		tx, ok := GetTransactionFromContext(ctx)
		if !ok {
			return errors.New("expected transaction in context")
		}

		// Modify record before insert
		record["value"] = record["value"].(int) * 2

		// Do some database work
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "before_hook", 1)
		return err
	}

	// After create hook with @transaction
	afterCreateFn := func(ctx context.Context, record map[string]interface{}) error {
		tx, ok := GetTransactionFromContext(ctx)
		if !ok {
			return errors.New("expected transaction in context")
		}

		// Do some post-insert work
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "after_hook", 2)
		return err
	}

	// Execute hooks within the same transaction
	err := executor.manager.WithTransaction(ctx, func(tx *sql.Tx) error {
		txCtx := WithContext(ctx, &Transaction{
			db:  db,
			tx:  tx,
			ctx: ctx,
		})

		// Execute before hook
		if err := beforeCreateFn(txCtx, record); err != nil {
			return err
		}

		// Insert main record
		_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)",
			record["name"], record["value"])
		if err != nil {
			return err
		}

		// Execute after hook
		if err := afterCreateFn(txCtx, record); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	// Verify all records were inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	// Verify record value was modified by before hook
	var value int
	err = db.QueryRow("SELECT value FROM test_records WHERE name = ?", "integration_test").Scan(&value)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if value != 84 { // 42 * 2
		t.Errorf("expected value 84, got %d", value)
	}
}
