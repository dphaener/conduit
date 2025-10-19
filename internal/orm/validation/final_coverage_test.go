package validation

import (
	"context"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestMinValidator_UnsupportedType(t *testing.T) {
	// Test min validator with a type that's not string, text, or numeric
	validator := &MinValidator{
		Min:       5,
		FieldType: schema.TypeBool, // Bool type doesn't support min
	}

	err := validator.Validate(true)
	// Should not error (returns nil for unsupported types)
	if err != nil {
		t.Errorf("unsupported type should not error, got %v", err)
	}
}

func TestMaxValidator_UnsupportedType(t *testing.T) {
	// Test max validator with a type that's not string, text, or numeric
	validator := &MaxValidator{
		Max:       100,
		FieldType: schema.TypeBool, // Bool type doesn't support max
	}

	err := validator.Validate(true)
	// Should not error (returns nil for unsupported types)
	if err != nil {
		t.Errorf("unsupported type should not error, got %v", err)
	}
}

func TestMinValidator_AllIntegerTypes(t *testing.T) {
	// Test conversion edge cases
	validator := &MinValidator{
		Min:       int64(10),
		FieldType: schema.TypeBigInt,
	}

	// Test with different integer value types
	testValues := []interface{}{
		int(15),
		int8(15),
		int16(15),
		int32(15),
		int64(15),
		uint(15),
		uint8(15),
		uint16(15),
		uint32(15),
		uint64(15),
	}

	for _, val := range testValues {
		err := validator.Validate(val)
		if err != nil {
			t.Errorf("MinValidator with value %T should not error, got %v", val, err)
		}
	}
}

func TestMaxValidator_AllIntegerTypes(t *testing.T) {
	// Test conversion edge cases
	validator := &MaxValidator{
		Max:       int64(100),
		FieldType: schema.TypeBigInt,
	}

	// Test with different integer value types
	testValues := []interface{}{
		int(50),
		int8(50),
		int16(50),
		int32(50),
		int64(50),
		uint(50),
		uint8(50),
		uint16(50),
		uint32(50),
		uint64(50),
	}

	for _, val := range testValues {
		err := validator.Validate(val)
		if err != nil {
			t.Errorf("MaxValidator with value %T should not error, got %v", val, err)
		}
	}
}

func TestMinValidator_FloatTypes(t *testing.T) {
	validator := &MinValidator{
		Min:       10.5,
		FieldType: schema.TypeDecimal,
	}

	// Test with float32 and float64
	testValues := []interface{}{
		float32(15.5),
		float64(15.5),
	}

	for _, val := range testValues {
		err := validator.Validate(val)
		if err != nil {
			t.Errorf("MinValidator with value %T should not error, got %v", val, err)
		}
	}
}

func TestMaxValidator_FloatTypes(t *testing.T) {
	validator := &MaxValidator{
		Max:       100.5,
		FieldType: schema.TypeDecimal,
	}

	// Test with float32 and float64
	testValues := []interface{}{
		float32(50.5),
		float64(50.5),
	}

	for _, val := range testValues {
		err := validator.Validate(val)
		if err != nil {
			t.Errorf("MaxValidator with value %T should not error, got %v", val, err)
		}
	}
}

func TestCRUDValidator_WithContext(t *testing.T) {
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["field"] = &schema.Field{
		Name: "field",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 3,
			},
		},
	}

	// Test with background context
	record := map[string]interface{}{
		"field": "value",
	}

	err := validator.Validate(context.Background(), resource, record, "create")
	if err != nil {
		t.Errorf("valid record should not error, got %v", err)
	}

	// Test with value that's too short
	shortRecord := map[string]interface{}{
		"field": "hi",
	}

	err = validator.Validate(context.Background(), resource, shortRecord, "create")
	if err == nil {
		t.Error("expected validation error for short field")
	}
}

func TestEngine_EmptyResourceSchema(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	// Empty resource
	resource := schema.NewResourceSchema("Empty")
	record := map[string]interface{}{}

	err := engine.Validate(context.Background(), resource, record, "create")
	if err != nil {
		t.Errorf("empty resource should not error, got %v", err)
	}
}

func TestEngine_NilValueHandling(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	resource := schema.NewResourceSchema("Test")
	resource.Fields["optional"] = &schema.Field{
		Name: "optional",
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

	// Nil value should not be validated for constraints (only nullability)
	record := map[string]interface{}{
		"optional": nil,
	}

	err := engine.Validate(context.Background(), resource, record, "create")
	if err != nil {
		t.Errorf("nil value for nullable field should not error, got %v", err)
	}
}
