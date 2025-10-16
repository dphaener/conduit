package typechecker

import (
	"strings"
	"testing"
)

// getBaseTypeName extracts the base type name without nullability suffix
func getBaseTypeName(t Type) string {
	switch v := t.(type) {
	case *PrimitiveType:
		return v.Name
	case *ArrayType:
		return "array"
	case *HashType:
		return "hash"
	case *StructType:
		return "struct"
	case *EnumType:
		return "enum"
	case *ResourceType:
		return v.Name
	default:
		// Fallback to string representation without suffix
		s := t.String()
		if strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") {
			return s[:len(s)-1]
		}
		return s
	}
}

// TestLookupStdlibFunction tests standard library function lookup
func TestLookupStdlibFunction(t *testing.T) {
	tests := []struct {
		namespace  string
		name       string
		shouldFind bool
	}{
		{"String", "slugify", true},
		{"String", "upcase", true},
		{"String", "nonexistent", false},
		{"", "slugify", false}, // No namespace
		{"Unknown", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"."+tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.name)
			if ok != tt.shouldFind {
				t.Errorf("Expected to find: %v, got: %v", tt.shouldFind, ok)
			}
			if tt.shouldFind && fn == nil {
				t.Error("Expected non-nil function")
			}
		})
	}
}

// TestFunctionFullName tests the FullName method
func TestFunctionFullName(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		expected  string
	}{
		{"String", "slugify", "String.slugify"},
		{"", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			fn := &Function{
				Name:      tt.name,
				Namespace: tt.namespace,
			}
			if fn.FullName() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, fn.FullName())
			}
		})
	}
}

