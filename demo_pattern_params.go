package main

import (
	"encoding/json"
	"fmt"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// This demo shows how to use PatternExtractor with custom parameters
// for the CON-65 iteration system.
func main() {
	// Sample resource metadata (simulating a blog application)
	resources := []metadata.ResourceMetadata{
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
			},
		},
	}

	fmt.Println("=== Default Parameters ===")
	demonstrateDefault(resources)

	fmt.Println("\n=== Custom Parameters (Permissive) ===")
	demonstratePermissive(resources)

	fmt.Println("\n=== Custom Parameters (High Quality) ===")
	demonstrateHighQuality(resources)

	fmt.Println("\n=== Custom Parameters (Verbose Names) ===")
	demonstrateVerbose(resources)

	fmt.Println("\n=== Custom Parameters (Compact Output) ===")
	demonstrateCompact(resources)
}

func demonstrateDefault(resources []metadata.ResourceMetadata) {
	pe := metadata.NewPatternExtractor()
	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d patterns\n", len(patterns))
	for _, pattern := range patterns {
		fmt.Printf("  - %s (freq: %d, conf: %.2f, examples: %d)\n",
			pattern.Name, pattern.Frequency, pattern.Confidence, len(pattern.Examples))
		if pattern.Description != "" {
			fmt.Printf("    Description: %s\n", pattern.Description)
		}
	}
}

func demonstratePermissive(resources []metadata.ResourceMetadata) {
	params := metadata.PatternExtractionParams{
		MinFrequency:        2,     // Lower threshold
		MinConfidence:       0.2,   // Lower confidence bar
		MaxExamples:         5,     // Standard examples
		IncludeDescriptions: true,  // Include descriptions
		VerboseNames:        false, // Concise names
	}
	pe := metadata.NewPatternExtractorWithParams(params)
	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d patterns (with MinFrequency=2)\n", len(patterns))
	for _, pattern := range patterns {
		fmt.Printf("  - %s (freq: %d)\n", pattern.Name, pattern.Frequency)
	}
}

func demonstrateHighQuality(resources []metadata.ResourceMetadata) {
	params := metadata.PatternExtractionParams{
		MinFrequency:        4,     // Higher threshold
		MinConfidence:       0.4,   // Higher confidence requirement
		MaxExamples:         5,     // Standard examples
		IncludeDescriptions: true,  // Include descriptions
		VerboseNames:        false, // Concise names
	}
	pe := metadata.NewPatternExtractorWithParams(params)
	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d high-quality patterns (MinFrequency=4, MinConfidence=0.4)\n", len(patterns))
	for _, pattern := range patterns {
		fmt.Printf("  - %s (freq: %d, conf: %.2f)\n",
			pattern.Name, pattern.Frequency, pattern.Confidence)
	}
}

func demonstrateVerbose(resources []metadata.ResourceMetadata) {
	params := metadata.DefaultParams()
	params.VerboseNames = true
	pe := metadata.NewPatternExtractorWithParams(params)
	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d patterns with verbose names\n", len(patterns))
	for _, pattern := range patterns {
		fmt.Printf("  - %s\n", pattern.Name)
		if len(pattern.Examples) > 0 {
			fmt.Printf("    Example: %s\n", pattern.Examples[0].Code)
		}
	}
}

func demonstrateCompact(resources []metadata.ResourceMetadata) {
	params := metadata.PatternExtractionParams{
		MinFrequency:        3,     // Standard
		MinConfidence:       0.3,   // Standard
		MaxExamples:         2,     // Limit examples
		IncludeDescriptions: false, // No descriptions
		VerboseNames:        false, // Concise names
	}
	pe := metadata.NewPatternExtractorWithParams(params)
	patterns := pe.ExtractMiddlewarePatterns(resources)

	fmt.Printf("Found %d patterns (compact output)\n", len(patterns))
	for _, pattern := range patterns {
		// Serialize to JSON to show size savings
		jsonBytes, _ := json.Marshal(pattern)
		fmt.Printf("  - %s (freq: %d, examples: %d, json_size: %d bytes)\n",
			pattern.Name, pattern.Frequency, len(pattern.Examples), len(jsonBytes))
	}
}
