package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/validation"
)

func TestRenderError(t *testing.T) {
	w := httptest.NewRecorder()
	err := fmt.Errorf("something went wrong")

	RenderError(w, http.StatusInternalServerError, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusInternalServerError)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "something went wrong" {
		t.Errorf("message = %v, want 'something went wrong'", resp.Message)
	}

	if resp.Code != "internal_error" {
		t.Errorf("code = %v, want 'internal_error'", resp.Code)
	}
}

func TestRenderErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	err := fmt.Errorf("custom error")

	RenderErrorWithCode(w, http.StatusBadRequest, err, "custom_code")

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code != "custom_code" {
		t.Errorf("code = %v, want 'custom_code'", resp.Code)
	}
}

func TestRenderValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	validationErr := validation.NewValidationErrors()
	validationErr.Add("email", "must be a valid email address")
	validationErr.Add("age", "must be at least 18")

	RenderValidationError(w, validationErr)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusUnprocessableEntity)
	}

	var resp ValidationErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "validation_error" {
		t.Errorf("code = %v, want 'validation_error'", resp.Code)
	}

	if len(resp.Fields) != 2 {
		t.Errorf("expected 2 field errors, got %v", len(resp.Fields))
	}

	if len(resp.Fields["email"]) == 0 {
		t.Error("email field error missing")
	}

	if len(resp.Fields["age"]) == 0 {
		t.Error("age field error missing")
	}
}

func TestRenderError_WithValidationErrors(t *testing.T) {
	w := httptest.NewRecorder()

	// RenderError should detect validation errors and format appropriately
	validationErr := validation.NewValidationErrors()
	validationErr.Add("name", "is required")

	RenderError(w, http.StatusBadRequest, validationErr)

	// Should use validation error formatting
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status code = %v, want %v for validation errors", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestRenderBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	RenderBadRequest(w, "invalid request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !strings.Contains(resp.Message, "invalid request") {
		t.Errorf("message should contain 'invalid request', got: %v", resp.Message)
	}
}

func TestRenderUnauthorized(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		w := httptest.NewRecorder()
		RenderUnauthorized(w, "invalid token")

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusUnauthorized)
		}

		var resp ErrorResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if !strings.Contains(resp.Message, "invalid token") {
			t.Errorf("message should contain 'invalid token', got: %v", resp.Message)
		}
	})

	t.Run("with default message", func(t *testing.T) {
		w := httptest.NewRecorder()
		RenderUnauthorized(w, "")

		var resp ErrorResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if !strings.Contains(resp.Message, "Authentication required") {
			t.Errorf("should use default message, got: %v", resp.Message)
		}
	})
}

func TestRenderForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	RenderForbidden(w, "insufficient permissions")

	if w.Code != http.StatusForbidden {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusForbidden)
	}
}

func TestRenderNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	RenderNotFound(w, "user not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestRenderMethodNotAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	allowedMethods := []string{"GET", "POST"}

	RenderMethodNotAllowed(w, allowedMethods)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}

	allow := w.Header().Get("Allow")
	if !strings.Contains(allow, "GET") || !strings.Contains(allow, "POST") {
		t.Errorf("Allow header = %v, should contain GET and POST", allow)
	}
}

func TestRenderConflict(t *testing.T) {
	w := httptest.NewRecorder()
	RenderConflict(w, "resource already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusConflict)
	}
}

func TestRenderTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()
	RenderTooManyRequests(w, 60)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusTooManyRequests)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("Retry-After header = %v, want 60", retryAfter)
	}
}

func TestRenderInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	err := fmt.Errorf("database connection failed")

	RenderInternalError(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestErrorCodeFromStatus(t *testing.T) {
	tests := []struct {
		status       int
		expectedCode string
	}{
		{http.StatusBadRequest, "bad_request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not_found"},
		{http.StatusMethodNotAllowed, "method_not_allowed"},
		{http.StatusConflict, "conflict"},
		{http.StatusUnprocessableEntity, "unprocessable_entity"},
		{http.StatusTooManyRequests, "too_many_requests"},
		{http.StatusInternalServerError, "internal_error"},
		{http.StatusServiceUnavailable, "service_unavailable"},
		{999, "error"}, // Unknown status
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			code := errorCodeFromStatus(tt.status)
			if code != tt.expectedCode {
				t.Errorf("errorCodeFromStatus(%d) = %v, want %v", tt.status, code, tt.expectedCode)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	err := NewHTTPError(http.StatusBadRequest, "invalid input")

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("status code = %v, want %v", err.StatusCode, http.StatusBadRequest)
	}

	if err.Message != "invalid input" {
		t.Errorf("message = %v, want 'invalid input'", err.Message)
	}

	if err.Error() != "invalid input" {
		t.Errorf("Error() = %v, want 'invalid input'", err.Error())
	}
}

func TestHTTPError_WithCode(t *testing.T) {
	err := NewHTTPError(http.StatusBadRequest, "error")
	err = err.WithCode("custom_error")

	if err.Code != "custom_error" {
		t.Errorf("code = %v, want 'custom_error'", err.Code)
	}
}

func TestHTTPError_WithDetails(t *testing.T) {
	details := map[string]interface{}{
		"field":  "email",
		"reason": "invalid format",
	}

	err := NewHTTPError(http.StatusBadRequest, "error")
	err = err.WithDetails(details)

	if len(err.Details) != 2 {
		t.Errorf("expected 2 details, got %v", len(err.Details))
	}

	if err.Details["field"] != "email" {
		t.Error("details not set correctly")
	}
}

func TestHTTPError_Render(t *testing.T) {
	t.Run("without details", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := NewHTTPError(http.StatusNotFound, "not found")
		err.Render(w)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := NewHTTPError(http.StatusBadRequest, "error")
		err = err.WithDetails(map[string]interface{}{"key": "value"})
		err.Render(w)

		var resp ErrorResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if len(resp.Details) == 0 {
			t.Error("details should be included in response")
		}
	})
}

func TestCommonHTTPErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *HTTPError
		wantCode int
	}{
		{"ErrBadRequest", ErrBadRequest, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, http.StatusForbidden},
		{"ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"ErrConflict", ErrConflict, http.StatusConflict},
		{"ErrInternalServer", ErrInternalServer, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode != tt.wantCode {
				t.Errorf("%s.StatusCode = %v, want %v", tt.name, tt.err.StatusCode, tt.wantCode)
			}
		})
	}
}

func TestRenderErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	err := fmt.Errorf("validation failed")
	details := map[string]interface{}{
		"fields": []string{"email", "password"},
		"count":  2,
	}

	RenderErrorWithDetails(w, http.StatusBadRequest, err, details)

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Details) == 0 {
		t.Error("details should be included in response")
	}

	// JSON unmarshaling converts numbers to float64
	count, ok := resp.Details["count"].(float64)
	if !ok {
		t.Errorf("details count type = %T, want float64", resp.Details["count"])
	} else if count != 2 {
		t.Errorf("details count = %v, want 2", count)
	}
}

// Benchmark tests
func BenchmarkRenderError(b *testing.B) {
	err := fmt.Errorf("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		RenderError(w, http.StatusBadRequest, err)
	}
}

func BenchmarkRenderValidationError(b *testing.B) {
	validationErr := validation.NewValidationErrors()
	validationErr.Add("field1", "error 1")
	validationErr.Add("field2", "error 2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		RenderValidationError(w, validationErr)
	}
}
