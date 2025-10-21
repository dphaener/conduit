package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockSessionStore is a mock implementation of SessionStore for testing
type mockSessionStore struct {
	sessions map[string]*Session
	getError error
	setError error
	delError error
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*Session),
	}
}

func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (m *mockSessionStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
	if m.setError != nil {
		return m.setError
	}
	m.sessions[sessionID] = session
	return nil
}

func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error {
	if m.delError != nil {
		return m.delError
	}
	delete(m.sessions, sessionID)
	return nil
}

func TestNewSessionAuthenticator(t *testing.T) {
	store := newMockSessionStore()
	auth := NewSessionAuthenticator(store)

	if auth == nil {
		t.Fatal("NewSessionAuthenticator() returned nil")
	}
	if auth.store != store {
		t.Error("NewSessionAuthenticator() did not set store correctly")
	}
}

func TestSessionMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupStore     func(*mockSessionStore)
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedUserID string
	}{
		{
			name: "allows valid session",
			setupStore: func(store *mockSessionStore) {
				store.sessions["session-123"] = &Session{
					ID:        "session-123",
					UserID:    "user-123",
					Data:      map[string]interface{}{"role": "admin"},
					ExpiresAt: time.Now().Add(time.Hour),
				}
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "session-123",
				})
			},
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
		},
		{
			name:       "rejects missing session cookie",
			setupStore: func(store *mockSessionStore) {},
			setupRequest: func(req *http.Request) {
				// No cookie set
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name:       "rejects invalid session ID",
			setupStore: func(store *mockSessionStore) {},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "invalid-session",
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name: "rejects expired session",
			setupStore: func(store *mockSessionStore) {
				store.sessions["expired-session"] = &Session{
					ID:        "expired-session",
					UserID:    "user-456",
					Data:      map[string]interface{}{},
					ExpiresAt: time.Now().Add(-time.Hour), // Expired 1 hour ago
				}
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "expired-session",
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock store
			store := newMockSessionStore()
			tt.setupStore(store)

			// Create authenticator
			auth := NewSessionAuthenticator(store)

			// Create test handler that checks context
			var capturedUserID string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUserID = GetCurrentUser(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with session middleware
			middleware := auth.Middleware()
			wrappedHandler := middleware(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Middleware() status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			// Check user ID in context (only for successful requests)
			if tt.expectedStatus == http.StatusOK && capturedUserID != tt.expectedUserID {
				t.Errorf("Middleware() userID in context = %v, want %v", capturedUserID, tt.expectedUserID)
			}
		})
	}
}

func TestSessionStoreOperations(t *testing.T) {
	store := newMockSessionStore()

	// Test Set
	session := &Session{
		ID:        "test-session",
		UserID:    "user-789",
		Data:      map[string]interface{}{"theme": "dark"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := store.Set(context.Background(), "test-session", session, time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Test Get
	retrieved, err := store.Get(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("Get() UserID = %v, want %v", retrieved.UserID, session.UserID)
	}
	if retrieved.Data["theme"] != "dark" {
		t.Errorf("Get() Data['theme'] = %v, want %v", retrieved.Data["theme"], "dark")
	}

	// Test Delete
	err = store.Delete(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = store.Get(context.Background(), "test-session")
	if err == nil {
		t.Error("Get() after Delete() should return error")
	}
}

func TestSessionStoreErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*mockSessionStore)
		operation   func(*mockSessionStore) error
		expectError bool
	}{
		{
			name: "Get returns error",
			setupStore: func(store *mockSessionStore) {
				store.getError = errors.New("database error")
			},
			operation: func(store *mockSessionStore) error {
				_, err := store.Get(context.Background(), "any-session")
				return err
			},
			expectError: true,
		},
		{
			name: "Set returns error",
			setupStore: func(store *mockSessionStore) {
				store.setError = errors.New("database error")
			},
			operation: func(store *mockSessionStore) error {
				session := &Session{ID: "test", UserID: "user-1", ExpiresAt: time.Now()}
				return store.Set(context.Background(), "test", session, time.Hour)
			},
			expectError: true,
		},
		{
			name: "Delete returns error",
			setupStore: func(store *mockSessionStore) {
				store.delError = errors.New("database error")
			},
			operation: func(store *mockSessionStore) error {
				return store.Delete(context.Background(), "test")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockSessionStore()
			tt.setupStore(store)

			err := tt.operation(store)
			if (err != nil) != tt.expectError {
				t.Errorf("operation() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSessionMiddlewareWithStoreError(t *testing.T) {
	store := newMockSessionStore()
	store.getError = errors.New("database connection failed")

	auth := NewSessionAuthenticator(store)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := auth.Middleware()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "any-session",
	})

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Middleware() with store error status = %v, want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionWithCustomData(t *testing.T) {
	store := newMockSessionStore()

	// Create session with custom data
	customData := map[string]interface{}{
		"preferences": map[string]interface{}{
			"language": "en",
			"timezone": "UTC",
		},
		"permissions": []string{"read", "write"},
	}

	session := &Session{
		ID:        "custom-session",
		UserID:    "user-custom",
		Data:      customData,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := store.Set(context.Background(), "custom-session", session, time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.Get(context.Background(), "custom-session")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	prefs, ok := retrieved.Data["preferences"].(map[string]interface{})
	if !ok {
		t.Fatal("Custom data 'preferences' not found or wrong type")
	}
	if prefs["language"] != "en" {
		t.Errorf("Custom data language = %v, want %v", prefs["language"], "en")
	}
}
