package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestRunIntrospectPatternsCommand(t *testing.T) {
	// Helper to create test metadata with patterns
	createTestMetadataWithPatterns := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{Name: "Post"},
				{Name: "Comment"},
				{Name: "User"},
			},
			Patterns: []metadata.PatternMetadata{
				{
					ID:          "auth_rate_limited",
					Name:        "authenticated_rate_limited",
					Category:    "Authentication",
					Description: "Authentication with rate limiting",
					Template:    "@on create: [auth, rate_limit(5, per: \"hour\")]",
					Frequency:   12,
					Confidence:  1.0,
					Examples: []metadata.PatternExample{
						{Resource: "Post", FilePath: "resources/post.cdt", LineNumber: 45},
						{Resource: "Comment", FilePath: "resources/comment.cdt", LineNumber: 23},
					},
				},
				{
					ID:          "owner_or_admin",
					Name:        "owner_or_admin",
					Category:    "Authentication",
					Description: "Ownership verification",
					Template:    "@on update: [auth, owner_or_admin]",
					Frequency:   8,
					Confidence:  0.8,
					Examples: []metadata.PatternExample{
						{Resource: "Post", FilePath: "resources/post.cdt", LineNumber: 50},
					},
				},
				{
					ID:          "list_cache",
					Name:        "list_cache",
					Category:    "Caching",
					Description: "Cache list operations",
					Template:    "@on list: [cache(300)]",
					Frequency:   5,
					Confidence:  0.9,
					Examples: []metadata.PatternExample{
						{Resource: "Post", FilePath: "resources/post.cdt", LineNumber: 60},
						{Resource: "Comment", FilePath: "resources/comment.cdt", LineNumber: 30},
					},
				},
				{
					ID:          "low_freq",
					Name:        "low_frequency_pattern",
					Category:    "Other",
					Description: "Low frequency pattern",
					Template:    "@on create: [custom]",
					Frequency:   1,
					Confidence:  0.5,
					Examples: []metadata.PatternExample{
						{Resource: "User", FilePath: "resources/user.cdt", LineNumber: 10},
					},
				},
			},
		}
	}

	t.Run("formats table output with all patterns", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Reset global flags
		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Check header
		assert.Contains(t, output, "PATTERNS")

		// Check categories
		assert.Contains(t, output, "Authentication")
		assert.Contains(t, output, "Caching")
		assert.Contains(t, output, "Other")

		// Check pattern names
		assert.Contains(t, output, "authenticated_rate_limited")
		assert.Contains(t, output, "owner_or_admin")
		assert.Contains(t, output, "list_cache")

		// Check frequency and confidence
		assert.Contains(t, output, "12 uses")
		assert.Contains(t, output, "confidence: 1.0")

		// Check templates
		assert.Contains(t, output, "Template:")
		assert.Contains(t, output, "@on create: [auth, rate_limit(5, per: \"hour\")]")

		// Check examples
		assert.Contains(t, output, "Used by:")
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "resources/post.cdt:45")

		// Check "when to use"
		assert.Contains(t, output, "When to use:")
	})

	t.Run("filters by category", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		// Filter by authentication category
		err = cmd.RunE(cmd, []string{"authentication"})
		require.NoError(t, err)

		output := buf.String()

		// Should contain authentication patterns
		assert.Contains(t, output, "authenticated_rate_limited")
		assert.Contains(t, output, "owner_or_admin")

		// Should NOT contain caching patterns
		assert.NotContains(t, output, "list_cache")
	})

	t.Run("filters by minimum frequency", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		// Set min-frequency flag
		cmd.Flags().Set("min-frequency", "5")

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Should contain patterns with frequency >= 5
		assert.Contains(t, output, "authenticated_rate_limited") // freq: 12
		assert.Contains(t, output, "owner_or_admin")            // freq: 8
		assert.Contains(t, output, "list_cache")                // freq: 5

		// Should NOT contain patterns with frequency < 5
		assert.NotContains(t, output, "low_frequency_pattern") // freq: 1
	})

	t.Run("returns error for negative min-frequency", func(t *testing.T) {
		// Setup test registry with patterns
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		cmd := newIntrospectPatternsCommand()
		cmd.SetArgs([]string{"--min-frequency", "-1"})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "min-frequency must be non-negative")
	})

	t.Run("sorts patterns by frequency", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		// Filter by authentication to check order
		err = cmd.RunE(cmd, []string{"authentication"})
		require.NoError(t, err)

		output := buf.String()

		// Find positions of patterns
		pos1 := strings.Index(output, "authenticated_rate_limited")
		pos2 := strings.Index(output, "owner_or_admin")

		// authenticated_rate_limited (12 uses) should come before owner_or_admin (8 uses)
		assert.True(t, pos1 < pos2, "Patterns should be sorted by frequency (descending)")
	})

	t.Run("formats JSON output", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPatterns()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set JSON format
		outputFormat = "json"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		// Parse JSON output
		var result struct {
			TotalCount int `json:"total_count"`
			Patterns   []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Category   string `json:"category"`
				Frequency  int    `json:"frequency"`
				Confidence float64 `json:"confidence"`
			} `json:"patterns"`
		}

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify JSON structure
		assert.Equal(t, 4, result.TotalCount)
		assert.Len(t, result.Patterns, 4)

		// Find authenticated_rate_limited pattern
		var authPattern *struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Category   string `json:"category"`
			Frequency  int    `json:"frequency"`
			Confidence float64 `json:"confidence"`
		}
		for i := range result.Patterns {
			if result.Patterns[i].Name == "authenticated_rate_limited" {
				authPattern = &result.Patterns[i]
				break
			}
		}

		require.NotNil(t, authPattern)
		assert.Equal(t, "auth_rate_limited", authPattern.ID)
		assert.Equal(t, "Authentication", authPattern.Category)
		assert.Equal(t, 12, authPattern.Frequency)
		assert.Equal(t, 1.0, authPattern.Confidence)

		// Reset format
		outputFormat = "table"
	})

	t.Run("handles no patterns", func(t *testing.T) {
		// Setup empty registry
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{Name: "Post"},
			},
			Patterns: []metadata.PatternMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No patterns found")
	})

	t.Run("limits examples to 5", func(t *testing.T) {
		// Setup test registry with pattern having many examples
		metadata.Reset()
		testMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{Name: "Post"},
				{Name: "Comment"},
				{Name: "User"},
				{Name: "Article"},
				{Name: "Page"},
				{Name: "Product"},
				{Name: "Order"},
			},
			Patterns: []metadata.PatternMetadata{
				{
					ID:         "many_examples",
					Name:       "pattern_with_many_examples",
					Category:   "Test",
					Template:   "@on create: [test]",
					Frequency:  7,
					Confidence: 1.0,
					Examples: []metadata.PatternExample{
						{Resource: "Post"},
						{Resource: "Comment"},
						{Resource: "User"},
						{Resource: "Article"},
						{Resource: "Page"},
						{Resource: "Product"},
						{Resource: "Order"},
					},
				},
			},
		}
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Should show first 5 examples
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "Comment")
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Article")
		assert.Contains(t, output, "Page")

		// Should show "... and N more" message
		assert.Contains(t, output, "[... and 2 more]")
	})

	// Cleanup after tests
	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}

