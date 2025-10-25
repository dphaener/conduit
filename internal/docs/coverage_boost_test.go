package docs

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Test more edge cases in generatePrimitiveExample
func TestExampleGenerator_GeneratePrimitiveExample_AllTypes(t *testing.T) {
	gen := NewExampleGenerator()

	types := []string{
		"string", "text", "uuid", "email", "url", "slug", "json", "jsonb",
		"int", "integer", "bigint", "float", "decimal", "money",
		"bool", "boolean",
		"date", "time", "datetime", "timestamp",
		"binary", "bytea",
		"unknown",
	}

	for _, typeName := range types {
		t.Run(typeName, func(t *testing.T) {
			result := gen.generatePrimitiveExample(typeName)
			if result == nil {
				t.Errorf("generatePrimitiveExample(%s) returned nil", typeName)
			}
		})
	}
}

// Test GenerateForType with all type kinds
func TestExampleGenerator_GenerateForType_AllKinds(t *testing.T) {
	gen := NewExampleGenerator()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
	}{
		{
			name: "primitive type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "string",
			},
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeArray,
				ElementType: &ast.TypeNode{
					Kind: ast.TypePrimitive,
					Name: "int",
				},
			},
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeHash,
				KeyType: &ast.TypeNode{
					Kind: ast.TypePrimitive,
					Name: "string",
				},
				ValueType: &ast.TypeNode{
					Kind: ast.TypePrimitive,
					Name: "int",
				},
			},
		},
		{
			name: "enum type with values",
			typeNode: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				EnumValues: []string{"active", "inactive", "pending"},
			},
		},
		{
			name: "enum type without values",
			typeNode: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				EnumValues: []string{},
			},
		},
		{
			name: "resource type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeResource,
				Name: "User",
			},
		},
		{
			name: "unknown type",
			typeNode: &ast.TypeNode{
				Kind: 999, // Unknown kind
			},
		},
		{
			name:     "nil type",
			typeNode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateForType(tt.typeNode)
			if result == nil {
				t.Error("GenerateForType returned nil")
			}
		})
	}
}

// Test formatType with all combinations
func TestExtractor_FormatType_AllCombinations(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		want     string
	}{
		{
			name:     "nil type",
			typeNode: nil,
			want:     "unknown",
		},
		{
			name: "required primitive",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "string",
				Nullable: false,
			},
			want: "string!",
		},
		{
			name: "optional primitive",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "int",
				Nullable: true,
			},
			want: "int?",
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeArray,
				Nullable: false,
				ElementType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
			},
			want: "array<string!>!",
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeHash,
				Nullable: false,
				KeyType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				ValueType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
			},
			want: "hash<string!,int!>!",
		},
		{
			name: "enum type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeEnum,
				Nullable: false,
			},
			want: "enum!",
		},
		{
			name: "resource type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeResource,
				Name:     "User",
				Nullable: false,
			},
			want: "User!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatType(tt.typeNode)
			if result != tt.want {
				t.Errorf("formatType() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Test formatConstraint with arguments
func TestExtractor_FormatConstraint_WithArguments(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name       string
		constraint *ast.ConstraintNode
		wantPrefix string
	}{
		{
			name: "constraint without arguments",
			constraint: &ast.ConstraintNode{
				Name:      "unique",
				Arguments: []ast.ExprNode{},
			},
			wantPrefix: "@unique",
		},
		{
			name: "constraint with one argument",
			constraint: &ast.ConstraintNode{
				Name: "min",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: 5},
				},
			},
			wantPrefix: "@min(5)",
		},
		{
			name: "constraint with multiple arguments",
			constraint: &ast.ConstraintNode{
				Name: "between",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: 1},
					&ast.LiteralExpr{Value: 100},
				},
			},
			wantPrefix: "@between(1, 100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatConstraint(tt.constraint)
			if result != tt.wantPrefix {
				t.Errorf("formatConstraint() = %v, want %v", result, tt.wantPrefix)
			}
		})
	}
}

