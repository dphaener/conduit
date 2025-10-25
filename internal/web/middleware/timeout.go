package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// TimeoutConfig holds configuration for the timeout middleware
type TimeoutConfig struct {
	// Timeout is the maximum duration for a request
	Timeout time.Duration

	// ErrorMessage is the message returned on timeout
	ErrorMessage string

	// StatusCode is the HTTP status code returned on timeout
	StatusCode int
}

// DefaultTimeoutConfig returns the default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:      30 * time.Second,
		ErrorMessage: "Request timeout",
		StatusCode:   http.StatusGatewayTimeout,
	}
}

// Timeout creates a timeout middleware with default configuration
func Timeout(timeout time.Duration) Middleware {
	config := DefaultTimeoutConfig()
	config.Timeout = timeout
	return TimeoutWithConfig(config)
}

// timeoutWriter wraps http.ResponseWriter to prevent writes after timeout
type timeoutWriter struct {
	w    http.ResponseWriter
	mu   sync.Mutex
	done bool
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.done {
		return 0, http.ErrHandlerTimeout
	}
	return tw.w.Write(b)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.done {
		return
	}
	tw.w.WriteHeader(code)
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *timeoutWriter) timeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.done = true
}

// TimeoutWithConfig creates a timeout middleware with custom configuration
func TimeoutWithConfig(config TimeoutConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})
			panicChan := make(chan interface{}, 1)

			// Wrap the response writer to prevent race conditions
			tw := &timeoutWriter{w: w}

			// Wrap the request with timeout context
			r = r.WithContext(ctx)

			// Run the handler in a goroutine
			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()

				next.ServeHTTP(tw, r)
				close(done)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Handler completed successfully
				return
			case p := <-panicChan:
				// Handler panicked
				panic(p)
			case <-ctx.Done():
				// Timeout occurred - mark writer as done before writing
				tw.timeout()
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, config.ErrorMessage, config.StatusCode)
				}
				return
			}
		})
	}
}

// FastTimeout creates a lightweight timeout middleware without goroutines
// This version is faster but doesn't interrupt long-running handlers
func FastTimeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
