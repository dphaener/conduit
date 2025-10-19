package transaction

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// HookExecutorWithTransactions wraps a hook executor with transaction support
// This ensures hooks marked with @transaction are executed within a transaction
type HookExecutorWithTransactions struct {
	manager *Manager
	db      *sql.DB
}

// NewHookExecutorWithTransactions creates a new hook executor with transaction support
func NewHookExecutorWithTransactions(db *sql.DB) *HookExecutorWithTransactions {
	return &HookExecutorWithTransactions{
		manager: NewManager(db),
		db:      db,
	}
}

// WrapHookWithTransaction wraps a hook function to execute within a transaction
// if the hook is marked with @transaction attribute
func (h *HookExecutorWithTransactions) WrapHookWithTransaction(
	fn func(ctx context.Context, record map[string]interface{}) error,
	needsTransaction bool,
) func(ctx context.Context, record map[string]interface{}) error {
	if !needsTransaction {
		// No transaction needed, return function as-is
		return fn
	}

	// Return wrapped function that executes within a transaction
	return func(ctx context.Context, record map[string]interface{}) error {
		// Check if there's already a transaction in context
		if _, ok := FromContext(ctx); ok {
			// Already in a transaction, just execute the hook
			return fn(ctx, record)
		}

		// No transaction in context, create one
		return h.manager.WithTransaction(ctx, func(tx *sql.Tx) error {
			// Create context with transaction
			txCtx := WithContext(ctx, &Transaction{
				db:  h.db,
				tx:  tx,
				ctx: ctx,
			})

			// Execute hook within transaction context
			return fn(txCtx, record)
		})
	}
}

// ExecuteHookWithTransaction executes a hook function within a transaction
// This is a convenience method for executing hooks marked with @transaction
func (h *HookExecutorWithTransactions) ExecuteHookWithTransaction(
	ctx context.Context,
	hookType schema.HookType,
	record map[string]interface{},
	fn func(ctx context.Context, record map[string]interface{}) error,
) error {
	// Check if there's already a transaction in context
	if _, ok := FromContext(ctx); ok {
		// Already in a transaction, just execute the hook
		return fn(ctx, record)
	}

	// No transaction in context, create one
	return h.manager.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Create context with transaction
		txCtx := WithContext(ctx, &Transaction{
			db:  h.db,
			tx:  tx,
			ctx: ctx,
		})

		// Execute hook within transaction context
		if err := fn(txCtx, record); err != nil {
			return fmt.Errorf("hook %s failed: %w", hookType.String(), err)
		}

		return nil
	})
}

// GetTransactionFromContext retrieves a transaction from context
// This is a helper for hooks that need to access the transaction
func GetTransactionFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := FromContext(ctx)
	if !ok {
		return nil, false
	}
	return tx.Tx(), true
}

// GetManagerFromContext retrieves a transaction manager from context
// This allows hooks to create nested transactions if needed
func GetManagerFromContext(ctx context.Context, db *sql.DB) *Manager {
	return NewManager(db)
}
