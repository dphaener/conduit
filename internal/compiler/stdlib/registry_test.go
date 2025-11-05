package stdlib

import (
	"testing"
)

// TestRegistryCompleteness verifies that the registry contains all expected namespaces
func TestRegistryCompleteness(t *testing.T) {
	expectedNamespaces := []string{"String", "Time", "Array", "Hash", "UUID"}

	for _, namespace := range expectedNamespaces {
		if _, exists := StdlibRegistry[namespace]; !exists {
			t.Errorf("Expected namespace %s to be in registry", namespace)
		}
	}

	// Verify no unexpected namespaces
	if len(StdlibRegistry) != len(expectedNamespaces) {
		t.Errorf("Expected %d namespaces, got %d", len(expectedNamespaces), len(StdlibRegistry))
	}
}

// TestFunctionCounts verifies that each namespace has the expected number of functions
func TestFunctionCounts(t *testing.T) {
	expectedCounts := map[string]int{
		"String": 7, // length, slugify, upcase, downcase, trim, contains, replace
		"Time":   4, // now, format, parse, add_days
		"Array":  2, // length, contains
		"Hash":   1, // has_key
		"UUID":   1, // generate
	}

	for namespace, expectedCount := range expectedCounts {
		funcs := GetFunctions(namespace)
		if funcs == nil {
			t.Errorf("Namespace %s not found in registry", namespace)
			continue
		}
		actualCount := len(funcs)
		if actualCount != expectedCount {
			t.Errorf("Expected %d functions in %s namespace, got %d", expectedCount, namespace, actualCount)
		}
	}
}

// TestTotalFunctionCount verifies the total function count
func TestTotalFunctionCount(t *testing.T) {
	expectedTotal := 15 // 7 + 4 + 2 + 1 + 1
	actualTotal := TotalFunctionCount()

	if actualTotal != expectedTotal {
		t.Errorf("Expected %d total functions, got %d", expectedTotal, actualTotal)
	}
}

// TestGetNamespaces verifies that GetNamespaces returns sorted namespaces
func TestGetNamespaces(t *testing.T) {
	namespaces := GetNamespaces()

	// Check count
	if len(namespaces) != 5 {
		t.Errorf("Expected 5 namespaces, got %d", len(namespaces))
	}

	// Verify sorted order
	expectedOrder := []string{"Array", "Hash", "String", "Time", "UUID"}
	for i, expected := range expectedOrder {
		if i >= len(namespaces) {
			t.Errorf("Missing namespace at index %d", i)
			continue
		}
		if namespaces[i] != expected {
			t.Errorf("Expected namespace at index %d to be %s, got %s", i, expected, namespaces[i])
		}
	}
}

// TestGetFunctions verifies that GetFunctions returns correct functions
func TestGetFunctions(t *testing.T) {
	// Test valid namespace
	stringFuncs := GetFunctions("String")
	if stringFuncs == nil {
		t.Fatal("Expected String namespace to exist")
	}
	if len(stringFuncs) != 7 {
		t.Errorf("Expected 7 String functions, got %d", len(stringFuncs))
	}

	// Test invalid namespace
	invalidFuncs := GetFunctions("InvalidNamespace")
	if invalidFuncs != nil {
		t.Error("Expected nil for invalid namespace")
	}
}

// TestFunctionSignatures verifies that all functions have proper signatures
func TestFunctionSignatures(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		signature string
	}{
		// String functions
		{"String", "length", "length(s: string!) -> int!"},
		{"String", "slugify", "slugify(s: string!) -> string!"},
		{"String", "upcase", "upcase(s: string!) -> string!"},
		{"String", "downcase", "downcase(s: string!) -> string!"},
		{"String", "trim", "trim(s: string!) -> string!"},
		{"String", "contains", "contains(s: string!, substr: string!) -> bool!"},
		{"String", "replace", "replace(s: string!, old: string!, new: string!) -> string!"},

		// Time functions
		{"Time", "now", "now() -> timestamp!"},
		{"Time", "format", "format(t: timestamp!, layout: string!) -> string!"},
		{"Time", "parse", "parse(s: string!, layout: string!) -> timestamp?"},
		{"Time", "add_days", "add_days(t: timestamp!, days: int!) -> timestamp!"},

		// Array functions
		{"Array", "length", "length(arr: array!) -> int!"},
		{"Array", "contains", "contains(arr: array!, value: any!) -> bool!"},

		// Hash functions
		{"Hash", "has_key", "has_key(h: hash!, key: any!) -> bool!"},

		// UUID functions
		{"UUID", "generate", "generate() -> uuid!"},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"."+tt.name, func(t *testing.T) {
			funcs := GetFunctions(tt.namespace)
			if funcs == nil {
				t.Fatalf("Namespace %s not found", tt.namespace)
			}

			// Find the function
			var found *FunctionDef
			for i := range funcs {
				if funcs[i].Name == tt.name {
					found = &funcs[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("Function %s not found in namespace %s", tt.name, tt.namespace)
			}

			if found.Signature != tt.signature {
				t.Errorf("Expected signature %s, got %s", tt.signature, found.Signature)
			}
		})
	}
}

