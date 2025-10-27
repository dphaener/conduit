package llm

import (
	"testing"
)

// TestValidator_ExactMatch tests exact matching validation.
func TestValidator_ExactMatch(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		response   string
		expected   string
		wantPassed bool
	}{
		{
			name:       "exact match",
			response:   "@on create: [auth]",
			expected:   "@on create: [auth]",
			wantPassed: true,
		},
		{
			name:       "match with extra whitespace",
			response:   "@on  create :  [ auth ]",
			expected:   "@on create: [auth]",
			wantPassed: true,
		},
		{
			name:       "different middleware",
			response:   "@on create: [cache]",
			expected:   "@on create: [auth]",
			wantPassed: false,
		},
		{
			name:       "different operation",
			response:   "@on update: [auth]",
			expected:   "@on create: [auth]",
			wantPassed: false,
		},
		{
			name:       "no declaration in response",
			response:   "Here is some text without a declaration",
			expected:   "@on create: [auth]",
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ExactMatch(tt.response, tt.expected)

			if result.Passed != tt.wantPassed {
				t.Errorf("Expected passed=%v, got %v (message: %s)",
					tt.wantPassed, result.Passed, result.Message)
			}

			if result.Passed {
				if result.MatchType != "exact" {
					t.Errorf("Expected match type 'exact', got '%s'", result.MatchType)
				}
				if result.Confidence != 1.0 {
					t.Errorf("Expected confidence 1.0, got %.2f", result.Confidence)
				}
			}
		})
	}
}

// TestValidator_SemanticMatch tests semantic matching validation.
func TestValidator_SemanticMatch(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		response   string
		expected   string
		wantPassed bool
	}{
		{
			name:       "exact match",
			response:   "@on create: [auth]",
			expected:   "@on create: [auth]",
			wantPassed: true,
		},
		{
			name:       "multiple middleware same order",
			response:   "@on create: [auth, rate_limit(10/hour)]",
			expected:   "@on create: [auth, rate_limit(10/hour)]",
			wantPassed: true,
		},
		{
			name:       "different middleware",
			response:   "@on create: [cache]",
			expected:   "@on create: [auth]",
			wantPassed: false,
		},
		{
			name:       "wrong order",
			response:   "@on create: [rate_limit(10/hour), auth]",
			expected:   "@on create: [auth, rate_limit(10/hour)]",
			wantPassed: false,
		},
		{
			name:       "different operation",
			response:   "@on update: [auth]",
			expected:   "@on create: [auth]",
			wantPassed: false,
		},
		{
			name:       "missing middleware",
			response:   "@on create: [auth]",
			expected:   "@on create: [auth, cache]",
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SemanticMatch(tt.response, tt.expected)

			if result.Passed != tt.wantPassed {
				t.Errorf("Expected passed=%v, got %v (message: %s)",
					tt.wantPassed, result.Passed, result.Message)
				if len(result.Differences) > 0 {
					t.Logf("Differences: %v", result.Differences)
				}
			}

			if result.Passed {
				if result.MatchType != "semantic" {
					t.Errorf("Expected match type 'semantic', got '%s'", result.MatchType)
				}
			}
		})
	}
}

// TestValidator_Validate tests the main validate method with different modes.
func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	response := "@on create: [auth]"
	expected := "@on create: [auth]"

	// Test exact mode
	result := validator.Validate(response, expected, "exact")
	if !result.Passed {
		t.Errorf("Exact validation should pass")
	}

	// Test semantic mode
	result = validator.Validate(response, expected, "semantic")
	if !result.Passed {
		t.Errorf("Semantic validation should pass")
	}

	// Test unknown mode
	result = validator.Validate(response, expected, "unknown")
	if result.Passed {
		t.Errorf("Unknown mode should fail")
	}
}

// TestValidator_MiddlewareEquals tests middleware equality checking.
func TestValidator_MiddlewareEquals(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name string
		m1   string
		m2   string
		want bool
	}{
		{
			name: "exact match",
			m1:   "auth",
			m2:   "auth",
			want: true,
		},
		{
			name: "different names",
			m1:   "auth",
			m2:   "cache",
			want: false,
		},
		{
			name: "same with params",
			m1:   "cache(300)",
			m2:   "cache(300)",
			want: true,
		},
		{
			name: "different params",
			m1:   "cache(300)",
			m2:   "cache(600)",
			want: false,
		},
		{
			name: "with and without params",
			m1:   "auth",
			m2:   "auth()",
			want: true, // Empty params () treated same as no params
		},
		{
			name: "whitespace normalized",
			m1:   "cache(300)",
			m2:   "cache( 300 )",
			want: true, // Whitespace in params is normalized
		},
		{
			name: "complex params",
			m1:   "rate_limit(10/hour)",
			m2:   "rate_limit(10/hour)",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.middlewareEquals(tt.m1, tt.m2)
			if got != tt.want {
				t.Errorf("middlewareEquals(%q, %q) = %v, want %v", tt.m1, tt.m2, got, tt.want)
			}
		})
	}
}

