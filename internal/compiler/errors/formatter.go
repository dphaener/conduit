package errors

import (
	"fmt"
	"strings"
)

// FormatError returns a human-readable error message for terminal output
func FormatError(e *CompilerError) string {
	var b strings.Builder

	// Severity icon
	icon := severityIcon(e.Severity)

	// Error header
	file := e.File
	if file == "" {
		file = "<source>"
	}

	categoryName := categoryDisplayName(e.Category)

	fmt.Fprintf(&b, "%s %s in %s\n", icon, categoryName, file)

	// Location
	fmt.Fprintf(&b, "Line %d, Column %d:\n", e.Location.Line, e.Location.Column)

	// Source context (if available)
	if e.Context != nil && len(e.Context.SourceLines) > 0 {
		for i, line := range e.Context.SourceLines {
			// Calculate line number for display
			lineNum := e.Location.Line - 1 + i
			if i == 1 {
				// This is the error line - add arrow
				fmt.Fprintf(&b, "%s  %s ← %s\n", formatLineNumber(lineNum), line, e.Message)
			} else {
				fmt.Fprintf(&b, "%s  %s\n", formatLineNumber(lineNum), line)
			}
		}
	} else {
		// No context, just show message
		fmt.Fprintf(&b, "  %s\n", e.Message)
	}

	// Expected vs Actual (if provided)
	if e.Expected != "" || e.Actual != "" {
		b.WriteString("\n")
		if e.Expected != "" {
			fmt.Fprintf(&b, "  Expected: %s\n", e.Expected)
		}
		if e.Actual != "" {
			fmt.Fprintf(&b, "  Actual:   %s\n", e.Actual)
		}
	}

	// Suggestion (if provided)
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n💡 %s\n", e.Suggestion)
	}

	// Examples (if provided)
	if len(e.Examples) > 0 {
		b.WriteString("\nQuick Fixes:\n")
		for i, example := range e.Examples {
			fmt.Fprintf(&b, "  %d. %s\n", i+1, example)
		}
	}

	// Documentation link
	if e.Documentation != "" {
		fmt.Fprintf(&b, "\nLearn more: %s\n", e.Documentation)
	}

	return b.String()
}

// FormatErrorList returns a formatted string of all errors
func FormatErrorList(errors ErrorList) string {
	if len(errors) == 0 {
		return "no errors"
	}

	var b strings.Builder

	// Summary header
	errCount, warnCount, infoCount := errors.ErrorCount()
	fmt.Fprintf(&b, "Compilation failed with %d error(s), %d warning(s), %d info\n\n",
		errCount, warnCount, infoCount)

	// Format each error
	for i, err := range errors {
		if i > 0 {
			b.WriteString("\n" + strings.Repeat("-", 80) + "\n\n")
		}
		b.WriteString(err.Format())
	}

	return b.String()
}

// FormatCompact returns a compact one-line error format
func FormatCompact(e *CompilerError) string {
	file := e.File
	if file == "" {
		file = "<source>"
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s [%s]",
		file, e.Location.Line, e.Location.Column,
		e.Severity, e.Message, e.Code)
}

// severityIcon returns the emoji/icon for a severity level
func severityIcon(severity ErrorSeverity) string {
	switch severity {
	case SeverityError:
		return "❌"
	case SeverityWarning:
		return "⚠️ "
	case SeverityInfo:
		return "ℹ️ "
	default:
		return "❓"
	}
}

// categoryDisplayName returns a human-readable category name
func categoryDisplayName(category ErrorCategory) string {
	switch category {
	case CategorySyntax:
		return "Syntax Error"
	case CategoryType:
		return "Type Error"
	case CategorySemantic:
		return "Semantic Error"
	case CategoryRelationship:
		return "Relationship Error"
	case CategoryPattern:
		return "Pattern Warning"
	case CategoryValidation:
		return "Validation Error"
	case CategoryCodeGen:
		return "Code Generation Error"
	case CategoryOptimization:
		return "Optimization Hint"
	default:
		return "Compiler Error"
	}
}

// formatLineNumber formats a line number for display
func formatLineNumber(lineNum int) string {
	return fmt.Sprintf("%3d |", lineNum)
}
