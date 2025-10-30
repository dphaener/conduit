package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/conduit-lang/conduit/internal/orm/validation"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation errors
type ValidationErrorResponse struct {
	Error   string              `json:"error"`
	Message string              `json:"message"`
	Code    string              `json:"code"`
	Fields  map[string][]string `json:"fields"`
}

// RenderError renders a standard error response
func RenderError(w http.ResponseWriter, statusCode int, err error) {
	RenderErrorWithCode(w, statusCode, err, "")
}

// RenderErrorWithCode renders an error with a specific error code
func RenderErrorWithCode(w http.ResponseWriter, statusCode int, err error, code string) {
	// Check if it's a validation error
	if validationErr, ok := err.(*validation.ValidationErrors); ok {
		RenderValidationError(w, validationErr)
		return
	}

	// Generate error code from status if not provided
	if code == "" {
		code = errorCodeFromStatus(statusCode)
	}

	response := &ErrorResponse{
		Error:   "error",
		Message: err.Error(),
		Code:    code,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// RenderErrorWithDetails renders an error with additional details
func RenderErrorWithDetails(w http.ResponseWriter, statusCode int, err error, details map[string]interface{}) {
	response := &ErrorResponse{
		Error:   "error",
		Message: err.Error(),
		Code:    errorCodeFromStatus(statusCode),
		Details: details,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// RenderValidationError renders validation errors
func RenderValidationError(w http.ResponseWriter, validationErr *validation.ValidationErrors) {
	response := &ValidationErrorResponse{
		Error:   "validation_failed",
		Message: "The request contains invalid data",
		Code:    "validation_error",
		Fields:  validationErr.Fields,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(response)
}

// RenderBadRequest renders a 400 Bad Request error
func RenderBadRequest(w http.ResponseWriter, message string) {
	RenderError(w, http.StatusBadRequest, fmt.Errorf("%s", message))
}

// RenderUnauthorized renders a 401 Unauthorized error
func RenderUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	RenderError(w, http.StatusUnauthorized, fmt.Errorf("%s", message))
}

// RenderForbidden renders a 403 Forbidden error
func RenderForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access denied"
	}
	RenderError(w, http.StatusForbidden, fmt.Errorf("%s", message))
}

// RenderNotFound renders a 404 Not Found error
func RenderNotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}
	RenderError(w, http.StatusNotFound, fmt.Errorf("%s", message))
}

// RenderMethodNotAllowed renders a 405 Method Not Allowed error
func RenderMethodNotAllowed(w http.ResponseWriter, allowedMethods []string) {
	w.Header().Set("Allow", joinMethods(allowedMethods))
	RenderError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
}

// RenderConflict renders a 409 Conflict error
func RenderConflict(w http.ResponseWriter, message string) {
	RenderError(w, http.StatusConflict, fmt.Errorf("%s", message))
}

// RenderUnprocessableEntity renders a 422 Unprocessable Entity error
func RenderUnprocessableEntity(w http.ResponseWriter, message string) {
	RenderError(w, http.StatusUnprocessableEntity, fmt.Errorf("%s", message))
}

// RenderTooManyRequests renders a 429 Too Many Requests error
func RenderTooManyRequests(w http.ResponseWriter, retryAfter int) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	RenderError(w, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded"))
}

// RenderInternalError renders a 500 Internal Server Error
func RenderInternalError(w http.ResponseWriter, err error) {
	message := "Internal server error"
	if err != nil {
		// In production, don't expose internal error details
		// For now, we'll include them for debugging
		message = err.Error()
	}
	RenderError(w, http.StatusInternalServerError, fmt.Errorf("%s", message))
}

// RenderServiceUnavailable renders a 503 Service Unavailable error
func RenderServiceUnavailable(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	RenderError(w, http.StatusServiceUnavailable, fmt.Errorf("%s", message))
}

// errorCodeFromStatus maps HTTP status codes to error codes
func errorCodeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusPaymentRequired:
		return "payment_required"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusNotAcceptable:
		return "not_acceptable"
	case http.StatusRequestTimeout:
		return "request_timeout"
	case http.StatusConflict:
		return "conflict"
	case http.StatusGone:
		return "gone"
	case http.StatusLengthRequired:
		return "length_required"
	case http.StatusPreconditionFailed:
		return "precondition_failed"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "too_many_requests"
	case http.StatusInternalServerError:
		return "internal_error"
	case http.StatusNotImplemented:
		return "not_implemented"
	case http.StatusBadGateway:
		return "bad_gateway"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	case http.StatusGatewayTimeout:
		return "gateway_timeout"
	default:
		return "error"
	}
}

// joinMethods joins HTTP methods with comma
func joinMethods(methods []string) string {
	result := ""
	for i, method := range methods {
		if i > 0 {
			result += ", "
		}
		result += method
	}
	return result
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
	Code       string
	Details    map[string]interface{}
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Code:       errorCodeFromStatus(statusCode),
	}
}

// WithCode sets a custom error code
func (e *HTTPError) WithCode(code string) *HTTPError {
	e.Code = code
	return e
}

// WithDetails adds details to the error
func (e *HTTPError) WithDetails(details map[string]interface{}) *HTTPError {
	e.Details = details
	return e
}

// Render renders the HTTP error as a response
func (e *HTTPError) Render(w http.ResponseWriter) {
	if len(e.Details) > 0 {
		RenderErrorWithDetails(w, e.StatusCode, e, e.Details)
	} else {
		RenderErrorWithCode(w, e.StatusCode, e, e.Code)
	}
}

// Common HTTP errors
var (
	ErrBadRequest          = NewHTTPError(http.StatusBadRequest, "Bad request")
	ErrUnauthorized        = NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden           = NewHTTPError(http.StatusForbidden, "Forbidden")
	ErrNotFound            = NewHTTPError(http.StatusNotFound, "Not found")
	ErrMethodNotAllowed    = NewHTTPError(http.StatusMethodNotAllowed, "Method not allowed")
	ErrConflict            = NewHTTPError(http.StatusConflict, "Conflict")
	ErrUnprocessableEntity = NewHTTPError(http.StatusUnprocessableEntity, "Unprocessable entity")
	ErrTooManyRequests     = NewHTTPError(http.StatusTooManyRequests, "Too many requests")
	ErrInternalServer      = NewHTTPError(http.StatusInternalServerError, "Internal server error")
	ErrServiceUnavailable  = NewHTTPError(http.StatusServiceUnavailable, "Service unavailable")
)
