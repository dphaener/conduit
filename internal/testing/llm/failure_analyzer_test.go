package llm

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestFailureAnalyzer_Analyze(t *testing.T) {
	analyzer := NewFailureAnalyzer()

	patterns := []metadata.PatternMetadata{
		{
			Name:     "authenticated_handler",
			Category: "authentication",
			Template: "@on <operation>: [auth]",
			Examples: []metadata.PatternExample{
				{Resource: "Post", Code: "@on create: [auth]"},
				{Resource: "Comment", Code: "@on create: [auth]"},
				{Resource: "Article", Code: "@on create: [auth]"},
			},
			Frequency:  3,
			Confidence: 0.6,
		},
		{
			Name:     "cached_handler",
			Category: "caching",
			Template: "@on <operation>: [cache(300)]",
			Examples: []metadata.PatternExample{
				{Resource: "Article", Code: "@on list: [cache(300)]"},
			},
			Frequency:  1,
			Confidence: 0.2,
		},
	}

	report := Report{
		Results: []TestResult{
			// Success case
			{
				TestCase: TestCase{
					Name:     "Apply authenticated_handler pattern to Post.create",
					Category: "authentication",
				},
				Provider: "test-provider",
				Response: "@on create: [auth]",
				Validation: ValidationResult{
					Passed:   true,
					Expected: "@on create: [auth]",
					Actual:   "@on create: [auth]",
				},
			},
			// Failure case - wrong middleware name
			{
				TestCase: TestCase{
					Name:     "Apply authenticated_handler pattern to Comment.create",
					Category: "authentication",
				},
				Provider: "test-provider",
				Response: "@on create: [login]",
				Validation: ValidationResult{
					Passed:   false,
					Expected: "@on create: [auth]",
					Actual:   "@on create: [login]",
				},
			},
			// Failure case - insufficient examples
			{
				TestCase: TestCase{
					Name:     "Apply cached_handler pattern to Article.list",
					Category: "caching",
				},
				Provider: "test-provider",
				Response: "@on list: [cache(300)]",
				Validation: ValidationResult{
					Passed:   false,
					Expected: "@on list: [cache(300)]",
					Actual:   "@on list: [cache(600)]",
				},
			},
		},
		Summary: ReportSummary{
			TotalTests:  3,
			PassedTests: 1,
			FailedTests: 2,
		},
	}

	analysis := analyzer.Analyze(report, patterns)

	// Verify total failures
	if analysis.TotalFailures != 2 {
		t.Errorf("Expected 2 total failures, got %d", analysis.TotalFailures)
	}

	// Verify failures are categorized
	if len(analysis.ByReason) == 0 {
		t.Error("Expected failures to be categorized by reason")
	}

	// Verify pattern failure tracking
	if len(analysis.ByPattern) == 0 {
		t.Error("Expected pattern failures to be tracked")
	}

	// Verify recommendations are generated
	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}

	// Check specific pattern stats
	authPattern, exists := analysis.ByPattern["authenticated_handler"]
	if !exists {
		t.Error("Expected authenticated_handler in pattern failures")
	} else {
		// 2 tests, 1 passed, 1 failed
		if authPattern.FailureCount != 1 {
			t.Errorf("Expected 1 failure for authenticated_handler, got %d", authPattern.FailureCount)
		}
		if authPattern.SuccessRate < 0.4 || authPattern.SuccessRate > 0.6 {
			t.Errorf("Expected ~0.5 success rate, got %f", authPattern.SuccessRate)
		}
	}
}

