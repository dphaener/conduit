package validation

import (
	"context"
	"regexp"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestEngine_ValidateFieldConstraints(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

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

	tests := []struct {
		name    string
		record  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid title",
			record: map[string]interface{}{
				"title": "Hello World",
			},
			wantErr: false,
		},
		{
			name: "title too short",
			record: map[string]interface{}{
				"title": "Hi",
			},
			wantErr: true,
		},
		{
			name: "title too long",
			record: map[string]interface{}{
				"title": "This is a very long title that exceeds the maximum length constraint of one hundred characters for testing purposes",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), resource, tt.record, "create")
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ValidateNullability(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	resource := schema.NewResourceSchema("Post")
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}
	resource.Fields["subtitle"] = &schema.Field{
		Name: "subtitle",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: true,
		},
	}

	tests := []struct {
		name    string
		record  map[string]interface{}
		wantErr bool
	}{
		{
			name: "all required fields present",
			record: map[string]interface{}{
				"title": "Hello World",
			},
			wantErr: false,
		},
		{
			name: "required field missing",
			record: map[string]interface{}{
				"subtitle": "A subtitle",
			},
			wantErr: true,
		},
		{
			name: "required field nil",
			record: map[string]interface{}{
				"title": nil,
			},
			wantErr: true,
		},
		{
			name: "optional field nil",
			record: map[string]interface{}{
				"title":    "Hello World",
				"subtitle": nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), resource, tt.record, "create")
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ValidateTypeSpecificFields(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	resource := schema.NewResourceSchema("User")
	resource.Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeEmail,
			Nullable: false,
		},
	}
	resource.Fields["website"] = &schema.Field{
		Name: "website",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeURL,
			Nullable: true,
		},
	}

	tests := []struct {
		name    string
		record  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid email",
			record: map[string]interface{}{
				"email": "user@example.com",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			record: map[string]interface{}{
				"email": "not-an-email",
			},
			wantErr: true,
		},
		{
			name: "valid URL",
			record: map[string]interface{}{
				"email":   "user@example.com",
				"website": "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "invalid URL",
			record: map[string]interface{}{
				"email":   "user@example.com",
				"website": "not-a-url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), resource, tt.record, "create")
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ValidateField(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	field := &schema.Field{
		Name: "age",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 18,
			},
			{
				Type:  schema.ConstraintMax,
				Value: 120,
			},
		},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid value",
			value:   25,
			wantErr: false,
		},
		{
			name:    "value too low",
			value:   10,
			wantErr: true,
		},
		{
			name:    "value too high",
			value:   150,
			wantErr: true,
		},
		{
			name:    "nil value for required field",
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateField("age", tt.value, field)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.ValidateField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ValidateMultipleErrors(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

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
		"email": "invalid-email",
	}

	err := engine.Validate(context.Background(), resource, record, "create")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	validationErr, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}

	if validationErr.Count() != 2 {
		t.Errorf("expected 2 validation errors, got %d", validationErr.Count())
	}

	if len(validationErr.Fields["title"]) == 0 {
		t.Error("expected error for title field")
	}

	if len(validationErr.Fields["email"]) == 0 {
		t.Error("expected error for email field")
	}
}

func TestEngine_ValidatePatternConstraint(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	pattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	resource := schema.NewResourceSchema("Post")
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintPattern,
				Value: pattern,
			},
		},
	}

	tests := []struct {
		name    string
		record  map[string]interface{}
		wantErr bool
	}{
		{
			name: "matching pattern",
			record: map[string]interface{}{
				"slug": "hello-world-123",
			},
			wantErr: false,
		},
		{
			name: "non-matching pattern",
			record: map[string]interface{}{
				"slug": "Hello World!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), resource, tt.record, "create")
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ValidateNumericConstraints(t *testing.T) {
	engine := NewEngineWithoutEvaluator()

	resource := schema.NewResourceSchema("Product")
	resource.Fields["price"] = &schema.Field{
		Name: "price",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeFloat,
			Nullable: false,
		},
		Constraints: []schema.Constraint{
			{
				Type:  schema.ConstraintMin,
				Value: 0.01,
			},
			{
				Type:  schema.ConstraintMax,
				Value: 9999.99,
			},
		},
	}

	tests := []struct {
		name    string
		record  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid price",
			record: map[string]interface{}{
				"price": 19.99,
			},
			wantErr: false,
		},
		{
			name: "price too low",
			record: map[string]interface{}{
				"price": 0.0,
			},
			wantErr: true,
		},
		{
			name: "price too high",
			record: map[string]interface{}{
				"price": 10000.0,
			},
			wantErr: true,
		},
		{
			name: "minimum price",
			record: map[string]interface{}{
				"price": 0.01,
			},
			wantErr: false,
		},
		{
			name: "maximum price",
			record: map[string]interface{}{
				"price": 9999.99,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), resource, tt.record, "create")
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
