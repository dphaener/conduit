package hooks

import (
	"context"
	"database/sql"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestNewContext(t *testing.T) {
	ctx := context.Background()
	db := &sql.DB{}
	resource := schema.NewResourceSchema("Test")

	hookCtx := NewContext(ctx, db, resource)

	if hookCtx.DB() != db {
		t.Error("DB not set correctly")
	}

	if hookCtx.Resource() != resource {
		t.Error("Resource not set correctly")
	}

	if hookCtx.Tx() != nil {
		t.Error("Transaction should be nil")
	}

	if hookCtx.HasTransaction() {
		t.Error("Should not have transaction")
	}
}

func TestContext_WithTransaction(t *testing.T) {
	ctx := context.Background()
	db := &sql.DB{}
	resource := schema.NewResourceSchema("Test")

	hookCtx := NewContext(ctx, db, resource)

	// Create a transaction (nil is fine for testing)
	tx := &sql.Tx{}
	txCtx := hookCtx.WithTransaction(tx)

	if txCtx.Tx() != tx {
		t.Error("Transaction not set correctly")
	}

	if !txCtx.HasTransaction() {
		t.Error("Should have transaction")
	}

	// Original context should not be modified
	if hookCtx.HasTransaction() {
		t.Error("Original context should not have transaction")
	}

	// DB and resource should be preserved
	if txCtx.DB() != db {
		t.Error("DB not preserved")
	}

	if txCtx.Resource() != resource {
		t.Error("Resource not preserved")
	}
}

func TestContext_ContextInterface(t *testing.T) {
	ctx := context.Background()
	db := &sql.DB{}
	resource := schema.NewResourceSchema("Test")

	hookCtx := NewContext(ctx, db, resource)

	// Should implement context.Context interface
	var _ context.Context = hookCtx

	// Test Done channel
	select {
	case <-hookCtx.Done():
		t.Error("Context should not be done")
	default:
		// Expected
	}

	// Test Err
	if hookCtx.Err() != nil {
		t.Error("Context should not have error")
	}
}

func TestContext_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	db := &sql.DB{}
	resource := schema.NewResourceSchema("Test")

	hookCtx := NewContext(ctx, db, resource)

	// Should reflect canceled state
	if hookCtx.Err() != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", hookCtx.Err())
	}

	select {
	case <-hookCtx.Done():
		// Expected
	default:
		t.Error("Context should be done")
	}
}

func TestContext_NilValues(t *testing.T) {
	ctx := context.Background()

	// Test with nil DB
	hookCtx := NewContext(ctx, nil, nil)

	if hookCtx.DB() != nil {
		t.Error("DB should be nil")
	}

	if hookCtx.Resource() != nil {
		t.Error("Resource should be nil")
	}
}