// TestStringNamespace tests all String namespace functions
func TestStringNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"slugify", "slugify", 1, "string", false},
		{"capitalize", "capitalize", 1, "string", false},
		{"upcase", "upcase", 1, "string", false},
		{"downcase", "downcase", 1, "string", false},
		{"trim", "trim", 1, "string", false},
		{"truncate", "truncate", 2, "string", false},
		{"split", "split", 2, "array", false},
		{"join", "join", 2, "string", false},
		{"replace", "replace", 3, "string", false},
		{"starts_with?", "starts_with?", 2, "bool", false},
		{"ends_with?", "ends_with?", 2, "bool", false},
		{"includes?", "includes?", 2, "bool", false},
		{"length", "length", 1, "int", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("String", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find String.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestTextNamespace tests all Text namespace functions
func TestTextNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"calculate_reading_time", "calculate_reading_time", 2, "int", false},
		{"word_count", "word_count", 1, "int", false},
		{"character_count", "character_count", 1, "int", false},
		{"excerpt", "excerpt", 2, "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Text", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Text.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestNumberNamespace tests all Number namespace functions
func TestNumberNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"format", "format", 2, "string", false},
		{"round", "round", 2, "float", false},
		{"abs", "abs", 1, "float", false},
		{"ceil", "ceil", 1, "int", false},
		{"floor", "floor", 1, "int", false},
		{"min", "min", 2, "float", false},
		{"max", "max", 2, "float", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Number", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Number.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestArrayNamespace tests all Array namespace functions
func TestArrayNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"first", "first", 1, "any", true}, // Returns nullable
		{"last", "last", 1, "any", true},   // Returns nullable
		{"length", "length", 1, "int", false},
		{"empty?", "empty?", 1, "bool", false},
		{"includes?", "includes?", 2, "bool", false},
		{"unique", "unique", 1, "array", false},
		{"sort", "sort", 1, "array", false},
		{"reverse", "reverse", 1, "array", false},
		{"push", "push", 2, "array", false},
		{"concat", "concat", 2, "array", false},
		{"map", "map", 2, "array", false},
		{"filter", "filter", 2, "array", false},
		{"reduce", "reduce", 3, "any", false},
		{"count", "count", 1, "int", false},
		{"contains", "contains", 2, "bool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Array", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Array.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestHashNamespace tests all Hash namespace functions
func TestHashNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		optionalParams int
		expectedReturn string
		nullable       bool
	}{
		{"keys", "keys", 1, 0, "array", false},
		{"values", "values", 1, 0, "array", false},
		{"merge", "merge", 2, 0, "hash", false},
		{"has_key?", "has_key?", 2, 0, "bool", false},
		{"get", "get", 3, 1, "any", true}, // Has optional param, returns nullable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Hash", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Hash.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			// Check optional parameters
			optionalCount := 0
			for _, p := range fn.Parameters {
				if p.Optional {
					optionalCount++
				}
			}
			if optionalCount != tt.optionalParams {
				t.Errorf("Expected %d optional parameters, got %d", tt.optionalParams, optionalCount)
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestTimeNamespace tests all Time namespace functions
func TestTimeNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		optionalParams int
		expectedReturn string
		nullable       bool
	}{
		{"now", "now", 0, 0, "timestamp", false},     // No parameters
		{"today", "today", 0, 0, "date", false},      // No parameters
		{"parse", "parse", 2, 1, "timestamp", false}, // Has optional format param
		{"format", "format", 2, 0, "string", false},
		{"year", "year", 1, 0, "int", false},
		{"month", "month", 1, 0, "int", false},
		{"day", "day", 1, 0, "int", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Time", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Time.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			// Check optional parameters
			optionalCount := 0
			for _, p := range fn.Parameters {
				if p.Optional {
					optionalCount++
				}
			}
			if optionalCount != tt.optionalParams {
				t.Errorf("Expected %d optional parameters, got %d", tt.optionalParams, optionalCount)
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestUUIDNamespace tests all UUID namespace functions
func TestUUIDNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"generate", "generate", 0, "uuid", false}, // No parameters
		{"validate", "validate", 1, "bool", false},
		{"parse", "parse", 1, "uuid", true}, // Returns nullable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("UUID", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find UUID.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestRandomNamespace tests all Random namespace functions
func TestRandomNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"int", "int", 2, "int", false},
		{"float", "float", 2, "float", false},
		{"uuid", "uuid", 0, "uuid", false}, // No parameters
		{"hex", "hex", 1, "string", false},
		{"alphanumeric", "alphanumeric", 1, "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Random", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Random.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestCryptoNamespace tests all Crypto namespace functions
func TestCryptoNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"hash", "hash", 2, "string", false},
		{"compare", "compare", 2, "bool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Crypto", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Crypto.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestHTMLNamespace tests all HTML namespace functions
func TestHTMLNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"strip_tags", "strip_tags", 1, "string", false},
		{"escape", "escape", 1, "string", false},
		{"unescape", "unescape", 1, "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("HTML", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find HTML.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestJSONNamespace tests all JSON namespace functions
func TestJSONNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		optionalParams int
		expectedReturn string
		nullable       bool
	}{
		{"parse", "parse", 1, 0, "json", false},
		{"stringify", "stringify", 2, 1, "string", false}, // Has optional pretty param
		{"validate", "validate", 1, 0, "bool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("JSON", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find JSON.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			// Check optional parameters
			optionalCount := 0
			for _, p := range fn.Parameters {
				if p.Optional {
					optionalCount++
				}
			}
			if optionalCount != tt.optionalParams {
				t.Errorf("Expected %d optional parameters, got %d", tt.optionalParams, optionalCount)
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestRegexNamespace tests all Regex namespace functions
func TestRegexNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"match", "match", 2, "array", true}, // Returns nullable array
		{"replace", "replace", 3, "string", false},
		{"test", "test", 2, "bool", false},
		{"split", "split", 2, "array", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Regex", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Regex.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestLoggerNamespace tests all Logger namespace functions
func TestLoggerNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"warn", "warn", 1, "void", false},
		{"debug", "debug", 1, "void", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Logger", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Logger.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestContextNamespace tests all Context namespace functions
func TestContextNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"current_user", "current_user", 0, "User", true},      // No params, returns nullable
		{"authenticated?", "authenticated?", 0, "bool", false}, // No params
		{"headers", "headers", 0, "hash", true},                // No params, returns nullable
		{"request_id", "request_id", 0, "string", false},       // No params
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Context", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Context.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestEnvNamespace tests all Env namespace functions
func TestEnvNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		optionalParams int
		expectedReturn string
		nullable       bool
	}{
		{"get", "get", 2, 1, "string", true}, // Has optional default param, returns nullable
		{"has?", "has?", 1, 0, "bool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := LookupStdlibFunction("Env", tt.funcName)
			if !ok {
				t.Fatalf("Expected to find Env.%s", tt.funcName)
			}
			if fn == nil {
				t.Fatal("Expected non-nil function")
			}
			if len(fn.Parameters) != tt.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tt.expectedParams, len(fn.Parameters))
			}
			// Check optional parameters
			optionalCount := 0
			for _, p := range fn.Parameters {
				if p.Optional {
					optionalCount++
				}
			}
			if optionalCount != tt.optionalParams {
				t.Errorf("Expected %d optional parameters, got %d", tt.optionalParams, optionalCount)
			}
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestFunctionsWithQuestionMarkSuffix tests functions with ? suffix
func TestFunctionsWithQuestionMarkSuffix(t *testing.T) {
	questionMarkFunctions := []struct {
		namespace string
		funcName  string
	}{
		{"String", "starts_with?"},
		{"String", "ends_with?"},
		{"String", "includes?"},
		{"Array", "empty?"},
		{"Array", "includes?"},
		{"Hash", "has_key?"},
		{"Context", "authenticated?"},
		{"Env", "has?"},
	}

	for _, tt := range questionMarkFunctions {
		t.Run(tt.namespace+"."+tt.funcName, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}
			if getBaseTypeName(fn.ReturnType) != "bool" {
				t.Errorf("Expected ? suffix function to return bool, got %s", getBaseTypeName(fn.ReturnType))
			}
		})
	}
}

// TestFunctionsWithNoParameters tests functions with no parameters
func TestFunctionsWithNoParameters(t *testing.T) {
	noParamFunctions := []struct {
		namespace string
		funcName  string
	}{
		{"Time", "now"},
		{"Time", "today"},
		{"UUID", "generate"},
		{"Random", "uuid"},
		{"Context", "current_user"},
		{"Context", "authenticated?"},
		{"Context", "headers"},
		{"Context", "request_id"},
	}

	for _, tt := range noParamFunctions {
		t.Run(tt.namespace+"."+tt.funcName, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}
			if len(fn.Parameters) != 0 {
				t.Errorf("Expected no parameters, got %d", len(fn.Parameters))
			}
		})
	}
}

