package validation

import (
	"context"
	"regexp"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/crud"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestCRUDValidator_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test would require a real database connection
	// For now, we'll test the validator interface implementation
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 5,
			},
			{
				Type:  schema.ConstraintMax,
				Value: 100,
			},
		},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}

	tests := []struct {
		name      string
		record    map[string]interface{}
		operation crud.Operation
		wantErr   bool
	}{
		{
			name: "valid record",
			record: map[string]interface{}{
				"title": "Hello World",
				"email": "user@example.com",
			},
			operation: crud.OperationCreate,
			wantErr:   false,
		},
		{
			name: "invalid title - too short",
			record: map[string]interface{}{
				"title": "Hi",
				"email": "user@example.com",
			},
			operation: crud.OperationCreate,
			wantErr:   true,
		},
		{
			name: "invalid email",
			record: map[string]interface{}{
				"title": "Hello World",
				"email": "not-an-email",
			},
			operation: crud.OperationCreate,
			wantErr:   true,
		},
		{
			name: "missing required field",
			record: map[string]interface{}{
				"email": "user@example.com",
			},
			operation: crud.OperationCreate,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), resource, tt.record, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify error structure
			if err != nil {
				validationErr, ok := err.(*ValidationErrors)
				if !ok {
					t.Errorf("expected *ValidationErrors, got %T", err)
				} else if !validationErr.HasErrors() {
					t.Error("ValidationErrors should have errors")
				}
			}
		})
	}
}

func TestValidationWithCRUDOperations(t *testing.T) {
	// This is a unit test that doesn't require a database
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("User")
	resource.Fields["username"] = &schema.Field{
		Name: "username",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 3,
			},
			{
				Type:  schema.ConstraintMax,
				Value: 20,
			},
			{
				Type:  schema.ConstraintPattern,
				Value: regexp.MustCompile(`^[a-zA-Z0-9_]+$`),
			},
		},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}
	resource.Fields["age"] = &schema.Field{
		Name: "age",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: true,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 13,
			},
			{
				Type:  schema.ConstraintMax,
				Value: 120,
			},
		},
	}

	tests := []struct {
		name          string
		record        map[string]interface{}
		operation     crud.Operation
		wantErr       bool
		expectedField string
	}{
		{
			name: "valid user creation",
			record: map[string]interface{}{
				"username": "johndoe",
				"email":    "john@example.com",
				"age":      25,
			},
			operation: crud.OperationCreate,
			wantErr:   false,
		},
		{
			name: "username too short",
			record: map[string]interface{}{
				"username": "jo",
				"email":    "john@example.com",
			},
			operation:     crud.OperationCreate,
			wantErr:       true,
			expectedField: "username",
		},
		{
			name: "username with invalid characters",
			record: map[string]interface{}{
				"username": "john-doe!",
				"email":    "john@example.com",
			},
			operation:     crud.OperationCreate,
			wantErr:       true,
			expectedField: "username",
		},
		{
			name: "invalid email format",
			record: map[string]interface{}{
				"username": "johndoe",
				"email":    "not-an-email",
			},
			operation:     crud.OperationCreate,
			wantErr:       true,
			expectedField: "email",
		},
		{
			name: "age too young",
			record: map[string]interface{}{
				"username": "johndoe",
				"email":    "john@example.com",
				"age":      10,
			},
			operation:     crud.OperationCreate,
			wantErr:       true,
			expectedField: "age",
		},
		{
			name: "age too old",
			record: map[string]interface{}{
				"username": "johndoe",
				"email":    "john@example.com",
				"age":      150,
			},
			operation:     crud.OperationCreate,
			wantErr:       true,
			expectedField: "age",
		},
		{
			name: "optional age field not provided",
			record: map[string]interface{}{
				"username": "johndoe",
				"email":    "john@example.com",
			},
			operation: crud.OperationCreate,
			wantErr:   false,
		},
		{
			name: "valid update operation",
			record: map[string]interface{}{
				"username": "johndoe_updated",
				"email":    "john.updated@example.com",
			},
			operation: crud.OperationUpdate,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), resource, tt.record, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectedField != "" {
				validationErr, ok := err.(*ValidationErrors)
				if !ok {
					t.Fatalf("expected *ValidationErrors, got %T", err)
				}
				if len(validationErr.Fields[tt.expectedField]) == 0 {
					t.Errorf("expected validation error for field %q, but got none", tt.expectedField)
				}
			}
		})
	}
}

func TestCRUDValidator_OperationConversion(t *testing.T) {
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
		wantErr   bool
	}{
		{
			name:      "string operation",
			operation: "create",
			wantErr:   false,
		},
		{
			name:      "int operation - create",
			operation: crud.OperationCreate,
			wantErr:   false,
		},
		{
			name:      "int operation - update",
			operation: crud.OperationUpdate,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), resource, record, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkValidation benchmarks the validation engine
func BenchmarkValidation(b *testing.B) {
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 5},
			{Type: schema.ConstraintMax, Value: 100},
		},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}

	record := map[string]interface{}{
		"title": "Hello World",
		"email": "user@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(context.Background(), resource, record, crud.OperationCreate)
	}
}

// TestValidationErrorFormat tests the error message formatting
func TestValidationErrorFormat(t *testing.T) {
	validator := NewCRUDValidatorWithoutEvaluator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMin, Value: 5},
		},
	}
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}

	record := map[string]interface{}{
		"title": "Hi",
		"email": "invalid",
	}

	err := validator.Validate(context.Background(), resource, record, crud.OperationCreate)
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErr, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}

	// Check error message format
	errMsg := validationErr.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}

	// Check JSON serialization
	_, jsonErr := validationErr.MarshalJSON()
	if jsonErr != nil {
		t.Errorf("failed to marshal validation error: %v", jsonErr)
	}
}
