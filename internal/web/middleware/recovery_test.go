package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recovery()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["error"] != "internal_server_error" {
		t.Errorf("Expected error field, got %v", response["error"])
	}
}

func TestRecoveryWithErrorPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("custom error"))
	})

	var loggedError error
	config := RecoveryConfig{
		EnableStackTrace: true,
		Logger: func(err error, stack []byte) {
			loggedError = err
		},
		ResponseHandler: defaultRecoveryResponse,
	}

	middleware := RecoveryWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if loggedError == nil {
		t.Error("Expected error to be logged")
	}
	if loggedError.Error() != "custom error" {
		t.Errorf("Expected 'custom error', got %v", loggedError.Error())
	}
}

func TestRecoveryWithNonErrorPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("string panic")
	})

	var loggedError error
	config := RecoveryConfig{
		EnableStackTrace: false,
		Logger: func(err error, stack []byte) {
			loggedError = err
		},
		ResponseHandler: defaultRecoveryResponse,
	}

	middleware := RecoveryWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if loggedError == nil {
		t.Error("Expected error to be logged")
	}
}

func TestRecoveryCustomResponseHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	customResponseCalled := false
	config := RecoveryConfig{
		EnableStackTrace: false,
		ResponseHandler: func(w http.ResponseWriter, r *http.Request, err interface{}) {
			customResponseCalled = true
			w.WriteHeader(http.StatusTeapot)
		},
	}

	middleware := RecoveryWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !customResponseCalled {
		t.Error("Custom response handler was not called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestRecoveryNoPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := Recovery()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", rec.Body.String())
	}
}

func TestRecoveryStackTrace(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	var stackTrace []byte
	config := RecoveryConfig{
		EnableStackTrace: true,
		Logger: func(err error, stack []byte) {
			stackTrace = stack
		},
		ResponseHandler: defaultRecoveryResponse,
	}

	middleware := RecoveryWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if len(stackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}
}

func TestRecoveryNoStackTrace(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	var stackTrace []byte
	config := RecoveryConfig{
		EnableStackTrace: false,
		Logger: func(err error, stack []byte) {
			stackTrace = stack
		},
		ResponseHandler: defaultRecoveryResponse,
	}

	middleware := RecoveryWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if len(stackTrace) != 0 {
		t.Error("Expected no stack trace when disabled")
	}
}

func TestPanicError(t *testing.T) {
	err := &panicError{value: "test error"}
	if err.Error() != "panic occurred" {
		t.Errorf("Expected 'panic occurred', got %s", err.Error())
	}

	err2 := &panicError{value: errors.New("actual error")}
	if err2.Error() != "actual error" {
		t.Errorf("Expected 'actual error', got %s", err2.Error())
	}
}
