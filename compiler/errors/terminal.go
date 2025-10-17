package errors

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// FormatForTerminal formats a CompilerError for terminal output with ANSI colors
func (e CompilerError) FormatForTerminal() string {
	var sb strings.Builder

	// Error header with severity color
	severityColor := getSeverityColor(e.Severity)
	sb.WriteString(fmt.Sprintf("%s%s%s: %s\n",
		colorBold+severityColor,
		strings.Title(e.Severity.String()),
		colorReset,
		e.Message))

	// Location
	sb.WriteString(fmt.Sprintf("  %s-->%s %s:%d:%d\n",
		colorCyan,
		colorReset,
		e.Location.File,
		e.Location.Line,
		e.Location.Column))

	// Source context if available
	if len(e.Context.SourceLines) > 0 {
		sb.WriteString(formatSourceContext(e.Context))
	}

	// Suggestion if available
	if e.Suggestion != nil {
		sb.WriteString(formatSuggestion(*e.Suggestion))
	}

	// Related errors if any
	if len(e.RelatedErrors) > 0 {
		sb.WriteString(fmt.Sprintf("\n%sRelated errors:%s\n", colorBold, colorReset))
		for i, related := range e.RelatedErrors {
			sb.WriteString(fmt.Sprintf("  %d. %s:%d:%d: %s\n",
				i+1,
				related.Location.File,
				related.Location.Line,
				related.Location.Column,
				related.Message))
		}
	}

	return sb.String()
}

// formatSourceContext formats the source code context with highlighting
func formatSourceContext(ctx ErrorContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("   %s|%s\n", colorBlue, colorReset))

	for i, line := range ctx.SourceLines {
		lineNum := i + 1
		isErrorLine := i == ctx.Highlight.Line

		if isErrorLine {
			// Error line with highlighting
			sb.WriteString(fmt.Sprintf("%s%2d%s %s|%s %s\n",
				colorBlue,
				lineNum,
				colorReset,
				colorBlue,
				colorReset,
				line))

			// Highlight marker (^^^)
			sb.WriteString(fmt.Sprintf("   %s|%s ",
				colorBlue,
				colorReset))

			// Spaces before the highlight
			for j := 0; j < ctx.Highlight.Start; j++ {
				sb.WriteString(" ")
			}

			// Highlight markers
			highlightLength := ctx.Highlight.End - ctx.Highlight.Start
			if highlightLength <= 0 {
				highlightLength = 1
			}
			sb.WriteString(fmt.Sprintf("%s%s%s\n",
				colorRed,
				strings.Repeat("^", highlightLength),
				colorReset))
		} else {
			// Regular context line
			sb.WriteString(fmt.Sprintf("%s%2d%s %s|%s %s\n",
				colorGray,
				lineNum,
				colorReset,
				colorBlue,
				colorReset,
				line))
		}
	}

	sb.WriteString(fmt.Sprintf("   %s|%s\n", colorBlue, colorReset))

	return sb.String()
}

// formatSuggestion formats a fix suggestion
func formatSuggestion(suggestion FixSuggestion) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%sHelp:%s %s\n",
		colorBold+colorCyan,
		colorReset,
		suggestion.Description))

	if suggestion.NewCode != "" {
		sb.WriteString(fmt.Sprintf("%sSuggestion:%s\n",
			colorBold+colorCyan,
			colorReset))

		// Show the suggested fix
		lines := strings.Split(suggestion.NewCode, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("    %s\n", line))
		}

		// Show confidence if less than 100%
		if suggestion.Confidence < 1.0 {
			confidencePercent := int(suggestion.Confidence * 100)
			sb.WriteString(fmt.Sprintf("%s(Confidence: %d%%)%s\n",
				colorGray,
				confidencePercent,
				colorReset))
		}
	}

	return sb.String()
}

// getSeverityColor returns the ANSI color for a severity level
func getSeverityColor(severity Severity) string {
	switch severity {
	case Info:
		return colorBlue
	case Warning:
		return colorYellow
	case Error:
		return colorRed
	case Fatal:
		return colorRed + colorBold
	default:
		return colorReset
	}
}

// FormatSummary formats a summary of errors and warnings
func FormatSummary(errorCount, warningCount int) string {
	var parts []string

	if errorCount > 0 {
		parts = append(parts, fmt.Sprintf("%s%d error(s)%s",
			colorRed,
			errorCount,
			colorReset))
	}

	if warningCount > 0 {
		parts = append(parts, fmt.Sprintf("%s%d warning(s)%s",
			colorYellow,
			warningCount,
			colorReset))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%sNo errors or warnings%s\n", colorBlue, colorReset)
	}

	return fmt.Sprintf("\n%sCompilation failed with %s%s\n",
		colorBold,
		strings.Join(parts, " and "),
		colorReset)
}

// StripColors removes ANSI color codes from a string (useful for testing)
func StripColors(s string) string {
	// Remove all ANSI escape sequences
	result := s
	result = strings.ReplaceAll(result, colorReset, "")
	result = strings.ReplaceAll(result, colorRed, "")
	result = strings.ReplaceAll(result, colorYellow, "")
	result = strings.ReplaceAll(result, colorBlue, "")
	result = strings.ReplaceAll(result, colorCyan, "")
	result = strings.ReplaceAll(result, colorGray, "")
	result = strings.ReplaceAll(result, colorBold, "")

	// Remove any remaining escape sequences
	for strings.Contains(result, "\033[") {
		start := strings.Index(result, "\033[")
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	return result
}
