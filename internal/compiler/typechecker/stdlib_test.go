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
		{"Array", "length", true},
		{"Array", "contains", true},
		{"Time", "now", true},
		{"Time", "format", true},
		{"Hash", "has_key", true},
		{"UUID", "generate", true},
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

// TestMVPStringNamespace tests MVP String namespace functions (7 functions)
func TestMVPStringNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"length", "length", 1, "int", false},
		{"slugify", "slugify", 1, "string", false},
		{"upcase", "upcase", 1, "string", false},
		{"downcase", "downcase", 1, "string", false},
		{"trim", "trim", 1, "string", false},
		{"contains", "contains", 2, "bool", false},
		{"replace", "replace", 3, "string", false},
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

// TestMVPTimeNamespace tests MVP Time namespace functions (4 functions)
func TestMVPTimeNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"now", "now", 0, "timestamp", false},       // No parameters
		{"format", "format", 2, "string", false},    // t, layout
		{"parse", "parse", 2, "timestamp", true},    // s, layout -> nullable!
		{"add_days", "add_days", 2, "timestamp", false}, // t, days
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
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestMVPArrayNamespace tests MVP Array namespace functions (2 functions)
func TestMVPArrayNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"length", "length", 1, "int", false},
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

// TestMVPHashNamespace tests MVP Hash namespace function (1 function)
func TestMVPHashNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"has_key", "has_key", 2, "bool", false},
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
			if getBaseTypeName(fn.ReturnType) != tt.expectedReturn {
				t.Errorf("Expected return type %s, got %s", tt.expectedReturn, getBaseTypeName(fn.ReturnType))
			}
			if fn.ReturnType.IsNullable() != tt.nullable {
				t.Errorf("Expected nullable=%v, got %v", tt.nullable, fn.ReturnType.IsNullable())
			}
		})
	}
}

// TestMVPUUIDNamespace tests MVP UUID namespace function (1 function)
func TestMVPUUIDNamespace(t *testing.T) {
	tests := []struct {
		name           string
		funcName       string
		expectedParams int
		expectedReturn string
		nullable       bool
	}{
		{"generate", "generate", 0, "uuid", false}, // No parameters
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

// TestMVPNamespacesExist tests that all MVP namespaces are registered
func TestMVPNamespacesExist(t *testing.T) {
	expectedNamespaces := []string{
		"String", "Time", "Array", "Hash", "UUID",
	}

	for _, namespace := range expectedNamespaces {
		t.Run(namespace, func(t *testing.T) {
			if _, ok := StdlibFunctions[namespace]; !ok {
				t.Errorf("Expected namespace %s to be registered", namespace)
			}
		})
	}
}

// TestMVPFunctionCount tests that exactly 15 MVP functions are registered
func TestMVPFunctionCount(t *testing.T) {
	expectedCounts := map[string]int{
		"String": 7, // length, slugify, upcase, downcase, trim, contains, replace
		"Time":   4, // now, format, parse, add_days
		"Array":  2, // length, contains
		"Hash":   1, // has_key
		"UUID":   1, // generate
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

	// Verify total count is exactly 15
	totalCount := 0
	for _, funcs := range StdlibFunctions {
		totalCount += len(funcs)
	}
	expectedTotal := 15
	if totalCount != expectedTotal {
		t.Errorf("Expected exactly %d MVP functions, got %d", expectedTotal, totalCount)
	}
}

// TestMVPFunctionsWithNoParameters tests MVP functions with no parameters
func TestMVPFunctionsWithNoParameters(t *testing.T) {
	noParamFunctions := []struct {
		namespace string
		funcName  string
	}{
		{"Time", "now"},
		{"UUID", "generate"},
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

// TestMVPNullableReturnTypes tests MVP functions that return nullable types
func TestMVPNullableReturnTypes(t *testing.T) {
	nullableReturnFunctions := []struct {
		namespace string
		funcName  string
	}{
		{"Time", "parse"}, // Only nullable return in MVP
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

// TestMVPParameterTypes tests that all MVP function parameters have correct types
func TestMVPParameterTypes(t *testing.T) {
	tests := []struct {
		namespace  string
		funcName   string
		paramIndex int
		paramName  string
		typeName   string
		nullable   bool
	}{
		{"String", "length", 0, "s", "string", false},
		{"String", "slugify", 0, "s", "string", false},
		{"String", "contains", 0, "s", "string", false},
		{"String", "contains", 1, "substr", "string", false},
		{"Time", "format", 0, "t", "timestamp", false},
		{"Time", "format", 1, "layout", "string", false},
		{"Time", "parse", 0, "s", "string", false},
		{"Time", "parse", 1, "layout", "string", false},
		{"Time", "add_days", 0, "t", "timestamp", false},
		{"Time", "add_days", 1, "days", "int", false},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"."+tt.funcName+"["+tt.paramName+"]", func(t *testing.T) {
			fn, ok := LookupStdlibFunction(tt.namespace, tt.funcName)
			if !ok {
				t.Fatalf("Expected to find %s.%s", tt.namespace, tt.funcName)
			}
			if tt.paramIndex >= len(fn.Parameters) {
				t.Fatalf("Parameter index %d out of range for %s.%s", tt.paramIndex, tt.namespace, tt.funcName)
			}
			param := fn.Parameters[tt.paramIndex]
			if param.Name != tt.paramName {
				t.Errorf("Expected parameter name %s, got %s", tt.paramName, param.Name)
			}
			if getBaseTypeName(param.Type) != tt.typeName {
				t.Errorf("Expected parameter type %s, got %s", tt.typeName, getBaseTypeName(param.Type))
			}
			if param.Type.IsNullable() != tt.nullable {
				t.Errorf("Expected parameter nullable=%v, got %v", tt.nullable, param.Type.IsNullable())
			}
		})
	}
}

// TestMVPNoOptionalParameters tests that MVP functions have no optional parameters
// (Optional parameters are a future feature, not in MVP)
func TestMVPNoOptionalParameters(t *testing.T) {
	for namespace, funcs := range StdlibFunctions {
		for funcName, fn := range funcs {
			t.Run(namespace+"."+funcName, func(t *testing.T) {
				for _, param := range fn.Parameters {
					if param.Optional {
						t.Errorf("MVP function %s.%s should not have optional parameters, but parameter %s is optional",
							namespace, funcName, param.Name)
					}
				}
			})
		}
	}
}