func TestCalculateCoverage(t *testing.T) {
	t.Run("calculates coverage correctly", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
				},
			},
			{
				Examples: []metadata.PatternExample{
					{Resource: "Post"}, // Duplicate - should be counted once
					{Resource: "User"},
				},
			},
		}

		// 3 unique resources (Post, Comment, User) out of 5 total = 60%
		coverage := calculateCoverage(patterns, 5)
		assert.Equal(t, 60.0, coverage)
	})

	t.Run("handles zero total resources", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
				},
			},
		}

		coverage := calculateCoverage(patterns, 0)
		assert.Equal(t, 0.0, coverage)
	})

	t.Run("handles no patterns", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{}

		coverage := calculateCoverage(patterns, 10)
		assert.Equal(t, 0.0, coverage)
	})

	t.Run("handles 100% coverage", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "User"},
				},
			},
		}

		coverage := calculateCoverage(patterns, 3)
		assert.Equal(t, 100.0, coverage)
	})
}

func TestGenerateWhenToUse(t *testing.T) {
	tests := []struct {
		name     string
		pattern  metadata.PatternMetadata
		expected string
	}{
		{
			name: "authentication pattern",
			pattern: metadata.PatternMetadata{
				Category: "Authentication",
				Template: "@on create: [auth]",
			},
			expected: "Endpoints requiring user authentication",
		},
		{
			name: "authentication with rate limiting",
			pattern: metadata.PatternMetadata{
				Category: "Authentication",
				Template: "@on create: [auth, rate_limit(5)]",
			},
			expected: "User-generated content creation needing spam protection",
		},
		{
			name: "authorization with owner check",
			pattern: metadata.PatternMetadata{
				Category: "Authorization",
				Template: "@on update: [owner_or_admin]",
			},
			expected: "Operations requiring ownership verification or admin privileges",
		},
		{
			name: "caching pattern",
			pattern: metadata.PatternMetadata{
				Category: "Caching",
				Template: "@on list: [cache(300)]",
			},
			expected: "Frequently accessed read-only data",
		},
		{
			name: "rate limiting pattern",
			pattern: metadata.PatternMetadata{
				Category: "Rate Limiting",
				Template: "@on create: [rate_limit(5)]",
			},
			expected: "User-generated content needing spam protection",
		},
		{
			name: "validation pattern",
			pattern: metadata.PatternMetadata{
				Category: "Validation",
				Template: "@constraint: [required]",
			},
			expected: "Data that requires validation before persistence",
		},
		{
			name: "before hook pattern",
			pattern: metadata.PatternMetadata{
				Category: "Hook",
				Template: "@before create: [validate]",
			},
			expected: "Operations requiring pre-processing or validation",
		},
		{
			name: "after hook pattern",
			pattern: metadata.PatternMetadata{
				Category: "Hook",
				Template: "@after create: [notify]",
			},
			expected: "Operations requiring post-processing or notifications",
		},
		{
			name: "constraint pattern",
			pattern: metadata.PatternMetadata{
				Category: "Constraint",
				Template: "@constraint: [unique]",
			},
			expected: "Business rules that must be enforced across operations",
		},
		{
			name: "transaction pattern",
			pattern: metadata.PatternMetadata{
				Category: "Transaction",
				Template: "@transaction",
			},
			expected: "Operations requiring atomicity and rollback support",
		},
		{
			name: "async pattern",
			pattern: metadata.PatternMetadata{
				Category: "Async",
				Template: "@async",
			},
			expected: "Long-running operations that should not block the response",
		},
		{
			name: "pattern with description",
			pattern: metadata.PatternMetadata{
				Category:    "Custom",
				Template:    "@custom",
				Description: "Custom pattern description",
			},
			expected: "Custom pattern description",
		},
		{
			name: "unknown pattern without description",
			pattern: metadata.PatternMetadata{
				Category: "Unknown",
				Template: "@unknown",
			},
			expected: "Common pattern in the codebase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateWhenToUse(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPatternsAsTable(t *testing.T) {
	t.Run("formats patterns correctly", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				ID:          "test_pattern",
				Name:        "test_pattern",
				Category:    "Test",
				Description: "Test pattern",
				Template:    "@on create: [test]",
				Frequency:   5,
				Confidence:  0.9,
				Examples: []metadata.PatternExample{
					{Resource: "Post", FilePath: "resources/post.cdt", LineNumber: 10},
				},
			},
		}

		buf := &bytes.Buffer{}
		noColor = true

		err := formatPatternsAsTable(patterns, buf, 10)
		require.NoError(t, err)

		output := buf.String()

		assert.Contains(t, output, "PATTERNS")
		assert.Contains(t, output, "Test (1 patterns, 10% coverage)")
		assert.Contains(t, output, "test_pattern (5 uses, confidence: 0.9)")
		assert.Contains(t, output, "Template:")
		assert.Contains(t, output, "@on create: [test]")
		assert.Contains(t, output, "Used by:")
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "When to use:")

		noColor = false
	})

	t.Run("handles patterns without category", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				Name:       "uncategorized",
				Category:   "",
				Template:   "@test",
				Frequency:  1,
				Confidence: 1.0,
			},
		}

		buf := &bytes.Buffer{}
		noColor = true

		err := formatPatternsAsTable(patterns, buf, 5)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Other (1 patterns")

		noColor = false
	})
}

