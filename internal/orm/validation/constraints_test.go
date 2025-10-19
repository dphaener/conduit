package validation

import (
	"context"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// MockExpressionEvaluator is a mock implementation of ExpressionEvaluator for testing
type MockExpressionEvaluator struct {
	evaluations map[interface{}]bool
	errors      map[interface{}]error
}

func NewMockEvaluator() *MockExpressionEvaluator {
	return &MockExpressionEvaluator{
		evaluations: make(map[interface{}]bool),
		errors:      make(map[interface{}]error),
	}
}

func (m *MockExpressionEvaluator) SetEvaluation(expr interface{}, result bool) {
	m.evaluations[expr] = result
}

func (m *MockExpressionEvaluator) SetError(expr interface{}, err error) {
	m.errors[expr] = err
}

func (m *MockExpressionEvaluator) EvaluateBool(ctx context.Context, expr interface{}, record map[string]interface{}) (bool, error) {
	if err, ok := m.errors[expr]; ok {
		return false, err
	}
	if result, ok := m.evaluations[expr]; ok {
		return result, nil
	}
	return true, nil // Default to true
}

func TestConstraintValidator_ValidateConstraintBlock(t *testing.T) {
	evaluator := NewMockEvaluator()
	validator := NewConstraintValidator(evaluator)

	// Use interface{} to represent expressions for testing
	whenExpr := interface{}("when_condition")
	conditionExpr := interface{}("main_condition")

	evaluator.SetEvaluation(whenExpr, true)
	evaluator.SetEvaluation(conditionExpr, true)

	constraint := &schema.ConstraintBlock{
		Name:      "test_constraint",
		On:        []string{"create", "update"},
		When:      nil, // Would be an AST node in real usage
		Condition: nil, // Would be an AST node in real usage
		Error:     "constraint failed",
	}

	tests := []struct {
		name      string
		operation string
		setup     func()
		wantErr   bool
	}{
		{
			name:      "constraint passes - nil condition",
			operation: "create",
			setup: func() {
				// No setup needed, nil condition should not error
			},
			wantErr: false,
		},
		{
			name:      "operation doesn't match",
			operation: "delete",
			setup: func() {
				// Even with nil conditions, if operation doesn't match, no error
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			record := map[string]interface{}{"status": "published"}
			err := validator.ValidateConstraintBlock(context.Background(), constraint, record, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraintBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConstraintValidator_ValidateInvariant(t *testing.T) {
	evaluator := NewMockEvaluator()
	validator := NewConstraintValidator(evaluator)

	invariant := &schema.Invariant{
		Name:      "test_invariant",
		Condition: nil, // Would be an AST node in real usage
		Error:     "invariant violated",
	}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "invariant with nil condition - no evaluator",
			setup: func() {
				// No setup needed, nil evaluator should not error
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			record := map[string]interface{}{"balance": 100}
			err := validator.ValidateInvariant(context.Background(), invariant, record)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInvariant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConstraintValidator_AppliesToOperation(t *testing.T) {
	validator := NewConstraintValidator(nil)

	tests := []struct {
		name       string
		constraint *schema.ConstraintBlock
		operation  string
		expected   bool
	}{
		{
			name: "no operations specified - applies to all",
			constraint: &schema.ConstraintBlock{
				Name: "test",
				On:   []string{},
			},
			operation: "create",
			expected:  true,
		},
		{
			name: "operation matches",
			constraint: &schema.ConstraintBlock{
				Name: "test",
				On:   []string{"create", "update"},
			},
			operation: "create",
			expected:  true,
		},
		{
			name: "operation doesn't match",
			constraint: &schema.ConstraintBlock{
				Name: "test",
				On:   []string{"create", "update"},
			},
			operation: "delete",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.appliesToOperation(tt.constraint, tt.operation)
			if result != tt.expected {
				t.Errorf("appliesToOperation() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestConstraintValidator_WithoutEvaluator(t *testing.T) {
	validator := NewConstraintValidator(nil)

	constraint := &schema.ConstraintBlock{
		Name:      "test",
		On:        []string{"create"},
		Condition: nil, // Would be an AST node in real usage
		Error:     "error message",
	}

	// Should not error when evaluator is nil
	record := map[string]interface{}{"field": "value"}
	err := validator.ValidateConstraintBlock(context.Background(), constraint, record, "create")
	if err != nil {
		t.Errorf("expected no error with nil evaluator, got %v", err)
	}
}
