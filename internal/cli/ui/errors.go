package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// ErrorLevel represents the severity of an error message
type ErrorLevel int

const (
	ErrorLevelError ErrorLevel = iota
	ErrorLevelWarning
	ErrorLevelInfo
)

// ErrorOptions configures the error message formatting
type ErrorOptions struct {
	Level        ErrorLevel
	Context      string
	Problem      string
	Consequence  string
	Suggestions  []string
	HelpCommands []string
	NoColor      bool
}

// FormatError creates a standardized error message with suggestions and help commands
//
// Example output:
//
//	❌ RESOURCE NOT FOUND: Pst
//	   Cannot find resource 'Pst'.
//
//	   Did you mean: Post, User, Product?
//
//	   → See all resources: conduit introspect resources
//	   → Get help: conduit introspect --help
func FormatError(opts ErrorOptions) string {
	var b strings.Builder

	// Determine colors and symbol based on level
	var headerColor, bodyColor *color.Color
	var symbol string

	switch opts.Level {
	case ErrorLevelError:
		headerColor = color.New(color.FgRed, color.Bold)
		bodyColor = color.New(color.FgRed)
		symbol = "❌"
	case ErrorLevelWarning:
		headerColor = color.New(color.FgYellow, color.Bold)
		bodyColor = color.New(color.FgYellow)
		symbol = "⚠️"
	case ErrorLevelInfo:
		headerColor = color.New(color.FgCyan, color.Bold)
		bodyColor = color.New(color.FgCyan)
		symbol = "ℹ️"
	}

	// Disable colors if requested
	if opts.NoColor {
		headerColor.DisableColor()
		bodyColor.DisableColor()
	}

	// Header line with context
	if opts.Context != "" {
		headerColor.Fprintf(&b, "%s %s: %s\n", symbol, strings.ToUpper(opts.Context), opts.Problem)
	} else {
		headerColor.Fprintf(&b, "%s %s\n", symbol, opts.Problem)
	}

	// Problem description with indentation
	if opts.Problem != "" && opts.Context != "" {
		bodyColor.Fprintf(&b, "   %s\n", opts.Problem)
	}

	// Consequence (if provided)
	if opts.Consequence != "" {
		b.WriteString("\n")
		bodyColor.Fprintf(&b, "   %s\n", opts.Consequence)
	}

	// Suggestions
	if len(opts.Suggestions) > 0 {
		b.WriteString("\n")
		yellow := color.New(color.FgYellow)
		if opts.NoColor {
			yellow.DisableColor()
		}
		yellow.Fprintf(&b, "   Did you mean: %s?\n", strings.Join(opts.Suggestions, ", "))
	}

	// Help commands
	if len(opts.HelpCommands) > 0 {
		b.WriteString("\n")
		cyan := color.New(color.FgCyan)
		if opts.NoColor {
			cyan.DisableColor()
		}
		for _, cmd := range opts.HelpCommands {
			cyan.Fprintf(&b, "   → %s\n", cmd)
		}
	}

	return b.String()
}

// WriteError writes a formatted error message to the writer
func WriteError(w io.Writer, opts ErrorOptions) {
	fmt.Fprint(w, FormatError(opts))
}

// FormatSuccess creates a success message
func FormatSuccess(message string, noColor bool) string {
	green := color.New(color.FgGreen, color.Bold)
	if noColor {
		green.DisableColor()
	}
	return green.Sprintf("✓ %s", message)
}

// WriteSuccess writes a success message to the writer
func WriteSuccess(w io.Writer, message string, noColor bool) {
	fmt.Fprintln(w, FormatSuccess(message, noColor))
}

// ResourceNotFoundError creates a standardized resource not found error
func ResourceNotFoundError(resourceName string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:       ErrorLevelError,
		Context:     "RESOURCE NOT FOUND",
		Problem:     fmt.Sprintf("Cannot find resource '%s'.", resourceName),
		Suggestions: suggestions,
		HelpCommands: []string{
			"See all resources: conduit introspect resources",
			"Get help: conduit introspect --help",
		},
		NoColor: noColor,
	}
	return FormatError(opts)
}

// PatternNotFoundError creates a standardized pattern not found error
func PatternNotFoundError(patternName string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:       ErrorLevelError,
		Context:     "PATTERN NOT FOUND",
		Problem:     fmt.Sprintf("Cannot find pattern '%s'.", patternName),
		Suggestions: suggestions,
		HelpCommands: []string{
			"See all patterns: conduit introspect patterns",
			"Get help: conduit introspect patterns --help",
		},
		NoColor: noColor,
	}
	return FormatError(opts)
}

// BuildError creates a standardized build error
func BuildError(message string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:        ErrorLevelError,
		Context:      "BUILD FAILED",
		Problem:      message,
		Suggestions:  suggestions,
		HelpCommands: []string{
			"Check syntax: conduit format --check",
			"Get help: conduit build --help",
		},
		NoColor: noColor,
	}
	return FormatError(opts)
}

// MigrationError creates a standardized migration error
func MigrationError(message string, consequence string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:        ErrorLevelError,
		Context:      "MIGRATION FAILED",
		Problem:      message,
		Consequence:  consequence,
		Suggestions:  suggestions,
		HelpCommands: []string{
			"Check migration status: conduit migrate status",
			"Rollback: conduit migrate rollback",
			"Get help: conduit migrate --help",
		},
		NoColor: noColor,
	}
	return FormatError(opts)
}

// ConfigError creates a standardized configuration error
func ConfigError(message string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:        ErrorLevelError,
		Context:      "CONFIGURATION ERROR",
		Problem:      message,
		Suggestions:  suggestions,
		HelpCommands: []string{
			"View config: cat conduit.yaml",
			"Get help: conduit --help",
		},
		NoColor: noColor,
	}
	return FormatError(opts)
}

// Warning creates a standardized warning message
func Warning(message string, suggestions []string, noColor bool) string {
	opts := ErrorOptions{
		Level:       ErrorLevelWarning,
		Problem:     message,
		Suggestions: suggestions,
		NoColor:     noColor,
	}
	return FormatError(opts)
}

// Info creates a standardized info message
func Info(message string, noColor bool) string {
	opts := ErrorOptions{
		Level:   ErrorLevelInfo,
		Problem: message,
		NoColor: noColor,
	}
	return FormatError(opts)
}
