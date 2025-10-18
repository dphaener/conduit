package schema

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestValidateNullability(t *testing.T) {
	validator := NewSchemaValidator()

	t.Run("valid nullability", func(t *testing.T) {
		schema := NewResourceSchema("Post")
		schema.Fields["title"] = &Field{
			Name: "title",
			Type: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
		}
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		err := validator.validateNullability(schema)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("optional field with default generates warning", func(t *testing.T) {
		schema := NewResourceSchema("Post")
		schema.Fields["status"] = &Field{
			Name: "status",
			Type: &TypeSpec{
				BaseType: TypeString,
				Nullable: true,
				Default:  "draft",
			},
		}
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		validator.validateNullability(schema)
		if len(validator.warnings) == 0 {
			t.Error("expected warning for optional field with default")
		}
	})
}

func TestValidatePrimaryKey(t *testing.T) {
	validator := NewSchemaValidator()

	t.Run("valid primary key", func(t *testing.T) {
		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		err := validator.validatePrimaryKey(schema)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(validator.errors) > 0 {
			t.Errorf("unexpected errors: %v", validator.errors)
		}
	})

	t.Run("missing primary key", func(t *testing.T) {
		validator := NewSchemaValidator()
		schema := NewResourceSchema("Post")
		schema.Fields["title"] = &Field{
			Name: "title",
			Type: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
		}

		validator.validatePrimaryKey(schema)
		if len(validator.errors) == 0 {
			t.Error("expected error for missing primary key")
		}
	})

	t.Run("nullable primary key", func(t *testing.T) {
		validator := NewSchemaValidator()
		schema := NewResourceSchema("Post")
		schema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: true,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		validator.validatePrimaryKey(schema)
		if len(validator.errors) == 0 {
			t.Error("expected error for nullable primary key")
		}
	})

	t.Run("multiple primary keys", func(t *testing.T) {
		validator := NewSchemaValidator()
		schema := NewResourceSchema("Post")
		schema.Fields["id1"] = &Field{
			Name: "id1",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		schema.Fields["id2"] = &Field{
			Name: "id2",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		validator.validatePrimaryKey(schema)
		if len(validator.errors) == 0 {
			t.Error("expected error for multiple primary keys")
		}
	})
}

func TestValidateFieldConstraint(t *testing.T) {
	validator := NewSchemaValidator()

	tests := []struct {
		name        string
		typeSpec    *TypeSpec
		constraint  Constraint
		expectError bool
	}{
		{
			name: "min on numeric type",
			typeSpec: &TypeSpec{
				BaseType: TypeInt,
				Nullable: false,
			},
			constraint: Constraint{
				Type:  ConstraintMin,
				Value: 0,
			},
			expectError: false,
		},
		{
			name: "min on text type",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
			constraint: Constraint{
				Type:  ConstraintMin,
				Value: 5,
			},
			expectError: false,
		},
		{
			name: "min on bool type",
			typeSpec: &TypeSpec{
				BaseType: TypeBool,
				Nullable: false,
			},
			constraint: Constraint{
				Type:  ConstraintMin,
				Value: 0,
			},
			expectError: true,
		},
		{
			name: "pattern on string type",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
			constraint: Constraint{
				Type:  ConstraintPattern,
				Value: "^[a-z]+$",
			},
			expectError: false,
		},
		{
			name: "pattern on int type",
			typeSpec: &TypeSpec{
				BaseType: TypeInt,
				Nullable: false,
			},
			constraint: Constraint{
				Type:  ConstraintPattern,
				Value: "^[0-9]+$",
			},
			expectError: true,
		},
		{
			name: "unique on string",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
				Nullable: false,
			},
			constraint: Constraint{
				Type: ConstraintUnique,
			},
			expectError: false,
		},
		{
			name: "unique on text",
			typeSpec: &TypeSpec{
				BaseType: TypeText,
				Nullable: false,
			},
			constraint: Constraint{
				Type: ConstraintUnique,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateFieldConstraint("Post", "field", tt.typeSpec, tt.constraint)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateRelationships(t *testing.T) {
	t.Run("valid relationship", func(t *testing.T) {
		validator := NewSchemaValidator()

		userSchema := NewResourceSchema("User")
		userSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		postSchema := NewResourceSchema("Post")
		postSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		postSchema.Relationships["author"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			Nullable:       false,
			OnDelete:       CascadeRestrict,
		}

		registry := map[string]*ResourceSchema{
			"User": userSchema,
			"Post": postSchema,
		}

		validator.schemas = registry
		err := validator.validateRelationships(postSchema)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(validator.errors) > 0 {
			t.Errorf("unexpected errors: %v", validator.errors)
		}
	})

	t.Run("unknown target resource", func(t *testing.T) {
		validator := NewSchemaValidator()

		postSchema := NewResourceSchema("Post")
		postSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		postSchema.Relationships["author"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "NonExistent",
			FieldName:      "author",
		}

		registry := map[string]*ResourceSchema{
			"Post": postSchema,
		}

		validator.schemas = registry
		validator.validateRelationships(postSchema)
		if len(validator.errors) == 0 {
			t.Error("expected error for unknown target resource")
		}
	})

	t.Run("set_null on non-nullable relationship", func(t *testing.T) {
		validator := NewSchemaValidator()

		userSchema := NewResourceSchema("User")
		userSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}

		postSchema := NewResourceSchema("Post")
		postSchema.Fields["id"] = &Field{
			Name: "id",
			Type: &TypeSpec{
				BaseType: TypeUUID,
				Nullable: false,
			},
			Annotations: []Annotation{{Name: "primary"}},
		}
		postSchema.Relationships["author"] = &Relationship{
			Type:           RelationshipBelongsTo,
			TargetResource: "User",
			FieldName:      "author",
			Nullable:       false,
			OnDelete:       CascadeSetNull,
		}

		registry := map[string]*ResourceSchema{
			"User": userSchema,
			"Post": postSchema,
		}

		validator.schemas = registry
		validator.validateRelationships(postSchema)
		if len(validator.errors) == 0 {
			t.Error("expected error for set_null on non-nullable relationship")
		}
	})
}

func TestCheckTypeMatch(t *testing.T) {
	validator := NewSchemaValidator()

	tests := []struct {
		name        string
		typeSpec    *TypeSpec
		value       interface{}
		expectError bool
	}{
		{
			name: "int matches int",
			typeSpec: &TypeSpec{
				BaseType: TypeInt,
			},
			value:       42,
			expectError: false,
		},
		{
			name: "string matches string type",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
			},
			value:       "hello",
			expectError: false,
		},
		{
			name: "int doesn't match string type",
			typeSpec: &TypeSpec{
				BaseType: TypeString,
			},
			value:       42,
			expectError: true,
		},
		{
			name: "bool matches bool",
			typeSpec: &TypeSpec{
				BaseType: TypeBool,
			},
			value:       true,
			expectError: false,
		},
		{
			name: "float matches float",
			typeSpec: &TypeSpec{
				BaseType: TypeFloat,
			},
			value:       3.14,
			expectError: false,
		},
		{
			name: "valid enum value",
			typeSpec: &TypeSpec{
				BaseType:   TypeEnum,
				EnumValues: []string{"draft", "published"},
			},
			value:       "draft",
			expectError: false,
		},
		{
			name: "invalid enum value",
			typeSpec: &TypeSpec{
				BaseType:   TypeEnum,
				EnumValues: []string{"draft", "published"},
			},
			value:       "archived",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkTypeMatch(tt.typeSpec, tt.value)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Resource: "Post",
		Field:    "title",
		Message:  "field is required",
		Location: ast.SourceLocation{
			Line:   10,
			Column: 5,
		},
		Hint: "Add a title field",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Post.title") {
		t.Errorf("error message should contain resource and field: %s", errMsg)
	}
	if !strings.Contains(errMsg, "line 10") {
		t.Errorf("error message should contain location: %s", errMsg)
	}
	if !strings.Contains(errMsg, "hint:") {
		t.Errorf("error message should contain hint: %s", errMsg)
	}
}
