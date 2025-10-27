package llm

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// TestCase represents a single test case for validating LLM code generation.
type TestCase struct {
	// Name is a descriptive name for the test case.
	Name string

	// Category is the pattern category being tested (e.g., "authentication", "caching").
	Category string

	// Prompt is the prompt to send to the LLM.
	Prompt string

	// IntrospectionPatterns are the patterns available via introspection API.
	IntrospectionPatterns []metadata.PatternMetadata

	// ExpectedPattern is the pattern the LLM should generate.
	ExpectedPattern string

	// ValidationMode determines how to validate the generated code.
	// Options: "exact" (exact string match), "semantic" (semantic equivalence).
	ValidationMode string
}

// TestCaseGenerator generates test cases from pattern metadata.
type TestCaseGenerator struct {
	// ResourceName is the name of the resource to generate code for.
	ResourceName string

	// Operation is the operation to add middleware to (e.g., "create", "update").
	Operation string
}

// NewTestCaseGenerator creates a new test case generator.
func NewTestCaseGenerator(resourceName, operation string) *TestCaseGenerator {
	return &TestCaseGenerator{
		ResourceName: resourceName,
		Operation:    operation,
	}
}

// GenerateFromPattern generates a test case from a pattern metadata.
// It creates a prompt that instructs the LLM to use introspection to apply the pattern.
func (g *TestCaseGenerator) GenerateFromPattern(pattern metadata.PatternMetadata) TestCase {
	// Generate prompt
	prompt := g.generatePrompt(pattern)

	// Generate expected output
	expected := g.generateExpectedOutput(pattern)

	return TestCase{
		Name:                  fmt.Sprintf("Apply %s pattern to %s.%s", pattern.Name, g.ResourceName, g.Operation),
		Category:              pattern.Category,
		Prompt:                prompt,
		IntrospectionPatterns: []metadata.PatternMetadata{pattern},
		ExpectedPattern:       expected,
		ValidationMode:        "semantic", // Default to semantic matching
	}
}

// GenerateFromPatterns generates a test case from multiple patterns.
// This is useful for testing complex multi-middleware scenarios.
func (g *TestCaseGenerator) GenerateFromPatterns(patterns []metadata.PatternMetadata, name string) TestCase {
	if len(patterns) == 0 {
		return TestCase{}
	}

	// Use the category of the first pattern
	category := patterns[0].Category

	// Generate combined prompt
	prompt := g.generateMultiPatternPrompt(patterns)

	// Generate expected output
	expected := g.generateMultiPatternExpected(patterns)

	return TestCase{
		Name:                  name,
		Category:              category,
		Prompt:                prompt,
		IntrospectionPatterns: patterns,
		ExpectedPattern:       expected,
		ValidationMode:        "semantic",
	}
}

