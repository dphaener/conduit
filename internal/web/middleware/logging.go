package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingConfig holds configuration for the logging middleware
type LoggingConfig struct {
	// Logger is an optional custom logger function
	Logger func(LogEntry)
	// SkipPaths is a list of paths to skip logging
	SkipPaths []string
}

// LogEntry represents a log entry for a request
type LogEntry struct {
	RequestID    string
	Method       string
	Path         string
	StatusCode   int
	Duration     time.Duration
	BytesWritten int
	RemoteAddr   string
	UserAgent    string
}

// DefaultLoggingConfig returns the default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:    defaultLogger,
		SkipPaths: []string{},
	}
}

// Logging creates a logging middleware with default configuration
func Logging() Middleware {
	return LoggingWithConfig(DefaultLoggingConfig())
}

// LoggingWithConfig creates a logging middleware with custom configuration
func LoggingWithConfig(config LoggingConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Start timing
			start := time.Now()

			// Wrap response writer to capture status code and bytes written
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status code
			}

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log the request
			if config.Logger != nil {
				config.Logger(LogEntry{
					RequestID:    requestID,
					Method:       r.Method,
					Path:         r.URL.Path,
					StatusCode:   rw.statusCode,
					Duration:     duration,
					BytesWritten: rw.bytesWritten,
					RemoteAddr:   r.RemoteAddr,
					UserAgent:    r.UserAgent(),
				})
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.statusCode = statusCode
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures bytes written
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// defaultLogger is the default logging function
func defaultLogger(entry LogEntry) {
	log.Printf("[%s] %s %s - %d (%v) %d bytes",
		entry.RequestID,
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.Duration,
		entry.BytesWritten,
	)
}
