package llm

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// FailureAnalyzer categorizes and analyzes why LLMs failed to use patterns correctly.
// It provides insights into pattern quality and suggests improvements.
type FailureAnalyzer struct{}

// NewFailureAnalyzer creates a new failure analyzer.
func NewFailureAnalyzer() *FailureAnalyzer {
	return &FailureAnalyzer{}
}

// FailureAnalysis contains detailed analysis of test failures.
type FailureAnalysis struct {
	// TotalFailures is the total number of failed tests.
	TotalFailures int

	// ByReason groups failures by their categorized reason.
	ByReason map[FailureReason][]TestResult

	// ByPattern groups failures by the pattern being tested.
	ByPattern map[string]PatternFailure

	// Recommendations are suggested improvements based on the analysis.
	Recommendations []string
}

// FailureReason categorizes why a test failed.
type FailureReason string

const (
	// ReasonInfrastructureError means there was an infrastructure-level error (network, timeout, API key, etc.).
	// This is not a pattern quality issue but an environmental problem.
	ReasonInfrastructureError FailureReason = "infrastructure_error"

	// ReasonPatternTooSpecific means the pattern is too specific and not general enough.
	// This happens when the pattern has very low frequency (< minFrequency threshold).
	ReasonPatternTooSpecific FailureReason = "pattern_too_specific"

	// ReasonPatternTooGeneric means the pattern is too vague or generic.
	// This happens when the pattern confidence is very low.
	ReasonPatternTooGeneric FailureReason = "pattern_too_generic"

	// ReasonNameUnclear means the pattern name doesn't clearly convey its purpose.
	// This happens when the LLM uses different middleware names or approaches.
	ReasonNameUnclear FailureReason = "name_unclear"

	// ReasonTemplateAmbiguous means the pattern template is ambiguous or unclear.
	// This happens when the LLM uses the correct concept but wrong syntax.
	ReasonTemplateAmbiguous FailureReason = "template_ambiguous"

	// ReasonInsufficientExamples means the pattern doesn't have enough examples.
	// This happens when the pattern has < 3 examples.
	ReasonInsufficientExamples FailureReason = "insufficient_examples"

	// ReasonLLMHallucination means the LLM generated completely unrelated code.
	// This happens when the response doesn't match any expected patterns.
	ReasonLLMHallucination FailureReason = "llm_hallucination"
)

// PatternFailure contains failure statistics for a single pattern.
type PatternFailure struct {
	// PatternName is the name of the pattern.
	PatternName string

	// FailureCount is the number of times this pattern failed.
	FailureCount int

	// SuccessRate is the success rate for this pattern (0.0-1.0).
	SuccessRate float64

	// Reason is the primary reason for failures.
	Reason FailureReason
}

// Analyze analyzes test results and categorizes failures.
func (fa *FailureAnalyzer) Analyze(report Report, patterns []metadata.PatternMetadata) FailureAnalysis {
	analysis := FailureAnalysis{
		ByReason:  make(map[FailureReason][]TestResult),
		ByPattern: make(map[string]PatternFailure),
	}

	// Build a map of pattern names to pattern metadata for quick lookup
	patternMap := make(map[string]metadata.PatternMetadata)
	for _, pattern := range patterns {
		patternMap[pattern.Name] = pattern
	}

	// Track pattern statistics
	patternStats := make(map[string]*patternStatsTracker)

	// Analyze each test result
	for _, result := range report.Results {
		// Skip passed tests
		if result.Error == nil && result.Validation.Passed {
			// Track success for pattern stats
			patternName := extractPatternName(result.TestCase.Name)
			if patternName != "" {
				if _, exists := patternStats[patternName]; !exists {
					patternStats[patternName] = &patternStatsTracker{}
				}
				patternStats[patternName].total++
				patternStats[patternName].passed++
			}
			continue
		}

		// This is a failure
		analysis.TotalFailures++

		// Extract pattern name from test case
		patternName := extractPatternName(result.TestCase.Name)
		if patternName == "" {
			continue
		}

		// Track failure for pattern stats
		if _, exists := patternStats[patternName]; !exists {
			patternStats[patternName] = &patternStatsTracker{}
		}
		patternStats[patternName].total++

		// Get the pattern metadata
		pattern, exists := patternMap[patternName]
		if !exists {
			continue
		}

		// Categorize the failure
		reason := fa.categorizeFailure(result, pattern)

		// Add to reason map
		analysis.ByReason[reason] = append(analysis.ByReason[reason], result)

		// Track reason for this pattern
		patternStats[patternName].reasons = append(patternStats[patternName].reasons, reason)
	}

	// Build pattern failure summary
	for patternName, stats := range patternStats {
		successRate := 0.0
		if stats.total > 0 {
			successRate = float64(stats.passed) / float64(stats.total)
		}

		failureCount := stats.total - stats.passed

		// Determine primary reason
		primaryReason := fa.determinePrimaryReason(stats.reasons)

		analysis.ByPattern[patternName] = PatternFailure{
			PatternName:  patternName,
			FailureCount: failureCount,
			SuccessRate:  successRate,
			Reason:       primaryReason,
		}
	}

	// Generate recommendations
	analysis.Recommendations = fa.generateRecommendations(analysis)

	return analysis
}

