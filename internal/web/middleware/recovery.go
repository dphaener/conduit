package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// RecoveryConfig holds configuration for the recovery middleware
type RecoveryConfig struct {
	// EnableStackTrace determines whether to log stack traces
	EnableStackTrace bool
	// Logger is an optional custom logger
	Logger func(error, []byte)
	// ResponseHandler is an optional custom response handler
	ResponseHandler func(http.ResponseWriter, *http.Request, interface{})
}

// DefaultRecoveryConfig returns the default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		EnableStackTrace: true,
		Logger: func(err error, stack []byte) {
			log.Printf("PANIC RECOVERED: %v\n%s", err, stack)
		},
		ResponseHandler: defaultRecoveryResponse,
	}
}

// Recovery creates a middleware that recovers from panics
func Recovery() Middleware {
	return RecoveryWithConfig(DefaultRecoveryConfig())
}

// RecoveryWithConfig creates a recovery middleware with custom configuration
func RecoveryWithConfig(config RecoveryConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Capture stack trace
					var stack []byte
					if config.EnableStackTrace {
						stack = debug.Stack()
					}

					// Log the panic
					if config.Logger != nil {
						// Convert err to error type if possible
						var errValue error
						switch e := err.(type) {
						case error:
							errValue = e
						default:
							errValue = &panicError{value: err}
						}
						config.Logger(errValue, stack)
					}

					// Send response
					if config.ResponseHandler != nil {
						config.ResponseHandler(w, r, err)
					} else {
						defaultRecoveryResponse(w, r, err)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// defaultRecoveryResponse sends a default JSON error response
func defaultRecoveryResponse(w http.ResponseWriter, r *http.Request, err interface{}) {
	// Prepare response payload
	response := map[string]interface{}{
		"error":   "internal_server_error",
		"message": "An unexpected error occurred",
	}

	// Attempt to marshal JSON first
	jsonData, encErr := json.Marshal(response)
	if encErr != nil {
		// Fallback to plain text if JSON encoding fails
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	// JSON encoding succeeded, send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonData)
}

// panicError wraps a panic value as an error
type panicError struct {
	value interface{}
}

func (e *panicError) Error() string {
	if err, ok := e.value.(error); ok {
		return err.Error()
	}
	return "panic occurred"
}
