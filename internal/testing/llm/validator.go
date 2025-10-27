package llm

import (
	"fmt"
	"strings"
)

// ValidationResult represents the result of validating generated code against expected patterns.
type ValidationResult struct {
	// Passed indicates whether validation passed.
	Passed bool

	// MatchType indicates the type of match ("exact", "semantic", "none").
	MatchType string

	// Confidence is a score from 0.0 to 1.0 indicating match quality.
	Confidence float64

	// Expected is the expected pattern.
	Expected string

	// Actual is the actual generated code.
	Actual string

	// Differences describes any differences found.
	Differences []string

	// Message provides a human-readable explanation.
	Message string
}

// Validator validates generated code against expected patterns.
type Validator struct {
	parser *Parser
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{
		parser: NewParser(),
	}
}

// Validate validates generated code against an expected pattern.
// It uses the validation mode from the test case to determine the type of matching.
func (v *Validator) Validate(response string, expected string, mode string) ValidationResult {
	switch mode {
	case "exact":
		return v.ExactMatch(response, expected)
	case "semantic":
		return v.SemanticMatch(response, expected)
	default:
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     response,
			Message:    fmt.Sprintf("unknown validation mode: %s", mode),
		}
	}
}

// ExactMatch checks if the generated code exactly matches the expected pattern.
// Whitespace and formatting are normalized before comparison.
func (v *Validator) ExactMatch(response string, expected string) ValidationResult {
	// Extract middleware declaration from response
	parsed, err := v.parser.Parse(response)
	if err != nil {
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     response,
			Message:    fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	if len(parsed.MiddlewareDeclarations) == 0 {
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     response,
			Message:    "no middleware declarations found in response",
		}
	}

	actual := parsed.MiddlewareDeclarations[0].Raw

	// Normalize both strings
	normalizedExpected := NormalizeMiddlewareDeclaration(expected)
	normalizedActual := NormalizeMiddlewareDeclaration(actual)

	if normalizedExpected == normalizedActual {
		return ValidationResult{
			Passed:     true,
			MatchType:  "exact",
			Confidence: 1.0,
			Expected:   expected,
			Actual:     actual,
			Message:    "exact match",
		}
	}

	return ValidationResult{
		Passed:      false,
		MatchType:   "none",
		Confidence:  0.0,
		Expected:    expected,
		Actual:      actual,
		Differences: []string{fmt.Sprintf("expected: %s, got: %s", normalizedExpected, normalizedActual)},
		Message:     "no exact match",
	}
}

// SemanticMatch checks if the generated code has the same semantic meaning as expected.
// This allows for variations in formatting, parameter values, and order (where appropriate).
func (v *Validator) SemanticMatch(response string, expected string) ValidationResult {
	// Extract middleware declarations
	parsed, err := v.parser.Parse(response)
	if err != nil {
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     response,
			Message:    fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	if len(parsed.MiddlewareDeclarations) == 0 {
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     response,
			Message:    "no middleware declarations found in response",
		}
	}

	actualDecl := parsed.MiddlewareDeclarations[0]

	// Parse expected pattern
	expectedParsed, err := v.parser.Parse(expected)
	if err != nil || len(expectedParsed.MiddlewareDeclarations) == 0 {
		return ValidationResult{
			Passed:     false,
			MatchType:  "none",
			Confidence: 0.0,
			Expected:   expected,
			Actual:     actualDecl.Raw,
			Message:    "failed to parse expected pattern",
		}
	}

	expectedDecl := expectedParsed.MiddlewareDeclarations[0]

	// Compare operations
	if actualDecl.Operation != expectedDecl.Operation {
		return ValidationResult{
			Passed:      false,
			MatchType:   "none",
			Confidence:  0.0,
			Expected:    expected,
			Actual:      actualDecl.Raw,
			Differences: []string{fmt.Sprintf("operation mismatch: expected %s, got %s", expectedDecl.Operation, actualDecl.Operation)},
			Message:     "operation mismatch",
		}
	}

	// Compare middleware lists
	result := v.compareMiddlewareLists(expectedDecl.Middleware, actualDecl.Middleware)

	result.Expected = expected
	result.Actual = actualDecl.Raw

	if result.Passed {
		result.MatchType = "semantic"
	}

	return result
}

// compareMiddlewareLists compares two middleware lists for semantic equivalence.
// Middleware order is significant, so lists must match in order.
func (v *Validator) compareMiddlewareLists(expected, actual []string) ValidationResult {
	if len(expected) != len(actual) {
		return ValidationResult{
			Passed:      false,
			Confidence:  0.0,
			Differences: []string{fmt.Sprintf("middleware count mismatch: expected %d, got %d", len(expected), len(actual))},
			Message:     "middleware count mismatch",
		}
	}

	var differences []string
	matchCount := 0

	for i := 0; i < len(expected); i++ {
		if v.middlewareEquals(expected[i], actual[i]) {
			matchCount++
		} else {
			differences = append(differences, fmt.Sprintf(
				"middleware at position %d: expected %s, got %s",
				i, expected[i], actual[i],
			))
		}
	}

	confidence := float64(matchCount) / float64(len(expected))
	passed := matchCount == len(expected)

	message := "semantic match"
	if !passed {
		message = "middleware mismatch"
	}

	return ValidationResult{
		Passed:      passed,
		Confidence:  confidence,
		Differences: differences,
		Message:     message,
	}
}

// middlewareEquals checks if two middleware strings are semantically equivalent.
// It normalizes formatting and compares base names and parameters.
func (v *Validator) middlewareEquals(m1, m2 string) bool {
	// Normalize whitespace
	m1 = strings.TrimSpace(m1)
	m2 = strings.TrimSpace(m2)

	// If exact match, return true
	if m1 == m2 {
		return true
	}

	// Extract base names (before parentheses)
	base1 := strings.Split(m1, "(")[0]
	base2 := strings.Split(m2, "(")[0]

	// Base names must match
	if base1 != base2 {
		return false
	}

	// If both have no parameters, they match
	if !strings.Contains(m1, "(") && !strings.Contains(m2, "(") {
		return true
	}

	// For parameterized middleware, extract and compare parameters
	params1 := v.extractParameters(m1)
	params2 := v.extractParameters(m2)

	// For semantic matching, we allow some flexibility in parameters
	// For now, we require exact parameter match
	return params1 == params2
}

// extractParameters extracts the parameter string from middleware.
// Examples:
//   - "auth" → ""
//   - "cache(300)" → "300"
//   - "rate_limit(10/hour)" → "10/hour"
func (v *Validator) extractParameters(middleware string) string {
	start := strings.Index(middleware, "(")
	if start == -1 {
		return ""
	}

	end := strings.LastIndex(middleware, ")")
	if end == -1 || end <= start {
		return ""
	}

	return strings.TrimSpace(middleware[start+1 : end])
}

// ValidateMultiple validates multiple responses against expected patterns.
// Returns the best match result.
func (v *Validator) ValidateMultiple(responses []string, expected string, mode string) ValidationResult {
	var bestResult ValidationResult
	bestResult.Confidence = 0.0

	for _, response := range responses {
		result := v.Validate(response, expected, mode)
		if result.Confidence > bestResult.Confidence {
			bestResult = result
		}

		// If we found a perfect match, return early
		if result.Passed && result.Confidence == 1.0 {
			return result
		}
	}

	return bestResult
}