// patternStatsTracker tracks statistics for a pattern during analysis.
type patternStatsTracker struct {
	total   int
	passed  int
	reasons []FailureReason
}

// isInfrastructureError checks if an error is infrastructure-related (network, timeout, API, etc.).
func isInfrastructureError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "api key") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "unavailable") ||
		strings.Contains(errStr, "service")
}

// categorizeFailure determines why a specific test failed.
func (fa *FailureAnalyzer) categorizeFailure(result TestResult, pattern metadata.PatternMetadata) FailureReason {
	// If there was an error (not just validation failure), check if it's infrastructure-related
	if result.Error != nil {
		if isInfrastructureError(result.Error) {
			return ReasonInfrastructureError
		}
		return ReasonLLMHallucination
	}

	// Check pattern quality issues
	if len(pattern.Examples) < 3 {
		return ReasonInsufficientExamples
	}

	if pattern.Confidence < 0.5 {
		return ReasonPatternTooSpecific
	}

	// Analyze the actual response vs expected
	response := normalizeCode(result.Response)
	expected := normalizeCode(result.Validation.Expected)

	// Check if the LLM used a completely different approach
	if !containsMiddlewareDeclaration(response) {
		return ReasonLLMHallucination
	}

	// Check if the LLM used the right middleware but wrong syntax
	// This should be checked before checking for similar concept
	if containsMiddlewareFromPattern(response, pattern.Template) && !matchesPattern(response, expected) {
		return ReasonTemplateAmbiguous
	}

	// Check if the LLM used the right concept but wrong middleware names
	if containsSimilarConcept(response, pattern.Category) && !matchesPattern(response, expected) {
		return ReasonNameUnclear
	}

	// Default to generic pattern issue
	return ReasonPatternTooGeneric
}

// determinePrimaryReason finds the most common failure reason.
func (fa *FailureAnalyzer) determinePrimaryReason(reasons []FailureReason) FailureReason {
	if len(reasons) == 0 {
		return ReasonPatternTooGeneric
	}

	// Count occurrences of each reason
	counts := make(map[FailureReason]int)
	for _, reason := range reasons {
		counts[reason]++
	}

	// Find the most common
	maxCount := 0
	var primaryReason FailureReason
	for reason, count := range counts {
		if count > maxCount {
			maxCount = count
			primaryReason = reason
		}
	}

	return primaryReason
}

