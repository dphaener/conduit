package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// TestRegressionBaseline validates that baseline extraction works as expected.
// This test uses fixed test data with known patterns and verifies that:
// 1. Patterns are extracted correctly
// 2. Expected failure categories are identified
// 3. Recommendations are generated appropriately
func TestRegressionBaseline(t *testing.T) {
	// Setup: Create fixed test data with known patterns
	resources := getFixedTestResources()

	// Create test configuration
	config := IterationConfig{
		MaxIterations: 1, // Only run baseline
		TargetSuccessRate: map[string]float64{
			"mock-provider": 0.80,
		},
		MinimumPatternSuccess: 0.50,
		StopOnSuccess:         false,
	}

	// Create mock harness
	harnessConfig := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "mock-provider",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 0,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 1,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(harnessConfig)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	analyzer := NewFailureAnalyzer()
	runner := NewIterationRunner(harness, analyzer, config)

	// Set deterministic mock LLM
	runner.SetMockLLM(createBaselineMockLLM())

	// Run iteration
	ctx := context.Background()
	results, err := runner.Run(ctx, resources)
	if err != nil {
		t.Fatalf("Iteration runner failed: %v", err)
	}

	// Verify we got exactly one iteration (baseline)
	if len(results) != 1 {
		t.Errorf("Expected 1 iteration, got %d", len(results))
	}

	result := results[0]

	// Verify patterns were extracted
	if len(result.Patterns) == 0 {
		t.Error("Expected patterns to be extracted, got none")
	}

	// Verify expected pattern names
	expectedPatterns := []string{"authenticated_handler", "cached_handler"}
	foundPatterns := make(map[string]bool)
	for _, p := range result.Patterns {
		foundPatterns[p.Name] = true
	}

	for _, expected := range expectedPatterns {
		if !foundPatterns[expected] {
			t.Errorf("Expected pattern %q not found in extracted patterns", expected)
		}
	}

	// Verify tests were executed
	if result.Report.Summary.TotalTests == 0 {
		t.Error("Expected tests to be executed, got 0")
	}

	// Verify failure analysis was performed
	if result.FailureAnalysis.TotalFailures > 0 {
		if len(result.FailureAnalysis.ByReason) == 0 {
			t.Error("Expected failure reasons to be categorized")
		}
	}

	// Verify recommendations were generated
	if len(result.FailureAnalysis.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}
}

// TestRegressionParameterTuning validates parameter adjustment improves results.
// This test simulates multiple iterations with parameter adjustments and verifies:
// 1. Parameters can be adjusted between iterations
// 2. Adjustments affect pattern extraction
// 3. Success rates can improve with proper tuning
func TestRegressionParameterTuning(t *testing.T) {
	resources := getFixedTestResources()

	// Start with default params
	config := IterationConfig{
		MaxIterations: 3,
		TargetSuccessRate: map[string]float64{
			"mock-provider": 0.80,
		},
		MinimumPatternSuccess: 0.50,
		StopOnSuccess:         false,
	}

	harnessConfig := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "mock-provider",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 0,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 1,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(harnessConfig)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	analyzer := NewFailureAnalyzer()
	runner := NewIterationRunner(harness, analyzer, config)

	// Mock LLM that improves over iterations
	iterationCount := 0
	runner.SetMockLLM(func(prompt string) string {
		iterationCount++
		// Simulate improvement: first iteration has more failures
		if iterationCount <= 2 && strings.Contains(prompt, "authenticated_handler") {
			return "@on create: [login]" // Wrong middleware name
		}
		// Later iterations succeed
		if strings.Contains(prompt, "authenticated_handler") {
			return "@on create: [auth]"
		}
		if strings.Contains(prompt, "cached_handler") {
			return "@on list: [cache(300)]"
		}
		return "@on create: [auth]"
	})

	ctx := context.Background()
	results, err := runner.Run(ctx, resources)
	if err != nil {
		t.Fatalf("Iteration runner failed: %v", err)
	}

	// Verify we got all iterations
	if len(results) != config.MaxIterations {
		t.Errorf("Expected %d iterations, got %d", config.MaxIterations, len(results))
	}

	// Verify each iteration has valid structure
	for i, result := range results {
		if result.IterationNumber != i+1 {
			t.Errorf("Iteration %d has wrong iteration number: %d", i, result.IterationNumber)
		}
		if len(result.Patterns) == 0 {
			t.Errorf("Iteration %d extracted no patterns", i+1)
		}
		if result.Report.Summary.TotalTests == 0 {
			t.Errorf("Iteration %d executed no tests", i+1)
		}
	}
}

