// Package errors provides structured error handling for the Conduit compiler.
// It defines error codes, categories, and formatting for both human-readable
// terminal output and machine-parseable JSON for LLM consumption.
package errors

import (
	"encoding/json"
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// ErrorCode represents a unique error code in the Conduit compiler
type ErrorCode string

// ErrorCategory represents the category of compiler error
type ErrorCategory string

const (
	// CategorySyntax represents syntax errors (SYN001-099)
	CategorySyntax ErrorCategory = "syntax"
	// CategoryType represents type errors (TYP100-199)
	CategoryType ErrorCategory = "type"
	// CategorySemantic represents semantic errors (SEM200-299)
	CategorySemantic ErrorCategory = "semantic"
	// CategoryRelationship represents relationship errors (REL300-399)
	CategoryRelationship ErrorCategory = "relationship"
	// CategoryPattern represents pattern warnings (PAT400-499)
	CategoryPattern ErrorCategory = "pattern"
	// CategoryValidation represents validation errors (VAL500-599)
	CategoryValidation ErrorCategory = "validation"
	// CategoryCodeGen represents code generation errors (GEN600-699)
	CategoryCodeGen ErrorCategory = "codegen"
	// CategoryOptimization represents optimization hints (OPT700-799)
	CategoryOptimization ErrorCategory = "optimization"
)

// ErrorSeverity indicates the severity level of an error
type ErrorSeverity string

const (
	// SeverityError indicates an error that prevents compilation
	SeverityError ErrorSeverity = "error"
	// SeverityWarning indicates a warning that suggests potential issues
	SeverityWarning ErrorSeverity = "warning"
	// SeverityInfo indicates informational messages (hints, optimizations)
	SeverityInfo ErrorSeverity = "info"
)

// ErrorContext provides source code context for an error
type ErrorContext struct {
	// Current is the line of code where the error occurred
	Current string `json:"current"`
	// SourceLines is a snippet of source code (before, error line, after)
	SourceLines []string `json:"source_lines"`
}

// CompilerError represents a structured compiler error with comprehensive information
// for both human-readable output and LLM consumption
type CompilerError struct {
	// Code is the unique error code (e.g., "TYP101", "SYN001")
	Code ErrorCode `json:"code"`
	// Type is a machine-readable error type identifier
	Type string `json:"type"`
	// Category is the error category
	Category ErrorCategory `json:"category"`
	// Severity is the error severity level
	Severity ErrorSeverity `json:"severity"`
	// Message is the primary error message
	Message string `json:"message"`
	// Location is the source location of the error
	Location ast.SourceLocation `json:"location"`
	// File is the source file name (optional)
	File string `json:"file,omitempty"`
	// Context provides source code context
	Context *ErrorContext `json:"context,omitempty"`
	// Expected describes what was expected (optional)
	Expected string `json:"expected,omitempty"`
	// Actual describes what was actually found (optional)
	Actual string `json:"actual,omitempty"`
	// Suggestion provides a hint for fixing the error (optional)
	Suggestion string `json:"suggestion,omitempty"`
	// Examples provides example fixes (optional)
	Examples []string `json:"examples,omitempty"`
	// Documentation is a URL to detailed error documentation
	Documentation string `json:"documentation,omitempty"`
}

// Error implements the error interface
func (e *CompilerError) Error() string {
	return e.Format()
}

// Format returns a human-readable error message for terminal output
func (e *CompilerError) Format() string {
	return FormatError(e)
}

// ToJSON returns the error as a JSON string for LLM consumption
func (e *CompilerError) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// WithFile sets the source file name for the error
func (e *CompilerError) WithFile(file string) *CompilerError {
	e.File = file
	return e
}

// WithContext sets the source code context for the error
func (e *CompilerError) WithContext(current string, sourceLines []string) *CompilerError {
	e.Context = &ErrorContext{
		Current:     current,
		SourceLines: sourceLines,
	}
	return e
}

// WithExpected sets the expected value for the error
func (e *CompilerError) WithExpected(expected string) *CompilerError {
	e.Expected = expected
	return e
}

// WithActual sets the actual value for the error
func (e *CompilerError) WithActual(actual string) *CompilerError {
	e.Actual = actual
	return e
}

// WithSuggestion sets a suggestion for fixing the error
func (e *CompilerError) WithSuggestion(suggestion string) *CompilerError {
	e.Suggestion = suggestion
	return e
}

// WithExamples sets example fixes for the error
func (e *CompilerError) WithExamples(examples ...string) *CompilerError {
	e.Examples = examples
	return e
}

// ErrorList is a collection of compiler errors
type ErrorList []*CompilerError

// Error implements the error interface
func (el ErrorList) Error() string {
	if len(el) == 0 {
		return "no errors"
	}
	return FormatErrorList(el)
}

// HasErrors returns true if the list contains any errors (excludes warnings/info)
func (el ErrorList) HasErrors() bool {
	for _, err := range el {
		if err.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the list contains any warnings
func (el ErrorList) HasWarnings() bool {
	for _, err := range el {
		if err.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// ToJSON returns all errors as a JSON array
func (el ErrorList) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(el, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ErrorCount returns the number of errors by severity
func (el ErrorList) ErrorCount() (errors, warnings, info int) {
	for _, err := range el {
		switch err.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		case SeverityInfo:
			info++
		}
	}
	return
}

// documentationURL returns the documentation URL for an error code
func documentationURL(code ErrorCode) string {
	return fmt.Sprintf("https://docs.conduit-lang.org/errors/%s", code)
}

// newError creates a new CompilerError with the given parameters
func newError(
	code ErrorCode,
	typ string,
	category ErrorCategory,
	severity ErrorSeverity,
	message string,
	loc ast.SourceLocation,
) *CompilerError {
	return &CompilerError{
		Code:          code,
		Type:          typ,
		Category:      category,
		Severity:      severity,
		Message:       message,
		Location:      loc,
		Documentation: documentationURL(code),
	}
}
