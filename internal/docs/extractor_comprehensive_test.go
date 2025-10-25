package docs

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestExtractor_ExtractRelationship(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		rel      *ast.RelationshipNode
		expected *RelationshipDoc
	}{
		{
			name: "belongs_to relationship",
			rel: &ast.RelationshipNode{
				Name:       "author",
				Type:       "User",
				Kind:       ast.RelationshipBelongsTo,
				ForeignKey: "author_id",
			},
			expected: &RelationshipDoc{
				Name:       "author",
				Type:       "User",
				Kind:       "belongs_to",
				ForeignKey: "author_id",
			},
		},
		{
			name: "has_many relationship",
			rel: &ast.RelationshipNode{
				Name:       "posts",
				Type:       "Post",
				Kind:       ast.RelationshipHasMany,
				ForeignKey: "user_id",
			},
			expected: &RelationshipDoc{
				Name:       "posts",
				Type:       "Post",
				Kind:       "has_many",
				ForeignKey: "user_id",
			},
		},
		{
			name: "has_many_through relationship",
			rel: &ast.RelationshipNode{
				Name:       "followers",
				Type:       "User",
				Kind:       ast.RelationshipHasManyThrough,
				ForeignKey: "",
			},
			expected: &RelationshipDoc{
				Name:       "followers",
				Type:       "User",
				Kind:       "has_many_through",
				ForeignKey: "user_id",
			},
		},
		{
			name: "relationship without foreign key",
			rel: &ast.RelationshipNode{
				Name: "owner",
				Type: "User",
				Kind: ast.RelationshipBelongsTo,
			},
			expected: &RelationshipDoc{
				Name:       "owner",
				Type:       "User",
				Kind:       "belongs_to",
				ForeignKey: "user_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.extractRelationship(tt.rel)

			if result.Name != tt.expected.Name {
				t.Errorf("Name = %v, want %v", result.Name, tt.expected.Name)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type = %v, want %v", result.Type, tt.expected.Type)
			}
			if result.Kind != tt.expected.Kind {
				t.Errorf("Kind = %v, want %v", result.Kind, tt.expected.Kind)
			}
			if result.ForeignKey != tt.expected.ForeignKey {
				t.Errorf("ForeignKey = %v, want %v", result.ForeignKey, tt.expected.ForeignKey)
			}
		})
	}
}

func TestExtractor_ExtractValidation(t *testing.T) {
	extractor := NewExtractor()

	validation := &ast.ValidationNode{
		Name:  "email_format",
		Error: "Invalid email format",
	}

	result := extractor.extractValidation(validation)

	if result.Name != "email_format" {
		t.Errorf("Name = %v, want email_format", result.Name)
	}

	if result.ErrorMessage != "Invalid email format" {
		t.Errorf("ErrorMessage = %v, want 'Invalid email format'", result.ErrorMessage)
	}

	if result.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestExtractor_ExtractConstraint(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name       string
		constraint *ast.ConstraintNode
		wantName   string
		wantArgs   int
		wantOn     []string
	}{
		{
			name: "constraint with arguments",
			constraint: &ast.ConstraintNode{
				Name: "min",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: 5},
				},
				On: []string{"create", "update"},
			},
			wantName: "min",
			wantArgs: 1,
			wantOn:   []string{"create", "update"},
		},
		{
			name: "constraint without arguments",
			constraint: &ast.ConstraintNode{
				Name:      "unique",
				Arguments: []ast.ExprNode{},
				On:        []string{"create"},
			},
			wantName: "unique",
			wantArgs: 0,
			wantOn:   []string{"create"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.extractConstraint(tt.constraint)

			if result.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", result.Name, tt.wantName)
			}

			if len(result.Arguments) != tt.wantArgs {
				t.Errorf("Arguments count = %v, want %v", len(result.Arguments), tt.wantArgs)
			}

			if len(result.On) != len(tt.wantOn) {
				t.Errorf("On count = %v, want %v", len(result.On), len(tt.wantOn))
			}
		})
	}
}

func TestExtractor_FormatExpression(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		expr     ast.ExprNode
		expected string
	}{
		{
			name:     "nil expression",
			expr:     nil,
			expected: "",
		},
		{
			name:     "literal expression",
			expr:     &ast.LiteralExpr{Value: 42},
			expected: "42",
		},
		{
			name:     "identifier expression",
			expr:     &ast.IdentifierExpr{Name: "self.status"},
			expected: "self.status",
		},
		{
			name: "binary expression",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "age"},
				Operator: ">",
				Right:    &ast.LiteralExpr{Value: 18},
			},
			expected: "age > 18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("formatExpression() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExampleGenerator_GenerateForField(t *testing.T) {
	gen := NewExampleGenerator()

	tests := []struct {
		name  string
		field *ast.FieldNode
	}{
		{
			name: "field with default",
			field: &ast.FieldNode{
				Name: "status",
				Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
				Default: &ast.LiteralExpr{Value: "active"},
			},
		},
		{
			name: "field without default",
			field: &ast.FieldNode{
				Name: "email",
				Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email", Nullable: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateForField(tt.field)
			if result == nil {
				t.Error("GenerateForField returned nil")
			}
		})
	}
}

func TestExampleGenerator_FormatDefault(t *testing.T) {
	gen := NewExampleGenerator()

	tests := []struct {
		name     string
		expr     ast.ExprNode
		expected interface{}
	}{
		{
			name:     "nil expression",
			expr:     nil,
			expected: nil,
		},
		{
			name:     "literal expression",
			expr:     &ast.LiteralExpr{Value: "default"},
			expected: "default",
		},
		{
			name:     "identifier expression",
			expr:     &ast.IdentifierExpr{Name: "pending"},
			expected: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.formatDefault(tt.expr)
			if result != tt.expected {
				t.Errorf("formatDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