// generatePrompt creates a prompt for the LLM based on a pattern.
func (g *TestCaseGenerator) generatePrompt(pattern metadata.PatternMetadata) string {
	var sb strings.Builder

	sb.WriteString("You are a developer working with the Conduit programming language.\n\n")
	sb.WriteString(fmt.Sprintf("Task: Add middleware to the %s resource's %s operation.\n\n", g.ResourceName, g.Operation))

	// Describe the pattern category
	sb.WriteString(fmt.Sprintf("The middleware should implement %s functionality.\n\n", pattern.Category))

	// Provide introspection query example
	sb.WriteString("You have access to an introspection API that provides common patterns.\n")
	sb.WriteString(fmt.Sprintf("Query patterns with: registry.Patterns('%s')\n\n", pattern.Category))

	// Show the pattern
	sb.WriteString("Available pattern:\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", pattern.Name))
	sb.WriteString(fmt.Sprintf("Template: %s\n", pattern.Template))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", pattern.Description))

	// Show examples if available
	if len(pattern.Examples) > 0 {
		sb.WriteString("Examples from the codebase:\n")
		for i, example := range pattern.Examples {
			if i >= 2 {
				break // Limit to 2 examples
			}
			sb.WriteString(fmt.Sprintf("- %s: %s\n", example.Resource, example.Code))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Generate the middleware declaration for the ")
	sb.WriteString(fmt.Sprintf("%s resource's %s operation using this pattern.\n", g.ResourceName, g.Operation))
	sb.WriteString("Return only the @on declaration, nothing else.")

	return sb.String()
}

// generateMultiPatternPrompt creates a prompt for multiple patterns.
func (g *TestCaseGenerator) generateMultiPatternPrompt(patterns []metadata.PatternMetadata) string {
	var sb strings.Builder

	sb.WriteString("You are a developer working with the Conduit programming language.\n\n")
	sb.WriteString(fmt.Sprintf("Task: Add middleware to the %s resource's %s operation.\n\n", g.ResourceName, g.Operation))

	// List all categories
	categories := make([]string, len(patterns))
	for i, p := range patterns {
		categories[i] = p.Category
	}
	sb.WriteString(fmt.Sprintf("The middleware should implement: %s.\n\n", strings.Join(categories, ", ")))

	sb.WriteString("You have access to an introspection API that provides common patterns.\n\n")

	// Show each pattern
	sb.WriteString("Available patterns:\n\n")
	for _, pattern := range patterns {
		sb.WriteString(fmt.Sprintf("Pattern: %s\n", pattern.Name))
		sb.WriteString(fmt.Sprintf("Category: %s\n", pattern.Category))
		sb.WriteString(fmt.Sprintf("Template: %s\n", pattern.Template))
		sb.WriteString(fmt.Sprintf("Description: %s\n\n", pattern.Description))
	}

	sb.WriteString("Generate the middleware declaration for the ")
	sb.WriteString(fmt.Sprintf("%s resource's %s operation, combining these patterns.\n", g.ResourceName, g.Operation))
	sb.WriteString("Return only the @on declaration, nothing else.")

	return sb.String()
}

// generateExpectedOutput creates the expected middleware declaration from a pattern.
func (g *TestCaseGenerator) generateExpectedOutput(pattern metadata.PatternMetadata) string {
	// Extract middleware list from template
	// Template format: "@on <operation>: [middleware1, middleware2, ...]"
	template := pattern.Template

	// Replace <operation> with actual operation
	expected := strings.Replace(template, "<operation>", g.Operation, 1)

	return expected
}

// generateMultiPatternExpected creates expected output for multiple patterns.
func (g *TestCaseGenerator) generateMultiPatternExpected(patterns []metadata.PatternMetadata) string {
	// Collect all middleware from all patterns
	var allMiddleware []string

	for _, pattern := range patterns {
		// Extract middleware from template
		// Template format: "@on <operation>: [middleware1, middleware2, ...]"
		template := pattern.Template
		start := strings.Index(template, "[")
		end := strings.Index(template, "]")

		if start != -1 && end != -1 {
			middlewareStr := template[start+1 : end]
			middleware := strings.Split(middlewareStr, ",")
			for _, m := range middleware {
				allMiddleware = append(allMiddleware, strings.TrimSpace(m))
			}
		}
	}

	// Generate expected declaration
	return fmt.Sprintf("@on %s: [%s]", g.Operation, strings.Join(allMiddleware, ", "))
}

// PredefinedTestCases returns a set of common test cases for validation.
func PredefinedTestCases() []TestCase {
	return []TestCase{
		{
			Name:     "Add authentication to Comment.create",
			Category: "authentication",
			Prompt: `You are a developer working with Conduit.
Task: Add authentication middleware to Comment.create.

Available pattern:
Name: authenticated_handler
Template: @on <operation>: [auth]
Description: Handler with auth middleware

Generate the middleware declaration. Return only the @on declaration.`,
			IntrospectionPatterns: []metadata.PatternMetadata{
				{
					Name:        "authenticated_handler",
					Category:    "authentication",
					Description: "Handler with auth middleware",
					Template:    "@on <operation>: [auth]",
				},
			},
			ExpectedPattern: "@on create: [auth]",
			ValidationMode:  "semantic",
		},
		{
			Name:     "Add caching to Article.list",
			Category: "caching",
			Prompt: `You are a developer working with Conduit.
Task: Add caching middleware to Article.list.

Available pattern:
Name: cached_handler
Template: @on <operation>: [cache(300)]
Description: Handler with cache middleware

Generate the middleware declaration. Return only the @on declaration.`,
			IntrospectionPatterns: []metadata.PatternMetadata{
				{
					Name:        "cached_handler",
					Category:    "caching",
					Description: "Handler with cache middleware",
					Template:    "@on <operation>: [cache(300)]",
				},
			},
			ExpectedPattern: "@on list: [cache(300)]",
			ValidationMode:  "semantic",
		},
		{
			Name:     "Add auth + rate limiting to Post.create",
			Category: "authentication",
			Prompt: `You are a developer working with Conduit.
Task: Add authentication and rate limiting to Post.create.

Available patterns:

Pattern: authenticated_handler
Template: @on <operation>: [auth]

Pattern: rate_limited_handler
Template: @on <operation>: [rate_limit(10/hour)]

Combine these patterns. Return only the @on declaration.`,
			IntrospectionPatterns: []metadata.PatternMetadata{
				{
					Name:        "authenticated_handler",
					Category:    "authentication",
					Template:    "@on <operation>: [auth]",
				},
				{
					Name:        "rate_limited_handler",
					Category:    "rate_limiting",
					Template:    "@on <operation>: [rate_limit(10/hour)]",
				},
			},
			ExpectedPattern: "@on create: [auth, rate_limit(10/hour)]",
			ValidationMode:  "semantic",
		},
	}
}
