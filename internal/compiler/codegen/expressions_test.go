package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateLiteral(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.LiteralExpr
		expected string
	}{
		{
			name:     "string literal",
			expr:     &ast.LiteralExpr{Value: "hello"},
			expected: `"hello"`,
		},
		{
			name:     "int literal",
			expr:     &ast.LiteralExpr{Value: int64(42)},
			expected: "42",
		},
		{
			name:     "float literal",
			expr:     &ast.LiteralExpr{Value: 3.14},
			expected: "3.14",
		},
		{
			name:     "bool true",
			expr:     &ast.LiteralExpr{Value: true},
			expected: "true",
		},
		{
			name:     "bool false",
			expr:     &ast.LiteralExpr{Value: false},
			expected: "false",
		},
		{
			name:     "nil literal",
			expr:     &ast.LiteralExpr{Value: nil},
			expected: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateFieldAccess(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.FieldAccessExpr
		expected string
	}{
		{
			name: "self.title",
			expr: &ast.FieldAccessExpr{
				Object: &ast.SelfExpr{},
				Field:  "title",
			},
			expected: "self.Title",
		},
		{
			name: "user.email",
			expr: &ast.FieldAccessExpr{
				Object: &ast.IdentifierExpr{Name: "user"},
				Field:  "email",
			},
			expected: "user.Email",
		},
		{
			name: "post.author_id",
			expr: &ast.FieldAccessExpr{
				Object: &ast.IdentifierExpr{Name: "post"},
				Field:  "author_id",
			},
			expected: "post.AuthorId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateBinaryExpr(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.BinaryExpr
		expected string
	}{
		{
			name: "addition",
			expr: &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: int64(1)},
				Operator: "+",
				Right:    &ast.LiteralExpr{Value: int64(2)},
			},
			expected: "1 + 2",
		},
		{
			name: "subtraction",
			expr: &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: int64(5)},
				Operator: "-",
				Right:    &ast.LiteralExpr{Value: int64(3)},
			},
			expected: "5 - 3",
		},
		{
			name: "multiplication",
			expr: &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: int64(4)},
				Operator: "*",
				Right:    &ast.LiteralExpr{Value: int64(5)},
			},
			expected: "4 * 5",
		},
		{
			name: "division",
			expr: &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: int64(10)},
				Operator: "/",
				Right:    &ast.LiteralExpr{Value: int64(2)},
			},
			expected: "10 / 2",
		},
		{
			name: "equality",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "x"},
				Operator: "==",
				Right:    &ast.LiteralExpr{Value: int64(42)},
			},
			expected: "x == 42",
		},
		{
			name: "inequality",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "y"},
				Operator: "!=",
				Right:    &ast.LiteralExpr{Value: int64(0)},
			},
			expected: "y != 0",
		},
		{
			name: "less than",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "age"},
				Operator: "<",
				Right:    &ast.LiteralExpr{Value: int64(18)},
			},
			expected: "age < 18",
		},
		{
			name: "exponentiation",
			expr: &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: int64(2)},
				Operator: "**",
				Right:    &ast.LiteralExpr{Value: int64(3)},
			},
			expected: "math.Pow(2, 3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateUnaryExpr(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.UnaryExpr
		expected string
	}{
		{
			name: "negation",
			expr: &ast.UnaryExpr{
				Operator: "-",
				Operand:  &ast.LiteralExpr{Value: int64(42)},
			},
			expected: "-42",
		},
		{
			name: "logical not",
			expr: &ast.UnaryExpr{
				Operator: "not",
				Operand:  &ast.IdentifierExpr{Name: "is_admin"},
			},
			expected: "!(is_admin)",
		},
		{
			name: "unwrap",
			expr: &ast.UnaryExpr{
				Operator: "!",
				Operand:  &ast.IdentifierExpr{Name: "maybe_value"},
			},
			expected: "*(maybe_value)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateLogicalExpr(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.LogicalExpr
		expected string
	}{
		{
			name: "logical and with &&",
			expr: &ast.LogicalExpr{
				Left:     &ast.IdentifierExpr{Name: "is_admin"},
				Operator: "&&",
				Right:    &ast.IdentifierExpr{Name: "is_active"},
			},
			expected: "is_admin && is_active",
		},
		{
			name: "logical and with 'and'",
			expr: &ast.LogicalExpr{
				Left:     &ast.IdentifierExpr{Name: "is_admin"},
				Operator: "and",
				Right:    &ast.IdentifierExpr{Name: "is_active"},
			},
			expected: "is_admin && is_active",
		},
		{
			name: "logical or with ||",
			expr: &ast.LogicalExpr{
				Left:     &ast.IdentifierExpr{Name: "is_admin"},
				Operator: "||",
				Right:    &ast.IdentifierExpr{Name: "is_moderator"},
			},
			expected: "is_admin || is_moderator",
		},
		{
			name: "logical or with 'or'",
			expr: &ast.LogicalExpr{
				Left:     &ast.IdentifierExpr{Name: "is_admin"},
				Operator: "or",
				Right:    &ast.IdentifierExpr{Name: "is_moderator"},
			},
			expected: "is_admin || is_moderator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateCallExpr(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.CallExpr
		contains string // Use contains for complex generated code
	}{
		{
			name: "String.slugify",
			expr: &ast.CallExpr{
				Namespace: "String",
				Function:  "slugify",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: "Hello World"},
				},
			},
			contains: "StringSlugify",
		},
		{
			name: "String.upcase",
			expr: &ast.CallExpr{
				Namespace: "String",
				Function:  "upcase",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: "hello"},
				},
			},
			contains: "strings.ToUpper",
		},
		{
			name: "String.downcase",
			expr: &ast.CallExpr{
				Namespace: "String",
				Function:  "downcase",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: "HELLO"},
				},
			},
			contains: "strings.ToLower",
		},
		{
			name: "Time.now",
			expr: &ast.CallExpr{
				Namespace: "Time",
				Function:  "now",
				Arguments: []ast.ExprNode{},
			},
			contains: "time.Now()",
		},
		{
			name: "Logger.info",
			expr: &ast.CallExpr{
				Namespace: "Logger",
				Function:  "info",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: "User logged in"},
				},
			},
			contains: "log.Printf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("generateExpr() = %v, should contain %v", result, tt.contains)
			}
		})
	}
}

