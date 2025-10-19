package validation

import (
	"context"
	"regexp"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Additional tests to improve coverage

func TestValidationErrors_EmptyFields(t *testing.T) {
	errors := &ValidationErrors{
		Fields: nil,
	}

	errors.Add("field", "error")
	if errors.Fields == nil {
		t.Error("Add should initialize Fields map if nil")
	}
}

func TestEngine_ValidateWithEmptyResource(t *testing.T) {
	engine := NewEngineWithoutEvaluator()
	resource := schema.NewResourceSchema("Empty")

	record := map[string]interface{}{}
	err := engine.Validate(context.Background(), resource, record, "create")
	if err != nil {
		t.Errorf("empty resource should not produce errors, got %v", err)
	}
}

func TestEngine_ValidateWithNonExistentField(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["real_field"] = &schema.Field{
		Name: "real_field",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}

	// Record contains a field not in schema
	record := map[string]interface{}{
		"real_field":  "value",
		"fake_field":  "ignored",
		"other_field": 123,
	}

	err := engine.Validate(context.Background(), resource, record, "create")
	if err != nil {
		t.Errorf("non-schema fields should be ignored, got %v", err)
	}
}

func TestEngine_ValidateFieldConstraint_InvalidPattern(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	// Test with invalid pattern constraint (not a regex)
	constraint := &schema.Constraint{
		Type:  schema.ConstraintPattern,
		Value: 123, // Invalid - not a regex
	}

	fieldType := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	err := engine.validateFieldConstraint(constraint, "test", fieldType)
	if err == nil {
		t.Error("expected error for invalid pattern constraint")
	}
}

func TestEngine_ValidateFieldConstraint_PatternAsString(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	// Test with pattern as string (should compile)
	constraint := &schema.Constraint{
		Type:  schema.ConstraintPattern,
		Value: `^[a-z]+$`,
	}

	fieldType := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	err := engine.validateFieldConstraint(constraint, "hello", fieldType)
	if err != nil {
		t.Errorf("valid pattern string should work, got %v", err)
	}

	err = engine.validateFieldConstraint(constraint, "Hello123", fieldType)
	if err == nil {
		t.Error("expected validation error for non-matching pattern")
	}
}

func TestEngine_ValidateFieldConstraint_InvalidPatternString(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	// Test with invalid regex pattern string
	constraint := &schema.Constraint{
		Type:  schema.ConstraintPattern,
		Value: `[invalid(regex`,
	}

	fieldType := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	err := engine.validateFieldConstraint(constraint, "test", fieldType)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestEngine_ValidateFieldConstraint_UnknownType(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	// Test with unknown constraint type
	constraint := &schema.Constraint{
		Type:  schema.ConstraintUnique, // This is handled elsewhere, not at runtime
		Value: true,
	}

	fieldType := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	err := engine.validateFieldConstraint(constraint, "test", fieldType)
	if err != nil {
		t.Errorf("unknown constraint types should be ignored, got %v", err)
	}
}

func TestCRUDValidator_OperationTypes(t *testing.T) {
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["field"] = &schema.Field{
		Name: "field",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}

	record := map[string]interface{}{
		"field": "value",
	}

	tests := []struct {
		name      string
		operation interface{}
	}{
		{
			name:      "string operation",
			operation: "create",
		},
		{
			name:      "int operation - read",
			operation: 1,
		},
		{
			name:      "int operation - delete",
			operation: 3,
		},
		{
			name:      "unknown int operation",
			operation: 99,
		},
		{
			name:      "other type operation",
			operation: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with any operation type
			_ = validator.Validate(context.Background(), resource, record, tt.operation)
		})
	}
}

func TestEngine_ValidateFieldNullable(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	field := &schema.Field{
		Name: "optional_field",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: true,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 5,
			},
		},
	}

	// Nil value for nullable field should not error
	err := engine.ValidateField("optional_field", nil, field)
	if err != nil {
		t.Errorf("nil value for nullable field should not error, got %v", err)
	}
}

func TestMinValidator_InvalidTypes(t *testing.T) {
	tests := []struct {
		name      string
		validator *MinValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "non-numeric value for numeric field",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeInt},
			value:     "not a number",
			wantErr:   true,
		},
		{
			name:      "non-string value for string field",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeString},
			value:     123,
			wantErr:   true,
		},
		{
			name:      "invalid min constraint for int",
			validator: &MinValidator{Min: "not a number", FieldType: schema.TypeInt},
			value:     10,
			wantErr:   true,
		},
		{
			name:      "invalid min constraint for string",
			validator: &MinValidator{Min: "not a number", FieldType: schema.TypeString},
			value:     "hello",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxValidator_InvalidTypes(t *testing.T) {
	tests := []struct {
		name      string
		validator *MaxValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "non-numeric value for numeric field",
			validator: &MaxValidator{Max: 100, FieldType: schema.TypeInt},
			value:     "not a number",
			wantErr:   true,
		},
		{
			name:      "non-string value for string field",
			validator: &MaxValidator{Max: 10, FieldType: schema.TypeString},
			value:     123,
			wantErr:   true,
		},
		{
			name:      "invalid max constraint for int",
			validator: &MaxValidator{Max: "not a number", FieldType: schema.TypeInt},
			value:     10,
			wantErr:   true,
		},
		{
			name:      "invalid max constraint for string",
			validator: &MaxValidator{Max: "not a number", FieldType: schema.TypeString},
			value:     "hello",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator_InvalidValue(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+$`)
	validator := &PatternValidator{Pattern: pattern}

	// Non-string value should error
	err := validator.Validate(123)
	if err == nil {
		t.Error("expected error for non-string value")
	}
}

func TestMinLengthValidator_InvalidValue(t *testing.T) {
	validator := &MinLengthValidator{MinLength: 5}

	// Non-array value should error
	err := validator.Validate("not an array")
	if err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestMaxLengthValidator_InvalidValue(t *testing.T) {
	validator := &MaxLengthValidator{MaxLength: 10}

	// Non-array value should error
	err := validator.Validate("not an array")
	if err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestEngine_ValidateResourceConstraints_EmptyConstraints(t *testing.T) {
	engine := NewEngineWithoutEvaluator()
	errors := NewValidationErrors()

	resource := schema.NewResourceSchema("Test")
	// No constraint blocks

	record := map[string]interface{}{
		"field": "value",
	}

	engine.validateResourceConstraints(context.Background(), resource, record, "create", errors)

	if errors.HasErrors() {
		t.Error("empty constraint blocks should not produce errors")
	}
}

func TestEngine_ValidateInvariants_EmptyInvariants(t *testing.T) {
	engine := NewEngineWithoutEvaluator()
	errors := NewValidationErrors()

	resource := schema.NewResourceSchema("Test")
	// No invariants

	record := map[string]interface{}{
		"field": "value",
	}

	engine.validateInvariants(context.Background(), resource, record, errors)

	if errors.HasErrors() {
		t.Error("empty invariants should not produce errors")
	}
}

func TestNewEngineWithEvaluator(t *testing.T) {
	evaluator := NewMockEvaluator()
	engine := NewEngine(evaluator)

	if engine == nil {
		t.Error("NewEngine should return non-nil engine")
	}

	if engine.constraintValidator == nil {
		t.Error("engine should have constraint validator")
	}
}
