package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// IterationRunner orchestrates the full pattern validation and improvement cycle.
// It runs multiple iterations of: extract patterns → generate test cases →
// validate with LLMs → analyze failures → adjust parameters → repeat.
type IterationRunner struct {
	harness          *Harness
	analyzer         *FailureAnalyzer
	config           IterationConfig
	verbose          bool
	patternExtractor *metadata.PatternExtractor
	mockLLMMu        sync.RWMutex
	mockLLM          func(string) string // For testing
}

// IterationConfig defines configuration for the iteration process.
type IterationConfig struct {
	// MaxIterations is the maximum number of iterations to run.
	MaxIterations int

	// TargetSuccessRate defines the minimum success rate required per provider.
	// Map keys are provider names (e.g., "claude-opus", "gpt-4").
	TargetSuccessRate map[string]float64

	// MinimumPatternSuccess is the minimum success rate for any individual pattern.
	// No pattern should have a success rate below this threshold.
	MinimumPatternSuccess float64

	// StopOnSuccess determines whether to stop iterations once success criteria are met.
	StopOnSuccess bool
}

// DefaultIterationConfig returns a configuration with sensible defaults.
// Default targets: Claude Opus 80%, GPT-4 70%, GPT-3.5 60%, minimum pattern success 50%.
func DefaultIterationConfig() IterationConfig {
	return IterationConfig{
		MaxIterations: 5,
		TargetSuccessRate: map[string]float64{
			"claude-opus":    0.80,
			"gpt-4":          0.70,
			"gpt-3.5-turbo":  0.60,
		},
		MinimumPatternSuccess: 0.50,
		StopOnSuccess:         true,
	}
}

// IterationResult captures the results from a single iteration.
type IterationResult struct {
	// IterationNumber is the iteration index (1-based).
	IterationNumber int

	// Patterns are the patterns extracted/used in this iteration.
	Patterns []metadata.PatternMetadata

	// Report contains the LLM validation results.
	Report Report

	// FailureAnalysis contains detailed analysis of failures.
	FailureAnalysis FailureAnalysis

	// MetCriteria indicates whether success criteria were met.
	MetCriteria bool

	// ExtractorParams are the parameters used for pattern extraction in this iteration.
	ExtractorParams metadata.PatternExtractionParams

	// ParameterChanges describes what parameters were changed from previous iteration.
	ParameterChanges []string
}

// NewIterationRunner creates a new iteration runner.
func NewIterationRunner(harness *Harness, analyzer *FailureAnalyzer, config IterationConfig) *IterationRunner {
	return &IterationRunner{
		harness:          harness,
		analyzer:         analyzer,
		config:           config,
		verbose:          false,
		patternExtractor: metadata.NewPatternExtractor(),
	}
}

// SetVerbose enables or disables verbose output.
func (ir *IterationRunner) SetVerbose(verbose bool) {
	ir.verbose = verbose
	ir.harness.SetVerbose(verbose)
}

// SetMockLLM sets a mock LLM function for testing.
func (ir *IterationRunner) SetMockLLM(mockFn func(string) string) {
	ir.mockLLMMu.Lock()
	defer ir.mockLLMMu.Unlock()
	ir.mockLLM = mockFn
}

