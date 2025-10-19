package router

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error  ErrorDetail `json:"error"`
	Status int         `json:"status"`
	Path   string      `json:"path,omitempty"`
	Method string      `json:"method,omitempty"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorHandler provides default error handlers
type ErrorHandler struct {
	// Include detailed errors in responses (disable in production)
	ShowDetails bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(showDetails bool) *ErrorHandler {
	return &ErrorHandler{
		ShowDetails: showDetails,
	}
}

// NotFoundHandler returns a handler for 404 Not Found errors
func (eh *ErrorHandler) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "The requested resource was not found",
			},
			Status: http.StatusNotFound,
			Path:   r.URL.Path,
			Method: r.Method,
		}

		if eh.ShowDetails {
			resp.Error.Details = map[string]interface{}{
				"path":   r.URL.Path,
				"method": r.Method,
			}
		}

		writeJSONError(w, http.StatusNotFound, resp)
	}
}

// MethodNotAllowedHandler returns a handler for 405 Method Not Allowed errors
func (eh *ErrorHandler) MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "METHOD_NOT_ALLOWED",
				Message: fmt.Sprintf("Method %s is not allowed for this resource", r.Method),
			},
			Status: http.StatusMethodNotAllowed,
			Path:   r.URL.Path,
			Method: r.Method,
		}

		if eh.ShowDetails {
			resp.Error.Details = map[string]interface{}{
				"path":            r.URL.Path,
				"method":          r.Method,
				"allowed_methods": getAllowedMethods(r),
			}
		}

		writeJSONError(w, http.StatusMethodNotAllowed, resp)
	}
}

// BadRequestHandler returns a handler for 400 Bad Request errors
func BadRequestHandler(message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: message,
			},
			Status: http.StatusBadRequest,
		}
		writeJSONError(w, http.StatusBadRequest, resp)
	}
}

// UnprocessableEntityHandler returns a handler for 422 Unprocessable Entity errors
func UnprocessableEntityHandler(message string, details map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "UNPROCESSABLE_ENTITY",
				Message: message,
				Details: details,
			},
			Status: http.StatusUnprocessableEntity,
		}
		writeJSONError(w, http.StatusUnprocessableEntity, resp)
	}
}

// InternalServerErrorHandler returns a handler for 500 Internal Server Error
func InternalServerErrorHandler(err error, showDetails bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		message := "An internal server error occurred"
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: message,
			},
			Status: http.StatusInternalServerError,
		}

		if showDetails && err != nil {
			resp.Error.Details = map[string]interface{}{
				"error": err.Error(),
			}
		}

		writeJSONError(w, http.StatusInternalServerError, resp)
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		Status: status,
	}
	writeJSONError(w, status, resp)
}

// WriteErrorWithDetails writes an error response with additional details
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Status: status,
	}
	writeJSONError(w, status, resp)
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, status int, resp ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp) // Error is logged elsewhere
}

// getAllowedMethods extracts allowed methods from the response (if available)
func getAllowedMethods(r *http.Request) []string {
	// This is a simplified implementation
	// In a real implementation, this would inspect the router to find allowed methods
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
}

// Common error helpers

// NotFound writes a 404 Not Found response
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}
	WriteError(w, http.StatusNotFound, "NOT_FOUND", message)
}

// BadRequest writes a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Bad request"
	}
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized writes a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403 Forbidden response
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Forbidden"
	}
	WriteError(w, http.StatusForbidden, "FORBIDDEN", message)
}

// UnprocessableEntity writes a 422 Unprocessable Entity response
func UnprocessableEntity(w http.ResponseWriter, message string, details map[string]interface{}) {
	if message == "" {
		message = "Validation failed"
	}
	WriteErrorWithDetails(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, details)
}

// InternalServerError writes a 500 Internal Server Error response
func InternalServerError(w http.ResponseWriter, err error) {
	message := "An internal server error occurred"
	details := make(map[string]interface{})
	if err != nil {
		details["error"] = err.Error()
	}
	WriteErrorWithDetails(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, details)
}

// SetupDefaultErrorHandlers configures the router with default error handlers
func SetupDefaultErrorHandlers(r *Router, showDetails bool) {
	eh := NewErrorHandler(showDetails)
	r.NotFound(eh.NotFoundHandler())
	r.MethodNotAllowed(eh.MethodNotAllowedHandler())
}
