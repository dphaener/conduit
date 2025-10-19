package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds configuration for CORS middleware
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins. Use "*" for all origins.
	AllowedOrigins []string
	// AllowedMethods is a list of allowed HTTP methods
	AllowedMethods []string
	// AllowedHeaders is a list of allowed request headers
	AllowedHeaders []string
	// ExposedHeaders is a list of headers exposed to the client
	ExposedHeaders []string
	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool
	// MaxAge indicates how long preflight results can be cached (in seconds)
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS creates a CORS middleware with default configuration
func CORS() Middleware {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig creates a CORS middleware with custom configuration
func CORSWithConfig(config CORSConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				// Set CORS headers
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			}

			// Handle preflight request
			if r.Method == http.MethodOptions {
				if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
					// Set preflight headers
					if len(config.AllowedMethods) > 0 {
						w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
					}

					if len(config.AllowedHeaders) > 0 {
						w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
					}

					if config.MaxAge > 0 {
						w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
					}
				}

				// Respond to preflight
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Continue with next handler
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if an origin is allowed
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			// Check if origin ends with .domain (subdomain match)
			// But not the domain itself
			if strings.HasSuffix(origin, "."+domain) {
				return true
			}
		}
	}
	return false
}
