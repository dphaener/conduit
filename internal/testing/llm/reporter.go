package llm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TestResult captures the result of a single test case execution.
type TestResult struct {
	// TestCase is the test case that was executed.
	TestCase TestCase

	// Provider is the LLM provider used.
	Provider string

	// Response is the raw response from the LLM.
	Response string

	// Validation is the validation result.
	Validation ValidationResult

	// Duration is how long the test took.
	Duration time.Duration

	// Error is any error that occurred during execution.
	Error error

	// Timestamp is when the test was executed.
	Timestamp time.Time
}

// Report contains aggregated results from test harness execution.
type Report struct {
	// Results contains all individual test results.
	Results []TestResult

	// Summary provides high-level statistics.
	Summary ReportSummary

	// ExecutionTime is the total time to run all tests.
	ExecutionTime time.Duration

	// Timestamp is when the report was generated.
	Timestamp time.Time
}

// ReportSummary provides aggregated statistics.
type ReportSummary struct {
	// TotalTests is the total number of tests executed.
	TotalTests int

	// PassedTests is the number of tests that passed.
	PassedTests int

	// FailedTests is the number of tests that failed.
	FailedTests int

	// ErrorTests is the number of tests that had errors.
	ErrorTests int

	// SuccessRate is the overall success rate (0.0-1.0).
	SuccessRate float64

	// ByProvider contains statistics per LLM provider.
	ByProvider map[string]ProviderStats

	// ByCategory contains statistics per pattern category.
	ByCategory map[string]CategoryStats
}

// ProviderStats contains statistics for a single LLM provider.
type ProviderStats struct {
	// Provider is the provider identifier.
	Provider string

	// TotalTests is the number of tests for this provider.
	TotalTests int

	// PassedTests is the number of passing tests.
	PassedTests int

	// SuccessRate is the success rate for this provider (0.0-1.0).
	SuccessRate float64

	// AverageDuration is the average test duration.
	AverageDuration time.Duration

	// AverageConfidence is the average validation confidence.
	AverageConfidence float64
}

// CategoryStats contains statistics for a pattern category.
type CategoryStats struct {
	// Category is the category name.
	Category string

	// TotalTests is the number of tests for this category.
	TotalTests int

	// PassedTests is the number of passing tests.
	PassedTests int

	// SuccessRate is the success rate for this category (0.0-1.0).
	SuccessRate float64
}

// Reporter generates reports from test results.
type Reporter struct{}

// NewReporter creates a new reporter.
func NewReporter() *Reporter {
	return &Reporter{}
}

// GenerateReport creates a report from test results.
func (r *Reporter) GenerateReport(results []TestResult, executionTime time.Duration) Report {
	summary := r.computeSummary(results)

	return Report{
		Results:       results,
		Summary:       summary,
		ExecutionTime: executionTime,
		Timestamp:     time.Now(),
	}
}

// computeSummary computes summary statistics from test results.
func (r *Reporter) computeSummary(results []TestResult) ReportSummary {
	summary := ReportSummary{
		TotalTests: len(results),
		ByProvider: make(map[string]ProviderStats),
		ByCategory: make(map[string]CategoryStats),
	}

	// Count passed/failed/error tests
	for _, result := range results {
		if result.Error != nil {
			summary.ErrorTests++
		} else if result.Validation.Passed {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}
	}

	// Calculate overall success rate
	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests)
	}

	// Compute per-provider stats
	r.computeProviderStats(results, summary.ByProvider)

	// Compute per-category stats
	r.computeCategoryStats(results, summary.ByCategory)

	return summary
}

// computeProviderStats computes statistics per provider.
func (r *Reporter) computeProviderStats(results []TestResult, stats map[string]ProviderStats) {
	// Group results by provider
	byProvider := make(map[string][]TestResult)
	for _, result := range results {
		byProvider[result.Provider] = append(byProvider[result.Provider], result)
	}

	// Calculate stats for each provider
	for provider, providerResults := range byProvider {
		stat := ProviderStats{
			Provider:   provider,
			TotalTests: len(providerResults),
		}

		var totalDuration time.Duration
		var totalConfidence float64
		validResults := 0

		for _, result := range providerResults {
			if result.Error == nil {
				if result.Validation.Passed {
					stat.PassedTests++
				}
				totalDuration += result.Duration
				totalConfidence += result.Validation.Confidence
				validResults++
			}
		}

		if stat.TotalTests > 0 {
			stat.SuccessRate = float64(stat.PassedTests) / float64(stat.TotalTests)
		}

		if validResults > 0 {
			stat.AverageDuration = totalDuration / time.Duration(validResults)
			stat.AverageConfidence = totalConfidence / float64(validResults)
		}

		stats[provider] = stat
	}
}

