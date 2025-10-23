package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey int

const (
	sessionKey contextKey = iota
)

// Middleware creates a session middleware with the given configuration
func Middleware(config *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session ID from cookie
			cookie, err := r.Cookie(config.CookieName)
			var sessionID string
			var sess *Session

			if err == nil && cookie.Value != "" {
				sessionID = cookie.Value
				// Try to load existing session
				sess, err = config.Store.Get(r.Context(), sessionID)
				if err != nil && err != ErrSessionNotFound && err != ErrSessionExpired {
					// Log error but continue with new session
					sess = nil
				}
			}

			// Create new session if needed
			if sess == nil {
				sessionID, err = generateSessionID()
				if err != nil {
					http.Error(w, "Failed to generate session ID", http.StatusInternalServerError)
					return
				}

				ttl := time.Duration(config.MaxAge) * time.Second
				sess = NewSession(sessionID, ttl)

				// Save the new session
				if err := config.Store.Set(r.Context(), sessionID, sess, ttl); err != nil {
					http.Error(w, "Failed to create session", http.StatusInternalServerError)
					return
				}
			}

			// Set session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     config.CookieName,
				Value:    sessionID,
				Path:     config.CookiePath,
				Domain:   config.CookieDomain,
				MaxAge:   config.MaxAge,
				HttpOnly: config.HttpOnly,
				Secure:   config.Secure,
				SameSite: sameSiteFromString(config.SameSite),
			})

			// Add session to context
			ctx := context.WithValue(r.Context(), sessionKey, sess)

			// If session has a user ID, add it to context for compatibility
			if sess.UserID != "" {
				ctx = webcontext.SetCurrentUser(ctx, sess.UserID)
			}

			r = r.WithContext(ctx)

			// Wrap response writer to save session after request
			sw := &sessionWriter{
				ResponseWriter: w,
				session:        sess,
				sessionID:      sessionID,
				store:          config.Store,
				ttl:            time.Duration(config.MaxAge) * time.Second,
				ctx:            ctx,
			}

			next.ServeHTTP(sw, r)
		})
	}
}

// sessionWriter wraps http.ResponseWriter to save session after response
type sessionWriter struct {
	http.ResponseWriter
	session   *Session
	sessionID string
	store     Store
	ttl       time.Duration
	ctx       context.Context
}

// WriteHeader saves the session before writing headers
func (sw *sessionWriter) WriteHeader(statusCode int) {
	// Only save session if it wasn't destroyed
	if !sw.session.destroyed {
		// Use request context with reasonable timeout for session save
		ctx, cancel := context.WithTimeout(sw.ctx, 5*time.Second)
		defer cancel()

		if err := sw.store.Set(ctx, sw.sessionID, sw.session, sw.ttl); err != nil {
			// Log to stderr at minimum
			fmt.Fprintf(os.Stderr, "WARNING: Failed to save session %s: %v\n", sw.sessionID, err)
		}
	}
	sw.ResponseWriter.WriteHeader(statusCode)
}

// GetSession retrieves the session from the request context
func GetSession(ctx context.Context) *Session {
	if sess, ok := ctx.Value(sessionKey).(*Session); ok {
		return sess
	}
	return nil
}

// GetSessionOrPanic retrieves the session from context or panics
// Use this only in handlers where session middleware is guaranteed to run
func GetSessionOrPanic(ctx context.Context) *Session {
	sess := GetSession(ctx)
	if sess == nil {
		panic("session not found in context - ensure session middleware is enabled")
	}
	return sess
}

// SetAuthenticatedUser sets the authenticated user ID in the session
func SetAuthenticatedUser(ctx context.Context, userID string) error {
	sess := GetSession(ctx)
	if sess == nil {
		return ErrSessionNotFound
	}
	sess.UserID = userID
	return nil
}

// GetAuthenticatedUser retrieves the authenticated user ID from the session
func GetAuthenticatedUser(ctx context.Context) string {
	sess := GetSession(ctx)
	if sess == nil {
		return ""
	}
	return sess.UserID
}

// ClearAuthenticatedUser removes the authenticated user from the session
func ClearAuthenticatedUser(ctx context.Context) {
	sess := GetSession(ctx)
	if sess != nil {
		sess.UserID = ""
	}
}

// DestroySession destroys the current session
func DestroySession(ctx context.Context, store Store, cookieName string, w http.ResponseWriter) error {
	sess := GetSession(ctx)
	if sess == nil {
		return nil
	}

	// Mark session as destroyed
	sess.destroyed = true

	// Delete from store
	if err := store.Delete(ctx, sess.ID); err != nil {
		return err
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	return nil
}

// RegenerateSessionID creates a new session ID while preserving session data.
// This should be called after authentication to prevent session fixation attacks.
//
// Example usage after successful login:
//
//	if err := session.RegenerateSessionID(r.Context(), store, config, w); err != nil {
//	    http.Error(w, "Session regeneration failed", http.StatusInternalServerError)
//	    return
//	}
func RegenerateSessionID(ctx context.Context, store Store, config *Config, w http.ResponseWriter) error {
	sess := GetSession(ctx)
	if sess == nil {
		return ErrSessionNotFound
	}

	// Generate new session ID
	newID, err := generateSessionID()
	if err != nil {
		return err
	}

	// Store old ID for deletion
	oldID := sess.ID

	// Delete old session from store
	if err := store.Delete(ctx, oldID); err != nil {
		return err
	}

	// Update session ID
	sess.ID = newID

	// Regenerate CSRF token to prevent CSRF session fixation
	if sess.CSRFToken != "" {
		newToken, err := generateCSRFToken(32)
		if err != nil {
			return fmt.Errorf("failed to regenerate CSRF token: %w", err)
		}
		sess.CSRFToken = newToken
	}

	// Save session with new ID
	ttl := time.Duration(config.MaxAge) * time.Second
	if err := store.Set(ctx, newID, sess, ttl); err != nil {
		return err
	}

	// Update cookie with new session ID
	http.SetCookie(w, &http.Cookie{
		Name:     config.CookieName,
		Value:    newID,
		Path:     config.CookiePath,
		Domain:   config.CookieDomain,
		MaxAge:   config.MaxAge,
		HttpOnly: config.HttpOnly,
		Secure:   config.Secure,
		SameSite: sameSiteFromString(config.SameSite),
	})

	return nil
}

// generateSessionID generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// sameSiteFromString converts string to http.SameSite
func sameSiteFromString(s string) http.SameSite {
	switch s {
	case "Strict":
		return http.SameSiteStrictMode
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
