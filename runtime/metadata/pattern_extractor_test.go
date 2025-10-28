package metadata

import (
	"testing"
	"time"
)

// TestNewPatternExtractor tests that NewPatternExtractor creates a properly initialized extractor.
func TestNewPatternExtractor(t *testing.T) {
	pe := NewPatternExtractor()

	if pe == nil {
		t.Fatal("NewPatternExtractor returned nil")
	}

	if pe.params.MinFrequency != 3 {
		t.Errorf("Expected MinFrequency to be 3, got %d", pe.params.MinFrequency)
	}

	if pe.params.MinConfidence != 0.3 {
		t.Errorf("Expected MinConfidence to be 0.3, got %f", pe.params.MinConfidence)
	}

	if pe.params.MaxExamples != 5 {
		t.Errorf("Expected MaxExamples to be 5, got %d", pe.params.MaxExamples)
	}

	if !pe.params.IncludeDescriptions {
		t.Error("Expected IncludeDescriptions to be true")
	}

	if pe.params.VerboseNames {
		t.Error("Expected VerboseNames to be false")
	}
}

// TestExtractMiddlewarePatterns_BasicExtraction tests basic pattern extraction.
func TestExtractMiddlewarePatterns_BasicExtraction(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/resources/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/resources/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"delete": {"auth"},
			},
		},
		{
			Name:     "Like",
			FilePath: "/app/resources/like.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	// Should extract one pattern for ["auth"] with frequency 5
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	pattern := patterns[0]
	if pattern.Name != "authenticated_handler" {
		t.Errorf("Expected pattern name 'authenticated_handler', got '%s'", pattern.Name)
	}

	if pattern.Category != "authentication" {
		t.Errorf("Expected category 'authentication', got '%s'", pattern.Category)
	}

	if pattern.Frequency != 5 {
		t.Errorf("Expected frequency 5, got %d", pattern.Frequency)
	}

	if pattern.Confidence != 0.5 {
		t.Errorf("Expected confidence 0.5, got %f", pattern.Confidence)
	}

	if pattern.Template != "@on <operation>: [auth]" {
		t.Errorf("Expected template '@on <operation>: [auth]', got '%s'", pattern.Template)
	}

	if len(pattern.Examples) != 5 {
		t.Errorf("Expected 5 examples, got %d", len(pattern.Examples))
	}
}

