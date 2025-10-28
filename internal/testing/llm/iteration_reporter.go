package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// IterationReporter provides comprehensive reporting for iteration runs.
// It generates both human-readable console output and machine-readable JSON.
type IterationReporter struct {
	writer io.Writer
}

// NewIterationReporter creates a new iteration reporter.
func NewIterationReporter(writer io.Writer) *IterationReporter {
	if writer == nil {
		writer = os.Stdout
	}
	return &IterationReporter{
		writer: writer,
	}
}

// PrintIterationResults prints a detailed report of all iterations.
// Shows summary, progress chart, per-iteration breakdown, and final recommendations.
func (r *IterationReporter) PrintIterationResults(results []IterationResult, config IterationConfig) {
	if len(results) == 0 {
		fmt.Fprintln(r.writer, "No iteration results to display.")
		return
	}

	// Print header
	r.printHeader()

	// Print progress chart
	r.PrintProgressChart(results, config)

	// Print per-iteration details
	for _, result := range results {
		r.PrintIterationSummary(result, config)
	}

	// Print final summary
	r.printFinalSummary(results, config)
}

// printHeader prints the report header.
func (r *IterationReporter) printHeader() {
	cyan := color.New(color.FgCyan, color.Bold)

	fmt.Fprintln(r.writer, strings.Repeat("═", 65))
	cyan.Fprintln(r.writer, "  PATTERN VALIDATION ITERATION SYSTEM")
	fmt.Fprintln(r.writer, strings.Repeat("═", 65))
	fmt.Fprintln(r.writer)
}