func TestFailureAnalyzer_CategorizeFailure(t *testing.T) {
	analyzer := NewFailureAnalyzer()

	tests := []struct {
		name           string
		result         TestResult
		pattern        metadata.PatternMetadata
		expectedReason FailureReason
	}{
		{
			name: "Infrastructure error - timeout",
			result: TestResult{
				Error: errors.New("request timeout after 30s"),
			},
			pattern: metadata.PatternMetadata{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonInfrastructureError,
		},
		{
			name: "Infrastructure error - API key",
			result: TestResult{
				Error: errors.New("invalid API key provided"),
			},
			pattern: metadata.PatternMetadata{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonInfrastructureError,
		},
		{
			name: "Infrastructure error - rate limit",
			result: TestResult{
				Error: errors.New("rate limit exceeded"),
			},
			pattern: metadata.PatternMetadata{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonInfrastructureError,
		},
		{
			name: "Insufficient examples",
			result: TestResult{
				Response: "@on create: [auth]",
				Validation: ValidationResult{
					Expected: "@on create: [auth]",
				},
			},
			pattern: metadata.PatternMetadata{
				Examples:   []metadata.PatternExample{{Resource: "Post"}}, // Only 1 example
				Confidence: 0.6,
			},
			expectedReason: ReasonInsufficientExamples,
		},
		{
			name: "Pattern too specific",
			result: TestResult{
				Response: "@on create: [auth]",
				Validation: ValidationResult{
					Expected: "@on create: [auth]",
				},
			},
			pattern: metadata.PatternMetadata{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.3, // Low confidence
			},
			expectedReason: ReasonPatternTooSpecific,
		},
		{
			name: "No middleware declaration",
			result: TestResult{
				Response: "function create() { ... }",
				Validation: ValidationResult{
					Expected: "@on create: [auth]",
				},
			},
			pattern: metadata.PatternMetadata{
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonLLMHallucination,
		},
		{
			name: "Similar concept but wrong name",
			result: TestResult{
				Response: "@on create: [login]",
				Validation: ValidationResult{
					Expected: "@on create: [auth]",
				},
			},
			pattern: metadata.PatternMetadata{
				Category: "authentication",
				Template: "@on <operation>: [auth]",
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonNameUnclear,
		},
		{
			name: "Right middleware, wrong syntax",
			result: TestResult{
				Response: "@on create: [auth(required)]",
				Validation: ValidationResult{
					Expected: "@on create: [auth]",
				},
			},
			pattern: metadata.PatternMetadata{
				Category: "authentication",
				Template: "@on <operation>: [auth]",
				Examples: []metadata.PatternExample{
					{Resource: "Post"},
					{Resource: "Comment"},
					{Resource: "Article"},
				},
				Confidence: 0.6,
			},
			expectedReason: ReasonTemplateAmbiguous,
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
}

func TestFailureAnalyzer_GenerateRecommendations(t *testing.T) {
	analyzer := NewFailureAnalyzer()

	tests := []struct {
		name                   string
		analysis               FailureAnalysis
		expectedRecommendation string
	}{
		{
			name: "Pattern too specific",
			analysis: FailureAnalysis{
				TotalFailures: 5,
				ByReason: map[FailureReason][]TestResult{
					ReasonPatternTooSpecific: {
						{}, {}, {}, {}, {}, // 5 failures
					},
				},
			},
			expectedRecommendation: "Reduce minimum pattern frequency",
		},
		{
			name: "Insufficient examples",
			analysis: FailureAnalysis{
				TotalFailures: 3,
				ByReason: map[FailureReason][]TestResult{
					ReasonInsufficientExamples: {
						{}, {}, {}, // 3 failures
					},
				},
			},
			expectedRecommendation: "Increase minimum examples",
		},
		{
			name: "Name unclear",
			analysis: FailureAnalysis{
				TotalFailures: 4,
				ByReason: map[FailureReason][]TestResult{
					ReasonNameUnclear: {
						{}, {}, {}, {}, // 4 failures
					},
				},
			},
			expectedRecommendation: "Improve pattern naming",
		},
		{
			name: "Template ambiguous",
			analysis: FailureAnalysis{
				TotalFailures: 2,
				ByReason: map[FailureReason][]TestResult{
					ReasonTemplateAmbiguous: {
						{}, {}, // 2 failures
					},
				},
			},
			expectedRecommendation: "Clarify pattern templates",
		},
		{
			name: "LLM hallucination",
			analysis: FailureAnalysis{
				TotalFailures: 6,
				ByReason: map[FailureReason][]TestResult{
					ReasonLLMHallucination: {
						{}, {}, {}, {}, {}, {}, // 6 failures
					},
				},
			},
			expectedRecommendation: "Review prompts",
		},
		{
			name: "Low success pattern",
			analysis: FailureAnalysis{
				TotalFailures: 3,
				ByPattern: map[string]PatternFailure{
					"bad_pattern": {
						PatternName:  "bad_pattern",
						FailureCount: 3,
						SuccessRate:  0.2,
					},
				},
			},
			expectedRecommendation: "Review pattern 'bad_pattern'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := analyzer.generateRecommendations(tt.analysis)
			if len(recommendations) == 0 {
				t.Error("Expected recommendations to be generated")
				return
			}

			found := false
			for _, rec := range recommendations {
				if strings.Contains(rec, tt.expectedRecommendation) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected recommendation containing %q, got %v",
					tt.expectedRecommendation, recommendations)
			}
		})
	}
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove extra spaces",
			input:    "@on  create:  [auth]",
			expected: "@on create: [auth]",
		},
		{
			name:     "Remove newlines",
			input:    "@on create:\n[auth]",
			expected: "@on create: [auth]",
		},
		{
			name:     "Remove tabs",
			input:    "@on\tcreate:\t[auth]",
			expected: "@on create: [auth]",
		},
		{
			name:     "Multiple whitespace types",
			input:    "  @on  \tcreate:\n  [auth]  ",
			expected: "@on create: [auth]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCode(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCode(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsMiddlewareDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Valid declaration",
			code:     "@on create: [auth]",
			expected: true,
		},
		{
			name:     "Missing @on",
			code:     "create: [auth]",
			expected: false,
		},
		{
			name:     "Missing colon",
			code:     "@on create [auth]",
			expected: false,
		},
		{
			name:     "Empty string",
			code:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMiddlewareDeclaration(tt.code)
			if result != tt.expected {
				t.Errorf("containsMiddlewareDeclaration(%q) = %v, expected %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestContainsSimilarConcept(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		category string
		expected bool
	}{
		{
			name:     "Auth concept",
			code:     "@on create: [auth]",
			category: "authentication",
			expected: true,
		},
		{
			name:     "Login concept for auth",
			code:     "@on create: [login]",
			category: "authentication",
			expected: true,
		},
		{
			name:     "Cache concept",
			code:     "@on list: [cache(300)]",
			category: "caching",
			expected: true,
		},
		{
			name:     "Rate limit concept",
			code:     "@on create: [rate_limit(10/hour)]",
			category: "rate_limiting",
			expected: true,
		},
		{
			name:     "CORS concept",
			code:     "@on options: [cors]",
			category: "cors",
			expected: true,
		},
		{
			name:     "Wrong concept",
			code:     "@on create: [auth]",
			category: "caching",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSimilarConcept(tt.code, tt.category)
			if result != tt.expected {
				t.Errorf("containsSimilarConcept(%q, %q) = %v, expected %v",
					tt.code, tt.category, result, tt.expected)
			}
		})
	}
}

func TestContainsMiddlewareFromPattern(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		template string
		expected bool
	}{
		{
			name:     "Exact match",
			code:     "@on create: [auth]",
			template: "@on <operation>: [auth]",
			expected: true,
		},
		{
			name:     "With parameters in template",
			code:     "@on list: [cache(300)]",
			template: "@on <operation>: [cache(300)]",
			expected: true,
		},
		{
			name:     "Base name match",
			code:     "@on list: [cache(600)]",
			template: "@on <operation>: [cache(300)]",
			expected: true,
		},
		{
			name:     "Multiple middleware",
			code:     "@on create: [auth, rate_limit(10/hour)]",
			template: "@on <operation>: [auth, rate_limit(5/hour)]",
			expected: true,
		},
		{
			name:     "No match",
			code:     "@on create: [cors]",
			template: "@on <operation>: [auth]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMiddlewareFromPattern(tt.code, tt.template)
			if result != tt.expected {
				t.Errorf("containsMiddlewareFromPattern(%q, %q) = %v, expected %v",
					tt.code, tt.template, result, tt.expected)
			}
		})
	}
}

func TestDeterminePrimaryReason(t *testing.T) {
	analyzer := NewFailureAnalyzer()

	tests := []struct {
		name     string
		reasons  []FailureReason
		expected FailureReason
	}{
		{
			name:     "Empty reasons",
			reasons:  []FailureReason{},
			expected: ReasonPatternTooGeneric,
		},
		{
			name: "Single reason",
			reasons: []FailureReason{
				ReasonNameUnclear,
			},
			expected: ReasonNameUnclear,
		},
		{
			name: "Multiple same reason",
			reasons: []FailureReason{
				ReasonInsufficientExamples,
				ReasonInsufficientExamples,
				ReasonInsufficientExamples,
			},
			expected: ReasonInsufficientExamples,
		},
		{
			name: "Multiple different reasons",
			reasons: []FailureReason{
				ReasonNameUnclear,
				ReasonNameUnclear,
				ReasonNameUnclear,
				ReasonTemplateAmbiguous,
				ReasonTemplateAmbiguous,
				ReasonLLMHallucination,
			},
			expected: ReasonNameUnclear, // Most common
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.determinePrimaryReason(tt.reasons)
			if result != tt.expected {
				t.Errorf("determinePrimaryReason() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
		matches  bool
	}{
		{
			name:     "Exact match",
			code:     "@on create: [auth]",
			expected: "@on create: [auth]",
			matches:  true,
		},
		{
			name:     "No match",
			code:     "@on create: [login]",
			expected: "@on create: [auth]",
			matches:  false,
		},
		{
			name:     "Different spacing",
			code:     "@on  create:  [auth]",
			expected: "@on create: [auth]",
			matches:  false, // matchesPattern expects normalized input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.code, tt.expected)
			if result != tt.matches {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v",
					tt.code, tt.expected, result, tt.matches)
			}
		})
	}
}

// Benchmark tests
func BenchmarkAnalyze(b *testing.B) {
	analyzer := NewFailureAnalyzer()

	patterns := []metadata.PatternMetadata{
		{
			Name:       "auth_handler",
			Category:   "authentication",
			Template:   "@on <operation>: [auth]",
			Examples:   []metadata.PatternExample{{}, {}, {}},
			Frequency:  3,
			Confidence: 0.6,
		},
	}

	report := Report{
		Results: make([]TestResult, 100),
	}

	for i := 0; i < 100; i++ {
		report.Results[i] = TestResult{
			TestCase: TestCase{
				Name:     "Apply auth_handler pattern to Post.create",
				Category: "authentication",
			},
			Provider:  "test",
			Response:  "@on create: [auth]",
			Timestamp: time.Now(),
			Validation: ValidationResult{
				Passed:   i%2 == 0, // 50% success rate
				Expected: "@on create: [auth]",
				Actual:   "@on create: [auth]",
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(report, patterns)
	}
}

func TestIsInfrastructureError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      errors.New("request timeout after 30s"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "API key error",
			err:      errors.New("invalid API key"),
			expected: true,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "LLM hallucination (not infrastructure)",
			err:      errors.New("invalid response format"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInfrastructureError(tt.err)
			if result != tt.expected {
				t.Errorf("isInfrastructureError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}
