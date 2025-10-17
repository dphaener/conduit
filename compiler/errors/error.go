package errors

import (
	"encoding/json"
	"fmt"
)

// Severity represents the severity level of an error
type Severity int

const (
	Info Severity = iota
	Warning
	Error
	Fatal
)

// String returns the string representation of the severity
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	case Fatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Severity
func (s Severity) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for Severity
func (s *Severity) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	switch str {
	case "info":
		*s = Info
	case "warning":
		*s = Warning
	case "error":
		*s = Error
	case "fatal":
		*s = Fatal
	default:
		*s = Error // Default to Error if unknown
	}
	return nil
}

// SourceLocation represents a location in source code
type SourceLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Length int    `json:"length"` // For multi-character tokens
}

// ErrorContext contains surrounding code for an error
type ErrorContext struct {
	SourceLines []string  `json:"source_lines"` // 3 lines before, error line, 3 lines after
	Highlight   Highlight `json:"highlight"`    // Which part to highlight
}

// Highlight specifies which part of the context to highlight
type Highlight struct {
	Line  int `json:"line"`  // Which line in SourceLines array
	Start int `json:"start"` // Column start
	End   int `json:"end"`   // Column end
}

// FixSuggestion represents an auto-fix suggestion
type FixSuggestion struct {
	Description string  `json:"description"`
	OldCode     string  `json:"old_code"`
	NewCode     string  `json:"new_code"`
	Confidence  float64 `json:"confidence"` // 0.0 to 1.0
}

// CompilerError represents a comprehensive compiler error
type CompilerError struct {
	Phase         string           // "lexer", "parser", "type_checker", "codegen"
	Code          string           // "E001", "E002", etc.
	Message       string           // Human-readable message
	Location      SourceLocation   // File, line, column
	Severity      Severity         // Error, Warning, Info
	Context       ErrorContext     // Surrounding code
	Suggestion    *FixSuggestion   // Optional auto-fix
	RelatedErrors []CompilerError  // Cascading errors
}

// Error implements the error interface
func (e CompilerError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		e.Location.File,
		e.Location.Line,
		e.Location.Column,
		e.Code,
		e.Message)
}

// NewCompilerError creates a new CompilerError
func NewCompilerError(phase, code, message string, location SourceLocation, severity Severity) CompilerError {
	return CompilerError{
		Phase:         phase,
		Code:          code,
		Message:       message,
		Location:      location,
		Severity:      severity,
		Context:       ErrorContext{},
		Suggestion:    nil,
		RelatedErrors: []CompilerError{},
	}
}

// WithContext adds context to the error
func (e CompilerError) WithContext(ctx ErrorContext) CompilerError {
	e.Context = ctx
	return e
}

// WithSuggestion adds a fix suggestion to the error
func (e CompilerError) WithSuggestion(suggestion FixSuggestion) CompilerError {
	e.Suggestion = &suggestion
	return e
}

// WithRelatedError adds a related error
func (e CompilerError) WithRelatedError(related CompilerError) CompilerError {
	e.RelatedErrors = append(e.RelatedErrors, related)
	return e
}

// MarshalJSON implements json.Marshaler
func (e CompilerError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Phase         string          `json:"phase"`
		Code          string          `json:"code"`
		Message       string          `json:"message"`
		Severity      Severity        `json:"severity"`
		Location      SourceLocation  `json:"location"`
		Context       ErrorContext    `json:"context"`
		Suggestion    *FixSuggestion  `json:"suggestion"`
		RelatedErrors []CompilerError `json:"related_errors"`
	}{
		Phase:         e.Phase,
		Code:          e.Code,
		Message:       e.Message,
		Severity:      e.Severity,
		Location:      e.Location,
		Context:       e.Context,
		Suggestion:    e.Suggestion,
		RelatedErrors: e.RelatedErrors,
	})
}

// IsError returns true if the error is at Error or Fatal severity
func (e CompilerError) IsError() bool {
	return e.Severity == Error || e.Severity == Fatal
}

// IsWarning returns true if the error is at Warning severity
func (e CompilerError) IsWarning() bool {
	return e.Severity == Warning
}

// IsInfo returns true if the error is at Info severity
func (e CompilerError) IsInfo() bool {
	return e.Severity == Info
}

// IsFatal returns true if the error is at Fatal severity
func (e CompilerError) IsFatal() bool {
	return e.Severity == Fatal
}
