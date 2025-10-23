package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionMiddleware(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false // Disable for testing

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		assert.NotEmpty(t, sess.ID)

		// Set some data
		sess.Set("test_key", "test_value")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check for session cookie
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "conduit_session", cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
	assert.Equal(t, "/", cookies[0].Path)
	assert.True(t, cookies[0].HttpOnly)
}

func TestSessionMiddlewareExistingSession(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	// Create a session manually
	sessionID := "existing-session-id"
	sess := NewSession(sessionID, 1*time.Hour)
	sess.Set("existing_key", "existing_value")
	err := store.Set(ctx, sessionID, sess, 1*time.Hour)
	require.NoError(t, err)

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		assert.Equal(t, sessionID, sess.ID)

		// Check existing data
		val, ok := sess.Get("existing_key")
		require.True(t, ok)
		assert.Equal(t, "existing_value", val)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "conduit_session",
		Value: sessionID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSessionMiddlewareExpiredSession(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	// Create an expired session
	sessionID := "expired-session-id"
	sess := NewSession(sessionID, -1*time.Hour) // Already expired
	err := store.Set(ctx, sessionID, sess, 1*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure expiration

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		// Should have a new session ID
		assert.NotEqual(t, sessionID, sess.ID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "conduit_session",
		Value: sessionID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSetAuthenticatedUser(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := SetAuthenticatedUser(r.Context(), "user-123")
		require.NoError(t, err)

		userID := GetAuthenticatedUser(r.Context())
		assert.Equal(t, "user-123", userID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestClearAuthenticatedUser(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set user
		err := SetAuthenticatedUser(r.Context(), "user-123")
		require.NoError(t, err)

		userID := GetAuthenticatedUser(r.Context())
		assert.Equal(t, "user-123", userID)

		// Clear user
		ClearAuthenticatedUser(r.Context())

		userID = GetAuthenticatedUser(r.Context())
		assert.Empty(t, userID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDestroySession(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	var sessionID string

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		sessionID = sess.ID

		// Destroy session
		err := DestroySession(r.Context(), store, "conduit_session", w)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Session should be deleted from store
	_, err := store.Get(context.Background(), sessionID)
	assert.Error(t, err)

	// Cookie should be cleared
	cookies := rec.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "conduit_session" && cookie.MaxAge == -1 {
			found = true
			break
		}
	}
	assert.True(t, found, "Session cookie should be cleared")
}

func TestGetSessionOrPanic(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not panic
		sess := GetSessionOrPanic(r.Context())
		assert.NotNil(t, sess)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetSessionOrPanicWithoutMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				assert.Contains(t, r.(string), "session not found in context")
			}
		}()

		GetSessionOrPanic(r.Context())
		t.Fatal("Should have panicked")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestSessionCookieConfiguration(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := &Config{
		CookieName:   "custom_session",
		CookiePath:   "/api",
		CookieDomain: "example.com",
		MaxAge:       3600,
		HttpOnly:     true,
		Secure:       true,
		SameSite:     "Strict",
		Store:        store,
	}

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "custom_session", cookie.Name)
	assert.Equal(t, "/api", cookie.Path)
	assert.Equal(t, "example.com", cookie.Domain)
	assert.Equal(t, 3600, cookie.MaxAge)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
}

func TestRegenerateSessionIDRegeneratesCSRFToken(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	ctx := context.Background()

	// Create a session with CSRF token
	oldID := "old-session-id"
	sess := NewSession(oldID, 1*time.Hour)
	sess.CSRFToken = "old-csrf-token"
	err := store.Set(ctx, oldID, sess, 1*time.Hour)
	require.NoError(t, err)

	ctx = context.WithValue(ctx, sessionKey, sess)
	rec := httptest.NewRecorder()

	err = RegenerateSessionID(ctx, store, config, rec)
	require.NoError(t, err)

	// Session ID should have changed
	assert.NotEqual(t, oldID, sess.ID)

	// CSRF token should have changed
	assert.NotEqual(t, "old-csrf-token", sess.CSRFToken)
	assert.NotEmpty(t, sess.CSRFToken)

	// Old session should be deleted
	_, err = store.Get(ctx, oldID)
	assert.Error(t, err)

	// New session should exist
	newSess, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, newSess.ID)
	assert.Equal(t, sess.CSRFToken, newSess.CSRFToken)
}