// TestNullableReturnTypes tests functions that return nullable types
func TestNullableReturnTypes(t *testing.T) {
	nullableReturnFunctions := []struct {
		namespace string
		funcName  string
	}{
		{"Array", "first"},
		{"Array", "last"},
		{"Hash", "get"},
		{"UUID", "parse"},
		{"Context", "current_user"},
		{"Context", "headers"},
		{"Env", "get"},
		{"Regex", "match"},
	}

	for _, tt := range nullableReturnFunctions {
		t.Run(tt.namespace+"."+tt.funcName, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}
			if !fn.ReturnType.IsNullable() {
				t.Errorf("Expected %s.%s to return nullable type", tt.namespace, tt.funcName)
			}
		})
	}
}

// TestOptionalParameters tests functions with optional parameters
func TestOptionalParameters(t *testing.T) {
	optionalParamFunctions := []struct {
		namespace      string
		funcName       string
		optionalParams []string
	}{
		{"Time", "parse", []string{"format"}},
		{"Hash", "get", []string{"default"}},
		{"Env", "get", []string{"default"}},
		{"JSON", "stringify", []string{"pretty"}},
	}

	for _, tt := range optionalParamFunctions {
		t.Run(tt.namespace+"."+tt.funcName, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}

			optionalCount := 0
			for _, p := range fn.Parameters {
				if p.Optional {
					optionalCount++
					found := false
					for _, expectedName := range tt.optionalParams {
						if p.Name == expectedName {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Unexpected optional parameter: %s", p.Name)
					}
				}
			}

			if optionalCount != len(tt.optionalParams) {
				t.Errorf("Expected %d optional parameters, got %d", len(tt.optionalParams), optionalCount)
			}
		})
	}
}

// TestParameterNullability tests parameter nullability markers
func TestParameterNullability(t *testing.T) {
	tests := []struct {
		namespace    string
		funcName     string
		paramIndex   int
		expectedNull bool
	}{
		{"String", "slugify", 0, false}, // Required string
		{"Time", "parse", 1, true},      // Optional format string?
		{"Hash", "get", 2, true},        // Optional default any?
		{"Env", "get", 1, true},         // Optional default string?
		{"JSON", "stringify", 1, true},  // Optional pretty bool?
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"."+tt.funcName, func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}
			if tt.paramIndex >= len(fn.Parameters) {
				t.Fatalf("Parameter index %d out of range for %s.%s", tt.paramIndex, tt.namespace, tt.funcName)
			}
			param := fn.Parameters[tt.paramIndex]
			if param.Type.IsNullable() != tt.expectedNull {
				t.Errorf("Expected parameter %s to be nullable=%v, got %v", param.Name, tt.expectedNull, param.Type.IsNullable())
			}
		})
	}
}

// TestAllNamespacesExist tests that all expected namespaces are registered
func TestAllNamespacesExist(t *testing.T) {
	expectedNamespaces := []string{
		"String", "Text", "Number", "Array", "Hash",
		"Time", "UUID", "Random", "Crypto", "HTML",
		"JSON", "Regex", "Logger", "Context", "Env",
	}

	for _, namespace := range expectedNamespaces {
		t.Run(namespace, func(t *testing.T) {
			if _, ok := StdlibFunctions[namespace]; !ok {
				t.Errorf("Expected namespace %s to be registered", namespace)
			}
		})
	}
}

// TestTotalFunctionCount tests that all expected functions are registered
func TestTotalFunctionCount(t *testing.T) {
	expectedCounts := map[string]int{
		"String":  13,
		"Text":    4,
		"Number":  7,
		"Array":   15,
		"Hash":    5,
		"Time":    7,
		"UUID":    3,
		"Random":  5,
		"Crypto":  2,
		"HTML":    3,
		"JSON":    3,
		"Regex":   4,
		"Logger":  2,
		"Context": 4,
		"Env":     2,
	}

	for namespace, expectedCount := range expectedCounts {
		t.Run(namespace, func(t *testing.T) {
			funcs, ok := StdlibFunctions[namespace]
			if !ok {
				t.Fatalf("Namespace %s not found", namespace)
			}
			actualCount := len(funcs)
			if actualCount != expectedCount {
				t.Errorf("Expected %d functions in %s namespace, got %d", expectedCount, namespace, actualCount)
			}
		})
	}

	// Verify total count
	totalCount := 0
	for _, funcs := range StdlibFunctions {
		totalCount += len(funcs)
	}
	expectedTotal := 0
	for _, count := range expectedCounts {
		expectedTotal += count
	}
	if totalCount != expectedTotal {
		t.Errorf("Expected %d total functions, got %d", expectedTotal, totalCount)
	}
}
