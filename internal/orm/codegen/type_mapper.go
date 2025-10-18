// Package codegen provides code generation for database schema DDL.
// It transforms validated Conduit resource definitions into PostgreSQL CREATE TABLE statements.
package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// TypeMapper maps Conduit types to PostgreSQL column types
type TypeMapper struct{}

// NewTypeMapper creates a new TypeMapper
func NewTypeMapper() *TypeMapper {
	return &TypeMapper{}
}

// MapType converts a Conduit TypeSpec to a PostgreSQL column type
func (tm *TypeMapper) MapType(typeSpec *schema.TypeSpec) (string, error) {
	if typeSpec == nil {
		return "", fmt.Errorf("type spec cannot be nil")
	}

	// Handle array types
	if typeSpec.ArrayElement != nil {
		elementType, err := tm.MapType(typeSpec.ArrayElement)
		if err != nil {
			return "", fmt.Errorf("array element: %w", err)
		}
		return elementType + "[]", nil
	}

	// Handle hash types (map to JSONB)
	if typeSpec.HashKey != nil && typeSpec.HashValue != nil {
		return "JSONB", nil
	}

	// Handle struct types (map to JSONB)
	if len(typeSpec.StructFields) > 0 {
		return "JSONB", nil
	}

	// Handle enum types (will be created as ENUM type)
	if len(typeSpec.EnumValues) > 0 {
		// Enum type name will be handled separately
		// This returns a placeholder that will be replaced with the actual enum type name
		return "ENUM", nil
	}

	// Handle primitive types
	return tm.mapPrimitiveType(typeSpec)
}

// mapPrimitiveType maps a primitive type to PostgreSQL
func (tm *TypeMapper) mapPrimitiveType(typeSpec *schema.TypeSpec) (string, error) {
	switch typeSpec.BaseType {
	case schema.TypeString:
		if typeSpec.Length != nil {
			return fmt.Sprintf("VARCHAR(%d)", *typeSpec.Length), nil
		}
		return "VARCHAR(255)", nil // Default length

	case schema.TypeText, schema.TypeMarkdown:
		return "TEXT", nil

	case schema.TypeInt:
		return "INTEGER", nil

	case schema.TypeBigInt:
		return "BIGINT", nil

	case schema.TypeFloat:
		return "DOUBLE PRECISION", nil

	case schema.TypeDecimal:
		if typeSpec.Precision != nil && typeSpec.Scale != nil {
			return fmt.Sprintf("NUMERIC(%d,%d)", *typeSpec.Precision, *typeSpec.Scale), nil
		}
		return "NUMERIC", nil

	case schema.TypeBool:
		return "BOOLEAN", nil

	case schema.TypeTimestamp:
		return "TIMESTAMP WITH TIME ZONE", nil

	case schema.TypeDate:
		return "DATE", nil

	case schema.TypeTime:
		return "TIME", nil

	case schema.TypeUUID:
		return "UUID", nil

	case schema.TypeULID:
		// ULID is stored as a 26-character string
		return "CHAR(26)", nil

	case schema.TypeEmail, schema.TypeURL, schema.TypePhone:
		// Validated types are stored as strings
		return "VARCHAR(255)", nil

	case schema.TypeJSON:
		return "JSON", nil

	case schema.TypeJSONB:
		return "JSONB", nil

	default:
		return "", fmt.Errorf("unsupported type: %s", typeSpec.BaseType)
	}
}

// MapNullability returns the NULL/NOT NULL constraint for a type
func (tm *TypeMapper) MapNullability(typeSpec *schema.TypeSpec) string {
	if typeSpec.Nullable {
		return "NULL"
	}
	return "NOT NULL"
}

// MapDefault generates a DEFAULT clause for a field if it has a default value
func (tm *TypeMapper) MapDefault(typeSpec *schema.TypeSpec) (string, error) {
	if typeSpec.Default == nil {
		return "", nil
	}

	// Format the default value based on type
	return tm.formatDefaultValue(typeSpec, typeSpec.Default)
}

