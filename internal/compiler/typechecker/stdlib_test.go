package typechecker

import "testing"

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
