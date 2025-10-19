package validation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidationErrors_Add(t *testing.T) {
	errors := NewValidationErrors()

	errors.Add("title", "must be at least 5 characters")
	errors.Add("email", "must be a valid email address")
	errors.Add("title", "must not contain special characters")

	if len(errors.Fields) != 2 {
		t.Errorf("expected 2 fields with errors, got %d", len(errors.Fields))
	}

	if len(errors.Fields["title"]) != 2 {
		t.Errorf("expected 2 errors for title, got %d", len(errors.Fields["title"]))
	}

	if len(errors.Fields["email"]) != 1 {
		t.Errorf("expected 1 error for email, got %d", len(errors.Fields["email"]))
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	errors := NewValidationErrors()

	if errors.HasErrors() {
		t.Error("expected HasErrors to return false for new ValidationErrors")
	}

	errors.Add("field", "error message")

	if !errors.HasErrors() {
		t.Error("expected HasErrors to return true after adding error")
	}
}

func TestValidationErrors_Count(t *testing.T) {
	errors := NewValidationErrors()

	if errors.Count() != 0 {
		t.Errorf("expected count 0, got %d", errors.Count())
	}

	errors.Add("field1", "error 1")
	errors.Add("field1", "error 2")
	errors.Add("field2", "error 3")

	if errors.Count() != 3 {
		t.Errorf("expected count 3, got %d", errors.Count())
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *ValidationErrors
		contains []string
	}{
		{
			name: "no errors",
			setup: func() *ValidationErrors {
				return NewValidationErrors()
			},
			contains: []string{"validation failed"},
		},
		{
			name: "single error",
			setup: func() *ValidationErrors {
				errors := NewValidationErrors()
				errors.Add("title", "must be at least 5 characters")
				return errors
			},
			contains: []string{"validation failed", "title", "must be at least 5 characters"},
		},
		{
			name: "multiple errors",
			setup: func() *ValidationErrors {
				errors := NewValidationErrors()
				errors.Add("title", "must be at least 5 characters")
				errors.Add("email", "must be a valid email address")
				return errors
			},
			contains: []string{"validation failed", "title", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.setup()
			message := errors.Error()

			for _, expected := range tt.contains {
				if !strings.Contains(message, expected) {
					t.Errorf("expected error message to contain %q, got %q", expected, message)
				}
			}
		})
	}
}

func TestValidationErrors_MarshalJSON(t *testing.T) {
	errors := NewValidationErrors()
	errors.Add("title", "must be at least 5 characters")
	errors.Add("email", "must be a valid email address")

	data, err := json.Marshal(errors)
	if err != nil {
		t.Fatalf("failed to marshal ValidationErrors: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["error"] != "validation_failed" {
		t.Errorf("expected error field to be 'validation_failed', got %v", result["error"])
	}

	fields, ok := result["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("expected fields to be a map")
	}

	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}

func TestFieldError_Error(t *testing.T) {
	err := NewFieldError("email", "must be a valid email address")

	expected := "email: must be a valid email address"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidationErrors_AddFieldError(t *testing.T) {
	errors := NewValidationErrors()
	fieldErr := NewFieldError("title", "is required")

	errors.AddFieldError(fieldErr)

	if len(errors.Fields["title"]) != 1 {
		t.Errorf("expected 1 error for title, got %d", len(errors.Fields["title"]))
	}

	if errors.Fields["title"][0] != "is required" {
		t.Errorf("expected 'is required', got %q", errors.Fields["title"][0])
	}
}
