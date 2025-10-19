package validation

import (
	"context"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// CRUDValidator implements the Validator interface for CRUD operations
type CRUDValidator struct {
	engine *Engine
}

// NewCRUDValidator creates a new CRUD validator
func NewCRUDValidator(evaluator ExpressionEvaluator) *CRUDValidator {
	return &CRUDValidator{
		engine: NewEngine(evaluator),
	}
}

// NewCRUDValidatorWithoutEvaluator creates a CRUD validator without expression evaluation
// This is useful for basic field-level validation without custom constraints
func NewCRUDValidatorWithoutEvaluator() *CRUDValidator {
	return &CRUDValidator{
		engine: NewEngineWithoutEvaluator(),
	}
}

// Validate implements the crud.Validator interface
func (v *CRUDValidator) Validate(
	ctx context.Context,
	resource *schema.ResourceSchema,
	record map[string]interface{},
	operation interface{},
) error {
	// Convert operation to string
	var opStr string
	switch op := operation.(type) {
	case string:
		opStr = op
	case int:
		// Handle Operation enum type from crud package
		switch op {
		case 0:
			opStr = "create"
		case 2:
			opStr = "update"
		default:
			opStr = "unknown"
		}
	default:
		opStr = "unknown"
	}

	return v.engine.Validate(ctx, resource, record, opStr)
}
