package transaction

import (
	"context"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// contextKeyTransaction is the key for storing a transaction in context
	contextKeyTransaction contextKey = "conduit:transaction"
)

// FromContext retrieves a transaction from the context
// Returns the transaction and true if found, nil and false otherwise
func FromContext(ctx context.Context) (*Transaction, bool) {
	tx, ok := ctx.Value(contextKeyTransaction).(*Transaction)
	return tx, ok
}

// WithContext returns a new context with the transaction embedded
func WithContext(ctx context.Context, tx *Transaction) context.Context {
	return context.WithValue(ctx, contextKeyTransaction, tx)
}

// MustFromContext retrieves a transaction from the context
// Panics if no transaction is found (use only when transaction is guaranteed)
func MustFromContext(ctx context.Context) *Transaction {
	tx, ok := FromContext(ctx)
	if !ok {
		panic("no transaction found in context")
	}
	return tx
}
