package session

import (
	"context"
	"errors"
	"time"
)

// ErrSessionNotFound is returned when a session is not found
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionExpired is returned when a session has expired
var ErrSessionExpired = errors.New("session expired")

// Store defines the interface for session storage backends
type Store interface {
	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Set stores a session with the given TTL
	Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// Refresh updates the expiration time of a session
	Refresh(ctx context.Context, sessionID string, ttl time.Duration) error

	// Close cleans up any resources used by the store
	Close() error
}

// Session represents a user session
type Session struct {
	// ID is the unique session identifier
	ID string `json:"id"`

	// UserID is the authenticated user ID (empty if not authenticated)
	UserID string `json:"user_id,omitempty"`

	// Data holds arbitrary session data
	Data map[string]interface{} `json:"data"`

	// FlashMessages holds one-time messages
	FlashMessages []FlashMessage `json:"flash_messages,omitempty"`

	// CreatedAt is when the session was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the session expires
	ExpiresAt time.Time `json:"expires_at"`

	// CSRFToken is the CSRF protection token for this session
	CSRFToken string `json:"csrf_token,omitempty"`

	// destroyed indicates if the session has been destroyed (not persisted)
	destroyed bool `json:"-"`
}

// FlashMessage represents a one-time message stored in the session
type FlashMessage struct {
	// Type indicates the message type (success, error, warning, info)
	Type string `json:"type"`

	// Message is the flash message content
	Message string `json:"message"`
}

// NewSession creates a new session with the given ID and TTL
func NewSession(id string, ttl time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:            id,
		Data:          make(map[string]interface{}),
		FlashMessages: []FlashMessage{},
		CreatedAt:     now,
		ExpiresAt:     now.Add(ttl),
	}
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Get retrieves a value from session data
func (s *Session) Get(key string) (interface{}, bool) {
	val, ok := s.Data[key]
	return val, ok
}

// Set stores a value in session data
func (s *Session) Set(key string, value interface{}) {
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data[key] = value
}

// Delete removes a value from session data
func (s *Session) Delete(key string) {
	delete(s.Data, key)
}

// AddFlash adds a flash message to the session
func (s *Session) AddFlash(messageType, message string) {
	s.FlashMessages = append(s.FlashMessages, FlashMessage{
		Type:    messageType,
		Message: message,
	})
}

// GetFlashes retrieves all flash messages and clears them from the session
func (s *Session) GetFlashes() []FlashMessage {
	messages := s.FlashMessages
	s.FlashMessages = []FlashMessage{}
	return messages
}

// Config holds session configuration
type Config struct {
	// CookieName is the name of the session cookie
	CookieName string

	// CookiePath is the path for the session cookie
	CookiePath string

	// CookieDomain is the domain for the session cookie
	CookieDomain string

	// MaxAge is the session TTL in seconds (0 = session cookie)
	MaxAge int

	// HttpOnly prevents JavaScript access to the cookie
	HttpOnly bool

	// Secure requires HTTPS for the cookie
	Secure bool

	// SameSite controls cross-site cookie behavior
	SameSite string // "Strict", "Lax", or "None"

	// Store is the session storage backend
	Store Store
}

// DefaultConfig returns default session configuration
func DefaultConfig(store Store) *Config {
	return &Config{
		CookieName:   "conduit_session",
		CookiePath:   "/",
		CookieDomain: "",
		MaxAge:       86400 * 7, // 7 days
		HttpOnly:     true,
		Secure:       true,
		SameSite:     "Lax",
		Store:        store,
	}
}