// generateRecommendations creates actionable recommendations based on failure analysis.
func (fa *FailureAnalyzer) generateRecommendations(analysis FailureAnalysis) []string {
	var recommendations []string

	// Recommendation based on failure reasons
	for reason, results := range analysis.ByReason {
		count := len(results)
		if count == 0 {
			continue
		}

		switch reason {
		case ReasonInfrastructureError:
			recommendations = append(recommendations,
				fmt.Sprintf("Fix infrastructure issues (network, API keys, timeouts) - %d errors detected", count))

		case ReasonPatternTooSpecific:
			recommendations = append(recommendations,
				fmt.Sprintf("Reduce minimum pattern frequency threshold (found %d patterns too specific)", count))

		case ReasonPatternTooGeneric:
			recommendations = append(recommendations,
				fmt.Sprintf("Increase minimum pattern frequency to filter vague patterns (%d cases)", count))

		case ReasonNameUnclear:
			recommendations = append(recommendations,
				fmt.Sprintf("Improve pattern naming to be more descriptive (%d cases)", count))

		case ReasonTemplateAmbiguous:
			recommendations = append(recommendations,
				fmt.Sprintf("Clarify pattern templates with more explicit syntax (%d cases)", count))

		case ReasonInsufficientExamples:
			recommendations = append(recommendations,
				fmt.Sprintf("Increase minimum examples required per pattern (%d cases)", count))

		case ReasonLLMHallucination:
			recommendations = append(recommendations,
				fmt.Sprintf("Review prompts for clarity and context (%d hallucinations detected)", count))
		}
	}

	// Pattern-specific recommendations
	for patternName, failure := range analysis.ByPattern {
		if failure.SuccessRate < 0.3 && failure.FailureCount > 2 {
			recommendations = append(recommendations,
				fmt.Sprintf("Review pattern '%s' (%.1f%% success rate) - consider removing or improving",
					patternName, failure.SuccessRate*100))
		}
	}

	// If no specific recommendations, provide general guidance
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Continue iterating to improve pattern quality and LLM prompts")
	}

	return recommendations
}

// normalizeCode normalizes code for comparison by removing extra whitespace.
func normalizeCode(code string) string {
	// Remove leading/trailing whitespace
	code = strings.TrimSpace(code)

	// Replace multiple spaces with single space
	normalized := strings.Builder{}
	prevSpace := false

	for _, c := range code {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				normalized.WriteRune(' ')
				prevSpace = true
			}
		} else {
			normalized.WriteRune(c)
			prevSpace = false
		}
	}

	return normalized.String()
}

// containsMiddlewareDeclaration checks if the code contains a middleware declaration.
func containsMiddlewareDeclaration(code string) bool {
	// Look for "@on <operation>: [...]" pattern
	return strings.Contains(code, "@on") && strings.Contains(code, ":")
}

// containsSimilarConcept checks if the code contains middleware related to the category.
func containsSimilarConcept(code, category string) bool {
	code = strings.ToLower(code)

	switch category {
	case "authentication":
		return strings.Contains(code, "auth") || strings.Contains(code, "login") || strings.Contains(code, "user")
	case "caching":
		return strings.Contains(code, "cache") || strings.Contains(code, "memo")
	case "rate_limiting":
		return strings.Contains(code, "rate") || strings.Contains(code, "limit") || strings.Contains(code, "throttle")
	case "cors":
		return strings.Contains(code, "cors") || strings.Contains(code, "origin")
	default:
		return false
	}
}

// containsMiddlewareFromPattern checks if the code contains middleware names from the pattern.
func containsMiddlewareFromPattern(code, template string) bool {
	// Extract middleware names from template
	// Template format: "@on <operation>: [middleware1, middleware2, ...]"
	start := strings.Index(template, "[")
	end := strings.Index(template, "]")

	if start == -1 || end == -1 || end <= start {
		return false
	}

	middlewareStr := template[start+1 : end]
	middlewares := strings.Split(middlewareStr, ",")

	// Check if any middleware name appears in the code
	for _, middleware := range middlewares {
		middleware = strings.TrimSpace(middleware)
		// Extract base name (before parentheses if any)
		if idx := strings.Index(middleware, "("); idx != -1 {
			middleware = middleware[:idx]
		}

		if strings.Contains(code, middleware) {
			return true
		}
	}

	return false
}

// matchesPattern checks if the code matches the expected pattern.
func matchesPattern(code, expected string) bool {
	// Simple check: do they match after normalization?
	return code == expected
}
