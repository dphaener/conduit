package validation

import (
	"context"
	"fmt"
	"regexp"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Engine is the main validation engine that coordinates all validation layers
type Engine struct {
	constraintValidator *ConstraintValidator
}

// NewEngine creates a new validation engine
func NewEngine(evaluator ExpressionEvaluator) *Engine {
	return &Engine{
		constraintValidator: NewConstraintValidator(evaluator),
	}
}

// NewEngineWithoutEvaluator creates a validation engine without an expression evaluator
// This is useful for basic field-level validation without custom constraint blocks
func NewEngineWithoutEvaluator() *Engine {
	return &Engine{
		constraintValidator: NewConstraintValidator(nil),
	}
}

// Validate performs multi-layer validation on a record
func (e *Engine) Validate(
	ctx context.Context,
	resource *schema.ResourceSchema,
	record map[string]interface{},
	operation string,
) error {
	errors := NewValidationErrors()

	// Layer 1: Field-level constraints (@min, @max, @pattern, etc.)
	e.validateFieldConstraints(resource, record, errors)

	// Layer 2: Type-specific validation (email, url, etc.)
	e.validateTypeSpecificFields(resource, record, errors)

	// Layer 3: Nullability validation
	e.validateNullability(resource, record, errors)

	// Layer 4: Resource-level constraints (@constraint blocks)
	e.validateResourceConstraints(ctx, resource, record, operation, errors)

	// Layer 5: Invariants (@invariant blocks)
	e.validateInvariants(ctx, resource, record, errors)

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// validateFieldConstraints validates field-level constraints
func (e *Engine) validateFieldConstraints(
	resource *schema.ResourceSchema,
	record map[string]interface{},
	errors *ValidationErrors,
) {
	for fieldName, field := range resource.Fields {
		value, exists := record[fieldName]
		if !exists {
			continue
		}

		// Skip validation for nil values (handled by nullability check)
		if value == nil {
			continue
		}

		// Validate each constraint on the field
		for _, constraint := range field.Constraints {
			if err := e.validateFieldConstraint(&constraint, value, field.Type); err != nil {
				errors.Add(fieldName, err.Error())
			}
		}
	}
}

// validateFieldConstraint validates a single field constraint
func (e *Engine) validateFieldConstraint(
	constraint *schema.Constraint,
	value interface{},
	fieldType *schema.TypeSpec,
) error {
	switch constraint.Type {
	case schema.ConstraintMin:
		validator := &MinValidator{
			Min:       constraint.Value,
			FieldType: fieldType.BaseType,
		}
		return validator.Validate(value)

	case schema.ConstraintMax:
		validator := &MaxValidator{
			Max:       constraint.Value,
			FieldType: fieldType.BaseType,
		}
		return validator.Validate(value)

	case schema.ConstraintPattern:
		// constraint.Value should be a compiled regex pattern
		pattern, ok := constraint.Value.(*regexp.Regexp)
		if !ok {
			// Try to compile if it's a string
			if patternStr, isStr := constraint.Value.(string); isStr {
				var err error
				pattern, err = regexp.Compile(patternStr)
				if err != nil {
					return fmt.Errorf("invalid pattern: %w", err)
				}
			} else {
				return fmt.Errorf("invalid pattern constraint")
			}
		}
		validator := &PatternValidator{Pattern: pattern}
		return validator.Validate(value)

	default:
		// Other constraint types (unique, index, primary, auto) are not validated at runtime
		return nil
	}
}

// validateTypeSpecificFields validates fields with built-in type validation
func (e *Engine) validateTypeSpecificFields(
	resource *schema.ResourceSchema,
	record map[string]interface{},
	errors *ValidationErrors,
) {
	for fieldName, field := range resource.Fields {
		value, exists := record[fieldName]
		if !exists || value == nil {
			continue
		}

		switch field.Type.BaseType {
		case schema.TypeEmail:
			validator := &EmailValidator{}
			if err := validator.Validate(value); err != nil {
				errors.Add(fieldName, err.Error())
			}

		case schema.TypeURL:
			validator := &URLValidator{}
			if err := validator.Validate(value); err != nil {
				errors.Add(fieldName, err.Error())
			}
		}
	}
}

// validateNullability validates that required fields are not null
func (e *Engine) validateNullability(
	resource *schema.ResourceSchema,
	record map[string]interface{},
	errors *ValidationErrors,
) {
	for fieldName, field := range resource.Fields {
		// Skip if field is nullable
		if field.Type.Nullable {
			continue
		}

		value, exists := record[fieldName]

		// Check if value is missing or nil
		if !exists || value == nil {
			errors.Add(fieldName, "is required")
		}
	}
}

// validateResourceConstraints validates custom constraint blocks
func (e *Engine) validateResourceConstraints(
	ctx context.Context,
	resource *schema.ResourceSchema,
	record map[string]interface{},
	operation string,
	errors *ValidationErrors,
) {
	for _, constraint := range resource.ConstraintBlocks {
		if err := e.constraintValidator.ValidateConstraintBlock(ctx, constraint, record, operation); err != nil {
			errors.Add(constraint.Name, err.Error())
		}
	}
}

// validateInvariants validates runtime invariants
func (e *Engine) validateInvariants(
	ctx context.Context,
	resource *schema.ResourceSchema,
	record map[string]interface{},
	errors *ValidationErrors,
) {
	for _, invariant := range resource.Invariants {
		if err := e.constraintValidator.ValidateInvariant(ctx, invariant, record); err != nil {
			errors.Add(invariant.Name, err.Error())
		}
	}
}

// ValidateField validates a single field value
// This is useful for partial validation or field-level feedback
func (e *Engine) ValidateField(
	fieldName string,
	value interface{},
	field *schema.Field,
) error {
	errors := NewValidationErrors()

	// Check nullability
	if !field.Type.Nullable && value == nil {
		errors.Add(fieldName, "is required")
		return errors
	}

	// Skip further validation for nil values
	if value == nil {
		return nil
	}

	// Validate field constraints
	for _, constraint := range field.Constraints {
		if err := e.validateFieldConstraint(&constraint, value, field.Type); err != nil {
			errors.Add(fieldName, err.Error())
		}
	}

	// Validate type-specific rules
	switch field.Type.BaseType {
	case schema.TypeEmail:
		validator := &EmailValidator{}
		if err := validator.Validate(value); err != nil {
			errors.Add(fieldName, err.Error())
		}

	case schema.TypeURL:
		validator := &URLValidator{}
		if err := validator.Validate(value); err != nil {
			errors.Add(fieldName, err.Error())
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