// computeCategoryStats computes statistics per category.
func (r *Reporter) computeCategoryStats(results []TestResult, stats map[string]CategoryStats) {
	// Group results by category
	byCategory := make(map[string][]TestResult)
	for _, result := range results {
		category := result.TestCase.Category
		byCategory[category] = append(byCategory[category], result)
	}

	// Calculate stats for each category
	for category, categoryResults := range byCategory {
		stat := CategoryStats{
			Category:   category,
			TotalTests: len(categoryResults),
		}

		for _, result := range categoryResults {
			if result.Error == nil && result.Validation.Passed {
				stat.PassedTests++
			}
		}

		if stat.TotalTests > 0 {
			stat.SuccessRate = float64(stat.PassedTests) / float64(stat.TotalTests)
		}

		stats[category] = stat
	}
}

// PrintReport prints a formatted report to stdout.
func (r *Reporter) PrintReport(report Report) {
	fmt.Println()
	fmt.Println("======================================")
	fmt.Println("LLM Validation Test Harness Report")
	fmt.Println("======================================")
	fmt.Println()

	// Overall summary
	fmt.Printf("Execution Time: %v\n", report.ExecutionTime)
	fmt.Printf("Timestamp: %v\n", report.Timestamp.Format(time.RFC3339))
	fmt.Println()

	fmt.Printf("Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("Errors: %d\n", report.Summary.ErrorTests)
	fmt.Printf("Success Rate: %.1f%%\n", report.Summary.SuccessRate*100)
	fmt.Println()

	// Per-provider stats
	fmt.Println("Results by Provider:")
	fmt.Println("--------------------")

	// Sort providers by name
	var providers []string
	for provider := range report.Summary.ByProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		stats := report.Summary.ByProvider[provider]
		fmt.Printf("\n%s:\n", stats.Provider)
		fmt.Printf("  Tests: %d\n", stats.TotalTests)
		fmt.Printf("  Passed: %d\n", stats.PassedTests)
		fmt.Printf("  Success Rate: %.1f%%\n", stats.SuccessRate*100)
		fmt.Printf("  Avg Duration: %v\n", stats.AverageDuration)
		fmt.Printf("  Avg Confidence: %.2f\n", stats.AverageConfidence)
	}
	fmt.Println()

	// Per-category stats
	fmt.Println("Results by Category:")
	fmt.Println("--------------------")

	// Sort categories by name
	var categories []string
	for category := range report.Summary.ByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		stats := report.Summary.ByCategory[category]
		fmt.Printf("\n%s:\n", stats.Category)
		fmt.Printf("  Tests: %d\n", stats.TotalTests)
		fmt.Printf("  Passed: %d\n", stats.PassedTests)
		fmt.Printf("  Success Rate: %.1f%%\n", stats.SuccessRate*100)
	}
	fmt.Println()

	// Failed tests detail
	failedCount := 0
	for _, result := range report.Results {
		if result.Error != nil || !result.Validation.Passed {
			failedCount++
		}
	}

	if failedCount > 0 {
		fmt.Println("Failed Tests:")
		fmt.Println("-------------")
		for i, result := range report.Results {
			if result.Error != nil {
				fmt.Printf("\n%d. %s [%s]\n", i+1, result.TestCase.Name, result.Provider)
				fmt.Printf("   Error: %v\n", result.Error)
			} else if !result.Validation.Passed {
				fmt.Printf("\n%d. %s [%s]\n", i+1, result.TestCase.Name, result.Provider)
				fmt.Printf("   Expected: %s\n", truncate(result.Validation.Expected, 80))
				fmt.Printf("   Actual: %s\n", truncate(result.Validation.Actual, 80))
				fmt.Printf("   Message: %s\n", result.Validation.Message)
				if len(result.Validation.Differences) > 0 {
					fmt.Printf("   Differences:\n")
					for _, diff := range result.Validation.Differences {
						fmt.Printf("     - %s\n", diff)
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("======================================")
}

// ExportJSON exports the report as JSON.
func (r *Reporter) ExportJSON(report Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}
	return string(data), nil
}

// PrintSummary prints a brief summary of the report.
func (r *Reporter) PrintSummary(report Report) {
	fmt.Printf("\nTest Results: %d total, %d passed, %d failed (%.1f%% success rate)\n",
		report.Summary.TotalTests,
		report.Summary.PassedTests,
		report.Summary.FailedTests+report.Summary.ErrorTests,
		report.Summary.SuccessRate*100,
	)

	// Show per-provider summary
	var providers []string
	for provider := range report.Summary.ByProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		stats := report.Summary.ByProvider[provider]
		fmt.Printf("  %s: %.1f%% (%d/%d)\n",
			stats.Provider,
			stats.SuccessRate*100,
			stats.PassedTests,
			stats.TotalTests,
		)
	}
}

// truncate truncates a string to a maximum length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