// TestFunctionDescriptions verifies that all functions have descriptions
func TestFunctionDescriptions(t *testing.T) {
	for namespace, funcs := range StdlibRegistry {
		for _, fn := range funcs {
			if fn.Description == "" {
				t.Errorf("Function %s.%s is missing a description", namespace, fn.Name)
			}

			// Description should not be too long (keep it concise)
			if len(fn.Description) > 200 {
				t.Errorf("Function %s.%s has a description that is too long (%d chars)", namespace, fn.Name, len(fn.Description))
			}
		}
	}
}

// TestFunctionNamesMatchSignatures verifies that function names in signatures match
func TestFunctionNamesMatchSignatures(t *testing.T) {
	for namespace, funcs := range StdlibRegistry {
		for _, fn := range funcs {
			// Signature should start with the function name followed by '('
			expectedPrefix := fn.Name + "("
			if len(fn.Signature) < len(expectedPrefix) || fn.Signature[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("Function %s.%s signature doesn't start with expected prefix %s: got %s",
					namespace, fn.Name, expectedPrefix, fn.Signature)
			}
		}
	}
}

// TestNullableReturnTypes verifies that nullable return types use '?' suffix
func TestNullableReturnTypes(t *testing.T) {
	// Time.parse is the only MVP function that returns a nullable type
	timeFuncs := GetFunctions("Time")
	if timeFuncs == nil {
		t.Fatal("Time namespace not found")
	}

	var parseFunc *FunctionDef
	for i := range timeFuncs {
		if timeFuncs[i].Name == "parse" {
			parseFunc = &timeFuncs[i]
			break
		}
	}

	if parseFunc == nil {
		t.Fatal("Time.parse function not found")
	}

	// Verify it returns timestamp?
	if parseFunc.Signature != "parse(s: string!, layout: string!) -> timestamp?" {
		t.Errorf("Expected Time.parse to return timestamp?, got: %s", parseFunc.Signature)
	}
}

// TestGetAllFunctions verifies that GetAllFunctions returns the complete registry
func TestGetAllFunctions(t *testing.T) {
	allFuncs := GetAllFunctions()

	if len(allFuncs) != len(StdlibRegistry) {
		t.Errorf("Expected GetAllFunctions to return %d namespaces, got %d", len(StdlibRegistry), len(allFuncs))
	}

	// Verify it's the same reference
	for namespace := range StdlibRegistry {
		if _, exists := allFuncs[namespace]; !exists {
			t.Errorf("Expected namespace %s in GetAllFunctions result", namespace)
		}
	}
}

// TestNoOptionalParameters verifies that MVP functions have no optional parameters
// (Optional parameters are a future feature, not in MVP)
func TestNoOptionalParameters(t *testing.T) {
	// All MVP functions have required parameters only
	// This is enforced by the signature format - no "?" on parameter types in MVP

	optionalParamPattern := []string{
		"param?:",   // param?: type pattern
		": string?", // : string? pattern (but this is actually for nullable types which is allowed)
	}

	for namespace, funcs := range StdlibRegistry {
		for _, fn := range funcs {
			for _, pattern := range optionalParamPattern[:1] { // Only check param?: pattern
				if contains(fn.Signature, pattern) {
					t.Errorf("Function %s.%s appears to have optional parameters (MVP doesn't support this): %s",
						namespace, fn.Name, fn.Signature)
				}
			}
		}
	}
}

// contains checks if a string contains a substring (simple helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

// findSubstring is a simple substring search
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
