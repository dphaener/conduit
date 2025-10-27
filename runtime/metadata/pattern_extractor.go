package metadata

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// PatternExtractor extracts common patterns from resource metadata.
// It identifies frequently occurring middleware chains, hook patterns,
// and other usage patterns to enable LLM learning and code generation.
type PatternExtractor struct {
	minFrequency int // Minimum occurrences for a pattern to be considered valid
}

// NewPatternExtractor creates a new pattern extractor with default settings.
// The default minimum frequency is 3, meaning a pattern must appear at least
// 3 times before it is extracted.
func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{
		minFrequency: 3,
	}
}

// ExtractMiddlewarePatterns extracts common middleware patterns from resource metadata.
// It analyzes all middleware chains across resources, groups identical chains together,
// and returns patterns that appear at least minFrequency times, sorted by frequency.
//
// Algorithm:
// 1. Collect all middleware chains from resource.Middleware map
// 2. Group identical chains by canonical key (middleware names joined with "|")
// 3. Filter chains by minFrequency (>= 3 occurrences)
// 4. Generate PatternMetadata for each valid pattern
// 5. Sort results by frequency (most common first)
func (pe *PatternExtractor) ExtractMiddlewarePatterns(resources []ResourceMetadata) []PatternMetadata {
	// Step 1: Collect all middleware chains
	chains := make(map[string]*middlewareChain)

	for _, resource := range resources {
		for operation, middleware := range resource.Middleware {
			// Create canonical key for this middleware chain.
			// NOTE: Order is preserved because middleware execution order is semantically
			// significant (e.g., auth must run before cache to avoid expensive operations
			// for unauthorized requests). Each unique ordering is treated as a distinct pattern.
			key := strings.Join(middleware, "|")

			if _, exists := chains[key]; !exists {
				chains[key] = &middlewareChain{
					middleware: middleware,
					usages:     []patternUsage{},
				}
			}

			chains[key].usages = append(chains[key].usages, patternUsage{
				resource:  resource.Name,
				operation: operation,
				filePath:  resource.FilePath,
			})
		}
	}

	// Step 2: Filter by frequency
	patterns := []PatternMetadata{}

	for _, chain := range chains {
		if len(chain.usages) >= pe.minFrequency {
			pattern := pe.generateMiddlewarePattern(chain)
			patterns = append(patterns, pattern)
		}
	}

	// Step 3: Sort by frequency (most common first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

// generateMiddlewarePattern creates a PatternMetadata from a middleware chain.
// It generates a descriptive name, template, and examples for the pattern.
func (pe *PatternExtractor) generateMiddlewarePattern(chain *middlewareChain) PatternMetadata {
	// Generate pattern name
	name := pe.generatePatternName(chain.middleware)

	// Generate template
	template := fmt.Sprintf("@on <operation>: [%s]", strings.Join(chain.middleware, ", "))

	// Generate examples
	examples := make([]PatternExample, 0, len(chain.usages))
	for _, usage := range chain.usages {
		examples = append(examples, PatternExample{
			Resource:   usage.resource,
			FilePath:   usage.filePath,
			LineNumber: 0, // Line numbers not tracked yet
			Code:       fmt.Sprintf("@on %s: [%s]", usage.operation, strings.Join(chain.middleware, ", ")),
		})
	}

	// Infer category
	category := pe.inferCategory(chain.middleware)

	return PatternMetadata{
		ID:          uuid.New().String(),
		Name:        name,
		Category:    category,
		Description: fmt.Sprintf("Handler with %s middleware", strings.Join(chain.middleware, " + ")),
		Template:    template,
		Examples:    examples,
		Frequency:   len(chain.usages),
		Confidence:  pe.calculateConfidence(len(chain.usages)),
	}
}

// generatePatternName creates a descriptive name for a middleware pattern.
// It extracts base names from middleware (ignoring parameters) and converts
// them to adjectives, then appends "handler".
//
// Examples:
//   - ["auth"] → "authenticated_handler"
//   - ["cache(300)"] → "cached_handler"
//   - ["auth", "rate_limit(5/hour)"] → "authenticated_rate_limited_handler"
func (pe *PatternExtractor) generatePatternName(middleware []string) string {
	parts := []string{}

	for _, m := range middleware {
		// Extract base name (ignore parameters)
		baseName := strings.Split(m, "(")[0]

		switch {
		case strings.Contains(baseName, "auth"):
			parts = append(parts, "authenticated")
		case strings.Contains(baseName, "cache"):
			parts = append(parts, "cached")
		case strings.Contains(baseName, "rate_limit"):
			parts = append(parts, "rate_limited")
		case strings.Contains(baseName, "cors"):
			parts = append(parts, "cors_enabled")
		case strings.Contains(baseName, "log"):
			parts = append(parts, "logged")
		default:
			// Use the base name as-is for unknown middleware
			parts = append(parts, baseName)
		}
	}

	parts = append(parts, "handler")
	return strings.Join(parts, "_")
}

// inferCategory determines the category of a middleware pattern based on
// the middleware types present in the chain.
//
// Priority order (first match wins):
//   1. Authentication (most security-critical)
//   2. Caching (performance optimization)
//   3. Rate limiting (abuse prevention)
//   4. CORS (cross-origin access)
//   5. General (fallback)
//
// Categories:
//   - "authentication" - if auth middleware is present
//   - "caching" - if cache middleware is present
//   - "rate_limiting" - if rate_limit middleware is present
//   - "cors" - if CORS middleware is present
//   - "general" - default category
//
// Examples:
//   - ["auth"] → "authentication"
//   - ["cache", "rate_limit"] → "caching" (cache wins)
//   - ["auth", "cache", "rate_limit"] → "authentication" (auth wins)
func (pe *PatternExtractor) inferCategory(middleware []string) string {
	for _, m := range middleware {
		if strings.Contains(m, "auth") {
			return "authentication"
		}
		if strings.Contains(m, "cache") {
			return "caching"
		}
		if strings.Contains(m, "rate_limit") {
			return "rate_limiting"
		}
		if strings.Contains(m, "cors") {
			return "cors"
		}
	}
	return "general"
}

// calculateConfidence calculates a confidence score for a pattern based on
// how frequently it appears in the codebase.
//
// Formula: confidence = frequency / 10.0, capped at 1.0
//
// Examples:
//   - frequency=3 → confidence=0.3
//   - frequency=5 → confidence=0.5
//   - frequency=12 → confidence=1.0 (capped)
func (pe *PatternExtractor) calculateConfidence(frequency int) float64 {
	confidence := float64(frequency) / 10.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

// middlewareChain represents a unique middleware chain and all places it's used.
type middlewareChain struct {
	middleware []string       // The middleware in this chain
	usages     []patternUsage // All places this chain appears
}

// patternUsage represents a single usage of a middleware chain.
type patternUsage struct {
	resource  string // Resource name where pattern is used
	operation string // Operation name (create, update, etc.)
	filePath  string // Source file path
}