func TestGenerateArrayLiteral(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.ArrayLiteralExpr
		expected string
	}{
		{
			name:     "empty array",
			expr:     &ast.ArrayLiteralExpr{Elements: []ast.ExprNode{}},
			expected: "[]interface{}{}",
		},
		{
			name: "array of integers",
			expr: &ast.ArrayLiteralExpr{
				Elements: []ast.ExprNode{
					&ast.LiteralExpr{Value: int64(1)},
					&ast.LiteralExpr{Value: int64(2)},
					&ast.LiteralExpr{Value: int64(3)},
				},
			},
			expected: "[]interface{}{1, 2, 3}",
		},
		{
			name: "array of strings",
			expr: &ast.ArrayLiteralExpr{
				Elements: []ast.ExprNode{
					&ast.LiteralExpr{Value: "foo"},
					&ast.LiteralExpr{Value: "bar"},
				},
			},
			expected: `[]interface{}{"foo", "bar"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateHashLiteral(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.HashLiteralExpr
		contains []string
	}{
		{
			name:     "empty hash",
			expr:     &ast.HashLiteralExpr{Pairs: []ast.HashPair{}},
			contains: []string{"map[string]interface{}{}"},
		},
		{
			name: "hash with string keys",
			expr: &ast.HashLiteralExpr{
				Pairs: []ast.HashPair{
					{
						Key:   &ast.LiteralExpr{Value: "name"},
						Value: &ast.LiteralExpr{Value: "John"},
					},
					{
						Key:   &ast.LiteralExpr{Value: "age"},
						Value: &ast.LiteralExpr{Value: int64(30)},
					},
				},
			},
			contains: []string{"map[string]interface{}", `"name": "John"`, `"age": 30`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("generateExpr() = %v, should contain %v", result, expected)
				}
			}
		})
	}
}

func TestGenerateIndexExpr(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		expr     *ast.IndexExpr
		expected string
	}{
		{
			name: "array index",
			expr: &ast.IndexExpr{
				Object: &ast.IdentifierExpr{Name: "arr"},
				Index:  &ast.LiteralExpr{Value: int64(0)},
			},
			expected: "arr[0]",
		},
		{
			name: "hash index",
			expr: &ast.IndexExpr{
				Object: &ast.IdentifierExpr{Name: "data"},
				Index:  &ast.LiteralExpr{Value: "key"},
			},
			expected: `data["key"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("generateExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateNullCoalesce(t *testing.T) {
	gen := NewGenerator()

	expr := &ast.NullCoalesceExpr{
		Left:  &ast.IdentifierExpr{Name: "maybe_value"},
		Right: &ast.LiteralExpr{Value: "default"},
	}

	result := gen.generateExpr(expr)

	// Check that it generates a function-style expression with nil check
	if !strings.Contains(result, "maybe_value") {
		t.Error("generateExpr() should reference left operand")
	}
	if !strings.Contains(result, "nil") {
		t.Error("generateExpr() should check for nil")
	}
	if !strings.Contains(result, `"default"`) {
		t.Error("generateExpr() should include default value")
	}
}

func TestGenerateParenExpr(t *testing.T) {
	gen := NewGenerator()

	expr := &ast.ParenExpr{
		Expr: &ast.BinaryExpr{
			Left:     &ast.LiteralExpr{Value: int64(1)},
			Operator: "+",
			Right:    &ast.LiteralExpr{Value: int64(2)},
		},
	}

	result := gen.generateExpr(expr)
	expected := "(1 + 2)"

	if result != expected {
		t.Errorf("generateExpr() = %v, want %v", result, expected)
	}
}

func TestGenerateComplexExpression(t *testing.T) {
	gen := NewGenerator()

	// Test: self.status == "published" && self.views > 100
	expr := &ast.LogicalExpr{
		Left: &ast.BinaryExpr{
			Left: &ast.FieldAccessExpr{
				Object: &ast.SelfExpr{},
				Field:  "status",
			},
			Operator: "==",
			Right:    &ast.LiteralExpr{Value: "published"},
		},
		Operator: "&&",
		Right: &ast.BinaryExpr{
			Left: &ast.FieldAccessExpr{
				Object: &ast.SelfExpr{},
				Field:  "views",
			},
			Operator: ">",
			Right:    &ast.LiteralExpr{Value: int64(100)},
		},
	}

	result := gen.generateExpr(expr)

	// Check that all components are present
	if !strings.Contains(result, "self.Status") {
		t.Error("Result should contain self.Status")
	}
	if !strings.Contains(result, `"published"`) {
		t.Error("Result should contain 'published'")
	}
	if !strings.Contains(result, "&&") {
		t.Error("Result should contain &&")
	}
	if !strings.Contains(result, "self.Views") {
		t.Error("Result should contain self.Views")
	}
	if !strings.Contains(result, "> 100") {
		t.Error("Result should contain > 100")
	}
}

func TestGenerateInterpolatedString(t *testing.T) {
	gen := NewGenerator()

	expr := &ast.InterpolatedStringExpr{
		Parts: []ast.ExprNode{
			&ast.LiteralExpr{Value: "Hello, "},
			&ast.IdentifierExpr{Name: "name"},
			&ast.LiteralExpr{Value: "!"},
		},
	}

	result := gen.generateExpr(expr)

	// Check that it uses fmt.Sprintf
	if !strings.Contains(result, "fmt.Sprintf") {
		t.Error("Result should use fmt.Sprintf for string interpolation")
	}
	if !strings.Contains(result, "name") {
		t.Error("Result should include the interpolated variable")
	}
}
