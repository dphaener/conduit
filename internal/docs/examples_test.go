package docs

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestExampleGenerator_GenerateForType(t *testing.T) {
	gen := NewExampleGenerator()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		checkFn  func(interface{}) bool
	}{
		{
			name:     "string type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
			checkFn: func(v interface{}) bool {
				s, ok := v.(string)
				return ok && s == "example string"
			},
		},
		{
			name:     "uuid type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"},
			checkFn: func(v interface{}) bool {
				s, ok := v.(string)
				return ok && len(s) == 36 // UUID format
			},
		},
		{
			name:     "email type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email"},
			checkFn: func(v interface{}) bool {
				s, ok := v.(string)
				return ok && s == "user@example.com"
			},
		},
		{
			name:     "int type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
			checkFn: func(v interface{}) bool {
				i, ok := v.(int)
				return ok && i == 42
			},
		},
		{
			name:     "float type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"},
			checkFn: func(v interface{}) bool {
				f, ok := v.(float64)
				return ok && f == 3.14
			},
		},
		{
			name:     "bool type",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"},
			checkFn: func(v interface{}) bool {
				b, ok := v.(bool)
				return ok && b == true
			},
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind:        ast.TypeArray,
				ElementType: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
			},
			checkFn: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 1
			},
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind:      ast.TypeHash,
				KeyType:   &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				ValueType: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
			},
			checkFn: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && len(m) > 0
			},
		},
		{
			name: "enum type",
			typeNode: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				EnumValues: []string{"active", "inactive"},
			},
			checkFn: func(v interface{}) bool {
				s, ok := v.(string)
				return ok && s == "active"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateForType(tt.typeNode)
			if !tt.checkFn(result) {
				t.Errorf("GenerateForType(%s) returned unexpected value: %v", tt.name, result)
			}
		})
	}
}

func TestExampleGenerator_GeneratePrimitiveExample(t *testing.T) {
	gen := NewExampleGenerator()

	tests := []struct {
		typeName string
		checkFn  func(interface{}) bool
	}{
		{
			typeName: "string",
			checkFn:  func(v interface{}) bool { _, ok := v.(string); return ok },
		},
		{
			typeName: "text",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) > 20 },
		},
		{
			typeName: "uuid",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) == 36 },
		},
		{
			typeName: "email",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) > 0 },
		},
		{
			typeName: "url",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) > 0 },
		},
		{
			typeName: "int",
			checkFn:  func(v interface{}) bool { _, ok := v.(int); return ok },
		},
		{
			typeName: "bigint",
			checkFn:  func(v interface{}) bool { _, ok := v.(int); return ok },
		},
		{
			typeName: "float",
			checkFn:  func(v interface{}) bool { _, ok := v.(float64); return ok },
		},
		{
			typeName: "decimal",
			checkFn:  func(v interface{}) bool { _, ok := v.(string); return ok },
		},
		{
			typeName: "bool",
			checkFn:  func(v interface{}) bool { _, ok := v.(bool); return ok },
		},
		{
			typeName: "date",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) == 10 },
		},
		{
			typeName: "datetime",
			checkFn:  func(v interface{}) bool { s, ok := v.(string); return ok && len(s) > 10 },
		},
		{
			typeName: "json",
			checkFn:  func(v interface{}) bool { _, ok := v.(map[string]interface{}); return ok },
		},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := gen.generatePrimitiveExample(tt.typeName)
			if !tt.checkFn(result) {
				t.Errorf("generatePrimitiveExample(%s) returned unexpected type: %T %v", tt.typeName, result, result)
			}
		})
	}
}

func TestExampleGenerator_Counter(t *testing.T) {
	gen := NewExampleGenerator()

	// Generate multiple examples to test counter
	for i := 0; i < 5; i++ {
		gen.GenerateForType(&ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"})
	}

	if gen.counter != 5 {
		t.Errorf("Expected counter to be 5, got %d", gen.counter)
	}
}
