package session

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
)

var (
	// ErrCSRFTokenMissing is returned when CSRF token is missing from request
	ErrCSRFTokenMissing = errors.New("CSRF token missing")

	// ErrCSRFTokenInvalid is returned when CSRF token is invalid
	ErrCSRFTokenInvalid = errors.New("CSRF token invalid")
)

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	// TokenLength is the length of the CSRF token in bytes
	TokenLength int

	// TokenHeader is the HTTP header name for CSRF token
	TokenHeader string

	// TokenField is the form field name for CSRF token
	TokenField string

	// SafeMethods are HTTP methods that don't require CSRF protection
	SafeMethods []string

	// SkipPaths are paths to skip CSRF protection
	SkipPaths []string

	// ErrorHandler is called when CSRF validation fails
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength: 32,
		TokenHeader: "X-CSRF-Token",
		TokenField:  "csrf_token",
		SafeMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
		SkipPaths:   []string{},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusForbidden)
		},
	}
}

// CSRFMiddleware creates a CSRF protection middleware
// This middleware requires session middleware to be enabled
func CSRFMiddleware(config *CSRFConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get session from context
			sess := GetSession(r.Context())
			if sess == nil {
				config.ErrorHandler(w, r, ErrSessionNotFound)
				return
			}

			// Ensure session has CSRF token
			if sess.CSRFToken == "" {
				token, err := generateCSRFToken(config.TokenLength)
				if err != nil {
					config.ErrorHandler(w, r, err)
					return
				}
				sess.CSRFToken = token
			}

			// Check if method requires CSRF protection
			requiresProtection := true
			for _, method := range config.SafeMethods {
				if r.Method == method {
					requiresProtection = false
					break
				}
			}

			if requiresProtection {
				// Extract token from request
				token := extractCSRFToken(r, config)
				if token == "" {
					config.ErrorHandler(w, r, ErrCSRFTokenMissing)
					return
				}

				// Validate token
				if !validateCSRFToken(token, sess.CSRFToken) {
					config.ErrorHandler(w, r, ErrCSRFTokenInvalid)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetCSRFToken retrieves the CSRF token from the current session
func GetCSRFToken(ctx context.Context) string {
	sess := GetSession(ctx)
	if sess == nil {
		return ""
	}

	// Generate token if it doesn't exist
	if sess.CSRFToken == "" {
		token, err := generateCSRFToken(32)
		if err != nil {
			return ""
		}
		sess.CSRFToken = token
	}

	return sess.CSRFToken
}

// generateCSRFToken generates a new CSRF token
func generateCSRFToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// extractCSRFToken extracts the CSRF token from the request
func extractCSRFToken(r *http.Request, config *CSRFConfig) string {
	// Try header first
	token := r.Header.Get(config.TokenHeader)
	if token != "" {
		return token
	}

	// Try form field
	if err := r.ParseForm(); err == nil {
		token = r.FormValue(config.TokenField)
		if token != "" {
			return token
		}
	}

	// Try multipart form
	if err := r.ParseMultipartForm(32 << 20); err == nil {
		if r.MultipartForm != nil {
			values := r.MultipartForm.Value[config.TokenField]
			if len(values) > 0 {
				return values[0]
			}
		}
	}

	return ""
}

// validateCSRFToken validates a CSRF token using constant-time comparison
func validateCSRFToken(token, expected string) bool {
	if token == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

// RegenerateCSRFToken generates a new CSRF token for the session
// This should be called after authentication to prevent session fixation
func RegenerateCSRFToken(ctx context.Context) error {
	sess := GetSession(ctx)
	if sess == nil {
		return ErrSessionNotFound
	}

	token, err := generateCSRFToken(32)
	if err != nil {
		return err
	}

	sess.CSRFToken = token
	return nil
}
