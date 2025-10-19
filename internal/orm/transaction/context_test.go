package transaction

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestFromContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	t.Run("NoTransaction", func(t *testing.T) {
		tx, ok := FromContext(ctx)
		if ok {
			t.Error("expected ok to be false")
		}
		if tx != nil {
			t.Error("expected nil transaction")
		}
	})

	t.Run("WithTransaction", func(t *testing.T) {
		tx, err := mgr.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		defer tx.Rollback()

		txCtx := WithContext(ctx, tx)
		retrievedTx, ok := FromContext(txCtx)

		if !ok {
			t.Error("expected ok to be true")
		}
		if retrievedTx != tx {
			t.Error("retrieved transaction is not the same as original")
		}
	})
}

func TestWithContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Create new context with transaction
	txCtx := WithContext(ctx, tx)

	// Verify transaction can be retrieved
	retrievedTx, ok := FromContext(txCtx)
	if !ok {
		t.Fatal("expected to find transaction in context")
	}

	if retrievedTx != tx {
		t.Error("retrieved transaction is not the same as original")
	}

	// Verify original context is unchanged
	_, ok = FromContext(ctx)
	if ok {
		t.Error("original context should not contain transaction")
	}
}

func TestMustFromContext(t *testing.T) {
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

		txCtx := WithContext(ctx, tx)

		// Should not panic
		retrievedTx := MustFromContext(txCtx)
		if retrievedTx != tx {
			t.Error("retrieved transaction is not the same as original")
		}
	})

	t.Run("NoTransaction", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got nil")
			}
		}()

		// Should panic
		MustFromContext(ctx)
	})
}

func TestTransaction_Context(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Get context from transaction
	txCtx := tx.Context()

	// Verify transaction is in context
	retrievedTx, ok := FromContext(txCtx)
	if !ok {
		t.Fatal("expected to find transaction in context")
	}

	if retrievedTx != tx {
		t.Error("retrieved transaction is not the same as original")
	}
}

func TestContextPropagation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Add some values to context
	type key string
	ctx = context.WithValue(ctx, key("user"), "test_user")
	ctx = context.WithValue(ctx, key("request_id"), "12345")

	tx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	txCtx := tx.Context()

	// Verify original context values are preserved
	if user := txCtx.Value(key("user")); user != "test_user" {
		t.Errorf("expected user value 'test_user', got %v", user)
	}

	if requestID := txCtx.Value(key("request_id")); requestID != "12345" {
		t.Errorf("expected request_id value '12345', got %v", requestID)
	}

	// Verify transaction is also in context
	retrievedTx, ok := FromContext(txCtx)
	if !ok {
		t.Fatal("expected to find transaction in context")
	}

	if retrievedTx != tx {
		t.Error("retrieved transaction is not the same as original")
	}
}

func TestNestedContextPropagation(t *testing.T) {
	// Skip for SQLite - requires PostgreSQL for savepoints
	t.Skip("Skipping nested transaction test for SQLite - requires PostgreSQL")

	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Create parent transaction
	parentTx, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin parent failed: %v", err)
	}
	defer parentTx.Rollback()

	parentCtx := parentTx.Context()

	// Create nested transaction
	nestedTx, err := parentTx.BeginNested(parentCtx)
	if err != nil {
		t.Fatalf("BeginNested failed: %v", err)
	}
	defer nestedTx.Rollback()

	nestedCtx := nestedTx.Context()

	// Verify nested transaction is in context
	retrievedTx, ok := FromContext(nestedCtx)
	if !ok {
		t.Fatal("expected to find transaction in nested context")
	}

	if retrievedTx != nestedTx {
		t.Error("retrieved transaction is not the nested transaction")
	}

	// Verify parent context still has parent transaction
	retrievedParentTx, ok := FromContext(parentCtx)
	if !ok {
		t.Fatal("expected to find parent transaction in parent context")
	}

	if retrievedParentTx != parentTx {
		t.Error("retrieved parent transaction is not correct")
	}
}

func TestMultipleContextValues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr := NewManager(db)
	ctx := context.Background()

	// Add multiple transactions to different contexts
	tx1, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin tx1 failed: %v", err)
	}
	defer tx1.Rollback()

	tx2, err := mgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin tx2 failed: %v", err)
	}
	defer tx2.Rollback()

	ctx1 := WithContext(ctx, tx1)
	ctx2 := WithContext(ctx, tx2)

	// Verify correct transactions are retrieved from each context
	retrieved1, ok := FromContext(ctx1)
	if !ok {
		t.Fatal("expected to find tx1 in ctx1")
	}
	if retrieved1 != tx1 {
		t.Error("ctx1 does not contain tx1")
	}

	retrieved2, ok := FromContext(ctx2)
	if !ok {
		t.Fatal("expected to find tx2 in ctx2")
	}
	if retrieved2 != tx2 {
		t.Error("ctx2 does not contain tx2")
	}

	// Verify they are different
	if retrieved1 == retrieved2 {
		t.Error("expected different transactions in different contexts")
	}
}