// TestExtractMiddlewarePatterns_FrequencyFiltering tests that patterns below minFrequency are filtered out.
func TestExtractMiddlewarePatterns_FrequencyFiltering(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/resources/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/resources/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
			},
		},
		{
			Name:     "Like",
			FilePath: "/app/resources/like.cdt",
			Middleware: map[string][]string{
				"create": {"cache"}, // Only appears twice
				"update": {"cache"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	// Should only extract ["auth"] pattern (frequency 3), not ["cache"] (frequency 2)
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	if patterns[0].Name != "authenticated_handler" {
		t.Errorf("Expected 'authenticated_handler', got '%s'", patterns[0].Name)
	}
}

// TestExtractMiddlewarePatterns_MultipleMiddleware tests extraction of multi-middleware chains.
func TestExtractMiddlewarePatterns_MultipleMiddleware(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/resources/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth", "rate_limit(5/hour)"},
				"update": {"auth", "rate_limit(5/hour)"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/resources/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth", "rate_limit(5/hour)"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	pattern := patterns[0]
	if pattern.Name != "authenticated_rate_limited_handler" {
		t.Errorf("Expected 'authenticated_rate_limited_handler', got '%s'", pattern.Name)
	}

	if pattern.Category != "authentication" {
		t.Errorf("Expected category 'authentication', got '%s'", pattern.Category)
	}

	expectedTemplate := "@on <operation>: [auth, rate_limit(5/hour)]"
	if pattern.Template != expectedTemplate {
		t.Errorf("Expected template '%s', got '%s'", expectedTemplate, pattern.Template)
	}
}

// TestExtractMiddlewarePatterns_OrderSensitivity verifies that middleware order matters
func TestExtractMiddlewarePatterns_OrderSensitivity(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth", "cache"},
				"update": {"auth", "cache"},
				"delete": {"auth", "cache"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/comment.cdt",
			Middleware: map[string][]string{
				"create": {"cache", "auth"}, // Different order
				"update": {"cache", "auth"},
				"delete": {"cache", "auth"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	// Should extract TWO patterns because order differs
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns (order-sensitive), got %d", len(patterns))
	}

	// Verify both patterns have frequency of 3
	for _, pattern := range patterns {
		if pattern.Frequency != 3 {
			t.Errorf("Expected frequency 3 for pattern %s, got %d", pattern.Name, pattern.Frequency)
		}
	}
}

// TestExtractMiddlewarePatterns_SortByFrequency tests that patterns are sorted by frequency.
func TestExtractMiddlewarePatterns_SortByFrequency(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		// auth appears 5 times
		{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{"create": {"auth"}, "update": {"auth"}}},
		{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{"create": {"auth"}}},
		{Name: "Like", FilePath: "/app/like.cdt", Middleware: map[string][]string{"create": {"auth"}, "delete": {"auth"}}},

		// cache appears 3 times
		{Name: "Article", FilePath: "/app/article.cdt", Middleware: map[string][]string{"list": {"cache"}, "show": {"cache"}}},
		{Name: "Tag", FilePath: "/app/tag.cdt", Middleware: map[string][]string{"list": {"cache"}}},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(patterns))
	}

	// Should be sorted by frequency: auth (5) before cache (3)
	if patterns[0].Name != "authenticated_handler" {
		t.Errorf("Expected first pattern to be 'authenticated_handler', got '%s'", patterns[0].Name)
	}

	if patterns[1].Name != "cached_handler" {
		t.Errorf("Expected second pattern to be 'cached_handler', got '%s'", patterns[1].Name)
	}

	if patterns[0].Frequency != 5 {
		t.Errorf("Expected first pattern frequency 5, got %d", patterns[0].Frequency)
	}

	if patterns[1].Frequency != 3 {
		t.Errorf("Expected second pattern frequency 3, got %d", patterns[1].Frequency)
	}
}

// TestExtractMiddlewarePatterns_EmptyResources tests extraction with no resources.
func TestExtractMiddlewarePatterns_EmptyResources(t *testing.T) {
	pe := NewPatternExtractor()

	patterns := pe.ExtractMiddlewarePatterns([]ResourceMetadata{})

	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns for empty resources, got %d", len(patterns))
	}
}

// TestExtractMiddlewarePatterns_NoMiddleware tests extraction when resources have no middleware.
func TestExtractMiddlewarePatterns_NoMiddleware(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{}},
		{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{}},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns when no middleware exists, got %d", len(patterns))
	}
}