func TestFormatPatternsAsJSON(t *testing.T) {
	t.Run("formats patterns as valid JSON", func(t *testing.T) {
		patterns := []metadata.PatternMetadata{
			{
				ID:         "test_pattern",
				Name:       "test_pattern",
				Category:   "Test",
				Template:   "@test",
				Frequency:  5,
				Confidence: 0.9,
			},
		}

		buf := &bytes.Buffer{}
		err := formatPatternsAsJSON(patterns, buf)
		require.NoError(t, err)

		// Verify it's valid JSON
		var result struct {
			TotalCount int                       `json:"total_count"`
			Patterns   []metadata.PatternMetadata `json:"patterns"`
		}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Patterns, 1)
		assert.Equal(t, "test_pattern", result.Patterns[0].Name)
	})
}

// BenchmarkIntrospectPatternsCommand benchmarks the patterns command performance
func BenchmarkIntrospectPatternsCommand(b *testing.B) {
	// Setup test registry with realistic pattern data
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: make([]metadata.ResourceMetadata, 0, 50),
		Patterns:  make([]metadata.PatternMetadata, 0, 20),
	}

	// Create 50 resources
	for i := 0; i < 50; i++ {
		testMeta.Resources = append(testMeta.Resources, metadata.ResourceMetadata{
			Name: fmt.Sprintf("Resource%d", i),
		})
	}

	// Create 20 patterns with multiple examples
	categories := []string{"Authentication", "Authorization", "Caching", "Validation", "Hook"}
	for i := 0; i < 20; i++ {
		examples := make([]metadata.PatternExample, 0, 10)
		for j := 0; j < 10; j++ {
			examples = append(examples, metadata.PatternExample{
				Resource:   fmt.Sprintf("Resource%d", j),
				FilePath:   fmt.Sprintf("resources/resource%d.cdt", j),
				LineNumber: j * 10,
			})
		}

		testMeta.Patterns = append(testMeta.Patterns, metadata.PatternMetadata{
			ID:          fmt.Sprintf("pattern_%d", i),
			Name:        fmt.Sprintf("pattern_%d", i),
			Category:    categories[i%len(categories)],
			Description: fmt.Sprintf("Pattern %d description", i),
			Template:    fmt.Sprintf("@on create: [middleware_%d]", i),
			Frequency:   10 + i,
			Confidence:  0.8 + (float64(i) * 0.01),
			Examples:    examples,
		})
	}

	metadata.Reset()
	data, err := json.Marshal(testMeta)
	if err != nil {
		b.Fatal(err)
	}
	err = metadata.RegisterMetadata(data)
	if err != nil {
		b.Fatal(err)
	}

	// Reset flags
	outputFormat = "table"
	verbose = false
	noColor = true

	cmd := newIntrospectPatternsCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := cmd.RunE(cmd, []string{})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Cleanup(func() {
		metadata.Reset()
	})
}
