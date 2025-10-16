package parser

import "fmt"

// ParseError represents a parsing error
type ParseError struct {
	Message  string
	Location SourceLocation
}

// Error implements the error interface
func (e ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Location.File, e.Location.Line, e.Location.Column, e.Message)
}

// ErrorCode returns a unique error code for this error type
func (e ParseError) ErrorCode() string {
	return "SYN001"
}

// Severity returns the severity level
func (e ParseError) Severity() string {
	return "error"
}

// ToJSON converts the error to a JSON-compatible structure
func (e ParseError) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"code":     e.ErrorCode(),
		"type":     "syntax",
		"severity": e.Severity(),
		"file":     e.Location.File,
		"line":     e.Location.Line,
		"column":   e.Location.Column,
		"message":  e.Message,
	}
}

// ParseErrorList is a collection of parse errors
type ParseErrorList []ParseError

// Error implements the error interface for error lists
func (el ParseErrorList) Error() string {
	if len(el) == 0 {
		return "no errors"
	}
	if len(el) == 1 {
		return el[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", el[0].Error(), len(el)-1)
}

// HasErrors returns true if there are any errors
func (el ParseErrorList) HasErrors() bool {
	return len(el) > 0
}

// Count returns the number of errors
func (el ParseErrorList) Count() int {
	return len(el)
}

// ToJSON converts all errors to JSON-compatible structures
func (el ParseErrorList) ToJSON() map[string]interface{} {
	errors := make([]map[string]interface{}, len(el))
	for i, err := range el {
		errors[i] = err.ToJSON()
	}

	return map[string]interface{}{
		"status": "error",
		"errors": errors,
	}
}

// Format formats all errors as a human-readable string
func (el ParseErrorList) Format() string {
	if len(el) == 0 {
		return "No errors"
	}

	result := fmt.Sprintf("Found %d parsing error(s):\n\n", len(el))
	for i, err := range el {
		result += fmt.Sprintf("%d. %s\n", i+1, err.Error())
	}
	return result
}
