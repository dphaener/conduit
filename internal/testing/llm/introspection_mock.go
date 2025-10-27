package llm

import (
	"fmt"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// IntrospectionMock simulates the runtime introspection API.
// It provides patterns based on category queries, simulating what an LLM
// would receive when querying the introspection system.
type IntrospectionMock struct {
	// patterns is the collection of all available patterns.
	patterns []metadata.PatternMetadata

	// patternsByCategory indexes patterns by category for fast lookup.
	patternsByCategory map[string][]metadata.PatternMetadata
}

// NewIntrospectionMock creates a new introspection mock with the given patterns.
func NewIntrospectionMock(patterns []metadata.PatternMetadata) *IntrospectionMock {
	mock := &IntrospectionMock{
		patterns:           patterns,
		patternsByCategory: make(map[string][]metadata.PatternMetadata),
	}

	// Index patterns by category
	for _, pattern := range patterns {
		mock.patternsByCategory[pattern.Category] = append(
			mock.patternsByCategory[pattern.Category],
			pattern,
		)
	}

	return mock
}

// Patterns returns all patterns matching the given category.
// This simulates the registry.Patterns(category) API call.
func (m *IntrospectionMock) Patterns(category string) []metadata.PatternMetadata {
	return m.patternsByCategory[category]
}

// AllPatterns returns all available patterns.
func (m *IntrospectionMock) AllPatterns() []metadata.PatternMetadata {
	return m.patterns
}

// PatternByName returns a specific pattern by name.
func (m *IntrospectionMock) PatternByName(name string) (metadata.PatternMetadata, error) {
	for _, pattern := range m.patterns {
		if pattern.Name == name {
			return pattern, nil
		}
	}
	return metadata.PatternMetadata{}, fmt.Errorf("pattern not found: %s", name)
}

// AddPattern adds a pattern to the mock.
func (m *IntrospectionMock) AddPattern(pattern metadata.PatternMetadata) {
	m.patterns = append(m.patterns, pattern)
	m.patternsByCategory[pattern.Category] = append(
		m.patternsByCategory[pattern.Category],
		pattern,
	)
}

// Categories returns all available pattern categories.
func (m *IntrospectionMock) Categories() []string {
	categories := make([]string, 0, len(m.patternsByCategory))
	for category := range m.patternsByCategory {
		categories = append(categories, category)
	}
	return categories
}

// NewDefaultIntrospectionMock creates a mock with commonly used patterns.
// This provides a baseline set of patterns for testing.
func NewDefaultIntrospectionMock() *IntrospectionMock {
	patterns := []metadata.PatternMetadata{
		{
			ID:          "auth-001",
			Name:        "authenticated_handler",
			Category:    "authentication",
			Description: "Handler with auth middleware",
			Template:    "@on <operation>: [auth]",
			Examples: []metadata.PatternExample{
				{
					Resource:   "Post",
					FilePath:   "/app/resources/post.cdt",
					LineNumber: 10,
					Code:       "@on create: [auth]",
				},
				{
					Resource:   "Comment",
					FilePath:   "/app/resources/comment.cdt",
					LineNumber: 15,
					Code:       "@on update: [auth]",
				},
			},
			Frequency:  5,
			Confidence: 0.95,
		},
		{
			ID:          "cache-001",
			Name:        "cached_handler",
			Category:    "caching",
			Description: "Handler with cache middleware",
			Template:    "@on <operation>: [cache(300)]",
			Examples: []metadata.PatternExample{
				{
					Resource:   "Article",
					FilePath:   "/app/resources/article.cdt",
					LineNumber: 20,
					Code:       "@on list: [cache(300)]",
				},
				{
					Resource:   "Tag",
					FilePath:   "/app/resources/tag.cdt",
					LineNumber: 12,
					Code:       "@on show: [cache(300)]",
				},
			},
			Frequency:  4,
			Confidence: 0.9,
		},
		{
			ID:          "rate-001",
			Name:        "rate_limited_handler",
			Category:    "rate_limiting",
			Description: "Handler with rate limit middleware",
			Template:    "@on <operation>: [rate_limit(10/hour)]",
			Examples: []metadata.PatternExample{
				{
					Resource:   "Post",
					FilePath:   "/app/resources/post.cdt",
					LineNumber: 25,
					Code:       "@on create: [rate_limit(10/hour)]",
				},
			},
			Frequency:  3,
			Confidence: 0.85,
		},
		{
			ID:          "auth-rate-001",
			Name:        "authenticated_rate_limited_handler",
			Category:    "authentication",
			Description: "Handler with auth + rate_limit middleware",
			Template:    "@on <operation>: [auth, rate_limit(10/hour)]",
			Examples: []metadata.PatternExample{
				{
					Resource:   "Post",
					FilePath:   "/app/resources/post.cdt",
					LineNumber: 30,
					Code:       "@on create: [auth, rate_limit(10/hour)]",
				},
			},
			Frequency:  2,
			Confidence: 0.8,
		},
		{
			ID:          "cors-001",
			Name:        "cors_enabled_handler",
			Category:    "cors",
			Description: "Handler with CORS middleware",
			Template:    "@on <operation>: [cors]",
			Examples: []metadata.PatternExample{
				{
					Resource:   "API",
					FilePath:   "/app/resources/api.cdt",
					LineNumber: 8,
					Code:       "@on list: [cors]",
				},
			},
			Frequency:  3,
			Confidence: 0.9,
		},
	}

	return NewIntrospectionMock(patterns)
}
