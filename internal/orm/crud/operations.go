package crud

import (
	"context"
	"database/sql"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Operation represents a CRUD operation type
type Operation int

const (
	// OperationCreate represents a create operation
	OperationCreate Operation = iota
	// OperationRead represents a read operation
	OperationRead
	// OperationUpdate represents an update operation
	OperationUpdate
	// OperationDelete represents a delete operation
	OperationDelete
)

// String returns the string representation of the operation
func (o Operation) String() string {
	switch o {
	case OperationCreate:
		return "create"
	case OperationRead:
		return "read"
	case OperationUpdate:
		return "update"
	case OperationDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Validator is an interface for validating records
type Validator interface {
	Validate(ctx context.Context, resource *schema.ResourceSchema, record map[string]interface{}, operation Operation) error
}

// HookExecutor is an interface for executing lifecycle hooks
type HookExecutor interface {
	ExecuteHooks(ctx context.Context, resource *schema.ResourceSchema, hookType schema.HookType, record map[string]interface{}) error
}

// TransactionManager is an interface for managing transactions
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

// Operations provides CRUD operations for a resource
type Operations struct {
	resource  *schema.ResourceSchema
	db        *sql.DB
	validator Validator
	hooks     HookExecutor
	txManager TransactionManager
}

// NewOperations creates a new Operations instance
func NewOperations(
	resource *schema.ResourceSchema,
	db *sql.DB,
	validator Validator,
	hooks HookExecutor,
	txManager TransactionManager,
) *Operations {
	return &Operations{
		resource:  resource,
		db:        db,
		validator: validator,
		hooks:     hooks,
		txManager: txManager,
	}
}

// Resource returns the resource schema
func (o *Operations) Resource() *schema.ResourceSchema {
	return o.resource
}

// DB returns the database connection
func (o *Operations) DB() *sql.DB {
	return o.db
}

// validateFieldIsColumn validates that a field is an actual database column
// and not a relationship field that cannot be used directly in WHERE clauses
func (o *Operations) validateFieldIsColumn(field string) error {
	// Check if field exists in the schema
	_, exists := o.resource.Fields[field]
	if !exists {
		// Field doesn't exist as a regular field - check if it's a relationship
		for _, rel := range o.resource.Relationships {
			if rel.FieldName == field {
				// This is a relationship field, not a column
				return ErrRelationshipField
			}
		}
		return ErrFieldNotFound
	}

	// Field exists - check if it's also defined as a relationship
	// (e.g., a belongs_to relationship might have both the field and relationship)
	for _, rel := range o.resource.Relationships {
		if rel.FieldName == field {
			// This is a relationship accessor, not a direct column
			// Exception: if it's a belongs_to with a foreign_key, the foreign_key column itself is valid
			if rel.Type == schema.RelationshipBelongsTo && rel.ForeignKey == field {
				// This is the actual foreign key column, which is valid
				return nil
			}
			return ErrRelationshipField
		}
	}

	return nil
}
