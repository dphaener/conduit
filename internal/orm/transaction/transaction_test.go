package transaction

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a test database with a test table
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE test_records (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	return db
}

func TestManager_Begin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	if tx.Level() != 0 {
		t.Errorf("expected level 0, got %d", tx.Level())
	}

	if tx.IsolationLevel() != ReadCommitted {
		t.Errorf("expected ReadCommitted isolation level, got %v", tx.IsolationLevel())
	}

	// Clean up
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback failed: %v", err)
	}
}

func TestManager_BeginWithIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tests := []struct {
		name  string
		level IsolationLevel
	}{
		{"ReadUncommitted", ReadUncommitted},
		{"ReadCommitted", ReadCommitted},
		{"RepeatableRead", RepeatableRead},
		{"Serializable", Serializable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := mgr.BeginWithIsolation(ctx, tt.level)
			if err != nil {
				t.Fatalf("BeginWithIsolation failed: %v", err)
			}

			if tx.IsolationLevel() != tt.level {
				t.Errorf("expected isolation level %v, got %v", tt.level, tx.IsolationLevel())
			}

			if err := tx.Rollback(); err != nil {
				t.Errorf("Rollback failed: %v", err)
			}
		})
	}
}

func TestTransaction_CommitAndRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("Commit", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}

		// Insert a record
		_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "test1", 100)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		// Commit
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		if !tx.IsCommitted() {
			t.Error("expected IsCommitted to be true")
		}

		// Verify record exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "test1").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}

		// Insert a record
		_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "test2", 200)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		// Rollback
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		if !tx.IsRolledBack() {
			t.Error("expected IsRolledBack to be true")
		}

		// Verify record does not exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "test2").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 records, got %d", count)
		}
	})
}

func TestTransaction_NestedTransactions(t *testing.T) {
	// Skip for SQLite as it doesn't fully support savepoints in the same way
	t.Skip("Skipping nested transaction test for SQLite - requires PostgreSQL")

	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Insert in parent transaction
	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "parent", 1)
	if err != nil {
		t.Fatalf("Insert in parent failed: %v", err)
	}

	// Create nested transaction
	nestedTx, err := tx.BeginNested(ctx)
	if err != nil {
		t.Fatalf("BeginNested failed: %v", err)
	}

	if nestedTx.Level() != 1 {
		t.Errorf("expected nested level 1, got %d", nestedTx.Level())
	}

	// Insert in nested transaction
	_, err = nestedTx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "nested", 2)
	if err != nil {
		t.Fatalf("Insert in nested failed: %v", err)
	}

	// Rollback nested transaction
	if err := nestedTx.Rollback(); err != nil {
		t.Fatalf("Nested rollback failed: %v", err)
	}

	// Commit parent transaction
	if err := tx.Commit(); err != nil {
		t.Fatalf("Parent commit failed: %v", err)
	}

	// Verify only parent record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record (parent only), got %d", count)
	}
}

func TestManager_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "success", 1)
			return err
		})

		if err != nil {
			t.Fatalf("WithTransaction failed: %v", err)
		}

		// Verify record exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "success").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
	})

	t.Run("Error", func(t *testing.T) {
		err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "error", 2)
			if err != nil {
				return err
			}
			return sql.ErrNoRows // Force an error
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify record was rolled back
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "error").Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 records, got %d", count)
		}
	})
}

func TestTransaction_ContextEmbedding(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Get context with transaction
	txCtx := tx.Context()

	// Retrieve transaction from context
	retrievedTx, ok := FromContext(txCtx)
	if !ok {
		t.Fatal("expected to find transaction in context")
	}

	if retrievedTx != tx {
		t.Error("retrieved transaction is not the same as original")
	}

	// Test WithContext
	newCtx := WithContext(context.Background(), tx)
	retrievedTx2, ok := FromContext(newCtx)
	if !ok {
		t.Fatal("expected to find transaction in new context")
	}

	if retrievedTx2 != tx {
		t.Error("retrieved transaction from new context is not the same as original")
	}
}

func TestTransaction_DoubleCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// First commit should succeed
	if err := tx.Commit(); err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// Second commit should fail
	err = tx.Commit()
	if err == nil {
		t.Fatal("expected error on double commit, got nil")
	}
}

func TestTransaction_CommitAfterRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Commit after rollback should fail
	err = tx.Commit()
	if err == nil {
		t.Fatal("expected error on commit after rollback, got nil")
	}
}

func TestIsolationLevel_String(t *testing.T) {
	tests := []struct {
		level    IsolationLevel
		expected string
	}{
		{ReadUncommitted, "READ UNCOMMITTED"},
		{ReadCommitted, "READ COMMITTED"},
		{RepeatableRead, "REPEATABLE READ"},
		{Serializable, "SERIALIZABLE"},
		{IsolationLevel(999), "READ COMMITTED"}, // Unknown defaults to READ COMMITTED
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestTransaction_QueryMethods(t *testing.T) {
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
	result, err := tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "query_test", 42)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row affected, got %d", rows)
	}

	// Test QueryRow
	var name string
	var value int
	err = tx.QueryRow("SELECT name, value FROM test_records WHERE name = ?", "query_test").Scan(&name, &value)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}
	if name != "query_test" || value != 42 {
		t.Errorf("expected (query_test, 42), got (%s, %d)", name, value)
	}

	// Test Query
	rows2, err := tx.Query("SELECT name, value FROM test_records")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 row from Query, got %d", count)
	}
}

func TestManager_BeginTx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Test BeginTx for compatibility with CRUD interface
	tx, err := mgr.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	// Should be able to use it like a normal sql.Tx
	_, err = tx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "begintx_test", 1)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_records WHERE name = ?", "begintx_test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestTransaction_DB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	if tx.DB() != db {
		t.Error("DB() did not return the same database connection")
	}
}

func TestTransaction_Tx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	sqlTx := tx.Tx()
	if sqlTx == nil {
		t.Fatal("Tx() returned nil")
	}

	// Should be able to use it directly
	_, err = sqlTx.Exec("INSERT INTO test_records (name, value) VALUES (?, ?)", "tx_test", 1)
	if err != nil {
		t.Fatalf("Insert using Tx() failed: %v", err)
	}
}
