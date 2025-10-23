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

// TestFullSessionLifecycle tests a complete session lifecycle
func TestFullSessionLifecycle(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	sessionMiddleware := Middleware(config)

	var sessionCookie *http.Cookie

	// Step 1: Create session and set data
	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		sess.Set("username", "testuser")
		sess.Set("email", "test@example.com")

		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Step 2: Retrieve session data
	handler2 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		username, ok := sess.Get("username")
		require.True(t, ok)
		assert.Equal(t, "testuser", username)

		email, ok := sess.Get("email")
		require.True(t, ok)
		assert.Equal(t, "test@example.com", email)

		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)

	// Step 3: Update session data
	handler3 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		sess.Set("username", "updateduser")
		sess.Delete("email")

		w.WriteHeader(http.StatusOK)
	}))

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(sessionCookie)
	rec3 := httptest.NewRecorder()
	handler3.ServeHTTP(rec3, req3)

	// Step 4: Verify updates
	handler4 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		username, ok := sess.Get("username")
		require.True(t, ok)
		assert.Equal(t, "updateduser", username)

		_, ok = sess.Get("email")
		assert.False(t, ok)

		w.WriteHeader(http.StatusOK)
	}))

	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.AddCookie(sessionCookie)
	rec4 := httptest.NewRecorder()
	handler4.ServeHTTP(rec4, req4)

	assert.Equal(t, http.StatusOK, rec4.Code)

	// Step 5: Destroy session
	handler5 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := DestroySession(r.Context(), store, config.CookieName, w)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))

	req5 := httptest.NewRequest(http.MethodGet, "/", nil)
	req5.AddCookie(sessionCookie)
	rec5 := httptest.NewRecorder()
	handler5.ServeHTTP(rec5, req5)

	assert.Equal(t, http.StatusOK, rec5.Code)
}

// TestSessionWithCSRFProtection tests session with CSRF middleware
func TestSessionWithCSRFProtection(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	var sessionCookie *http.Cookie
	var csrfToken string

	// Step 1: Get session and CSRF token
	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csrfToken = GetCSRFToken(r.Context())
		require.NotEmpty(t, csrfToken)
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Step 2: Make protected POST request with CSRF token
	handler2 := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler2 = sessionMiddleware(handler2)

	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.AddCookie(sessionCookie)
	req2.Header.Set("X-CSRF-Token", csrfToken)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)

	// Step 3: Regenerate CSRF token (e.g., after login)
	handler3 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldToken := GetCSRFToken(r.Context())
		require.Equal(t, csrfToken, oldToken)

		err := RegenerateCSRFToken(r.Context())
		require.NoError(t, err)

		newToken := GetCSRFToken(r.Context())
		require.NotEqual(t, oldToken, newToken)

		csrfToken = newToken

		w.WriteHeader(http.StatusOK)
	}))

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(sessionCookie)
	rec3 := httptest.NewRecorder()
	handler3.ServeHTTP(rec3, req3)

	// Step 4: Old token should not work
	req4 := httptest.NewRequest(http.MethodPost, "/", nil)
	req4.AddCookie(sessionCookie)
	req4.Header.Set("X-CSRF-Token", "old-token")
	rec4 := httptest.NewRecorder()
	handler2.ServeHTTP(rec4, req4)

	assert.Equal(t, http.StatusForbidden, rec4.Code)

	// Step 5: New token should work
	req5 := httptest.NewRequest(http.MethodPost, "/", nil)
	req5.AddCookie(sessionCookie)
	req5.Header.Set("X-CSRF-Token", csrfToken)
	rec5 := httptest.NewRecorder()
	handler2.ServeHTTP(rec5, req5)

	assert.Equal(t, http.StatusOK, rec5.Code)
}

