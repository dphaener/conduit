package metadata

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// PatternExtractionParams controls how patterns are extracted and what quality bar they must meet.
type PatternExtractionParams struct {
	// MinFrequency is minimum occurrences for a pattern (default: 3)
	MinFrequency int

	// MinConfidence is minimum confidence score 0.0-1.0 (default: 0.3)
	MinConfidence float64

	// MaxExamples limits examples per pattern (default: 5)
	MaxExamples int

	// IncludeDescriptions adds detailed descriptions (default: true)
	IncludeDescriptions bool

	// VerboseNames uses more descriptive pattern names (default: false)
	VerboseNames bool
}

// DefaultParams returns the default extraction parameters.
func DefaultParams() PatternExtractionParams {
	return PatternExtractionParams{
		MinFrequency:        3,
		MinConfidence:       0.3,
		MaxExamples:         5,
		IncludeDescriptions: true,
		VerboseNames:        false,
	}
}

// PatternExtractor extracts common patterns from resource metadata.
// It identifies frequently occurring middleware chains, hook patterns,
// and other usage patterns to enable LLM learning and code generation.
type PatternExtractor struct {
	params PatternExtractionParams
}

// NewPatternExtractor creates a new pattern extractor with default settings.
// The default minimum frequency is 3, meaning a pattern must appear at least
// 3 times before it is extracted.
func NewPatternExtractor() *PatternExtractor {
	return NewPatternExtractorWithParams(DefaultParams())
}

// NewPatternExtractorWithParams creates a pattern extractor with custom parameters.
func NewPatternExtractorWithParams(params PatternExtractionParams) *PatternExtractor {
	return &PatternExtractor{
		params: params,
	}
}

// GetParams returns the current extraction parameters.
func (pe *PatternExtractor) GetParams() PatternExtractionParams {
	return pe.params
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

	// Step 2: Filter by frequency and confidence
	patterns := []PatternMetadata{}

	for _, chain := range chains {
		if len(chain.usages) >= pe.params.MinFrequency {
			pattern := pe.generateMiddlewarePattern(chain)
			// Filter by minimum confidence
			if pattern.Confidence >= pe.params.MinConfidence {
				patterns = append(patterns, pattern)
			}
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
	name := pe.generatePatternName(chain.middleware, chain.usages)

	// Generate template
	template := fmt.Sprintf("@on <operation>: [%s]", strings.Join(chain.middleware, ", "))

	// Generate examples (limit to MaxExamples)
	maxExamples := pe.params.MaxExamples
	if maxExamples <= 0 || maxExamples > len(chain.usages) {
		maxExamples = len(chain.usages)
	}
	examples := make([]PatternExample, 0, maxExamples)
	for i := 0; i < maxExamples; i++ {
		usage := chain.usages[i]
		examples = append(examples, PatternExample{
			Resource:   usage.resource,
			FilePath:   usage.filePath,
			LineNumber: 0, // Line numbers not tracked yet
			Code:       fmt.Sprintf("@on %s: [%s]", usage.operation, strings.Join(chain.middleware, ", ")),
		})
	}

	// Infer category
	category := pe.inferCategory(chain.middleware)

	// Generate description (if enabled)
	description := ""
	if pe.params.IncludeDescriptions {
		description = fmt.Sprintf("Handler with %s middleware", strings.Join(chain.middleware, " + "))
	}

	return PatternMetadata{
		ID:          uuid.New().String(),
		Name:        name,
		Category:    category,
		Description: description,
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
// When VerboseNames is enabled, adds operation context and parameter details.
//
// Examples (VerboseNames=false):
//   - ["auth"] → "authenticated_handler"
//   - ["cache(300)"] → "cached_handler"
//   - ["auth", "rate_limit(5/hour)"] → "authenticated_rate_limited_handler"
//
// Examples (VerboseNames=true):
//   - ["auth"] for create → "authenticated_handler_for_create"
//   - ["cache(300)"] for list → "cached_handler_with_300_for_list"
func (pe *PatternExtractor) generatePatternName(middleware []string, usages []patternUsage) string {
	parts := []string{}
	paramsParts := []string{}

	for _, m := range middleware {
		// Extract base name and parameters
		baseName := strings.Split(m, "(")[0]
		params := ""
		if idx := strings.Index(m, "("); idx != -1 {
			params = strings.TrimSuffix(m[idx+1:], ")")
		}

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

		// Collect parameter details if VerboseNames is enabled and params exist
		if pe.params.VerboseNames && params != "" {
			// Simplify params for name (e.g., "300" from "cache(300)")
			cleanParams := strings.ReplaceAll(params, "/", "_per_")
			cleanParams = strings.ReplaceAll(cleanParams, " ", "_")
			paramsParts = append(paramsParts, cleanParams)
		}
	}

	parts = append(parts, "handler")

	// Add all parameter details at the end if any exist
	if len(paramsParts) > 0 {
		parts = append(parts, "with_"+strings.Join(paramsParts, "_"))
	}

	// Add operation context if VerboseNames is enabled
	if pe.params.VerboseNames && len(usages) > 0 {
		// Find most common operation
		operationCounts := make(map[string]int)
		for _, usage := range usages {
			operationCounts[usage.operation]++
		}

		maxCount := 0
		commonOperation := ""
		for op, count := range operationCounts {
			if count > maxCount {
				maxCount = count
				commonOperation = op
			}
		}

		if commonOperation != "" {
			parts = append(parts, "for_"+commonOperation)
		}
	}

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
