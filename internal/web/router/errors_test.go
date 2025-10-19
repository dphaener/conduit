package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorHandler(t *testing.T) {
	eh := NewErrorHandler(true)
	assert.NotNil(t, eh)
	assert.True(t, eh.ShowDetails)

	eh = NewErrorHandler(false)
	assert.False(t, eh.ShowDetails)
}

func TestNotFoundHandler(t *testing.T) {
	router := NewRouter()
	eh := NewErrorHandler(true)
	router.NotFound(eh.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "NOT_FOUND", response.Error.Code)
	assert.Equal(t, "The requested resource was not found", response.Error.Message)
	assert.Equal(t, http.StatusNotFound, response.Status)
	assert.Equal(t, "/nonexistent", response.Path)
	assert.Equal(t, http.MethodGet, response.Method)
}

func TestNotFoundHandlerWithoutDetails(t *testing.T) {
	router := NewRouter()
	eh := NewErrorHandler(false)
	router.NotFound(eh.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "NOT_FOUND", response.Error.Code)
	assert.Nil(t, response.Error.Details)
}

func TestMethodNotAllowedHandler(t *testing.T) {
	router := NewRouter()
	eh := NewErrorHandler(true)
	router.MethodNotAllowed(eh.MethodNotAllowedHandler())

	// Register a GET route
	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Try to POST (not allowed)
	req := httptest.NewRequest(http.MethodPost, "/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "METHOD_NOT_ALLOWED", response.Error.Code)
	assert.Contains(t, response.Error.Message, "POST")
	assert.Equal(t, http.StatusMethodNotAllowed, response.Status)
}

func TestBadRequestHandler(t *testing.T) {
	handler := BadRequestHandler("Invalid request format")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", response.Error.Code)
	assert.Equal(t, "Invalid request format", response.Error.Message)
	assert.Equal(t, http.StatusBadRequest, response.Status)
}

func TestUnprocessableEntityHandler(t *testing.T) {
	details := map[string]interface{}{
		"title": "Title is required",
		"body":  "Body must be at least 100 characters",
	}
	handler := UnprocessableEntityHandler("Validation failed", details)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "UNPROCESSABLE_ENTITY", response.Error.Code)
	assert.Equal(t, "Validation failed", response.Error.Message)
	assert.Equal(t, http.StatusUnprocessableEntity, response.Status)
	assert.Equal(t, "Title is required", response.Error.Details["title"])
	assert.Equal(t, "Body must be at least 100 characters", response.Error.Details["body"])
}

func TestInternalServerErrorHandler(t *testing.T) {
	err := errors.New("database connection failed")
	handler := InternalServerErrorHandler(err, true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "INTERNAL_SERVER_ERROR", response.Error.Code)
	assert.Equal(t, "An internal server error occurred", response.Error.Message)
	assert.Equal(t, http.StatusInternalServerError, response.Status)
	assert.Equal(t, "database connection failed", response.Error.Details["error"])
}

func TestInternalServerErrorHandlerWithoutDetails(t *testing.T) {
	err := errors.New("database connection failed")
	handler := InternalServerErrorHandler(err, false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Nil(t, response.Error.Details)
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "The input provided is invalid")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_INPUT", response.Error.Code)
	assert.Equal(t, "The input provided is invalid", response.Error.Message)
	assert.Equal(t, http.StatusBadRequest, response.Status)
}

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]interface{}{
		"field": "email",
		"issue": "invalid format",
	}
	WriteErrorWithDetails(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", details)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Equal(t, "Validation failed", response.Error.Message)
	assert.Equal(t, "email", response.Error.Details["field"])
	assert.Equal(t, "invalid format", response.Error.Details["issue"])
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "Post not found")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "NOT_FOUND", response.Error.Code)
	assert.Equal(t, "Post not found", response.Error.Message)
}

func TestNotFoundDefaultMessage(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "")

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Resource not found", response.Error.Message)
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "Invalid JSON")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", response.Error.Code)
	assert.Equal(t, "Invalid JSON", response.Error.Message)
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "Authentication required")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)
	assert.Equal(t, "Authentication required", response.Error.Message)
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "Access denied")

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "FORBIDDEN", response.Error.Code)
	assert.Equal(t, "Access denied", response.Error.Message)
}

func TestUnprocessableEntity(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]interface{}{
		"email": "Email is required",
	}
	UnprocessableEntity(w, "Validation failed", details)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Equal(t, "Validation failed", response.Error.Message)
	assert.Equal(t, "Email is required", response.Error.Details["email"])
}

func TestInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	err := errors.New("database error")
	InternalServerError(w, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "INTERNAL_SERVER_ERROR", response.Error.Code)
	assert.Equal(t, "An internal server error occurred", response.Error.Message)
	assert.Equal(t, "database error", response.Error.Details["error"])
}

func TestSetupDefaultErrorHandlers(t *testing.T) {
	router := NewRouter()
	SetupDefaultErrorHandlers(router, true)

	// Test 404
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "NOT_FOUND", response.Error.Code)

	// Test 405
	router.Get("/posts", func(w http.ResponseWriter, r *http.Request) {})
	req = httptest.NewRequest(http.MethodPost, "/posts", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "METHOD_NOT_ALLOWED", response.Error.Code)
}

func TestErrorResponseStructure(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]interface{}{
		"field":  "title",
		"reason": "too short",
	}
	WriteErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", details)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify structure
	assert.NotEmpty(t, response.Error.Code)
	assert.NotEmpty(t, response.Error.Message)
	assert.NotNil(t, response.Error.Details)
	assert.Equal(t, http.StatusBadRequest, response.Status)
}