// Run executes the full iteration loop.
// It extracts patterns from resources, generates test cases, validates with LLMs,
// analyzes failures, and repeats until success criteria are met or max iterations reached.
func (ir *IterationRunner) Run(ctx context.Context, resources []metadata.ResourceMetadata) ([]IterationResult, error) {
	if ir.verbose {
		fmt.Printf("Starting pattern validation iterations (max: %d)\n", ir.config.MaxIterations)
		fmt.Println("Success criteria:")
		for provider, target := range ir.config.TargetSuccessRate {
			fmt.Printf("  - %s: %.1f%%\n", provider, target*100)
		}
		fmt.Printf("  - Minimum pattern success: %.1f%%\n", ir.config.MinimumPatternSuccess*100)
		fmt.Println()
	}

	var results []IterationResult

	for i := 1; i <= ir.config.MaxIterations; i++ {
		if ir.verbose {
			fmt.Printf("=== Iteration %d/%d ===\n", i, ir.config.MaxIterations)
		}

		// Run a single iteration
		result, err := ir.runIteration(ctx, i, resources)
		if err != nil {
			return results, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		results = append(results, result)

		if ir.verbose {
			ir.printIterationSummary(result)
		}

		// Check if success criteria are met
		if result.MetCriteria && ir.config.StopOnSuccess {
			if ir.verbose {
				fmt.Printf("\nSuccess criteria met after %d iterations!\n", i)
			}
			break
		}

		if !result.MetCriteria && ir.verbose {
			fmt.Println("\nSuccess criteria not met. Recommendations:")
			for _, rec := range result.FailureAnalysis.Recommendations {
				fmt.Printf("  - %s\n", rec)
			}
			fmt.Println()
		}

		// Apply parameter adjustments for next iteration
		if i < ir.config.MaxIterations {
			newParams := ir.adjustParameters(result.FailureAnalysis, i, len(results))
			ir.patternExtractor = metadata.NewPatternExtractorWithParams(newParams)

			if ir.verbose && len(result.ParameterChanges) > 0 {
				fmt.Println("\nParameter adjustments for next iteration:")
				for _, change := range result.ParameterChanges {
					fmt.Printf("  - %s\n", change)
				}
				fmt.Println()
			}
		}
	}

	return results, nil
}

// runIteration executes a single iteration of the validation loop.
func (ir *IterationRunner) runIteration(ctx context.Context, iteration int, resources []metadata.ResourceMetadata) (IterationResult, error) {
	result := IterationResult{
		IterationNumber:  iteration,
		ExtractorParams:  ir.patternExtractor.GetParams(),
		ParameterChanges: ir.describeParameterChanges(iteration),
	}

	// Step 1: Extract patterns from resources
	if ir.verbose {
		fmt.Println("Extracting patterns from resources...")
	}

	patterns := ir.patternExtractor.ExtractMiddlewarePatterns(resources)
	result.Patterns = patterns

	if ir.verbose {
		fmt.Printf("Extracted %d patterns\n", len(patterns))
	}

	// If no patterns found, can't proceed
	if len(patterns) == 0 {
		return result, fmt.Errorf("no patterns extracted from resources")
	}

	// Step 2: Generate test cases from patterns
	if ir.verbose {
		fmt.Println("Generating test cases from patterns...")
	}

	testCases := ir.generateTestCases(patterns)

	if ir.verbose {
		fmt.Printf("Generated %d test cases\n", len(testCases))
	}

	// Step 3: Run tests with LLMs
	if ir.verbose {
		fmt.Println("Running tests with LLMs...")
	}

	var report Report
	var err error

	// Thread-safe mock access
	ir.mockLLMMu.RLock()
	mockFn := ir.mockLLM
	ir.mockLLMMu.RUnlock()

	// Check if we have a mock function for testing
	if mockFn != nil {
		report, err = ir.harness.RunWithMockLLM(ctx, testCases, mockFn)
	} else {
		report, err = ir.harness.Run(ctx, testCases)
	}

	if err != nil {
		return result, fmt.Errorf("harness run failed: %w", err)
	}

	result.Report = report

	if ir.verbose {
		fmt.Printf("Tests complete: %d total, %d passed, %d failed\n",
			report.Summary.TotalTests,
			report.Summary.PassedTests,
			report.Summary.FailedTests+report.Summary.ErrorTests)
	}

	// Step 4: Analyze failures
	if ir.verbose {
		fmt.Println("Analyzing failures...")
	}

	analysis := ir.analyzer.Analyze(report, patterns)
	result.FailureAnalysis = analysis

	// Step 5: Check success criteria
	result.MetCriteria = ir.checkSuccessCriteria(report, patterns)

	return result, nil
}

// generateTestCases creates test cases from patterns.
// For each pattern, it generates a test case that prompts the LLM to use the pattern.
func (ir *IterationRunner) generateTestCases(patterns []metadata.PatternMetadata) []TestCase {
	var testCases []TestCase

	for _, pattern := range patterns {
		// Generate a test case for each pattern
		// Use the first example's resource and infer an operation
		operation := "create"
		resourceName := "TestResource"

		if len(pattern.Examples) > 0 {
			resourceName = pattern.Examples[0].Resource
			// Try to extract operation from the example
			if pattern.Examples[0].Code != "" {
				// Parse "@on <operation>: [...]" to extract operation
				// This is a simple extraction - could be improved
				if idx := findOperationInCode(pattern.Examples[0].Code); idx != "" {
					operation = idx
				}
			}
		}

		generator := NewTestCaseGenerator(resourceName, operation)
		testCase := generator.GenerateFromPattern(pattern)
		testCases = append(testCases, testCase)
	}

	return testCases
}

// findOperationInCode extracts the operation name from a middleware declaration.
// Input: "@on create: [auth]" → Output: "create"
func findOperationInCode(code string) string {
	// Simple parser: find "@on <operation>:"
	// Look for "@on " followed by word followed by ":"
	const onPrefix = "@on "

	idx := 0
	for i := 0; i < len(code); i++ {
		if i+len(onPrefix) <= len(code) && code[i:i+len(onPrefix)] == onPrefix {
			idx = i + len(onPrefix)
			break
		}
	}

	if idx == 0 {
		return ""
	}

	// Find the next colon
	end := idx
	for end < len(code) && code[end] != ':' {
		end++
	}

	if end >= len(code) {
		return ""
	}

	operation := code[idx:end]
	// Trim whitespace
	operation = trimSpace(operation)

	return operation
}

// trimSpace removes leading and trailing whitespace from a string.
func trimSpace(s string) string {
	// Trim leading space
	start := 0
	for start < len(s) && isSpace(s[start]) {
		start++
	}

	// Trim trailing space
	end := len(s)
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isSpace checks if a byte is a whitespace character.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// checkSuccessCriteria checks if the iteration met all success criteria.
func (ir *IterationRunner) checkSuccessCriteria(report Report, patterns []metadata.PatternMetadata) bool {
	// Criterion 1: Check per-provider success rates
	for provider, targetRate := range ir.config.TargetSuccessRate {
		stats, exists := report.Summary.ByProvider[provider]
		if !exists {
			// Provider not tested, criteria not met
			return false
		}

		if stats.SuccessRate < targetRate {
			return false
		}
	}

	// Criterion 2: Check minimum pattern success rate
	// Calculate success rate per pattern
	patternSuccess := ir.calculatePatternSuccessRates(report, patterns)

	// If we have patterns but couldn't evaluate any, fail
	if len(patternSuccess) == 0 && len(patterns) > 0 {
		return false
	}

	for _, rate := range patternSuccess {
		if rate < ir.config.MinimumPatternSuccess {
			return false
		}
	}

	return true
}

// calculatePatternSuccessRates calculates the success rate for each pattern.
// Returns a map of pattern name to success rate.
func (ir *IterationRunner) calculatePatternSuccessRates(report Report, patterns []metadata.PatternMetadata) map[string]float64 {
	// Build a map of pattern name to test cases
	patternTests := make(map[string][]TestResult)

	for _, result := range report.Results {
		// The test case name contains the pattern name
		// Format: "Apply <pattern_name> pattern to <resource>.<operation>"
		patternName := extractPatternName(result.TestCase.Name)
		if patternName != "" {
			patternTests[patternName] = append(patternTests[patternName], result)
		}
	}

	// Calculate success rates
	rates := make(map[string]float64)

	for patternName, results := range patternTests {
		if len(results) == 0 {
			rates[patternName] = 0.0
			continue
		}

		passed := 0
		for _, result := range results {
			if result.Error == nil && result.Validation.Passed {
				passed++
			}
		}

		rates[patternName] = float64(passed) / float64(len(results))
	}

	return rates
}

// extractPatternName extracts the pattern name from a test case name.
// Input: "Apply authenticated_handler pattern to Post.create" → Output: "authenticated_handler"
func extractPatternName(testCaseName string) string {
	// Look for "Apply <pattern_name> pattern to"
	const prefix = "Apply "
	const suffix = " pattern to"

	start := strings.Index(testCaseName, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)

	end := strings.Index(testCaseName[start:], suffix)
	if end == -1 {
		return ""
	}

	return testCaseName[start : start+end]
}

// printIterationSummary prints a summary of the iteration results.
func (ir *IterationRunner) printIterationSummary(result IterationResult) {
	fmt.Printf("\nIteration %d Summary:\n", result.IterationNumber)
	fmt.Printf("  Patterns: %d\n", len(result.Patterns))
	fmt.Printf("  Tests: %d total, %d passed, %d failed\n",
		result.Report.Summary.TotalTests,
		result.Report.Summary.PassedTests,
		result.Report.Summary.FailedTests+result.Report.Summary.ErrorTests)
	fmt.Printf("  Overall success rate: %.1f%%\n", result.Report.Summary.SuccessRate*100)

	fmt.Println("\n  Per-provider results:")
	for provider, stats := range result.Report.Summary.ByProvider {
		target, hasTarget := ir.config.TargetSuccessRate[provider]
		status := ""
		if hasTarget {
			if stats.SuccessRate >= target {
				status = " [PASS]"
			} else {
				status = fmt.Sprintf(" [FAIL - need %.1f%%]", target*100)
			}
		}
		fmt.Printf("    %s: %.1f%%%s\n", provider, stats.SuccessRate*100, status)
	}

	if result.FailureAnalysis.TotalFailures > 0 {
		fmt.Println("\n  Failure analysis:")
		fmt.Printf("    Total failures: %d\n", result.FailureAnalysis.TotalFailures)

		if len(result.FailureAnalysis.ByReason) > 0 {
			fmt.Println("    By reason:")
			for reason, results := range result.FailureAnalysis.ByReason {
				fmt.Printf("      %s: %d\n", reason, len(results))
			}
		}
	}

	if result.MetCriteria {
		fmt.Println("\n  SUCCESS CRITERIA MET")
	} else {
		fmt.Println("\n  Success criteria not yet met")
	}
}

// adjustParameters updates extraction parameters based on failure analysis.
// It analyzes the dominant failure reasons and adjusts parameters accordingly.
func (ir *IterationRunner) adjustParameters(analysis FailureAnalysis, iteration int, totalResults int) metadata.PatternExtractionParams {
	params := ir.patternExtractor.GetParams()

	// Count failure reasons
	reasonCounts := make(map[FailureReason]int)
	for reason, results := range analysis.ByReason {
		reasonCounts[reason] = len(results)
	}

	totalFailures := analysis.TotalFailures
	if totalFailures == 0 {
		return params  // No failures, no adjustment needed
	}

	// Adjust based on dominant failure reason
	// 1. If name_unclear > 30% of failures, enable verbose names
	if float64(reasonCounts[ReasonNameUnclear])/float64(totalFailures) > 0.3 {
		params.VerboseNames = true
	}

	// 2. If pattern_too_specific > 30%, lower frequency threshold
	if float64(reasonCounts[ReasonPatternTooSpecific])/float64(totalFailures) > 0.3 {
		params.MinFrequency = max(1, params.MinFrequency-1)
	}

	// 3. If pattern_too_generic > 30%, raise confidence threshold
	if float64(reasonCounts[ReasonPatternTooGeneric])/float64(totalFailures) > 0.3 {
		params.MinConfidence = min(1.0, params.MinConfidence+0.1)
	}

	// 4. If insufficient_examples, lower min frequency to get more patterns with examples
	if float64(reasonCounts[ReasonInsufficientExamples])/float64(totalFailures) > 0.3 {
		params.MinFrequency = max(2, params.MinFrequency-1)
	}

	return params
}

// describeParameterChanges creates human-readable descriptions of parameter changes.
func (ir *IterationRunner) describeParameterChanges(iteration int) []string {
	if iteration == 1 {
		return nil // No changes in first iteration
	}

	// This will be populated by comparing current and previous params
	// For now, return empty slice; actual changes will be computed in Run
	return []string{}
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
