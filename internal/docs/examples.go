package docs

import (
	"fmt"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// ExampleGenerator generates example values for field types
type ExampleGenerator struct {
	counter int
}

// NewExampleGenerator creates a new example generator
func NewExampleGenerator() *ExampleGenerator {
	return &ExampleGenerator{
		counter: 0,
	}
}

// GenerateForType generates an example value for a given type
func (g *ExampleGenerator) GenerateForType(typeNode *ast.TypeNode) interface{} {
	if typeNode == nil {
		return "example"
	}

	g.counter++

	switch typeNode.Kind {
	case ast.TypePrimitive:
		return g.generatePrimitiveExample(typeNode.Name)
	case ast.TypeArray:
		itemExample := g.GenerateForType(typeNode.ElementType)
		return []interface{}{itemExample}
	case ast.TypeHash:
		keyExample := g.GenerateForType(typeNode.KeyType)
		valueExample := g.GenerateForType(typeNode.ValueType)
		return map[string]interface{}{
			fmt.Sprintf("%v", keyExample): valueExample,
		}
	case ast.TypeEnum:
		if len(typeNode.EnumValues) > 0 {
			return typeNode.EnumValues[0]
		}
		return "value1"
	case ast.TypeResource:
		// For resource types, return the ID
		return "550e8400-e29b-41d4-a716-446655440000"
	default:
		return "example"
	}
}

// generatePrimitiveExample generates an example for a primitive type
func (g *ExampleGenerator) generatePrimitiveExample(typeName string) interface{} {
	switch typeName {
	// String types
	case "string":
		return "example string"
	case "text":
		return "This is example text content that can be much longer than a regular string field."
	case "uuid":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "email":
		return "user@example.com"
	case "url":
		return "https://example.com"
	case "slug":
		return "example-slug"
	case "json":
		return map[string]interface{}{"key": "value"}
	case "jsonb":
		return map[string]interface{}{"key": "value"}

	// Numeric types
	case "int", "integer":
		return 42
	case "bigint":
		return 9223372036854775807
	case "float":
		return 3.14
	case "decimal":
		return "99.99"
	case "money":
		return "100.00"

	// Boolean
	case "bool", "boolean":
		return true

	// Date/time types
	case "date":
		return time.Now().Format("2006-01-02")
	case "time":
		return time.Now().Format("15:04:05")
	case "datetime", "timestamp":
		return time.Now().Format(time.RFC3339)

	// Binary types
	case "binary", "bytea":
		return "base64encodedcontent=="

	default:
		return "example"
	}
}

// GenerateForField generates an example value for a field
func (g *ExampleGenerator) GenerateForField(field *ast.FieldNode) interface{} {
	// If field has a default, use that
	if field.Default != nil {
		return g.formatDefault(field.Default)
	}

	// Otherwise generate based on type
	return g.GenerateForType(field.Type)
}

// formatDefault formats a default value expression
func (g *ExampleGenerator) formatDefault(expr ast.ExprNode) interface{} {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return e.Value
	case *ast.IdentifierExpr:
		return e.Name
	default:
		return "default value"
	}
}
