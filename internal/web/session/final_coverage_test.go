package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMiddlewareGenerateSessionIDError tests error handling when session ID generation fails
func TestMiddlewareSessionCreationFlow(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	// Test multiple requests to ensure session generation works repeatedly
	for i := 0; i < 5; i++ {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			require.NotNil(t, sess)
			assert.NotEmpty(t, sess.ID)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

// TestCSRFExtractFromHeader tests CSRF token extraction from header
func TestCSRFExtractFromHeader(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	// Get token
	var token string
	var sessionCookie *http.Cookie

	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = GetCSRFToken(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Test with header
	handler2 := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler2 = sessionMiddleware(handler2)

	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.AddCookie(sessionCookie)
	req2.Header.Set("X-CSRF-Token", token)
	rec2 := httptest.NewRecorder()

	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

// TestCSRFExtractFromFormUrlEncoded tests CSRF token from URL-encoded form
func TestCSRFExtractFromFormUrlEncoded(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	// Get token
	var token string
	var sessionCookie *http.Cookie

	handler1 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = GetCSRFToken(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Test with form data
	handler2 := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler2 = sessionMiddleware(handler2)

	formData := "csrf_token=" + token + "&field=value"
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
	req2.AddCookie(sessionCookie)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()

	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

// TestDatabaseStoreNullFields tests handling of NULL fields
func TestDatabaseStoreNullFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "null-fields-test"
	ttl := 1 * time.Hour

	// Create session with no user ID, CSRF token, or flash messages
	sess := NewSession(sessionID, ttl)
	sess.UserID = "" // Explicitly empty
	sess.CSRFToken = ""
	sess.FlashMessages = nil

	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Retrieve and verify NULL handling
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Empty(t, retrieved.UserID)
	assert.Empty(t, retrieved.CSRFToken)
	assert.Empty(t, retrieved.FlashMessages)
}

// TestDatabaseStoreWithUserAndCSRF tests all fields populated
func TestDatabaseStoreWithUserAndCSRF(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "full-fields-test"
	ttl := 1 * time.Hour

	// Create session with all fields
	sess := NewSession(sessionID, ttl)
	sess.UserID = "user-456"
	sess.CSRFToken = "csrf-token-789"
	sess.AddFlash(FlashWarning, "Warning")
	sess.AddFlash(FlashInfo, "Info")

	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "user-456", retrieved.UserID)
	assert.Equal(t, "csrf-token-789", retrieved.CSRFToken)
	assert.Len(t, retrieved.FlashMessages, 2)
}

// TestHasFlashesByTypeWithMultipleTypes tests HasFlashesByType
func TestHasFlashesByTypeVariety(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add multiple types
		AddFlashSuccess(r.Context(), "Success")
		AddFlashError(r.Context(), "Error")

		// Check each type
		assert.True(t, HasFlashesByType(r.Context(), FlashSuccess))
		assert.True(t, HasFlashesByType(r.Context(), FlashError))
		assert.False(t, HasFlashesByType(r.Context(), FlashWarning))
		assert.False(t, HasFlashesByType(r.Context(), FlashInfo))

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestMemoryStoreCount tests the Count method
func TestMemoryStoreCountMethod(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	assert.Equal(t, 0, store.Count())

	// Add sessions
	for i := 0; i < 10; i++ {
		sessionID := "count-test-" + string(rune('0'+i))
		sess := NewSession(sessionID, 1*time.Hour)
		store.Set(ctx, sessionID, sess, 1*time.Hour)
	}

	assert.Equal(t, 10, store.Count())

	// Delete some
	for i := 0; i < 5; i++ {
		sessionID := "count-test-" + string(rune('0'+i))
		store.Delete(ctx, sessionID)
	}

	assert.Equal(t, 5, store.Count())
}

// TestDestroySessionDestroysFlag tests that destroyed flag is set
func TestDestroySessionSetsDestroyedFlag(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		// Should not be destroyed initially
		assert.False(t, sess.destroyed)

		// Destroy
		err := DestroySession(r.Context(), store, config.CookieName, w)
		require.NoError(t, err)

		// Should be marked as destroyed
		assert.True(t, sess.destroyed)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestDatabaseStoreTableCreation tests table and index creation
func TestDatabaseStoreTableCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.TableName = "custom_sessions"
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_sessions'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "custom_sessions", tableName)

	// Verify index exists
	var indexName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_custom_sessions_%'").Scan(&indexName)
	require.NoError(t, err)
	assert.Contains(t, indexName, "idx_custom_sessions_")
}

// TestGenerateCSRFTokenDifferentLengths tests token generation with different lengths
func TestGenerateCSRFTokenDifferentLengths(t *testing.T) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		token, err := generateCSRFToken(length)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		// Base64 encoding increases size
		assert.Greater(t, len(token), length)
	}
}

// TestNewDatabaseStoreWithCleanup tests store creation with cleanup enabled
func TestNewDatabaseStoreWithCleanup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	// Enable cleanup with very long interval so it doesn't run during test
	config.CleanupInterval = 1 * time.Hour

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Close immediately to stop cleanup goroutine
	store.Close()
}