// Test schemaTypeForFieldType with all types
func TestExtractor_SchemaTypeForFieldType_AllTypes(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		want     string
	}{
		{
			name:     "nil type",
			typeNode: nil,
			want:     "string",
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeArray,
			},
			want: "array",
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeHash,
			},
			want: "object",
		},
		{
			name: "int primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "int",
			},
			want: "integer",
		},
		{
			name: "integer primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "integer",
			},
			want: "integer",
		},
		{
			name: "float primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "float",
			},
			want: "number",
		},
		{
			name: "decimal primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "decimal",
			},
			want: "number",
		},
		{
			name: "bool primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "bool",
			},
			want: "boolean",
		},
		{
			name: "boolean primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "boolean",
			},
			want: "boolean",
		},
		{
			name: "string primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "string",
			},
			want: "string",
		},
		{
			name: "unknown primitive",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "custom",
			},
			want: "string",
		},
		{
			name: "unknown kind",
			typeNode: &ast.TypeNode{
				Kind: 999,
			},
			want: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.schemaTypeForFieldType(tt.typeNode)
			if result != tt.want {
				t.Errorf("schemaTypeForFieldType() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Test schemaFormatForFieldType with all formats
func TestExtractor_SchemaFormatForFieldType_AllFormats(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		want     string
	}{
		{
			name:     "nil type",
			typeNode: nil,
			want:     "",
		},
		{
			name: "non-primitive type",
			typeNode: &ast.TypeNode{
				Kind: ast.TypeArray,
			},
			want: "",
		},
		{
			name: "uuid format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "uuid",
			},
			want: "uuid",
		},
		{
			name: "date format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "date",
			},
			want: "date",
		},
		{
			name: "datetime format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "datetime",
			},
			want: "date-time",
		},
		{
			name: "timestamp format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "timestamp",
			},
			want: "date-time",
		},
		{
			name: "email format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "email",
			},
			want: "email",
		},
		{
			name: "url format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "url",
			},
			want: "uri",
		},
		{
			name: "no special format",
			typeNode: &ast.TypeNode{
				Kind: ast.TypePrimitive,
				Name: "string",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.schemaFormatForFieldType(tt.typeNode)
			if result != tt.want {
				t.Errorf("schemaFormatForFieldType() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Test extractResource with hooks, validations, and constraints
func TestExtractor_ExtractResource_Complete(t *testing.T) {
	extractor := NewExtractor()

	resource := &ast.ResourceNode{
		Name:          "Post",
		Documentation: "Blog post",
		Fields: []*ast.FieldNode{
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
				Nullable: false,
			},
		},
		Relationships: []*ast.RelationshipNode{
			{
				Name: "author",
				Type: "User",
				Kind: ast.RelationshipBelongsTo,
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
			},
		},
		Validations: []*ast.ValidationNode{
			{
				Name:  "title_length",
				Error: "Title too short",
			},
		},
		Constraints: []*ast.ConstraintNode{
			{
				Name: "unique",
				On:   []string{"create"},
			},
		},
	}

	result := extractor.extractResource(resource)

	if result.Name != "Post" {
		t.Errorf("Name = %v, want Post", result.Name)
	}

	if len(result.Fields) != 1 {
		t.Errorf("Fields count = %v, want 1", len(result.Fields))
	}

	if len(result.Relationships) != 1 {
		t.Errorf("Relationships count = %v, want 1", len(result.Relationships))
	}

	if len(result.Hooks) != 1 {
		t.Errorf("Hooks count = %v, want 1", len(result.Hooks))
	}

	if len(result.Validations) != 1 {
		t.Errorf("Validations count = %v, want 1", len(result.Validations))
	}

	if len(result.Constraints) != 1 {
		t.Errorf("Constraints count = %v, want 1", len(result.Constraints))
	}

	// Endpoints should be auto-generated
	if len(result.Endpoints) < 5 {
		t.Errorf("Expected at least 5 endpoints, got %d", len(result.Endpoints))
	}
}
