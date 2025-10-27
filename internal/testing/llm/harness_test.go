package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// TestHarnessIntegration tests the full harness with mock LLM responses.
func TestHarnessIntegration(t *testing.T) {
	config := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "claude-test",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 2,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(config)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	// Use predefined test cases
	testCases := PredefinedTestCases()

	// Mock LLM that returns expected patterns
	mockFn := func(prompt string) string {
		// Parse prompt to determine expected response
		if strings.Contains(prompt, "Comment.create") && strings.Contains(prompt, "authentication") {
			return "@on create: [auth]"
		}
		if strings.Contains(prompt, "Article.list") && strings.Contains(prompt, "caching") {
			return "@on list: [cache(300)]"
		}
		if strings.Contains(prompt, "Post.create") && strings.Contains(prompt, "rate limiting") {
			return "@on create: [auth, rate_limit(10/hour)]"
		}
		return "@on create: [auth]"
	}

	ctx := context.Background()
	report, err := harness.RunWithMockLLM(ctx, testCases, mockFn)
	if err != nil {
		t.Fatalf("Failed to run harness: %v", err)
	}

	// Verify report structure
	if report.Summary.TotalTests != len(testCases) {
		t.Errorf("Expected %d total tests, got %d", len(testCases), report.Summary.TotalTests)
	}

	if report.Summary.PassedTests != len(testCases) {
		t.Errorf("Expected all tests to pass, got %d/%d", report.Summary.PassedTests, report.Summary.TotalTests)
	}

	// Verify per-provider stats exist
	if len(report.Summary.ByProvider) == 0 {
		t.Error("Expected per-provider stats")
	}

	// Verify per-category stats exist
	if len(report.Summary.ByCategory) == 0 {
		t.Error("Expected per-category stats")
	}
}

// TestHarness_TenTestCases tests 10+ different test cases covering various patterns.
func TestHarness_TenTestCases(t *testing.T) {
	config := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "claude-test",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 2,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(config)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	// Create 10+ comprehensive test cases
	testCases := []TestCase{
		// 1. Authentication middleware
		{
			Name:            "Add authentication to Comment.create",
			Category:        "authentication",
			Prompt:          "Add authentication middleware to Comment.create. Return: @on create: [auth]",
			ExpectedPattern: "@on create: [auth]",
			ValidationMode:  "semantic",
		},
		// 2. Caching middleware
		{
			Name:            "Add caching to Article.list",
			Category:        "caching",
			Prompt:          "Add cache middleware to Article.list. Return: @on list: [cache(300)]",
			ExpectedPattern: "@on list: [cache(300)]",
			ValidationMode:  "semantic",
		},
		// 3. Rate limiting
		{
			Name:            "Add rate limiting to Post.create",
			Category:        "rate_limiting",
			Prompt:          "Add rate limiting to Post.create. Return: @on create: [rate_limit(10/hour)]",
			ExpectedPattern: "@on create: [rate_limit(10/hour)]",
			ValidationMode:  "semantic",
		},
		// 4. Auth + rate limiting combo
		{
			Name:            "Add auth and rate limiting to Post.create",
			Category:        "authentication",
			Prompt:          "Add auth and rate limiting to Post.create. Return: @on create: [auth, rate_limit(10/hour)]",
			ExpectedPattern: "@on create: [auth, rate_limit(10/hour)]",
			ValidationMode:  "semantic",
		},
		// 5. Caching on show operation
		{
			Name:            "Add caching to User.show",
			Category:        "caching",
			Prompt:          "Add cache to User.show. Return: @on show: [cache(300)]",
			ExpectedPattern: "@on show: [cache(300)]",
			ValidationMode:  "semantic",
		},
		// 6. Auth on update
		{
			Name:            "Add authentication to Post.update",
			Category:        "authentication",
			Prompt:          "Add auth to Post.update. Return: @on update: [auth]",
			ExpectedPattern: "@on update: [auth]",
			ValidationMode:  "semantic",
		},
		// 7. Auth on delete
		{
			Name:            "Add authentication to Comment.delete",
			Category:        "authentication",
			Prompt:          "Add auth to Comment.delete. Return: @on delete: [auth]",
			ExpectedPattern: "@on delete: [auth]",
			ValidationMode:  "semantic",
		},
		// 8. CORS middleware
		{
			Name:            "Add CORS to API.list",
			Category:        "cors",
			Prompt:          "Add CORS middleware to API.list. Return: @on list: [cors]",
			ExpectedPattern: "@on list: [cors]",
			ValidationMode:  "semantic",
		},
		// 9. Multiple middleware - auth + cache
		{
			Name:            "Add auth and cache to Article.show",
			Category:        "authentication",
			Prompt:          "Add auth and cache to Article.show. Return: @on show: [auth, cache(300)]",
			ExpectedPattern: "@on show: [auth, cache(300)]",
			ValidationMode:  "semantic",
		},
		// 10. Rate limiting with different params
		{
			Name:            "Add rate limiting to Like.create",
			Category:        "rate_limiting",
			Prompt:          "Add rate limiting to Like.create. Return: @on create: [rate_limit(100/hour)]",
			ExpectedPattern: "@on create: [rate_limit(100/hour)]",
			ValidationMode:  "semantic",
		},
		// 11. Three middleware combo
		{
			Name:            "Add auth, cache, and rate limit to Post.list",
			Category:        "authentication",
			Prompt:          "Add auth, cache, and rate limit to Post.list. Return: @on list: [auth, cache(300), rate_limit(100/hour)]",
			ExpectedPattern: "@on list: [auth, cache(300), rate_limit(100/hour)]",
			ValidationMode:  "semantic",
		},
	}

	// Mock that returns the expected pattern
	mockFn := func(prompt string) string {
		// Extract expected pattern from prompt
		startIdx := strings.Index(prompt, "Return:")
		if startIdx != -1 {
			response := strings.TrimSpace(prompt[startIdx+7:])
			return response
		}
		return "@on create: [auth]"
	}

	ctx := context.Background()
	report, err := harness.RunWithMockLLM(ctx, testCases, mockFn)
	if err != nil {
		t.Fatalf("Failed to run harness: %v", err)
	}

	// Verify all tests passed
	if report.Summary.PassedTests != len(testCases) {
		t.Errorf("Expected all tests to pass, got %d/%d passed",
			report.Summary.PassedTests, report.Summary.TotalTests)

		// Print failures for debugging
		for _, result := range report.Results {
			if !result.Validation.Passed {
				t.Logf("Failed: %s - %s", result.TestCase.Name, result.Validation.Message)
			}
		}
	}

	// Verify we tested multiple categories
	if len(report.Summary.ByCategory) < 3 {
		t.Errorf("Expected at least 3 categories, got %d", len(report.Summary.ByCategory))
	}
}

