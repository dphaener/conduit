package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "request_id"
)

// RequestIDConfig holds configuration for the request ID middleware
type RequestIDConfig struct {
	// HeaderName is the name of the header to read/write the request ID
	HeaderName string
	// Generator is a custom function to generate request IDs
	Generator func() string
}

// DefaultRequestIDConfig returns the default request ID configuration
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		HeaderName: "X-Request-ID",
		Generator:  defaultRequestIDGenerator,
	}
}

// RequestID creates a middleware that adds a unique request ID to each request
func RequestID() Middleware {
	return RequestIDWithConfig(DefaultRequestIDConfig())
}

// RequestIDWithConfig creates a request ID middleware with custom configuration
func RequestIDWithConfig(config RequestIDConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get request ID from header
			requestID := r.Header.Get(config.HeaderName)
			if requestID == "" {
				// Generate new request ID
				requestID = config.Generator()
			}

			// Add request ID to context
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			r = r.WithContext(ctx)

			// Add request ID to response header
			w.Header().Set(config.HeaderName, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID extracts the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// defaultRequestIDGenerator generates a UUID v4 request ID
func defaultRequestIDGenerator() string {
	return uuid.New().String()
}