// PrintIterationSummary prints a brief summary for each iteration.
// Shows iteration number, success rates per provider, top failure reasons, and patterns tested.
func (r *IterationReporter) PrintIterationSummary(result IterationResult, config IterationConfig) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	// Section header
	fmt.Fprintf(r.writer, "\n%s ITERATION %d", strings.Repeat("━", 3), result.IterationNumber)

	// Add label based on iteration number
	if result.IterationNumber == 1 {
		fmt.Fprintf(r.writer, ": Baseline")
	}

	fmt.Fprintf(r.writer, " %s\n\n", strings.Repeat("━", 50-len(fmt.Sprintf("ITERATION %d", result.IterationNumber))))

	// Patterns extracted
	fmt.Fprintf(r.writer, "Extracted %d patterns from resources\n", len(result.Patterns))
	fmt.Fprintf(r.writer, "Generated %d test cases (%d patterns × %d providers)\n\n",
		result.Report.Summary.TotalTests,
		len(result.Patterns),
		len(result.Report.Summary.ByProvider))

	// Results section
	bold.Fprintln(r.writer, "Results:")

	// Sort providers for consistent output
	providers := make([]string, 0, len(result.Report.Summary.ByProvider))
	for provider := range result.Report.Summary.ByProvider {
		providers = append(providers, provider)
	}
	// Sort by target rate (highest first)
	for i := 0; i < len(providers); i++ {
		for j := i + 1; j < len(providers); j++ {
			targetI := config.TargetSuccessRate[providers[i]]
			targetJ := config.TargetSuccessRate[providers[j]]
			if targetJ > targetI {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	for _, provider := range providers {
		stats := result.Report.Summary.ByProvider[provider]
		target, hasTarget := config.TargetSuccessRate[provider]

		// Calculate pass/total
		total := stats.TotalTests
		passed := stats.PassedTests

		// Format provider line
		fmt.Fprintf(r.writer, "  %-12s %d/%d  (%.1f%%)  ",
			provider+":", passed, total, stats.SuccessRate*100)

		// Progress bar
		progressBar := r.generateProgressBar(stats.SuccessRate, 10)
		fmt.Fprintf(r.writer, "%s  ", progressBar)

		// Target indicator
		if hasTarget {
			targetStr := fmt.Sprintf("Target: %.0f%%", target*100)
			if stats.SuccessRate >= target {
				green.Fprintf(r.writer, "✓ %s", targetStr)
			} else {
				fmt.Fprintf(r.writer, "%s", targetStr)
			}
		}
		fmt.Fprintln(r.writer)
	}

	// Failure analysis
	if result.FailureAnalysis.TotalFailures > 0 {
		fmt.Fprintln(r.writer)
		yellow.Fprintln(r.writer, "Top Failure Reasons:")

		// Find top 3 failure reasons by count
		type reasonCount struct {
			reason FailureReason
			count  int
		}
		var counts []reasonCount
		for reason, results := range result.FailureAnalysis.ByReason {
			counts = append(counts, reasonCount{reason, len(results)})
		}
		// Sort by count descending
		for i := 0; i < len(counts); i++ {
			for j := i + 1; j < len(counts); j++ {
				if counts[j].count > counts[i].count {
					counts[i], counts[j] = counts[j], counts[i]
				}
			}
		}

		// Show top 3
		maxReasons := 3
		if len(counts) < maxReasons {
			maxReasons = len(counts)
		}
		for i := 0; i < maxReasons; i++ {
			desc := getFailureReasonDescription(counts[i].reason)
			fmt.Fprintf(r.writer, "  %d. %s (%d failures) - %s\n",
				i+1, counts[i].reason, counts[i].count, desc)
		}
	}

	// Recommendations
	if !result.MetCriteria && len(result.FailureAnalysis.Recommendations) > 0 {
		fmt.Fprintln(r.writer)
		cyan.Fprintln(r.writer, "Recommendations:")
		for _, rec := range result.FailureAnalysis.Recommendations {
			fmt.Fprintf(r.writer, "  • %s\n", rec)
		}
	}

	// Success indicator
	if result.MetCriteria {
		fmt.Fprintln(r.writer)
		green.Fprintln(r.writer, "✅ SUCCESS CRITERIA MET!")
	} else if result.IterationNumber > 1 {
		// Show progress indicator for iterations after first
		prevResult := findPreviousResult(result.IterationNumber, []IterationResult{result})
		if prevResult != nil {
			improvement := result.Report.Summary.SuccessRate - prevResult.Report.Summary.SuccessRate
			if improvement > 0 {
				green.Fprintf(r.writer, "\nProgress: ⬆ Improvement across all providers! (+%.1f%%)\n", improvement*100)
			} else if improvement < 0 {
				red.Fprintf(r.writer, "\nProgress: ⬇ Decreased performance (-%.1f%%)\n", -improvement*100)
			} else {
				yellow.Fprintln(r.writer, "\nProgress: ⮕ No change")
			}
		}
	}
}

// PrintProgressChart prints an ASCII chart showing improvement over iterations.
// Displays visual chart of success rates, target lines, and trend indicators.
func (r *IterationReporter) PrintProgressChart(results []IterationResult, config IterationConfig) {
	if len(results) == 0 {
		return
	}

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	bold.Fprintln(r.writer, "PROGRESS OVERVIEW")
	fmt.Fprintln(r.writer)

	// Get list of providers
	var providers []string
	if len(results) > 0 && len(results[0].Report.Summary.ByProvider) > 0 {
		for provider := range results[0].Report.Summary.ByProvider {
			providers = append(providers, provider)
		}
	}

	// Show progress for each provider
	for _, provider := range providers {
		target := config.TargetSuccessRate[provider]

		cyan.Fprintf(r.writer, "%s (target: %.0f%%)\n", provider, target*100)

		// Show progress line for each iteration
		for i, result := range results {
			stats := result.Report.Summary.ByProvider[provider]

			fmt.Fprintf(r.writer, "  Iter %d: ", i+1)

			// Progress bar (scaled to 100%)
			barWidth := 30
			progressBar := r.generateProgressBar(stats.SuccessRate, barWidth)
			fmt.Fprintf(r.writer, "%s %.1f%%", progressBar, stats.SuccessRate*100)

			// Met target indicator
			if stats.SuccessRate >= target {
				green := color.New(color.FgGreen)
				green.Fprintf(r.writer, " ✓")
			}

			fmt.Fprintln(r.writer)
		}
		fmt.Fprintln(r.writer)
	}

	// Overall trend
	if len(results) > 1 {
		firstRate := results[0].Report.Summary.SuccessRate
		lastRate := results[len(results)-1].Report.Summary.SuccessRate
		improvement := lastRate - firstRate

		bold.Fprintf(r.writer, "Overall Improvement: ")
		if improvement > 0 {
			green := color.New(color.FgGreen)
			green.Fprintf(r.writer, "+%.1f%%\n", improvement*100)
		} else if improvement < 0 {
			red := color.New(color.FgRed)
			red.Fprintf(r.writer, "%.1f%%\n", improvement*100)
		} else {
			fmt.Fprintf(r.writer, "0.0%%\n")
		}
		fmt.Fprintln(r.writer)
	}
}

// ExportIterationReport exports full report to JSON.
func (r *IterationReporter) ExportIterationReport(results []IterationResult, config IterationConfig) (string, error) {
	type JSONOutput struct {
		Timestamp      string            `json:"timestamp"`
		Iterations     []IterationResult `json:"iterations"`
		TotalCount     int               `json:"total_iterations"`
		FinalSuccess   bool              `json:"final_success"`
		Improvement    float64           `json:"improvement"`
		SuccessCriteria map[string]float64 `json:"success_criteria"`
	}

	output := JSONOutput{
		Timestamp:       time.Now().Format(time.RFC3339),
		Iterations:      results,
		TotalCount:      len(results),
		SuccessCriteria: config.TargetSuccessRate,
	}

	// Calculate final success
	if len(results) > 0 {
		output.FinalSuccess = results[len(results)-1].MetCriteria
	}

	// Calculate improvement
	if len(results) > 1 {
		firstRate := results[0].Report.Summary.SuccessRate
		lastRate := results[len(results)-1].Report.Summary.SuccessRate
		output.Improvement = lastRate - firstRate
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	return string(data), nil
}

// printFinalSummary prints the final summary after all iterations.
func (r *IterationReporter) printFinalSummary(results []IterationResult, config IterationConfig) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer, strings.Repeat("═", 65))

	lastResult := results[len(results)-1]

	if lastResult.MetCriteria {
		// Success banner
		green.Fprintln(r.writer, "  ✅ SUCCESS CRITERIA MET!")
		fmt.Fprintln(r.writer)

		bold.Fprintln(r.writer, "Final Report:")
		fmt.Fprintf(r.writer, "  - Total iterations: %d\n", len(results))
		fmt.Fprintf(r.writer, "  - Patterns validated: %d\n", len(lastResult.Patterns))

		if len(results) > 1 {
			improvement := lastResult.Report.Summary.SuccessRate - results[0].Report.Summary.SuccessRate
			fmt.Fprintf(r.writer, "  - Overall improvement: +%.1f%%\n", improvement*100)
		}

		fmt.Fprintln(r.writer, "  - All patterns above 50%% threshold")

		// Provider success summary
		fmt.Fprintln(r.writer)
		bold.Fprintln(r.writer, "Provider Success Rates:")
		for provider, target := range config.TargetSuccessRate {
			if stats, ok := lastResult.Report.Summary.ByProvider[provider]; ok {
				green.Fprintf(r.writer, "  ✓ %-12s %.1f%% (target: %.0f%%)\n",
					provider+":", stats.SuccessRate*100, target*100)
			}
		}
	} else {
		// Failure banner
		red.Fprintln(r.writer, "  ❌ MORE WORK NEEDED")
		fmt.Fprintln(r.writer)

		bold.Fprintln(r.writer, "Final Report:")
		fmt.Fprintf(r.writer, "  - Total iterations: %d (max: %d)\n", len(results), config.MaxIterations)
		fmt.Fprintf(r.writer, "  - Patterns tested: %d\n", len(lastResult.Patterns))

		// Show what's missing
		fmt.Fprintln(r.writer)
		bold.Fprintln(r.writer, "Remaining Issues:")
		for provider, target := range config.TargetSuccessRate {
			if stats, ok := lastResult.Report.Summary.ByProvider[provider]; ok {
				if stats.SuccessRate < target {
					fmt.Fprintf(r.writer, "  • %-12s %.1f%% (need: %.0f%%, gap: %.1f%%)\n",
						provider+":", stats.SuccessRate*100, target*100, (target-stats.SuccessRate)*100)
				}
			}
		}

		// Top recommendations
		if len(lastResult.FailureAnalysis.Recommendations) > 0 {
			fmt.Fprintln(r.writer)
			bold.Fprintln(r.writer, "Next Steps:")
			for _, rec := range lastResult.FailureAnalysis.Recommendations {
				fmt.Fprintf(r.writer, "  • %s\n", rec)
			}
		}
	}

	fmt.Fprintln(r.writer, strings.Repeat("═", 65))
}

// generateProgressBar generates an ASCII progress bar.
// width is the number of characters for the full bar.
func (r *IterationReporter) generateProgressBar(rate float64, width int) string {
	filled := int(rate * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// getFailureReasonDescription returns a human-readable description for a failure reason.
func getFailureReasonDescription(reason FailureReason) string {
	switch reason {
	case ReasonPatternTooSpecific:
		return "Low confidence"
	case ReasonPatternTooGeneric:
		return "Pattern too vague"
	case ReasonNameUnclear:
		return "LLM used different names"
	case ReasonTemplateAmbiguous:
		return "Syntax unclear"
	case ReasonInsufficientExamples:
		return "Need more examples"
	case ReasonLLMHallucination:
		return "LLM generated unrelated code"
	default:
		return "Unknown reason"
	}
}

// findPreviousResult finds the previous iteration result.
// Returns nil if not found or if this is the first iteration.
func findPreviousResult(currentIteration int, results []IterationResult) *IterationResult {
	if currentIteration <= 1 {
		return nil
	}

	for _, result := range results {
		if result.IterationNumber == currentIteration-1 {
			return &result
		}
	}

	return nil
}