// TestHarness_FailingCases tests that the harness correctly identifies failures.
func TestHarness_FailingCases(t *testing.T) {
	config := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "claude-test",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 1,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(config)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	testCases := []TestCase{
		{
			Name:            "Should fail - wrong middleware",
			Category:        "authentication",
			Prompt:          "Add auth",
			ExpectedPattern: "@on create: [auth]",
			ValidationMode:  "semantic",
		},
	}

	// Mock that returns wrong response
	mockFn := func(prompt string) string {
		return "@on create: [cache]" // Wrong middleware
	}

	ctx := context.Background()
	report, err := harness.RunWithMockLLM(ctx, testCases, mockFn)
	if err != nil {
		t.Fatalf("Failed to run harness: %v", err)
	}

	// Should have failures
	if report.Summary.FailedTests == 0 {
		t.Error("Expected failures but got none")
	}

	if report.Summary.SuccessRate >= 1.0 {
		t.Errorf("Expected success rate < 1.0, got %.2f", report.Summary.SuccessRate)
	}
}

// TestTestCaseGenerator tests the test case generator.
func TestTestCaseGenerator(t *testing.T) {
	generator := NewTestCaseGenerator("Comment", "create")

	pattern := metadata.PatternMetadata{
		Name:        "authenticated_handler",
		Category:    "authentication",
		Description: "Handler with auth middleware",
		Template:    "@on <operation>: [auth]",
		Examples: []metadata.PatternExample{
			{
				Resource: "Post",
				Code:     "@on create: [auth]",
			},
		},
	}

	testCase := generator.GenerateFromPattern(pattern)

	// Verify test case structure
	if testCase.Name == "" {
		t.Error("Test case name should not be empty")
	}

	if testCase.Category != "authentication" {
		t.Errorf("Expected category 'authentication', got '%s'", testCase.Category)
	}

	if testCase.ExpectedPattern != "@on create: [auth]" {
		t.Errorf("Expected pattern '@on create: [auth]', got '%s'", testCase.ExpectedPattern)
	}

	if !strings.Contains(testCase.Prompt, "Comment") {
		t.Error("Prompt should mention the resource name")
	}

	if !strings.Contains(testCase.Prompt, "create") {
		t.Error("Prompt should mention the operation")
	}
}

