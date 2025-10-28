package llm

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestIterationReporter_PrintIterationSummary(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	config := DefaultIterationConfig()

	result := IterationResult{
		IterationNumber: 1,
		Patterns: []metadata.PatternMetadata{
			{Name: "pattern1", Category: "auth"},
			{Name: "pattern2", Category: "auth"},
		},
		Report: Report{
			Summary: ReportSummary{
				TotalTests:  6,
				PassedTests: 4,
				FailedTests: 2,
				SuccessRate: 0.667,
				ByProvider: map[string]ProviderStats{
					"claude-opus": {
						TotalTests:  3,
						PassedTests: 2,
						SuccessRate: 0.667,
					},
					"gpt-4": {
						TotalTests:  3,
						PassedTests: 2,
						SuccessRate: 0.667,
					},
				},
			},
		},
		FailureAnalysis: FailureAnalysis{
			TotalFailures: 2,
			ByReason: map[FailureReason][]TestResult{
				ReasonNameUnclear: {
					{},
				},
				ReasonPatternTooSpecific: {
					{},
				},
			},
			Recommendations: []string{
				"Improve pattern naming",
				"Lower MinFrequency threshold",
			},
		},
		MetCriteria: false,
	}

	reporter.PrintIterationSummary(result, config)

	output := buf.String()

	// Check for key elements
	if !strings.Contains(output, "ITERATION 1") {
		t.Errorf("Expected iteration header, got: %s", output)
	}

	if !strings.Contains(output, "Baseline") {
		t.Errorf("Expected 'Baseline' label for iteration 1, got: %s", output)
	}

	if !strings.Contains(output, "Extracted 2 patterns") {
		t.Errorf("Expected pattern count, got: %s", output)
	}

	if !strings.Contains(output, "Results:") {
		t.Errorf("Expected results section, got: %s", output)
	}

	if !strings.Contains(output, "claude-opus") {
		t.Errorf("Expected provider name, got: %s", output)
	}

	if !strings.Contains(output, "Top Failure Reasons:") {
		t.Errorf("Expected failure reasons, got: %s", output)
	}

	if !strings.Contains(output, "Recommendations:") {
		t.Errorf("Expected recommendations, got: %s", output)
	}
}

func TestIterationReporter_PrintProgressChart(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	config := DefaultIterationConfig()

	results := []IterationResult{
		{
			IterationNumber: 1,
			Report: Report{
				Summary: ReportSummary{
					SuccessRate: 0.6,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {SuccessRate: 0.6},
					},
				},
			},
		},
		{
			IterationNumber: 2,
			Report: Report{
				Summary: ReportSummary{
					SuccessRate: 0.75,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {SuccessRate: 0.75},
					},
				},
			},
		},
		{
			IterationNumber: 3,
			Report: Report{
				Summary: ReportSummary{
					SuccessRate: 0.85,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {SuccessRate: 0.85},
					},
				},
			},
		},
	}

	reporter.PrintProgressChart(results, config)

	output := buf.String()

	// Check for key elements
	if !strings.Contains(output, "PROGRESS OVERVIEW") {
		t.Errorf("Expected progress header, got: %s", output)
	}

	if !strings.Contains(output, "claude-opus") {
		t.Errorf("Expected provider name, got: %s", output)
	}

	if !strings.Contains(output, "Iter 1:") {
		t.Errorf("Expected iteration 1, got: %s", output)
	}

	if !strings.Contains(output, "Iter 2:") {
		t.Errorf("Expected iteration 2, got: %s", output)
	}

	if !strings.Contains(output, "Iter 3:") {
		t.Errorf("Expected iteration 3, got: %s", output)
	}

	if !strings.Contains(output, "Overall Improvement:") {
		t.Errorf("Expected overall improvement, got: %s", output)
	}

	// Should show improvement
	if !strings.Contains(output, "+25.0%") {
		t.Errorf("Expected improvement percentage, got: %s", output)
	}
}

func TestIterationReporter_ExportIterationReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	config := DefaultIterationConfig()

	results := []IterationResult{
		{
			IterationNumber: 1,
			Patterns: []metadata.PatternMetadata{
				{Name: "pattern1"},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  3,
					PassedTests: 2,
					SuccessRate: 0.667,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {SuccessRate: 0.667},
					},
				},
			},
			FailureAnalysis: FailureAnalysis{
				TotalFailures:   1,
				Recommendations: []string{"Test recommendation"},
			},
			MetCriteria: false,
		},
		{
			IterationNumber: 2,
			Patterns: []metadata.PatternMetadata{
				{Name: "pattern1"},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  3,
					PassedTests: 3,
					SuccessRate: 1.0,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {SuccessRate: 1.0},
					},
				},
			},
			FailureAnalysis: FailureAnalysis{},
			MetCriteria:     true,
		},
	}

	jsonReport, err := reporter.ExportIterationReport(results, config)
	if err != nil {
		t.Fatalf("Failed to export report: %v", err)
	}

	// Check JSON structure
	if !strings.Contains(jsonReport, `"timestamp"`) {
		t.Errorf("Expected timestamp field, got: %s", jsonReport)
	}

	if !strings.Contains(jsonReport, `"iterations"`) {
		t.Errorf("Expected iterations field, got: %s", jsonReport)
	}

	if !strings.Contains(jsonReport, `"total_iterations": 2`) {
		t.Errorf("Expected total_iterations: 2, got: %s", jsonReport)
	}

	if !strings.Contains(jsonReport, `"final_success": true`) {
		t.Errorf("Expected final_success: true, got: %s", jsonReport)
	}

	// Check improvement calculation (1.0 - 0.667 ≈ 0.333)
	if !strings.Contains(jsonReport, `"improvement"`) {
		t.Errorf("Expected improvement field, got: %s", jsonReport)
	}

	if !strings.Contains(jsonReport, `"success_criteria"`) {
		t.Errorf("Expected success_criteria field, got: %s", jsonReport)
	}
}

