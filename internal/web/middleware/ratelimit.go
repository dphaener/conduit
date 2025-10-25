package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/web/ratelimit"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	// Limiter is the rate limiter implementation to use
	Limiter ratelimit.RateLimiter
	// KeyFunc extracts the rate limit key from the request
	KeyFunc RateLimitKeyFunc
	// BypassFunc determines if rate limiting should be skipped for a request
	BypassFunc RateLimitBypassFunc
	// ErrorHandler handles rate limit exceeded errors
	ErrorHandler RateLimitErrorHandler
	// FailOpen determines behavior when rate limiter returns an error
	// If true, allows the request; if false, denies it
	FailOpen bool
}

// RateLimitKeyFunc extracts a rate limit key from a request
type RateLimitKeyFunc func(*http.Request) string

// RateLimitBypassFunc determines if rate limiting should be bypassed
type RateLimitBypassFunc func(*http.Request) bool

// RateLimitErrorHandler handles rate limit errors
type RateLimitErrorHandler func(http.ResponseWriter, *http.Request, error)

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig(limiter ratelimit.RateLimiter) RateLimitConfig {
	return RateLimitConfig{
		Limiter:      limiter,
		KeyFunc:      IPKeyFunc,
		BypassFunc:   nil,
		ErrorHandler: DefaultRateLimitErrorHandler,
		FailOpen:     true,
	}
}

// RateLimit creates a rate limiting middleware with the given limiter and IP-based key
func RateLimit(limiter ratelimit.RateLimiter) Middleware {
	return RateLimitWithConfig(DefaultRateLimitConfig(limiter))
}

// RateLimitWithConfig creates a rate limiting middleware with custom configuration
func RateLimitWithConfig(config RateLimitConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check bypass function
			if config.BypassFunc != nil && config.BypassFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract key
			key := config.KeyFunc(r)
			if key == "" {
				// If no key, fail open or closed based on config
				if config.FailOpen {
					next.ServeHTTP(w, r)
				} else {
					http.Error(w, "Rate limit key extraction failed", http.StatusInternalServerError)
				}
				return
			}

			// Check rate limit
			info, err := config.Limiter.Allow(r.Context(), key)
			if err != nil {
				if config.FailOpen {
					// Log error but allow request
					next.ServeHTTP(w, r)
				} else {
					config.ErrorHandler(w, r, err)
				}
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt.Unix(), 10))

			// Check if allowed
			if !info.Allowed {
				// Calculate seconds until reset
				retryAfter := int64(info.ResetAt.Sub(time.Now()).Seconds())
				if retryAfter < 0 {
					retryAfter = 0
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyFunc extracts the IP address from the request
// Checks X-Forwarded-For header first, then falls back to RemoteAddr
func IPKeyFunc(r *http.Request) string {
	// Try X-Forwarded-For first (proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Try X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in format "ip:port", extract just the IP
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// UserKeyFunc extracts the user ID from the request context
func UserKeyFunc(r *http.Request) string {
	userID := GetUserID(r.Context())
	if userID == "" {
		return ""
	}
	return "user:" + userID
}

// EndpointKeyFunc uses the request path as the key
func EndpointKeyFunc(r *http.Request) string {
	return "endpoint:" + r.URL.Path
}

// UserEndpointKeyFunc combines user ID and endpoint
func UserEndpointKeyFunc(r *http.Request) string {
	userID := GetUserID(r.Context())
	if userID == "" {
		// Fall back to IP if no user
		return "ip:" + IPKeyFunc(r) + ":" + r.URL.Path
	}
	return "user:" + userID + ":" + r.URL.Path
}

// CombinedKeyFunc creates a key function that combines multiple key functions
func CombinedKeyFunc(funcs ...RateLimitKeyFunc) RateLimitKeyFunc {
	return func(r *http.Request) string {
		var parts []string
		for _, f := range funcs {
			if key := f(r); key != "" {
				parts = append(parts, key)
			}
		}
		return strings.Join(parts, ":")
	}
}

// DefaultRateLimitErrorHandler is the default error handler for rate limit errors
func DefaultRateLimitErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("Rate limit check failed: %v", err), http.StatusInternalServerError)
}

// AdminBypassFunc bypasses rate limiting for admin users
func AdminBypassFunc(r *http.Request) bool {
	roles := GetUserRoles(r.Context())
	for _, role := range roles {
		if role == "admin" || role == "superadmin" {
			return true
		}
	}
	return false
}

// InternalBypassFunc bypasses rate limiting for internal requests
// Checks for X-Internal header
func InternalBypassFunc(r *http.Request) bool {
	return r.Header.Get("X-Internal") == "true"
}

// CombinedBypassFunc combines multiple bypass functions (OR logic)
func CombinedBypassFunc(funcs ...RateLimitBypassFunc) RateLimitBypassFunc {
	return func(r *http.Request) bool {
		for _, f := range funcs {
			if f(r) {
				return true
			}
		}
		return false
	}
}