// TestTestCaseGenerator_MultiplePatterns tests generating from multiple patterns.
func TestTestCaseGenerator_MultiplePatterns(t *testing.T) {
	generator := NewTestCaseGenerator("Post", "create")

	patterns := []metadata.PatternMetadata{
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
	}

	testCase := generator.GenerateFromPatterns(patterns, "Test combined patterns")

	if testCase.Name != "Test combined patterns" {
		t.Errorf("Expected name 'Test combined patterns', got '%s'", testCase.Name)
	}

	expected := "@on create: [auth, rate_limit(10/hour)]"
	if testCase.ExpectedPattern != expected {
		t.Errorf("Expected pattern '%s', got '%s'", expected, testCase.ExpectedPattern)
	}
}

// TestIntrospectionMock tests the introspection mock.
func TestIntrospectionMock(t *testing.T) {
	mock := NewDefaultIntrospectionMock()

	// Test Patterns method
	authPatterns := mock.Patterns("authentication")
	if len(authPatterns) == 0 {
		t.Error("Expected authentication patterns")
	}

	cachePatterns := mock.Patterns("caching")
	if len(cachePatterns) == 0 {
		t.Error("Expected caching patterns")
	}

	// Test AllPatterns method
	allPatterns := mock.AllPatterns()
	if len(allPatterns) < 3 {
		t.Errorf("Expected at least 3 patterns, got %d", len(allPatterns))
	}

	// Test PatternByName method
	pattern, err := mock.PatternByName("authenticated_handler")
	if err != nil {
		t.Errorf("Failed to find pattern by name: %v", err)
	}
	if pattern.Name != "authenticated_handler" {
		t.Errorf("Expected pattern name 'authenticated_handler', got '%s'", pattern.Name)
	}

	// Test Categories method
	categories := mock.Categories()
	if len(categories) < 3 {
		t.Errorf("Expected at least 3 categories, got %d", len(categories))
	}
}

// TestReporter_GenerateReport tests report generation.
func TestReporter_GenerateReport(t *testing.T) {
	reporter := NewReporter()

	results := []TestResult{
		{
			TestCase: TestCase{
				Name:     "Test 1",
				Category: "authentication",
			},
			Provider: "claude:test",
			Validation: ValidationResult{
				Passed:     true,
				Confidence: 1.0,
			},
			Duration: 100 * time.Millisecond,
		},
		{
			TestCase: TestCase{
				Name:     "Test 2",
				Category: "caching",
			},
			Provider: "claude:test",
			Validation: ValidationResult{
				Passed:     false,
				Confidence: 0.5,
			},
			Duration: 150 * time.Millisecond,
		},
	}

	report := reporter.GenerateReport(results, 300*time.Millisecond)

	// Verify summary
	if report.Summary.TotalTests != 2 {
		t.Errorf("Expected 2 total tests, got %d", report.Summary.TotalTests)
	}

	if report.Summary.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", report.Summary.PassedTests)
	}

	if report.Summary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", report.Summary.FailedTests)
	}

	if report.Summary.SuccessRate != 0.5 {
		t.Errorf("Expected success rate 0.5, got %.2f", report.Summary.SuccessRate)
	}

	// Verify provider stats
	if len(report.Summary.ByProvider) == 0 {
		t.Error("Expected provider stats")
	}

	// Verify category stats
	if len(report.Summary.ByCategory) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(report.Summary.ByCategory))
	}
}

// TestReporter_ExportJSON tests JSON export.
func TestReporter_ExportJSON(t *testing.T) {
	reporter := NewReporter()

	results := []TestResult{
		{
			TestCase: TestCase{
				Name:     "Test 1",
				Category: "authentication",
			},
			Provider: "claude:test",
			Validation: ValidationResult{
				Passed: true,
			},
		},
	}

	report := reporter.GenerateReport(results, 100*time.Millisecond)

	json, err := reporter.ExportJSON(report)
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	if !strings.Contains(json, "Test 1") {
		t.Error("JSON should contain test name")
	}

	if !strings.Contains(json, "authentication") {
		t.Error("JSON should contain category")
	}
}

// BenchmarkHarness_MockLLM benchmarks the harness with mock LLM.
func BenchmarkHarness_MockLLM(b *testing.B) {
	config := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "claude-test",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 5,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(config)
	if err != nil {
		b.Fatalf("Failed to create harness: %v", err)
	}

	testCases := PredefinedTestCases()

	mockFn := func(prompt string) string {
		return "@on create: [auth]"
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := harness.RunWithMockLLM(ctx, testCases, mockFn)
		if err != nil {
			b.Fatalf("Failed to run harness: %v", err)
		}
	}
}