func TestIterationReporter_GenerateProgressBar(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	tests := []struct {
		rate     float64
		width    int
		expected string
	}{
		{0.0, 10, "░░░░░░░░░░"},
		{0.5, 10, "█████░░░░░"},
		{1.0, 10, "██████████"},
		{0.3, 10, "███░░░░░░░"},
		{0.75, 10, "███████░░░"},
	}

	for _, tt := range tests {
		result := reporter.generateProgressBar(tt.rate, tt.width)
		if result != tt.expected {
			t.Errorf("generateProgressBar(%.1f, %d) = %s, want %s",
				tt.rate, tt.width, result, tt.expected)
		}
	}
}

func TestIterationReporter_PrintIterationResults(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	config := DefaultIterationConfig()

	results := []IterationResult{
		{
			IterationNumber: 1,
			Patterns: []metadata.PatternMetadata{
				{Name: "auth_pattern", Category: "authentication"},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  6,
					PassedTests: 4,
					SuccessRate: 0.667,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {
							TotalTests:  3,
							PassedTests: 2,
							SuccessRate: 0.667,
						},
						"gpt-4": {
							TotalTests:  3,
							PassedTests: 2,
							SuccessRate: 0.667,
						},
					},
				},
				ExecutionTime: 5 * time.Second,
			},
			FailureAnalysis: FailureAnalysis{
				TotalFailures: 2,
				ByReason: map[FailureReason][]TestResult{
					ReasonNameUnclear: {{}, {}},
				},
				Recommendations: []string{"Improve naming"},
			},
			MetCriteria: false,
		},
		{
			IterationNumber: 2,
			Patterns: []metadata.PatternMetadata{
				{Name: "auth_pattern", Category: "authentication"},
			},
			Report: Report{
				Summary: ReportSummary{
					TotalTests:  6,
					PassedTests: 6,
					SuccessRate: 1.0,
					ByProvider: map[string]ProviderStats{
						"claude-opus": {
							TotalTests:  3,
							PassedTests: 3,
							SuccessRate: 1.0,
						},
						"gpt-4": {
							TotalTests:  3,
							PassedTests: 3,
							SuccessRate: 1.0,
						},
					},
				},
				ExecutionTime: 5 * time.Second,
			},
			FailureAnalysis: FailureAnalysis{},
			MetCriteria:     true,
		},
	}

	reporter.PrintIterationResults(results, config)

	output := buf.String()

	// Check for header
	if !strings.Contains(output, "PATTERN VALIDATION ITERATION SYSTEM") {
		t.Errorf("Expected header, got: %s", output)
	}

	// Check for progress overview
	if !strings.Contains(output, "PROGRESS OVERVIEW") {
		t.Errorf("Expected progress overview, got: %s", output)
	}

	// Check for iteration summaries
	if !strings.Contains(output, "ITERATION 1") {
		t.Errorf("Expected iteration 1, got: %s", output)
	}

	if !strings.Contains(output, "ITERATION 2") {
		t.Errorf("Expected iteration 2, got: %s", output)
	}

	// Check for final summary
	if !strings.Contains(output, "SUCCESS CRITERIA MET") {
		t.Errorf("Expected success message, got: %s", output)
	}

	if !strings.Contains(output, "Final Report:") {
		t.Errorf("Expected final report, got: %s", output)
	}
}

func TestIterationReporter_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewIterationReporter(&buf)

	config := DefaultIterationConfig()

	reporter.PrintIterationResults([]IterationResult{}, config)

	output := buf.String()

	if !strings.Contains(output, "No iteration results to display") {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestGetFailureReasonDescription(t *testing.T) {
	tests := []struct {
		reason   FailureReason
		expected string
	}{
		{ReasonPatternTooSpecific, "Low confidence"},
		{ReasonPatternTooGeneric, "Pattern too vague"},
		{ReasonNameUnclear, "LLM used different names"},
		{ReasonTemplateAmbiguous, "Syntax unclear"},
		{ReasonInsufficientExamples, "Need more examples"},
		{ReasonLLMHallucination, "LLM generated unrelated code"},
		{FailureReason("unknown"), "Unknown reason"},
	}

	for _, tt := range tests {
		result := getFailureReasonDescription(tt.reason)
		if result != tt.expected {
			t.Errorf("getFailureReasonDescription(%s) = %s, want %s",
				tt.reason, result, tt.expected)
		}
	}
}