// formatDefaultValue formats a default value for SQL
func (tm *TypeMapper) formatDefaultValue(typeSpec *schema.TypeSpec, value interface{}) (string, error) {
	switch typeSpec.BaseType {
	case schema.TypeString, schema.TypeText, schema.TypeMarkdown,
		schema.TypeEmail, schema.TypeURL, schema.TypePhone:
		if str, ok := value.(string); ok {
			// Escape single quotes by doubling them
			escaped := strings.ReplaceAll(str, "'", "''")
			return fmt.Sprintf("'%s'", escaped), nil
		}
		return "", fmt.Errorf("expected string for text type, got %T", value)

	case schema.TypeInt, schema.TypeBigInt:
		if num, ok := value.(int); ok {
			return fmt.Sprintf("%d", num), nil
		}
		if num, ok := value.(int64); ok {
			return fmt.Sprintf("%d", num), nil
		}
		return "", fmt.Errorf("expected int for integer type, got %T", value)

	case schema.TypeFloat, schema.TypeDecimal:
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("%f", v), nil
		case float32:
			return fmt.Sprintf("%f", v), nil
		case int:
			return fmt.Sprintf("%d", v), nil
		default:
			return "", fmt.Errorf("expected numeric for float/decimal type, got %T", value)
		}

	case schema.TypeBool:
		if b, ok := value.(bool); ok {
			if b {
				return "TRUE", nil
			}
			return "FALSE", nil
		}
		return "", fmt.Errorf("expected bool for boolean type, got %T", value)

	case schema.TypeUUID:
		if str, ok := value.(string); ok {
			return fmt.Sprintf("'%s'::uuid", str), nil
		}
		return "", fmt.Errorf("expected string for UUID type, got %T", value)

	case schema.TypeTimestamp:
		if str, ok := value.(string); ok {
			if str == "now()" || str == "CURRENT_TIMESTAMP" {
				return "CURRENT_TIMESTAMP", nil
			}
			return fmt.Sprintf("'%s'::timestamp", str), nil
		}
		return "", fmt.Errorf("expected string for timestamp type, got %T", value)

	case schema.TypeDate:
		if str, ok := value.(string); ok {
			if str == "today()" || str == "CURRENT_DATE" {
				return "CURRENT_DATE", nil
			}
			return fmt.Sprintf("'%s'::date", str), nil
		}
		return "", fmt.Errorf("expected string for date type, got %T", value)

	case schema.TypeTime:
		if str, ok := value.(string); ok {
			if str == "now()" || str == "CURRENT_TIME" {
				return "CURRENT_TIME", nil
			}
			return fmt.Sprintf("'%s'::time", str), nil
		}
		return "", fmt.Errorf("expected string for time type, got %T", value)

	case schema.TypeJSON, schema.TypeJSONB:
		// JSON values should be strings representing JSON
		if str, ok := value.(string); ok {
			escaped := strings.ReplaceAll(str, "'", "''")
			return fmt.Sprintf("'%s'::%s", escaped, strings.ToLower(typeSpec.BaseType.String())), nil
		}
		return "", fmt.Errorf("expected string for JSON type, got %T", value)

	case schema.TypeEnum:
		if str, ok := value.(string); ok {
			escaped := strings.ReplaceAll(str, "'", "''")
			return fmt.Sprintf("'%s'", escaped), nil
		}
		return "", fmt.Errorf("expected string for enum type, got %T", value)

	default:
		return "", fmt.Errorf("unsupported default value type: %s", typeSpec.BaseType)
	}
}

// GetEnumTypeName generates a PostgreSQL enum type name from a resource and field name
func (tm *TypeMapper) GetEnumTypeName(resourceName, fieldName string) string {
	return fmt.Sprintf("%s_%s_enum", toSnakeCase(resourceName), toSnakeCase(fieldName))
}

// toSnakeCase converts a string to snake_case
// Sanitizes identifiers to only allow alphanumeric characters and underscores
// Handles acronyms properly (HTTPRequest -> http_request, userID -> user_id)
func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		// Skip invalid characters (only allow alphanumeric and underscore)
		if !isAlphanumeric(r) && r != '_' {
			continue
		}

		// Handle uppercase letters
		if r >= 'A' && r <= 'Z' {
			// Add underscore before uppercase letter if needed
			if i > 0 && len(result) > 0 {
				prev := runes[i-1]

				// Case 1: transition from lowercase to uppercase (userName -> user_name)
				if prev >= 'a' && prev <= 'z' {
					result = append(result, '_')
				} else if prev >= 'A' && prev <= 'Z' {
					// Case 2: acronym followed by lowercase (HTTPRequest -> http_request)
					// Look ahead to see if next char is lowercase
					if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
						result = append(result, '_')
					}
				}
			}

			// Convert to lowercase
			result = append(result, r+('a'-'A'))
		} else {
			// Keep lowercase, digits, and underscores as-is
			result = append(result, r)
		}
	}

	// Ensure identifier starts with letter or underscore, not a digit
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]rune{'_'}, result...)
	}

	return string(result)
}

// isAlphanumeric checks if a rune is alphanumeric
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// QuoteIdentifier wraps a SQL identifier in double quotes and escapes internal quotes
// This prevents SQL injection in table and column names
func QuoteIdentifier(identifier string) string {
	// Escape internal double quotes by doubling them
	escaped := strings.ReplaceAll(identifier, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}
