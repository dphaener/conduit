package errors

import (
	"fmt"
	"strings"
)

// MaxErrors is the maximum number of errors to collect before stopping
const MaxErrors = 100

// ErrorRecovery manages error collection and recovery
type ErrorRecovery struct {
	errors   []CompilerError
	warnings []CompilerError
	maxCount int
}

// NewErrorRecovery creates a new ErrorRecovery instance
func NewErrorRecovery() *ErrorRecovery {
	return &ErrorRecovery{
		errors:   make([]CompilerError, 0),
		warnings: make([]CompilerError, 0),
		maxCount: MaxErrors,
	}
}

// NewErrorRecoveryWithMax creates a new ErrorRecovery with custom max count
func NewErrorRecoveryWithMax(maxCount int) *ErrorRecovery {
	return &ErrorRecovery{
		errors:   make([]CompilerError, 0),
		warnings: make([]CompilerError, 0),
		maxCount: maxCount,
	}
}

// Recover adds an error to the collection
func (r *ErrorRecovery) Recover(err CompilerError) {
	// Stop if we've already hit the max error count
	if len(r.errors) >= r.maxCount && (err.IsError() || err.IsFatal()) {
		// Don't add more errors, but allow warnings
		return
	}

	// Enrich error with context if file is available
	if err.Location.File != "" && len(err.Context.SourceLines) == 0 {
		err = EnrichErrorFromFile(err)
	}

	// Add to appropriate collection
	if err.IsWarning() || err.IsInfo() {
		r.warnings = append(r.warnings, err)
	} else {
		r.errors = append(r.errors, err)
	}
}

// RecoverMultiple adds multiple errors to the collection
func (r *ErrorRecovery) RecoverMultiple(errs []CompilerError) {
	for _, err := range errs {
		if len(r.errors) >= r.maxCount {
			break
		}
		r.Recover(err)
	}
}

// HasErrors returns true if there are any errors (not just warnings)
func (r *ErrorRecovery) HasErrors() bool {
	return len(r.errors) > 0
}

// HasWarnings returns true if there are any warnings
func (r *ErrorRecovery) HasWarnings() bool {
	return len(r.warnings) > 0
}

// HasFatals returns true if there are any fatal errors
func (r *ErrorRecovery) HasFatals() bool {
	for _, err := range r.errors {
		if err.IsFatal() {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of errors
func (r *ErrorRecovery) ErrorCount() int {
	return len(r.errors)
}

// WarningCount returns the number of warnings
func (r *ErrorRecovery) WarningCount() int {
	return len(r.warnings)
}

// TotalCount returns the total number of errors and warnings
func (r *ErrorRecovery) TotalCount() int {
	return len(r.errors) + len(r.warnings)
}

// GetErrors returns all errors
func (r *ErrorRecovery) GetErrors() []CompilerError {
	return r.errors
}

// GetWarnings returns all warnings
func (r *ErrorRecovery) GetWarnings() []CompilerError {
	return r.warnings
}

// GetAll returns all errors and warnings combined
func (r *ErrorRecovery) GetAll() []CompilerError {
	all := make([]CompilerError, 0, len(r.errors)+len(r.warnings))
	all = append(all, r.errors...)
	all = append(all, r.warnings...)
	return all
}

// Clear resets all errors and warnings
func (r *ErrorRecovery) Clear() {
	r.errors = make([]CompilerError, 0)
	r.warnings = make([]CompilerError, 0)
}

// FormatForTerminal formats all errors for terminal output
func (r *ErrorRecovery) FormatForTerminal() string {
	var sb strings.Builder

	// Format all errors
	for i, err := range r.errors {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(err.FormatForTerminal())
	}

	// Format all warnings
	for i, warn := range r.warnings {
		if len(r.errors) > 0 || i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(warn.FormatForTerminal())
	}

	// Add summary
	if r.TotalCount() > 0 {
		sb.WriteString(FormatSummary(len(r.errors), len(r.warnings)))
	}

	// Add truncation notice if we hit the limit
	if len(r.errors) >= r.maxCount {
		sb.WriteString(fmt.Sprintf("\n%sNote: Error limit reached (%d). Additional errors not shown.%s\n",
			colorYellow,
			r.maxCount,
			colorReset))
	}

	return sb.String()
}

// FormatAsJSON formats all errors as JSON
func (r *ErrorRecovery) FormatAsJSON() (string, error) {
	all := r.GetAll()
	return FormatErrorsAsJSON(all)
}

// FormatAsJSONCompact formats all errors as compact JSON
func (r *ErrorRecovery) FormatAsJSONCompact() (string, error) {
	all := r.GetAll()
	return FormatErrorsAsJSONCompact(all)
}

// FirstError returns the first error, or nil if there are none
func (r *ErrorRecovery) FirstError() *CompilerError {
	if len(r.errors) == 0 {
		return nil
	}
	return &r.errors[0]
}

// FirstFatal returns the first fatal error, or nil if there are none
func (r *ErrorRecovery) FirstFatal() *CompilerError {
	for _, err := range r.errors {
		if err.IsFatal() {
			return &err
		}
	}
	return nil
}

// Error implements the error interface
func (r *ErrorRecovery) Error() string {
	if len(r.errors) == 0 && len(r.warnings) == 0 {
		return "no errors"
	}

	if len(r.errors) == 1 && len(r.warnings) == 0 {
		return r.errors[0].Error()
	}

	return fmt.Sprintf("%d error(s) and %d warning(s)", len(r.errors), len(r.warnings))
}

// Summary returns a human-readable summary
func (r *ErrorRecovery) Summary() string {
	if len(r.errors) == 0 && len(r.warnings) == 0 {
		return "No errors or warnings"
	}

	var parts []string
	if len(r.errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", len(r.errors)))
	}
	if len(r.warnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", len(r.warnings)))
	}

	return "Found " + strings.Join(parts, " and ")
}

// GetErrorsByPhase returns errors for a specific phase
func (r *ErrorRecovery) GetErrorsByPhase(phase string) []CompilerError {
	var result []CompilerError
	for _, err := range r.errors {
		if err.Phase == phase {
			result = append(result, err)
		}
	}
	return result
}

// GetErrorsByCode returns errors with a specific error code
func (r *ErrorRecovery) GetErrorsByCode(code string) []CompilerError {
	var result []CompilerError
	for _, err := range r.errors {
		if err.Code == code {
			result = append(result, err)
		}
	}
	for _, warn := range r.warnings {
		if warn.Code == code {
			result = append(result, warn)
		}
	}
	return result
}

// GetErrorsBySeverity returns errors with a specific severity
func (r *ErrorRecovery) GetErrorsBySeverity(severity Severity) []CompilerError {
	var result []CompilerError
	for _, err := range r.errors {
		if err.Severity == severity {
			result = append(result, err)
		}
	}
	for _, warn := range r.warnings {
		if warn.Severity == severity {
			result = append(result, warn)
		}
	}
	return result
}