// TestSessionWithFlashMessages tests flash message workflow
func TestSessionWithFlashMessages(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	sessionMiddleware := Middleware(config)

	var sessionCookie *http.Cookie

	// Step 1: Add flash messages
	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddFlashSuccess(r.Context(), "User created successfully")
		AddFlashInfo(r.Context(), "Welcome to the platform")
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Step 2: Retrieve flash messages (should clear them)
	handler2 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flashes := GetFlashes(r.Context())
		require.Len(t, flashes, 2)

		assert.Equal(t, FlashSuccess, flashes[0].Type)
		assert.Equal(t, "User created successfully", flashes[0].Message)

		assert.Equal(t, FlashInfo, flashes[1].Type)
		assert.Equal(t, "Welcome to the platform", flashes[1].Message)

		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	// Step 3: Flash messages should be gone
	handler3 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flashes := GetFlashes(r.Context())
		assert.Len(t, flashes, 0)
		w.WriteHeader(http.StatusOK)
	}))

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(sessionCookie)
	rec3 := httptest.NewRecorder()
	handler3.ServeHTTP(rec3, req3)

	assert.Equal(t, http.StatusOK, rec3.Code)
}

// TestSessionAuthentication tests user authentication workflow
func TestSessionAuthentication(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	sessionMiddleware := Middleware(config)

	var sessionCookie *http.Cookie

	// Step 1: Anonymous session
	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetAuthenticatedUser(r.Context())
		assert.Empty(t, userID)
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Step 2: Authenticate user
	handler2 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := SetAuthenticatedUser(r.Context(), "user-123")
		require.NoError(t, err)

		userID := GetAuthenticatedUser(r.Context())
		assert.Equal(t, "user-123", userID)

		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	// Step 3: Verify authentication persists
	handler3 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetAuthenticatedUser(r.Context())
		assert.Equal(t, "user-123", userID)
		w.WriteHeader(http.StatusOK)
	}))

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(sessionCookie)
	rec3 := httptest.NewRecorder()
	handler3.ServeHTTP(rec3, req3)

	// Step 4: Logout
	handler4 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ClearAuthenticatedUser(r.Context())

		userID := GetAuthenticatedUser(r.Context())
		assert.Empty(t, userID)

		w.WriteHeader(http.StatusOK)
	}))

	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.AddCookie(sessionCookie)
	rec4 := httptest.NewRecorder()
	handler4.ServeHTTP(rec4, req4)

	assert.Equal(t, http.StatusOK, rec4.Code)
}

// TestSessionExpiration tests session expiration and renewal
func TestSessionExpiration(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false
	config.MaxAge = 1 // 1 second TTL

	sessionMiddleware := Middleware(config)

	var sessionCookie *http.Cookie
	var originalSessionID string

	// Step 1: Create session
	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		originalSessionID = sess.ID
		sess.Set("data", "value")
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Step 2: Wait for expiration
	time.Sleep(2 * time.Second)

	// Step 3: Try to use expired session (should create new one)
	handler2 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		// Should have new session ID
		assert.NotEqual(t, originalSessionID, sess.ID)

		// Old data should be gone
		_, ok := sess.Get("data")
		assert.False(t, ok)

		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

// TestMultipleStoresInteroperability tests that different stores work correctly
func TestMultipleStoresInteroperability(t *testing.T) {
	ctx := context.Background()
	sessionID := "test-session"
	ttl := 1 * time.Hour

	// Create session
	sess := NewSession(sessionID, ttl)
	sess.Set("key1", "value1")
	sess.UserID = "user-123"
	sess.AddFlash(FlashSuccess, "Test message")

	stores := []struct {
		name  string
		store Store
	}{
		{"Memory", NewMemoryStore()},
	}

	for _, tc := range stores {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.store.Close()

			// Set
			err := tc.store.Set(ctx, sessionID, sess, ttl)
			require.NoError(t, err)

			// Get
			retrieved, err := tc.store.Get(ctx, sessionID)
			require.NoError(t, err)
			assert.Equal(t, sessionID, retrieved.ID)
			assert.Equal(t, "user-123", retrieved.UserID)

			val, ok := retrieved.Get("key1")
			require.True(t, ok)
			assert.Equal(t, "value1", val)

			assert.Len(t, retrieved.FlashMessages, 1)
			assert.Equal(t, FlashSuccess, retrieved.FlashMessages[0].Type)

			// Refresh
			err = tc.store.Refresh(ctx, sessionID, ttl)
			require.NoError(t, err)

			// Delete
			err = tc.store.Delete(ctx, sessionID)
			require.NoError(t, err)

			// Verify deleted
			_, err = tc.store.Get(ctx, sessionID)
			assert.Error(t, err)
		})
	}
}
