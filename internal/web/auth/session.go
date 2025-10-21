package auth

import (
	"context"
	"net/http"
	"time"
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	Data      map[string]interface{}
	ExpiresAt time.Time
}

// SessionStore defines the interface for session storage implementations
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*Session, error)
	Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error
	Delete(ctx context.Context, sessionID string) error
}

// SessionAuthenticator handles session-based authentication
type SessionAuthenticator struct {
	store SessionStore
}

// NewSessionAuthenticator creates a new session authenticator with the given store
func NewSessionAuthenticator(store SessionStore) *SessionAuthenticator {
	return &SessionAuthenticator{
		store: store,
	}
}

// Middleware returns a middleware function that validates sessions from cookies
func (a *SessionAuthenticator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session ID from cookie
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Retrieve session from store
			session, err := a.store.Get(r.Context(), cookie.Value)
			if err != nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check session validity and expiration
			if session == nil || session.ExpiresAt.Before(time.Now()) {
				if session != nil {
					// Cleanup expired session
					_ = a.store.Delete(r.Context(), cookie.Value)
				}
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Add to context
			ctx := SetCurrentUser(r.Context(), session.UserID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
