package validation

import (
	"context"
	"fmt"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// ConstraintValidator executes custom constraint blocks
type ConstraintValidator struct {
	// Interpreter for evaluating constraint expressions
	// For now, this is a placeholder - full implementation would require the runtime interpreter
	evaluator ExpressionEvaluator
}

// ExpressionEvaluator defines the interface for evaluating constraint expressions
type ExpressionEvaluator interface {
	// EvaluateBool evaluates an expression and returns a boolean result
	EvaluateBool(ctx context.Context, expr interface{}, record map[string]interface{}) (bool, error)
}

// NewConstraintValidator creates a new ConstraintValidator
func NewConstraintValidator(evaluator ExpressionEvaluator) *ConstraintValidator {
	return &ConstraintValidator{
		evaluator: evaluator,
	}
}

// ValidateConstraintBlock validates a custom constraint block
func (cv *ConstraintValidator) ValidateConstraintBlock(
	ctx context.Context,
	constraint *schema.ConstraintBlock,
	record map[string]interface{},
	operation string,
) error {
	// Check if constraint applies to this operation
	if !cv.appliesToOperation(constraint, operation) {
		return nil
	}

	// Evaluate "when" condition if present
	if constraint.When != nil {
		if cv.evaluator == nil {
			// Skip constraint evaluation if no evaluator is available
			// This allows the validation engine to work without full runtime support
			return nil
		}

		applies, err := cv.evaluator.EvaluateBool(ctx, constraint.When, record)
		if err != nil {
			return fmt.Errorf("error evaluating 'when' condition for constraint %s: %w", constraint.Name, err)
		}
		if !applies {
			// Constraint doesn't apply in this case
			return nil
		}
	}

	// Evaluate constraint condition
	if constraint.Condition != nil {
		if cv.evaluator == nil {
			// Skip constraint evaluation if no evaluator is available
			return nil
		}

		valid, err := cv.evaluator.EvaluateBool(ctx, constraint.Condition, record)
		if err != nil {
			return fmt.Errorf("error evaluating constraint %s: %w", constraint.Name, err)
		}

		if !valid {
			// Use custom error message if provided, otherwise use constraint name
			if constraint.Error != "" {
				return fmt.Errorf("%s", constraint.Error)
			}
			return fmt.Errorf("constraint %s failed", constraint.Name)
		}
	}

	return nil
}

// appliesToOperation checks if a constraint applies to a given operation
func (cv *ConstraintValidator) appliesToOperation(constraint *schema.ConstraintBlock, operation string) bool {
	if len(constraint.On) == 0 {
		// If no operations specified, apply to all
		return true
	}

	for _, op := range constraint.On {
		if op == operation {
			return true
		}
	}

	return false
}

// ValidateInvariant validates a runtime invariant
func (cv *ConstraintValidator) ValidateInvariant(
	ctx context.Context,
	invariant *schema.Invariant,
	record map[string]interface{},
) error {
	if cv.evaluator == nil {
		// Skip invariant validation if no evaluator is available
		return nil
	}

	valid, err := cv.evaluator.EvaluateBool(ctx, invariant.Condition, record)
	if err != nil {
		return fmt.Errorf("error evaluating invariant %s: %w", invariant.Name, err)
	}

	if !valid {
		if invariant.Error != "" {
			return fmt.Errorf("%s", invariant.Error)
		}
		return fmt.Errorf("invariant %s failed", invariant.Name)
	}

	return nil
}
