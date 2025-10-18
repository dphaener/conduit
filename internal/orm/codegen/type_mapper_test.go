package codegen

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestTypeMapper_MapPrimitiveTypes(t *testing.T) {
	mapper := NewTypeMapper()

	tests := []struct {
		name     string
		baseType schema.PrimitiveType
		length   *int
		precision *int
		scale    *int
		expected string
	}{
		{"string default", schema.TypeString, nil, nil, nil, "VARCHAR(255)"},
		{"string with length", schema.TypeString, intPtr(50), nil, nil, "VARCHAR(50)"},
		{"text", schema.TypeText, nil, nil, nil, "TEXT"},
		{"markdown", schema.TypeMarkdown, nil, nil, nil, "TEXT"},
		{"int", schema.TypeInt, nil, nil, nil, "INTEGER"},
		{"bigint", schema.TypeBigInt, nil, nil, nil, "BIGINT"},
		{"float", schema.TypeFloat, nil, nil, nil, "DOUBLE PRECISION"},
		{"decimal default", schema.TypeDecimal, nil, nil, nil, "NUMERIC"},
		{"decimal with precision", schema.TypeDecimal, nil, intPtr(10), intPtr(2), "NUMERIC(10,2)"},
		{"bool", schema.TypeBool, nil, nil, nil, "BOOLEAN"},
		{"timestamp", schema.TypeTimestamp, nil, nil, nil, "TIMESTAMP WITH TIME ZONE"},
		{"date", schema.TypeDate, nil, nil, nil, "DATE"},
		{"time", schema.TypeTime, nil, nil, nil, "TIME"},
		{"uuid", schema.TypeUUID, nil, nil, nil, "UUID"},
		{"ulid", schema.TypeULID, nil, nil, nil, "CHAR(26)"},
		{"email", schema.TypeEmail, nil, nil, nil, "VARCHAR(255)"},
		{"url", schema.TypeURL, nil, nil, nil, "VARCHAR(255)"},
		{"phone", schema.TypePhone, nil, nil, nil, "VARCHAR(255)"},
		{"json", schema.TypeJSON, nil, nil, nil, "JSON"},
		{"jsonb", schema.TypeJSONB, nil, nil, nil, "JSONB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeSpec := &schema.TypeSpec{
				BaseType:  tt.baseType,
				Nullable:  false,
				Length:    tt.length,
				Precision: tt.precision,
				Scale:     tt.scale,
			}

			result, err := mapper.MapType(typeSpec)
			if err != nil {
				t.Fatalf("MapType() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("MapType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeMapper_MapArrayTypes(t *testing.T) {
	mapper := NewTypeMapper()

	tests := []struct {
		name     string
		element  schema.PrimitiveType
		expected string
	}{
		{"int array", schema.TypeInt, "INTEGER[]"},
		{"string array", schema.TypeString, "VARCHAR(255)[]"},
		{"uuid array", schema.TypeUUID, "UUID[]"},
		{"text array", schema.TypeText, "TEXT[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeSpec := &schema.TypeSpec{
				ArrayElement: &schema.TypeSpec{
					BaseType: tt.element,
					Nullable: false,
				},
			}

			result, err := mapper.MapType(typeSpec)
			if err != nil {
				t.Fatalf("MapType() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("MapType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeMapper_MapHashTypes(t *testing.T) {
	mapper := NewTypeMapper()

	typeSpec := &schema.TypeSpec{
		HashKey: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
		HashValue: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
		},
	}

	result, err := mapper.MapType(typeSpec)
	if err != nil {
		t.Fatalf("MapType() error = %v", err)
	}

	if result != "JSONB" {
		t.Errorf("MapType() = %v, want JSONB", result)
	}
}

func TestTypeMapper_MapNullability(t *testing.T) {
	mapper := NewTypeMapper()

	tests := []struct {
		name     string
		nullable bool
		expected string
	}{
		{"non-nullable", false, "NOT NULL"},
		{"nullable", true, "NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeSpec := &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: tt.nullable,
			}

			result := mapper.MapNullability(typeSpec)
			if result != tt.expected {
				t.Errorf("MapNullability() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeMapper_MapDefault(t *testing.T) {
	mapper := NewTypeMapper()

	tests := []struct {
		name     string
		baseType schema.PrimitiveType
		value    interface{}
		expected string
		wantErr  bool
	}{
		{"string", schema.TypeString, "hello", "'hello'", false},
		{"string with quotes", schema.TypeString, "it's", "'it''s'", false},
		{"int", schema.TypeInt, 42, "42", false},
		{"bigint", schema.TypeBigInt, int64(1234567890), "1234567890", false},
		{"float", schema.TypeFloat, 3.14, "3.140000", false},
		{"float from int", schema.TypeFloat, 42, "42", false},
		{"decimal from int", schema.TypeDecimal, 100, "100", false},
		{"bool true", schema.TypeBool, true, "TRUE", false},
		{"bool false", schema.TypeBool, false, "FALSE", false},
		{"uuid", schema.TypeUUID, "550e8400-e29b-41d4-a716-446655440000", "'550e8400-e29b-41d4-a716-446655440000'::uuid", false},
		{"timestamp now", schema.TypeTimestamp, "now()", "CURRENT_TIMESTAMP", false},
		{"timestamp value", schema.TypeTimestamp, "2023-01-01 00:00:00", "'2023-01-01 00:00:00'::timestamp", false},
		{"date today", schema.TypeDate, "today()", "CURRENT_DATE", false},
		{"time now", schema.TypeTime, "now()", "CURRENT_TIME", false},
		{"json", schema.TypeJSON, `{"key":"value"}`, `'{"key":"value"}'::json`, false},
		{"enum", schema.TypeEnum, "active", "'active'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeSpec := &schema.TypeSpec{
				BaseType: tt.baseType,
				Nullable: false,
				Default:  tt.value,
			}

			result, err := mapper.MapDefault(typeSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapDefault() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("MapDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeMapper_MapDefaultNoValue(t *testing.T) {
	mapper := NewTypeMapper()

	typeSpec := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
		Default:  nil,
	}

	result, err := mapper.MapDefault(typeSpec)
	if err != nil {
		t.Fatalf("MapDefault() error = %v", err)
	}

	if result != "" {
		t.Errorf("MapDefault() = %v, want empty string", result)
	}
}

func TestTypeMapper_GetEnumTypeName(t *testing.T) {
	mapper := NewTypeMapper()

	tests := []struct {
		name         string
		resourceName string
		fieldName    string
		expected     string
	}{
		{"simple", "Post", "status", "post_status_enum"},
		{"camel case", "BlogPost", "publishStatus", "blog_post_publish_status_enum"},
		{"acronym", "HTTPRequest", "method", "http_request_method_enum"}, // Fixed: acronym handling
		{"userID field", "User", "userID", "user_user_id_enum"},          // Fixed: acronym handling
		{"XML parser", "XMLParser", "format", "xml_parser_format_enum"},  // Fixed: acronym handling
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.GetEnumTypeName(tt.resourceName, tt.fieldName)
			if result != tt.expected {
				t.Errorf("GetEnumTypeName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeMapper_MapEnumType(t *testing.T) {
	mapper := NewTypeMapper()

	typeSpec := &schema.TypeSpec{
		EnumValues: []string{"draft", "published", "archived"},
	}

	result, err := mapper.MapType(typeSpec)
	if err != nil {
		t.Fatalf("MapType() error = %v", err)
	}

	if result != "ENUM" {
		t.Errorf("MapType() = %v, want ENUM", result)
	}
}

func TestTypeMapper_MapStructType(t *testing.T) {
	mapper := NewTypeMapper()

	typeSpec := &schema.TypeSpec{
		StructFields: map[string]*schema.TypeSpec{
			"lat": {BaseType: schema.TypeFloat, Nullable: false},
			"lng": {BaseType: schema.TypeFloat, Nullable: false},
		},
	}

	result, err := mapper.MapType(typeSpec)
	if err != nil {
		t.Fatalf("MapType() error = %v", err)
	}

	if result != "JSONB" {
		t.Errorf("MapType() = %v, want JSONB", result)
	}
}

func TestTypeMapper_ErrorCases(t *testing.T) {
	mapper := NewTypeMapper()

	t.Run("nil type spec", func(t *testing.T) {
		_, err := mapper.MapType(nil)
		if err == nil {
			t.Error("MapType() expected error for nil type spec")
		}
	})

	t.Run("wrong default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Default:  "not an int",
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong default type")
		}
	})

	t.Run("wrong float default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeFloat,
			Default:  "not a float",
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong float default type")
		}
	})

	t.Run("wrong bool default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeBool,
			Default:  "not a bool",
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong bool default type")
		}
	})

	t.Run("wrong uuid default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeUUID,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong uuid default type")
		}
	})

	t.Run("wrong timestamp default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeTimestamp,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong timestamp default type")
		}
	})

	t.Run("wrong date default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeDate,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong date default type")
		}
	})

	t.Run("wrong time default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeTime,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong time default type")
		}
	})

	t.Run("wrong json default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeJSON,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong json default type")
		}
	})

	t.Run("wrong enum default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeEnum,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong enum default type")
		}
	})

	t.Run("wrong string default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeString,
			Default:  123,
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong string default type")
		}
	})

	t.Run("wrong bigint default type", func(t *testing.T) {
		typeSpec := &schema.TypeSpec{
			BaseType: schema.TypeBigInt,
			Default:  "not an int",
		}

		_, err := mapper.MapDefault(typeSpec)
		if err == nil {
			t.Error("MapDefault() expected error for wrong bigint default type")
		}
	})
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases
		{"Post", "post"},
		{"BlogPost", "blog_post"},
		{"firstName", "first_name"},

		// Acronym handling (HIGH priority fix)
		{"HTTPRequest", "http_request"},
		{"userID", "user_id"},
		{"XMLParser", "xml_parser"},
		{"API", "api"},
		{"myHTTPSServer", "my_https_server"},
		{"parseHTMLContent", "parse_html_content"},
		{"getURLPath", "get_url_path"},
		{"HTTPSConnection", "https_connection"},

		// SQL injection prevention (HIGH priority security fix)
		// Note: Sanitization removes invalid characters entirely - this prevents SQL injection
		{"field'; DROP TABLE users; --", "fielddroptableusers"},
		{"field\"; DROP TABLE users; --", "fielddroptableusers"},
		{"field with spaces", "fieldwithspaces"},      // Spaces are removed
		{"field-with-dashes", "fieldwithdashes"},      // Dashes are removed
		{"field.with.dots", "fieldwithdots"},          // Dots are removed
		{"field!@#$%", "field"},                       // Special chars removed
		{"123field", "_123field"},                     // Ensure starts with underscore if begins with digit
		{"_field", "_field"},                          // Preserve leading underscore
		{"__field", "__field"},                        // Preserve multiple underscores

		// Edge cases
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"AB", "ab"},
		{"ABC", "abc"},
		{"ABCDef", "abc_def"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple identifier", "users", `"users"`},
		{"identifier with underscore", "user_name", `"user_name"`},
		{"identifier with double quote", `user"name`, `"user""name"`},
		{"identifier with multiple quotes", `user"table"name`, `"user""table""name"`},
		{"empty string", "", `""`},
		{"single quote (not escaped by QuoteIdentifier)", "user'name", `"user'name"`},
		{"SQL injection attempt", `users"; DROP TABLE users; --`, `"users""; DROP TABLE users; --"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuoteIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		{"lowercase a", 'a', true},
		{"lowercase z", 'z', true},
		{"uppercase A", 'A', true},
		{"uppercase Z", 'Z', true},
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"space", ' ', false},
		{"underscore", '_', false},
		{"dash", '-', false},
		{"quote", '\'', false},
		{"double quote", '"', false},
		{"semicolon", ';', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlphanumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}
