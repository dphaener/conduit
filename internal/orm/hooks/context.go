package hooks

import (
	"context"
	"database/sql"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Context wraps the standard context with ORM-specific information
// for hook execution
type Context struct {
	context.Context
	db       *sql.DB
	tx       *sql.Tx
	resource *schema.ResourceSchema
}

// NewContext creates a new hook context
func NewContext(ctx context.Context, db *sql.DB, resource *schema.ResourceSchema) *Context {
	return &Context{
		Context:  ctx,
		db:       db,
		resource: resource,
	}
}

// WithTransaction creates a new context with a transaction
func (c *Context) WithTransaction(tx *sql.Tx) *Context {
	return &Context{
		Context:  c.Context,
		db:       c.db,
		tx:       tx,
		resource: c.resource,
	}
}

// DB returns the database connection
func (c *Context) DB() *sql.DB {
	return c.db
}

// Tx returns the transaction (may be nil)
func (c *Context) Tx() *sql.Tx {
	return c.tx
}

// Resource returns the resource schema
func (c *Context) Resource() *schema.ResourceSchema {
	return c.resource
}

// HasTransaction returns true if a transaction is active
func (c *Context) HasTransaction() bool {
	return c.tx != nil
}
