package validation

import (
	"context"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Simple tests to improve coverage without AST complexity

func TestNewCRUDValidator(t *testing.T) {
	evaluator := NewMockEvaluator()
	validator := NewCRUDValidator(evaluator)

	if validator == nil {
		t.Error("NewCRUDValidator should return non-nil validator")
	}
}

func TestToInt64_AllTypes(t *testing.T) {
	// Test all numeric types for toInt64
	values := []interface{}{
		int(42),
		int8(42),
		int16(42),
		int32(42),
		int64(42),
		uint(42),
		uint8(42),
		uint16(42),
		uint32(42),
		uint64(42),
		float32(42.0),
		float64(42.0),
	}

	for _, v := range values {
		result, ok := toInt64(v)
		if !ok {
			t.Errorf("toInt64(%T) should convert successfully", v)
		}
		if result != 42 {
			t.Errorf("toInt64(%T) = %d, want 42", v, result)
		}
	}
}

func TestToFloat64_AllTypes(t *testing.T) {
	// Test all numeric types for toFloat64
	values := []interface{}{
		int(42),
		int8(42),
		int16(42),
		int32(42),
		int64(42),
		uint(42),
		uint8(42),
		uint16(42),
		uint32(42),
		uint64(42),
		float32(42.0),
		float64(42.0),
	}

	for _, v := range values {
		result, ok := toFloat64(v)
		if !ok {
			t.Errorf("toFloat64(%T) should convert successfully", v)
		}
		if result != 42.0 {
			t.Errorf("toFloat64(%T) = %f, want 42.0", v, result)
		}
	}
}

func TestMinValidator_EdgeCases(t *testing.T) {
	// Test with invalid min constraint value types
	tests := []struct {
		name      string
		validator *MinValidator
		value     interface{}
	}{
		{
			name: "invalid min type for int field",
			validator: &MinValidator{
				Min:       []byte{},
				FieldType: schema.TypeInt,
			},
			value: 10,
		},
		{
			name: "invalid min type for float field",
			validator: &MinValidator{
				Min:       "invalid",
				FieldType: schema.TypeFloat,
			},
			value: 10.5,
		},
		{
			name: "invalid min type for string field",
			validator: &MinValidator{
				Min:       []int{},
				FieldType: schema.TypeString,
			},
			value: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should return error due to invalid constraint type
			err := tt.validator.Validate(tt.value)
			if err == nil {
				t.Error("expected error for invalid constraint type")
			}
		})
	}
}

func TestMaxValidator_EdgeCases(t *testing.T) {
	// Test with invalid max constraint value types
	tests := []struct {
		name      string
		validator *MaxValidator
		value     interface{}
	}{
		{
			name: "invalid max type for int field",
			validator: &MaxValidator{
				Max:       []byte{},
				FieldType: schema.TypeInt,
			},
			value: 10,
		},
		{
			name: "invalid max type for float field",
			validator: &MaxValidator{
				Max:       "invalid",
				FieldType: schema.TypeFloat,
			},
			value: 10.5,
		},
		{
			name: "invalid max type for string field",
			validator: &MaxValidator{
				Max:       []int{},
				FieldType: schema.TypeString,
			},
			value: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should return error due to invalid constraint type
			err := tt.validator.Validate(tt.value)
			if err == nil {
				t.Error("expected error for invalid constraint type")
			}
		})
	}
}

func TestCRUDValidator_AllOperationTypes(t *testing.T) {
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

	// Test all operation conversion paths
	operations := []interface{}{
		"create",
		"update",
		"delete",
		0, // OperationCreate
		1, // OperationRead
		2, // OperationUpdate
		3, // OperationDelete
		99, // Unknown int
		true, // Other type
		nil,
	}

	for _, op := range operations {
		t.Run("operation conversion", func(t *testing.T) {
			// Should not panic with any operation type
			_ = validator.Validate(context.Background(), resource, record, op)
		})
	}
}