// TestValidator_ExtractParameters tests parameter extraction.
func TestValidator_ExtractParameters(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		middleware string
		want       string
	}{
		{
			name:       "no params",
			middleware: "auth",
			want:       "",
		},
		{
			name:       "simple param",
			middleware: "cache(300)",
			want:       "300",
		},
		{
			name:       "complex param",
			middleware: "rate_limit(10/hour)",
			want:       "10/hour",
		},
		{
			name:       "empty params",
			middleware: "auth()",
			want:       "",
		},
		{
			name:       "multiple values",
			middleware: "middleware(foo, bar, baz)",
			want:       "foo, bar, baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.extractParameters(tt.middleware)
			if got != tt.want {
				t.Errorf("extractParameters(%q) = %q, want %q", tt.middleware, got, tt.want)
			}
		})
	}
}

// TestValidator_CompareMiddlewareLists tests middleware list comparison.
func TestValidator_CompareMiddlewareLists(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		expected   []string
		actual     []string
		wantPassed bool
		wantConf   float64
	}{
		{
			name:       "exact match",
			expected:   []string{"auth"},
			actual:     []string{"auth"},
			wantPassed: true,
			wantConf:   1.0,
		},
		{
			name:       "multiple exact match",
			expected:   []string{"auth", "cache"},
			actual:     []string{"auth", "cache"},
			wantPassed: true,
			wantConf:   1.0,
		},
		{
			name:       "wrong order",
			expected:   []string{"auth", "cache"},
			actual:     []string{"cache", "auth"},
			wantPassed: false,
			wantConf:   0.0,
		},
		{
			name:       "partial match",
			expected:   []string{"auth", "cache"},
			actual:     []string{"auth", "rate_limit"},
			wantPassed: false,
			wantConf:   0.5,
		},
		{
			name:       "length mismatch",
			expected:   []string{"auth"},
			actual:     []string{"auth", "cache"},
			wantPassed: false,
			wantConf:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.compareMiddlewareLists(tt.expected, tt.actual)

			if result.Passed != tt.wantPassed {
				t.Errorf("Expected passed=%v, got %v", tt.wantPassed, result.Passed)
			}

			if result.Confidence != tt.wantConf {
				t.Errorf("Expected confidence=%.2f, got %.2f", tt.wantConf, result.Confidence)
			}
		})
	}
}

// TestValidator_ValidateMultiple tests validating multiple responses.
func TestValidator_ValidateMultiple(t *testing.T) {
	validator := NewValidator()

	responses := []string{
		"@on create: [cache]",        // Wrong
		"@on update: [auth]",         // Wrong operation
		"@on create: [auth]",         // Correct
		"@on create: [auth, cache]",  // Extra middleware
	}

	expected := "@on create: [auth]"

	result := validator.ValidateMultiple(responses, expected, "semantic")

	if !result.Passed {
		t.Error("Expected to find a passing match")
	}

	if result.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0 for perfect match, got %.2f", result.Confidence)
	}
}

// TestValidator_SemanticMatch_RealWorldCases tests realistic scenarios.
func TestValidator_SemanticMatch_RealWorldCases(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		response   string
		expected   string
		wantPassed bool
	}{
		{
			name: "LLM adds explanation",
			response: `To add authentication, use:

@on create: [auth]

This ensures only authenticated users can create resources.`,
			expected:   "@on create: [auth]",
			wantPassed: true,
		},
		{
			name:       "LLM uses code block",
			response:   "```conduit\n@on create: [auth, rate_limit(10/hour)]\n```",
			expected:   "@on create: [auth, rate_limit(10/hour)]",
			wantPassed: true,
		},
		{
			name: "LLM suggests variation",
			response: `You could use:

@on create: [auth]

This is the recommended approach for simple authentication.`,
			expected:   "@on create: [auth]",
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SemanticMatch(tt.response, tt.expected)

			if result.Passed != tt.wantPassed {
				t.Errorf("Expected passed=%v, got %v (message: %s)",
					tt.wantPassed, result.Passed, result.Message)
			}
		})
	}
}

// TestValidator_ValidationResult tests the ValidationResult structure.
func TestValidator_ValidationResult(t *testing.T) {
	result := ValidationResult{
		Passed:      true,
		MatchType:   "semantic",
		Confidence:  0.95,
		Expected:    "@on create: [auth]",
		Actual:      "@on create: [auth]",
		Differences: []string{},
		Message:     "semantic match",
	}

	if !result.Passed {
		t.Error("Result should be passed")
	}

	if result.MatchType != "semantic" {
		t.Errorf("Expected match type 'semantic', got '%s'", result.MatchType)
	}

	if result.Confidence < 0.9 {
		t.Errorf("Expected high confidence, got %.2f", result.Confidence)
	}
}

// TestValidator_EdgeCases tests edge cases and error conditions.
func TestValidator_EdgeCases(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		response string
		expected string
		mode     string
	}{
		{
			name:     "empty response",
			response: "",
			expected: "@on create: [auth]",
			mode:     "semantic",
		},
		{
			name:     "malformed declaration",
			response: "@on create [auth]", // Missing colon
			expected: "@on create: [auth]",
			mode:     "semantic",
		},
		{
			name:     "empty middleware list",
			response: "@on create: []",
			expected: "@on create: [auth]",
			mode:     "semantic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.response, tt.expected, tt.mode)

			// These should all fail gracefully
			if result.Passed {
				t.Error("Expected validation to fail for edge case")
			}
		})
	}
}