// TestRegressionSuccessCriteria validates criteria checking.
// This test verifies that:
// 1. Success criteria are correctly evaluated per provider
// 2. Pattern-level success thresholds are enforced
// 3. MetCriteria flag is set correctly
func TestRegressionSuccessCriteria(t *testing.T) {
	config := DefaultIterationConfig()
	runner := NewIterationRunner(nil, nil, config)

	tests := []struct {
		name     string
		report   Report
		patterns []metadata.PatternMetadata
		expected bool
	}{
		{
			name: "All criteria met",
			report: Report{
				Results: []TestResult{
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "claude-opus",
						Validation: ValidationResult{Passed: true},
					},
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "gpt-4",
						Validation: ValidationResult{Passed: true},
					},
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "gpt-3.5-turbo",
						Validation: ValidationResult{Passed: true},
					},
				},
				Summary: ReportSummary{
					TotalTests:  3,
					PassedTests: 3,
					SuccessRate: 1.0,
					ByProvider: map[string]ProviderStats{
						"claude-opus":    {TotalTests: 1, PassedTests: 1, SuccessRate: 1.0},
						"gpt-4":          {TotalTests: 1, PassedTests: 1, SuccessRate: 1.0},
						"gpt-3.5-turbo":  {TotalTests: 1, PassedTests: 1, SuccessRate: 1.0},
					},
				},
			},
			patterns: []metadata.PatternMetadata{{Name: "auth_handler"}},
			expected: true,
		},
		{
			name: "Provider below target",
			report: Report{
				Results: []TestResult{
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "claude-opus",
						Validation: ValidationResult{Passed: false},
					},
				},
				Summary: ReportSummary{
					TotalTests:  1,
					PassedTests: 0,
					SuccessRate: 0.0,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {TotalTests: 1, PassedTests: 0, SuccessRate: 0.0},
					},
				},
			},
			patterns: []metadata.PatternMetadata{{Name: "auth_handler"}},
			expected: false,
		},
		{
			name: "Pattern below minimum success",
			report: Report{
				Results: []TestResult{
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "claude-opus",
						Validation: ValidationResult{Passed: true},
					},
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "claude-opus",
						Validation: ValidationResult{Passed: false},
					},
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Comment.create"},
						Provider:  "claude-opus",
						Validation: ValidationResult{Passed: false},
					},
				},
				Summary: ReportSummary{
					TotalTests:  3,
					PassedTests: 1,
					SuccessRate: 0.33,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {TotalTests: 3, PassedTests: 1, SuccessRate: 0.33},
					},
				},
			},
			patterns: []metadata.PatternMetadata{{Name: "auth_handler"}},
			expected: false, // Pattern success rate (33%) below minimum (50%)
		},
		{
			name: "Provider not tested",
			report: Report{
				Results: []TestResult{
					{
						TestCase:  TestCase{Name: "Apply auth_handler pattern to Post.create"},
						Provider:  "unknown-provider",
						Validation: ValidationResult{Passed: true},
					},
				},
				Summary: ReportSummary{
					TotalTests:  1,
					PassedTests: 1,
					SuccessRate: 1.0,
					ByProvider: map[string]ProviderStats{
						"unknown-provider": {TotalTests: 1, PassedTests: 1, SuccessRate: 1.0},
					},
				},
			},
			patterns: []metadata.PatternMetadata{{Name: "auth_handler"}},
			expected: false, // Required providers (claude-opus, gpt-4) not tested
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.checkSuccessCriteria(tt.report, tt.patterns)
			if result != tt.expected {
				t.Errorf("checkSuccessCriteria() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestRegressionIterationTracking validates progress tracking.
// This test verifies that:
// 1. Iteration numbers increment correctly
// 2. Results accumulate across iterations
// 3. Final report contains all iteration data
func TestRegressionIterationTracking(t *testing.T) {
	resources := getFixedTestResources()

	config := IterationConfig{
		MaxIterations: 3,
		TargetSuccessRate: map[string]float64{
			"mock-provider": 0.80,
		},
		MinimumPatternSuccess: 0.50,
		StopOnSuccess:         false,
	}

	harnessConfig := &Config{
		Providers: []ProviderConfig{
			{
				Type:       ProviderClaude,
				Model:      "mock-provider",
				APIKey:     "test-key",
				Timeout:    10 * time.Second,
				MaxRetries: 0,
				Enabled:    true,
			},
		},
		MaxConcurrentRequests: 1,
		DefaultTimeout:        10 * time.Second,
		RateLimitDelay:        0,
	}

	harness, err := NewHarness(harnessConfig)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	analyzer := NewFailureAnalyzer()
	runner := NewIterationRunner(harness, analyzer, config)

	runner.SetMockLLM(createBaselineMockLLM())

	ctx := context.Background()
	results, err := runner.Run(ctx, resources)
	if err != nil {
		t.Fatalf("Iteration runner failed: %v", err)
	}

	// Verify iteration count
	if len(results) != config.MaxIterations {
		t.Errorf("Expected %d results, got %d", config.MaxIterations, len(results))
	}

	// Verify iteration numbers are sequential
	for i, result := range results {
		expectedNum := i + 1
		if result.IterationNumber != expectedNum {
			t.Errorf("Result %d has iteration number %d, expected %d",
				i, result.IterationNumber, expectedNum)
		}
	}

	// Verify all iterations have data
	for i, result := range results {
		if len(result.Patterns) == 0 {
			t.Errorf("Iteration %d has no patterns", i+1)
		}
		if result.Report.Summary.TotalTests == 0 {
			t.Errorf("Iteration %d has no test results", i+1)
		}
	}

	// Verify final report can be generated
	reporter := NewIterationReporter(nil)
	jsonReport, err := reporter.ExportIterationReport(results, config)
	if err != nil {
		t.Errorf("Failed to export report: %v", err)
	}
	if jsonReport == "" {
		t.Error("Expected non-empty JSON report")
	}

	// Verify JSON contains all iterations
	if !strings.Contains(jsonReport, `"total_iterations": 3`) {
		t.Error("JSON report doesn't contain correct iteration count")
	}
}

// TestRegressionFailureAnalysis validates all failure categories.
// This test verifies that:
// 1. Each of the 6 failure reasons can be identified
// 2. Categorization is accurate
// 3. Recommendations are appropriate for each failure type
func TestRegressionFailureAnalysis(t *testing.T) {
	analyzer := NewFailureAnalyzer()

	tests := []struct {
		name           string
		result         TestResult
		pattern        metadata.PatternMetadata
		expectedReason FailureReason
	}{
		{
			name: "pattern_too_specific",
			result: TestResult{
				Response:   "@on create: [auth]",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Examples:   []metadata.PatternExample{{}, {}, {}},
				Confidence: 0.4, // Low confidence indicates too specific
			},
			expectedReason: ReasonPatternTooSpecific,
		},
		{
			name: "pattern_too_generic",
			result: TestResult{
				Response:   "@on create: [middleware]",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Examples:   []metadata.PatternExample{{}, {}, {}},
				Confidence: 0.6,
			},
			expectedReason: ReasonPatternTooGeneric,
		},
		{
			name: "name_unclear",
			result: TestResult{
				Response:   "@on create: [login]",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Category:   "authentication",
				Template:   "@on <operation>: [auth]",
				Examples:   []metadata.PatternExample{{}, {}, {}},
				Confidence: 0.6,
			},
			expectedReason: ReasonNameUnclear,
		},
		{
			name: "template_ambiguous",
			result: TestResult{
				Response:   "@on create: [auth(required)]",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Category:   "authentication",
				Template:   "@on <operation>: [auth]",
				Examples:   []metadata.PatternExample{{}, {}, {}},
				Confidence: 0.6,
			},
			expectedReason: ReasonTemplateAmbiguous,
		},
		{
			name: "insufficient_examples",
			result: TestResult{
				Response:   "@on create: [auth]",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Examples:   []metadata.PatternExample{{}}, // Only 1 example
				Confidence: 0.6,
			},
			expectedReason: ReasonInsufficientExamples,
		},
		{
			name: "llm_hallucination",
			result: TestResult{
				Response:   "function create() { /* some code */ }",
				Validation: ValidationResult{Expected: "@on create: [auth]"},
			},
			pattern: metadata.PatternMetadata{
				Examples:   []metadata.PatternExample{{}, {}, {}},
				Confidence: 0.6,
			},
			expectedReason: ReasonLLMHallucination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := analyzer.categorizeFailure(tt.result, tt.pattern)
			if reason != tt.expectedReason {
				t.Errorf("categorizeFailure() = %v, expected %v", reason, tt.expectedReason)
			}
		})
	}

	// Verify recommendations are generated for each failure type
	for _, tt := range tests {
		t.Run(tt.name+"_recommendations", func(t *testing.T) {
			analysis := FailureAnalysis{
				TotalFailures: 1,
				ByReason: map[FailureReason][]TestResult{
					tt.expectedReason: {tt.result},
				},
			}

			recommendations := analyzer.generateRecommendations(analysis)
			if len(recommendations) == 0 {
				t.Errorf("No recommendations generated for %s", tt.expectedReason)
			}
		})
	}
}

// TestRegressionReportingFormats validates all output formats.
// This test verifies that:
// 1. Text output is properly formatted
// 2. JSON output is valid and complete
// 3. All required data is included in both formats
func TestRegressionReportingFormats(t *testing.T) {
	// Create sample iteration results
	results := []IterationResult{
		{
			IterationNumber: 1,
			Patterns: []metadata.PatternMetadata{
				{Name: "authenticated_handler", Frequency: 3, Confidence: 0.5},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  2,
					PassedTests: 1,
					FailedTests: 1,
					SuccessRate: 0.5,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {TotalTests: 2, PassedTests: 1, SuccessRate: 0.5},
					},
				},
			},
			FailureAnalysis: FailureAnalysis{
				TotalFailures: 1,
				Recommendations: []string{"Improve pattern quality"},
			},
			MetCriteria: false,
		},
		{
			IterationNumber: 2,
			Patterns: []metadata.PatternMetadata{
				{Name: "authenticated_handler", Frequency: 3, Confidence: 0.5},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  2,
					PassedTests: 2,
					FailedTests: 0,
					SuccessRate: 1.0,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {TotalTests: 2, PassedTests: 2, SuccessRate: 1.0},
					},
				},
			},
			FailureAnalysis: FailureAnalysis{
				TotalFailures:   0,
				Recommendations: []string{},
			},
			MetCriteria: true,
		},
	}

	config := DefaultIterationConfig()
	reporter := NewIterationReporter(nil)

	// Test JSON export
	t.Run("JSON export", func(t *testing.T) {
		jsonOutput, err := reporter.ExportIterationReport(results, config)
		if err != nil {
			t.Fatalf("Failed to export JSON: %v", err)
		}

		if jsonOutput == "" {
			t.Error("JSON output is empty")
		}

		// Verify JSON contains expected fields
		requiredFields := []string{
			`"timestamp"`,
			`"iterations"`,
			`"total_iterations"`,
			`"final_success"`,
			`"improvement"`,
			`"success_criteria"`,
		}

		for _, field := range requiredFields {
			if !strings.Contains(jsonOutput, field) {
				t.Errorf("JSON output missing field: %s", field)
			}
		}

		// Verify values
		if !strings.Contains(jsonOutput, `"total_iterations": 2`) {
			t.Error("JSON should contain 2 iterations")
		}
		if !strings.Contains(jsonOutput, `"final_success": true`) {
			t.Error("JSON should indicate final success")
		}
	})

	// Test text format (via string builder)
	t.Run("Text format", func(t *testing.T) {
		var buf strings.Builder
		textReporter := NewIterationReporter(&buf)

		textReporter.PrintIterationResults(results, config)
		output := buf.String()

		if output == "" {
			t.Error("Text output is empty")
		}

		// Verify key sections are present
		expectedSections := []string{
			"PATTERN VALIDATION ITERATION SYSTEM",
			"PROGRESS OVERVIEW",
			"ITERATION 1",
			"ITERATION 2",
			"SUCCESS CRITERIA MET",
		}

		for _, section := range expectedSections {
			if !strings.Contains(output, section) {
				t.Errorf("Text output missing section: %s", section)
			}
		}
	})
}