// TestGeneratePatternName tests pattern name generation for various middleware combinations.
func TestGeneratePatternName(t *testing.T) {
	pe := NewPatternExtractor()

	tests := []struct {
		name       string
		middleware []string
		expected   string
	}{
		{
			name:       "single auth",
			middleware: []string{"auth"},
			expected:   "authenticated_handler",
		},
		{
			name:       "single cache",
			middleware: []string{"cache(300)"},
			expected:   "cached_handler",
		},
		{
			name:       "single rate_limit",
			middleware: []string{"rate_limit(5/hour)"},
			expected:   "rate_limited_handler",
		},
		{
			name:       "auth and rate_limit",
			middleware: []string{"auth", "rate_limit(5/hour)"},
			expected:   "authenticated_rate_limited_handler",
		},
		{
			name:       "auth and cache",
			middleware: []string{"auth", "cache(300)"},
			expected:   "authenticated_cached_handler",
		},
		{
			name:       "cors middleware",
			middleware: []string{"cors"},
			expected:   "cors_enabled_handler",
		},
		{
			name:       "log middleware",
			middleware: []string{"log"},
			expected:   "logged_handler",
		},
		{
			name:       "unknown middleware",
			middleware: []string{"custom_middleware"},
			expected:   "custom_middleware_handler",
		},
		{
			name:       "multiple middleware",
			middleware: []string{"auth", "cache", "rate_limit"},
			expected:   "authenticated_cached_rate_limited_handler",
		},
	}

	// Create empty usages for testing non-verbose names
	emptyUsages := []patternUsage{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pe.generatePatternName(tt.middleware, emptyUsages)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestInferCategory tests category inference for various middleware types.
func TestInferCategory(t *testing.T) {
	pe := NewPatternExtractor()

	tests := []struct {
		name       string
		middleware []string
		expected   string
	}{
		{
			name:       "auth category",
			middleware: []string{"auth"},
			expected:   "authentication",
		},
		{
			name:       "cache category",
			middleware: []string{"cache"},
			expected:   "caching",
		},
		{
			name:       "rate_limit category",
			middleware: []string{"rate_limit"},
			expected:   "rate_limiting",
		},
		{
			name:       "cors category",
			middleware: []string{"cors"},
			expected:   "cors",
		},
		{
			name:       "auth takes precedence",
			middleware: []string{"auth", "cache", "rate_limit"},
			expected:   "authentication",
		},
		{
			name:       "unknown defaults to general",
			middleware: []string{"custom"},
			expected:   "general",
		},
		{
			name:       "empty defaults to general",
			middleware: []string{},
			expected:   "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pe.inferCategory(tt.middleware)
			if result != tt.expected {
				t.Errorf("Expected category '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCalculateConfidence tests confidence calculation for various frequencies.
func TestCalculateConfidence(t *testing.T) {
	pe := NewPatternExtractor()

	tests := []struct {
		frequency int
		expected  float64
	}{
		{frequency: 1, expected: 0.1},
		{frequency: 3, expected: 0.3},
		{frequency: 5, expected: 0.5},
		{frequency: 10, expected: 1.0},
		{frequency: 12, expected: 1.0}, // Capped at 1.0
		{frequency: 100, expected: 1.0}, // Capped at 1.0
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := pe.calculateConfidence(tt.frequency)
			if result != tt.expected {
				t.Errorf("For frequency %d, expected confidence %f, got %f", tt.frequency, tt.expected, result)
			}
		})
	}
}

// TestExtractMiddlewarePatterns_ExamplesTracking tests that examples are properly tracked.
func TestExtractMiddlewarePatterns_ExamplesTracking(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/resources/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/resources/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
			},
		},
		{
			Name:     "Like",
			FilePath: "/app/resources/like.cdt",
			Middleware: map[string][]string{
				"delete": {"auth"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	pattern := patterns[0]
	if len(pattern.Examples) != 4 {
		t.Fatalf("Expected 4 examples, got %d", len(pattern.Examples))
	}

	// Verify examples contain correct resource and operation information
	exampleMap := make(map[string]bool)
	for _, example := range pattern.Examples {
		key := example.Resource + ":" + example.Code
		exampleMap[key] = true

		// Verify FilePath is set
		if example.FilePath == "" {
			t.Errorf("Example for resource '%s' has empty FilePath", example.Resource)
		}

		// Verify Code format
		expectedPrefix := "@on "
		if len(example.Code) < len(expectedPrefix) || example.Code[:4] != expectedPrefix {
			t.Errorf("Example code should start with '@on ', got '%s'", example.Code)
		}
	}

	// Check that we have examples from all resources
	expectedExamples := []string{
		"Post:@on create: [auth]",
		"Post:@on update: [auth]",
		"Comment:@on create: [auth]",
		"Like:@on delete: [auth]",
	}

	for _, expected := range expectedExamples {
		if !exampleMap[expected] {
			t.Errorf("Missing expected example: %s", expected)
		}
	}
}

// TestExtractMiddlewarePatterns_UniqueIDs tests that each pattern gets a unique ID.
func TestExtractMiddlewarePatterns_UniqueIDs(t *testing.T) {
	pe := NewPatternExtractor()

	resources := []ResourceMetadata{
		// Create multiple patterns
		{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
			"create": {"auth"}, "update": {"auth"}, "delete": {"auth"},
		}},
		{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
			"create": {"auth"}, "update": {"auth"},
		}},
		{Name: "Article", FilePath: "/app/article.cdt", Middleware: map[string][]string{
			"list": {"cache"}, "show": {"cache"}, "search": {"cache"},
		}},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(patterns))
	}

	// Check that IDs are unique and non-empty
	ids := make(map[string]bool)
	for _, pattern := range patterns {
		if pattern.ID == "" {
			t.Error("Pattern has empty ID")
		}

		if ids[pattern.ID] {
			t.Errorf("Duplicate pattern ID: %s", pattern.ID)
		}
		ids[pattern.ID] = true
	}
}

// TestExtractMiddlewarePatterns_BlogExample tests extraction with a realistic blog example.
func TestExtractMiddlewarePatterns_BlogExample(t *testing.T) {
	pe := NewPatternExtractor()

	// Realistic blog application resources
	resources := []ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/app/resources/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth", "rate_limit(10/hour)"},
				"update": {"auth"},
				"delete": {"auth"},
				"list":   {"cache(300)"},
				"show":   {"cache(300)"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/app/resources/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth", "rate_limit(10/hour)"},
				"update": {"auth"},
				"delete": {"auth"},
			},
		},
		{
			Name:     "User",
			FilePath: "/app/resources/user.cdt",
			Middleware: map[string][]string{
				"update": {"auth"},
				"delete": {"auth"},
				"show":   {"cache(300)"},
			},
		},
		{
			Name:     "Tag",
			FilePath: "/app/resources/tag.cdt",
			Middleware: map[string][]string{
				"list": {"cache(300)"},
				"show": {"cache(300)"},
			},
		},
		{
			Name:     "Category",
			FilePath: "/app/resources/category.cdt",
			Middleware: map[string][]string{
				"list": {"cache(300)"},
			},
		},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	// Expected patterns (frequency >= 3):
	// - ["auth"] - 6 times
	// - ["cache(300)"] - 6 times
	// - ["auth", "rate_limit(10/hour)"] - 2 times (should NOT appear)

	if len(patterns) < 2 || len(patterns) > 5 {
		t.Errorf("Expected 2-5 patterns for blog example, got %d", len(patterns))
	}

	// Verify top patterns are auth and cache
	foundAuth := false
	foundCache := false

	for _, pattern := range patterns {
		if pattern.Name == "authenticated_handler" && pattern.Frequency >= 6 {
			foundAuth = true
		}
		if pattern.Name == "cached_handler" && pattern.Frequency >= 6 {
			foundCache = true
		}
	}

	if !foundAuth {
		t.Error("Expected to find 'authenticated_handler' pattern with frequency >= 6")
	}

	if !foundCache {
		t.Error("Expected to find 'cached_handler' pattern with frequency >= 6")
	}
}

// BenchmarkExtractMiddlewarePatterns_50Resources benchmarks pattern extraction with 50 resources.
func BenchmarkExtractMiddlewarePatterns_50Resources(b *testing.B) {
	pe := NewPatternExtractor()

	// Create 50 resources with varied middleware
	resources := make([]ResourceMetadata, 50)
	for i := 0; i < 50; i++ {
		resources[i] = ResourceMetadata{
			Name:     "Resource" + string(rune('A'+i%26)),
			FilePath: "/app/resources/resource.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth", "cache"},
				"delete": {"auth"},
				"list":   {"cache"},
			},
		}
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		pe.ExtractMiddlewarePatterns(resources)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(b.N)

	b.Logf("Average time for 50 resources: %v", avgTime)

	// Verify performance target (<50ms)
	if avgTime > 50*time.Millisecond {
		b.Errorf("Performance target missed: expected <50ms, got %v", avgTime)
	}
}

// TestPatternExtractorWithCustomParams tests pattern extraction with custom parameters.
func TestPatternExtractorWithCustomParams(t *testing.T) {
	t.Run("MinFrequency=2 includes more patterns", func(t *testing.T) {
		params := DefaultParams()
		params.MinFrequency = 2
		params.MinConfidence = 0.0 // Disable confidence filtering for this test
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"cache"}, // Only appears twice
				"list":   {"cache"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		// With MinFrequency=2, both patterns should be extracted
		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns with MinFrequency=2, got %d", len(patterns))
		}
	})

	t.Run("MinConfidence filters low-confidence patterns", func(t *testing.T) {
		params := DefaultParams()
		params.MinFrequency = 2
		params.MinConfidence = 0.5 // Requires frequency >= 5
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"auth"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		// Frequency is 3, confidence is 0.3, should be filtered out
		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns with MinConfidence=0.5, got %d", len(patterns))
		}
	})

	t.Run("MaxExamples limits example count", func(t *testing.T) {
		params := DefaultParams()
		params.MaxExamples = 2
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"auth"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		// Should only have 2 examples despite 3 usages
		if len(patterns[0].Examples) != 2 {
			t.Errorf("Expected 2 examples with MaxExamples=2, got %d", len(patterns[0].Examples))
		}

		// Frequency should still reflect total usages
		if patterns[0].Frequency != 3 {
			t.Errorf("Expected frequency 3, got %d", patterns[0].Frequency)
		}
	})

	t.Run("IncludeDescriptions=false removes descriptions", func(t *testing.T) {
		params := DefaultParams()
		params.IncludeDescriptions = false
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"auth"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Description != "" {
			t.Errorf("Expected empty description with IncludeDescriptions=false, got '%s'", patterns[0].Description)
		}
	})

	t.Run("VerboseNames generates detailed names", func(t *testing.T) {
		params := DefaultParams()
		params.VerboseNames = true
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"auth"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		// Should include operation context (most common operation)
		// We have 2 "create" and 1 "update", so "create" should be selected
		if patterns[0].Name != "authenticated_handler_for_create" {
			t.Errorf("Expected 'authenticated_handler_for_create' with VerboseNames, got '%s'", patterns[0].Name)
		}
	})

	t.Run("VerboseNames with parameters", func(t *testing.T) {
		params := DefaultParams()
		params.VerboseNames = true
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"list": {"cache(300)"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"list": {"cache(300)"},
			}},
			{Name: "Tag", FilePath: "/app/tag.cdt", Middleware: map[string][]string{
				"list": {"cache(300)"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		// Should include parameter details and operation
		// Format: <middleware>_handler_with_<params>_for_<operation>
		expectedName := "cached_handler_with_300_for_list"
		if patterns[0].Name != expectedName {
			t.Errorf("Expected '%s' with VerboseNames, got '%s'", expectedName, patterns[0].Name)
		}
	})

	t.Run("Combined custom params", func(t *testing.T) {
		params := PatternExtractionParams{
			MinFrequency:        2,
			MinConfidence:       0.2,
			MaxExamples:         1,
			IncludeDescriptions: false,
			VerboseNames:        true,
		}
		pe := NewPatternExtractorWithParams(params)

		resources := []ResourceMetadata{
			{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
				"create": {"auth", "rate_limit(5/hour)"},
			}},
			{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
				"create": {"auth", "rate_limit(5/hour)"},
			}},
		}

		patterns := pe.ExtractMiddlewarePatterns(resources)

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		pattern := patterns[0]

		// Check VerboseNames - params are combined at the end
		// Format: <middleware1>_<middleware2>_handler_with_<params>_for_<operation>
		expectedName := "authenticated_rate_limited_handler_with_5_per_hour_for_create"
		if pattern.Name != expectedName {
			t.Errorf("Expected name '%s', got '%s'", expectedName, pattern.Name)
		}

		// Check MaxExamples
		if len(pattern.Examples) != 1 {
			t.Errorf("Expected 1 example, got %d", len(pattern.Examples))
		}

		// Check IncludeDescriptions
		if pattern.Description != "" {
			t.Errorf("Expected empty description, got '%s'", pattern.Description)
		}

		// Check frequency still correct
		if pattern.Frequency != 2 {
			t.Errorf("Expected frequency 2, got %d", pattern.Frequency)
		}
	})
}

// TestDefaultParams tests that DefaultParams returns correct defaults.
func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	if params.MinFrequency != 3 {
		t.Errorf("Expected MinFrequency 3, got %d", params.MinFrequency)
	}

	if params.MinConfidence != 0.3 {
		t.Errorf("Expected MinConfidence 0.3, got %f", params.MinConfidence)
	}

	if params.MaxExamples != 5 {
		t.Errorf("Expected MaxExamples 5, got %d", params.MaxExamples)
	}

	if !params.IncludeDescriptions {
		t.Error("Expected IncludeDescriptions true")
	}

	if params.VerboseNames {
		t.Error("Expected VerboseNames false")
	}
}
