package validation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationErrors contains multiple validation errors for a record
type ValidationErrors struct {
	Fields map[string][]string `json:"fields"`
}

// NewValidationErrors creates a new ValidationErrors instance
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Fields: make(map[string][]string),
	}
}

// Add adds a validation error for a specific field
func (ve *ValidationErrors) Add(field, message string) {
	if ve.Fields == nil {
		ve.Fields = make(map[string][]string)
	}
	ve.Fields[field] = append(ve.Fields[field], message)
}

// AddFieldError adds a FieldError to the validation errors
func (ve *ValidationErrors) AddFieldError(err FieldError) {
	ve.Add(err.Field, err.Message)
}

// HasErrors returns true if there are any validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Fields) > 0
}

// Count returns the total number of validation errors across all fields
func (ve *ValidationErrors) Count() int {
	count := 0
	for _, messages := range ve.Fields {
		count += len(messages)
	}
	return count
}

// Error implements the error interface
func (ve *ValidationErrors) Error() string {
	if !ve.HasErrors() {
		return "validation failed"
	}

	var messages []string
	for field, errs := range ve.Fields {
		for _, msg := range errs {
			messages = append(messages, fmt.Sprintf("  - %s: %s", field, msg))
		}
	}

	if len(messages) == 1 {
		return fmt.Sprintf("validation failed: %s", strings.TrimPrefix(messages[0], "  - "))
	}

	return fmt.Sprintf("validation failed:\n%s", strings.Join(messages, "\n"))
}

// MarshalJSON implements json.Marshaler for custom JSON serialization
func (ve *ValidationErrors) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error  string              `json:"error"`
		Fields map[string][]string `json:"fields"`
	}{
		Error:  "validation_failed",
		Fields: ve.Fields,
	})
}

// FieldError represents a validation error on a specific field
type FieldError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (fe FieldError) Error() string {
	return fmt.Sprintf("%s: %s", fe.Field, fe.Message)
}

// NewFieldError creates a new FieldError
func NewFieldError(field, message string) FieldError {
	return FieldError{
		Field:   field,
		Message: message,
	}
}