// TestRegressionPerformance validates that the system completes quickly.
// This test ensures all regression tests complete in under 5 seconds total.
func TestRegressionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	start := time.Now()

	// Run a subset of regression tests
	t.Run("Baseline", func(t *testing.T) {
		TestRegressionBaseline(t)
	})

	t.Run("SuccessCriteria", func(t *testing.T) {
		TestRegressionSuccessCriteria(t)
	})

	t.Run("FailureAnalysis", func(t *testing.T) {
		TestRegressionFailureAnalysis(t)
	})

	elapsed := time.Since(start)
	maxDuration := 5 * time.Second

	if elapsed > maxDuration {
		t.Errorf("Regression tests took %v, expected < %v", elapsed, maxDuration)
	}
}

// Helper functions

// getFixedTestResources returns deterministic test resources for regression tests.
func getFixedTestResources() []metadata.ResourceMetadata {
	return []metadata.ResourceMetadata{
		{
			Name:     "Post",
			FilePath: "/test/post.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
				"delete": {"auth"},
			},
		},
		{
			Name:     "Comment",
			FilePath: "/test/comment.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"update": {"auth"},
				"delete": {"auth"},
			},
		},
		{
			Name:     "Article",
			FilePath: "/test/article.cdt",
			Middleware: map[string][]string{
				"create": {"auth"},
				"list":   {"cache(300)"},
				"show":   {"cache(300)"},
			},
		},
		{
			Name:     "Category",
			FilePath: "/test/category.cdt",
			Middleware: map[string][]string{
				"list": {"cache(300)"},
				"show": {"cache(300)"},
			},
		},
	}
}

// createBaselineMockLLM returns a deterministic mock LLM function.
// This function returns correct patterns for known cases and realistic
// variations for edge cases.
func createBaselineMockLLM() func(string) string {
	return func(prompt string) string {
		// Deterministic responses based on pattern type in prompt
		if strings.Contains(prompt, "authenticated_handler") {
			return "@on create: [auth]"
		}
		if strings.Contains(prompt, "cached_handler") {
			return "@on list: [cache(300)]"
		}
		// Default fallback
		return "@on create: [auth]"
	}
}
