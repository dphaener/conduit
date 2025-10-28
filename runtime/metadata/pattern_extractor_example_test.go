package metadata_test

import (
	"fmt"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// ExamplePatternExtractor_defaultParams demonstrates basic pattern extraction with default parameters.
func ExamplePatternExtractor_defaultParams() {
	// Create extractor with default params:
	// - MinFrequency: 3 (patterns must appear at least 3 times)
	// - MinConfidence: 0.3 (confidence score 0.0-1.0)
	// - MaxExamples: 5 (limit examples per pattern)
	// - IncludeDescriptions: true
	// - VerboseNames: false
	pe := metadata.NewPatternExtractor()

	resources := []metadata.ResourceMetadata{
		{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
			"create": {"auth"},
			"update": {"auth"},
		}},
		{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
			"create": {"auth"},
		}},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	for _, pattern := range patterns {
		fmt.Printf("Pattern: %s (frequency: %d, confidence: %.1f)\n",
			pattern.Name, pattern.Frequency, pattern.Confidence)
	}

	// Output:
	// Pattern: authenticated_handler (frequency: 3, confidence: 0.3)
}

// ExamplePatternExtractor_customParams demonstrates pattern extraction with custom parameters.
func ExamplePatternExtractor_customParams() {
	// Create extractor with custom params for more aggressive pattern extraction
	params := metadata.PatternExtractionParams{
		MinFrequency:        2,     // Lower threshold - patterns appear at least 2 times
		MinConfidence:       0.2,   // Lower confidence bar
		MaxExamples:         3,     // Fewer examples to reduce output size
		IncludeDescriptions: false, // Skip descriptions for compact output
		VerboseNames:        false, // Use concise names
	}
	pe := metadata.NewPatternExtractorWithParams(params)

	resources := []metadata.ResourceMetadata{
		{Name: "Post", FilePath: "/app/post.cdt", Middleware: map[string][]string{
			"create": {"auth"},
			"update": {"auth"},
		}},
		{Name: "Comment", FilePath: "/app/comment.cdt", Middleware: map[string][]string{
			"create": {"cache"},
			"list":   {"cache"},
		}},
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d patterns with MinFrequency=2\n", len(patterns))

	for _, pattern := range patterns {
		fmt.Printf("- %s (frequency: %d, examples: %d)\n",
			pattern.Name, pattern.Frequency, len(pattern.Examples))
	}

	// Output:
	// Found 2 patterns with MinFrequency=2
	// - authenticated_handler (frequency: 2, examples: 2)
	// - cached_handler (frequency: 2, examples: 2)
}

// ExamplePatternExtractor_verboseNames demonstrates verbose naming mode for detailed pattern names.
func ExamplePatternExtractor_verboseNames() {
	// Enable verbose names to include operation context and parameters
	params := metadata.DefaultParams()
	params.VerboseNames = true
	pe := metadata.NewPatternExtractorWithParams(params)

	resources := []metadata.ResourceMetadata{
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

	for _, pattern := range patterns {
		fmt.Printf("Pattern: %s\n", pattern.Name)
		fmt.Printf("Description: %s\n", pattern.Description)
	}

	// Output:
	// Pattern: cached_handler_with_300_for_list
	// Description: Handler with cache(300) middleware
}

// ExamplePatternExtractor_highQualityPatterns demonstrates filtering for high-confidence patterns.
func ExamplePatternExtractor_highQualityPatterns() {
	// Configure for high-quality patterns only
	params := metadata.PatternExtractionParams{
		MinFrequency:        5,    // Require 5+ occurrences
		MinConfidence:       0.5,  // Require 50%+ confidence
		MaxExamples:         10,   // More examples for confident patterns
		IncludeDescriptions: true, // Include detailed descriptions
		VerboseNames:        false,
	}
	pe := metadata.NewPatternExtractorWithParams(params)

	// Create resources with a common auth pattern
	resources := make([]metadata.ResourceMetadata, 6)
	for i := 0; i < 6; i++ {
		resources[i] = metadata.ResourceMetadata{
			Name:     fmt.Sprintf("Resource%d", i),
			FilePath: fmt.Sprintf("/app/resource%d.cdt", i),
			Middleware: map[string][]string{
				"create": {"auth"},
			},
		}
	}

	patterns := pe.ExtractMiddlewarePatterns(resources)

	if len(patterns) > 0 {
		pattern := patterns[0]
		fmt.Printf("High-confidence pattern: %s\n", pattern.Name)
		fmt.Printf("Frequency: %d\n", pattern.Frequency)
		fmt.Printf("Confidence: %.1f\n", pattern.Confidence)
		fmt.Printf("Examples: %d\n", len(pattern.Examples))
	}

	// Output:
	// High-confidence pattern: authenticated_handler
	// Frequency: 6
	// Confidence: 0.6
	// Examples: 6
}
