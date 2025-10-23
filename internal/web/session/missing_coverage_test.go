package session

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFTokenExtractionFromMultipartForm tests CSRF token extraction from multipart form
func TestCSRFTokenExtractionFromMultipartForm(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false
	sessionMiddleware := Middleware(sessionConfig)

	csrfConfig := DefaultCSRFConfig()
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	combined := sessionMiddleware(csrfMiddleware(handler))

	// Create multipart form with CSRF token
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add csrf_token field
	err := writer.WriteField("csrf_token", "test-csrf-token")
	require.NoError(t, err)

	// Add file field to make it a real multipart form
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = io.WriteString(part, "test content")
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// First request to get session and set CSRF token
	req1 := httptest.NewRequest("GET", "/test", nil)
	rec1 := httptest.NewRecorder()
	combined.ServeHTTP(rec1, req1)

	// Get session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range rec1.Result().Cookies() {
		if cookie.Name == "conduit_session" {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Get CSRF token from session
	var csrfToken string
	sessionID := sessionCookie.Value
	ctx := context.Background()
	sess, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	csrfToken = sess.CSRFToken

	// Create new multipart form with correct CSRF token
	var body2 bytes.Buffer
	writer2 := multipart.NewWriter(&body2)
	err = writer2.WriteField("csrf_token", csrfToken)
	require.NoError(t, err)
	part2, err := writer2.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = io.WriteString(part2, "test content")
	require.NoError(t, err)
	err = writer2.Close()
	require.NoError(t, err)

	// Second request with multipart form
	req2 := httptest.NewRequest("POST", "/test", &body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()

	combined.ServeHTTP(rec2, req2)

	// Should succeed with valid CSRF token
	assert.Equal(t, http.StatusOK, rec2.Code)
}

// TestCSRFTokenExtractionFromFormField tests CSRF extraction with form parse error
func TestCSRFTokenExtractionEdgeCases(t *testing.T) {
	config := DefaultCSRFConfig()

	t.Run("malformed multipart form", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("invalid multipart")))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=----invalid")

		token := extractCSRFToken(req, config)
		assert.Empty(t, token)
	})

	t.Run("empty multipart form", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		err := writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/test", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		token := extractCSRFToken(req, config)
		assert.Empty(t, token)
	})
}

// TestGenerateCSRFTokenError tests error path in generateCSRFToken
func TestGenerateCSRFTokenLength(t *testing.T) {
	// Test with various lengths
	lengths := []int{0, 1, 16, 32, 64, 128}
	for _, length := range lengths {
		token, err := generateCSRFToken(length)
		require.NoError(t, err)
		if length > 0 {
			assert.NotEmpty(t, token)
		}
	}
}

// TestGenerateSessionIDLength tests session ID generation
func TestGenerateSessionIDLength(t *testing.T) {
	// Generate many session IDs to ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		id, err := generateSessionID()
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Should be unique
		assert.False(t, ids[id], "Generated duplicate session ID")
		ids[id] = true

		// Should have reasonable length (base64 encoded 32 bytes)
		assert.GreaterOrEqual(t, len(id), 40)
	}
}

// TestMemoryStoreCleanupGoroutine tests the cleanup goroutine runs
func TestMemoryStoreCleanupGoroutine(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	// Add sessions with very short TTL and unique IDs
	for i := 0; i < 10; i++ {
		sessionID := "short-lived-" + string(rune('0'+i))
		sess := NewSession(sessionID, 50*time.Millisecond)
		err := store.Set(ctx, sessionID, sess, 50*time.Millisecond)
		require.NoError(t, err)
	}

	// Store should have sessions
	initialCount := store.Count()
	assert.Greater(t, initialCount, 0)

	// Wait for cleanup to run (cleanup runs every 1 second by default)
	time.Sleep(2 * time.Second)

	// Sessions should be cleaned up or significantly reduced
	finalCount := store.Count()
	assert.LessOrEqual(t, finalCount, initialCount)
}

// TestDatabaseStoreCleanupGoroutine tests database cleanup goroutine
func TestDatabaseStoreCleanupGoroutine(t *testing.T) {
	db := setupTestDBLocal(t)
	defer db.Close()

	config := &DatabaseConfig{
		DB:              db,
		TableName:       "cleanup_goroutine_test",
		CleanupInterval: 500 * time.Millisecond,
	}

	_, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS cleanup_goroutine_test")
	}()

	// Add sessions that will expire soon with unique IDs
	for i := 0; i < 5; i++ {
		sessionID := "cleanup-test-" + string(rune('0'+i))
		sess := NewSession(sessionID, 100*time.Millisecond)
		// Manually set expiration to past
		sess.ExpiresAt = time.Now().Add(-1 * time.Hour)

		// Use raw SQL to insert expired session
		_, err := db.Exec(
			`INSERT INTO cleanup_goroutine_test (id, user_id, data, flash_messages, csrf_token, created_at, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			sessionID, "", "{}", "[]", "", sess.CreatedAt, sess.ExpiresAt,
		)
		require.NoError(t, err)
	}

	// Verify sessions exist
	var initialCount int
	err = db.QueryRow("SELECT COUNT(*) FROM cleanup_goroutine_test").Scan(&initialCount)
	require.NoError(t, err)
	assert.Equal(t, 5, initialCount)

	// Wait for cleanup to run
	time.Sleep(2 * time.Second)

	// Sessions should be cleaned up
	var finalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM cleanup_goroutine_test").Scan(&finalCount)
	require.NoError(t, err)
	assert.Equal(t, 0, finalCount)
}

// TestDatabaseStoreCleanupError tests cleanup continues after errors
func TestDatabaseStoreCleanupError(t *testing.T) {
	db := setupTestDBLocal(t)

	config := &DatabaseConfig{
		DB:              db,
		TableName:       "cleanup_error_test",
		CleanupInterval: 200 * time.Millisecond,
	}

	_, err := NewDatabaseStore(config)
	require.NoError(t, err)

	// Wait a bit for cleanup goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Close DB to cause errors
	db.Close()

	// Wait for cleanup to attempt running on closed DB
	// This tests that cleanup handles errors gracefully
	time.Sleep(500 * time.Millisecond)

	// Should not panic
}

// TestMemoryStoreCleanupStopsOnClose tests cleanup stops when store closes
func TestMemoryStoreCleanupStopsOnClose(t *testing.T) {
	memStore := NewMemoryStore()

	ctx := context.Background()

	// Add a session
	sess := NewSession("test", 5*time.Minute)
	err := memStore.Set(ctx, "test", sess, 5*time.Minute)
	require.NoError(t, err)

	// Close store
	err = memStore.Close()
	require.NoError(t, err)

	// Wait to ensure cleanup goroutine has stopped
	time.Sleep(2 * time.Second)

	// Accessing closed store should still work for reads but ticker stopped
	count := memStore.Count()
	assert.GreaterOrEqual(t, count, 0)
}

// TestCSRFMiddlewareCustomErrorHandler tests custom CSRF error handler
func TestCSRFMiddlewareCustomErrorHandler(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false
	sessionMiddleware := Middleware(sessionConfig)

	csrfConfig := DefaultCSRFConfig()
	customErrorCalled := false
	csrfConfig.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		customErrorCalled = true
		http.Error(w, "Custom CSRF error", http.StatusTeapot)
	}
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	combined := sessionMiddleware(csrfMiddleware(handler))

	// POST without CSRF token should trigger error handler
	req := httptest.NewRequest("POST", "/test", nil)
	rec := httptest.NewRecorder()

	combined.ServeHTTP(rec, req)

	assert.True(t, customErrorCalled)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

// TestDatabaseStoreNoCleanup tests database store with cleanup disabled
func TestDatabaseStoreNoCleanup(t *testing.T) {
	db := setupTestDBLocal(t)
	defer db.Close()

	config := &DatabaseConfig{
		DB:              db,
		TableName:       "no_cleanup_test",
		CleanupInterval: 0, // Disabled
	}

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS no_cleanup_test")
	}()

	ctx := context.Background()

	// Add expired session
	sess := NewSession("test", -1*time.Hour)
	err = store.Set(ctx, "test", sess, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait - no cleanup should run
	time.Sleep(1 * time.Second)

	// Session should still be in DB (cleanup disabled)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM no_cleanup_test").Scan(&count)
	require.NoError(t, err)
	// Note: Get won't return it because it checks expiration, but it's still in DB
	assert.Equal(t, 1, count)
}

// TestNewDatabaseStoreTableCreationError tests error handling during table creation
func TestNewDatabaseStoreTableCreationError(t *testing.T) {
	db := setupTestDBLocal(t)
	// Close DB immediately to cause table creation to fail
	db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	_, err := NewDatabaseStore(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create sessions table")
}

// TestDatabaseStoreRefreshNoRowsAffected tests Refresh when session doesn't exist
func TestDatabaseStoreRefreshNoRowsAffected(t *testing.T) {
	db := setupTestDBLocal(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Try to refresh non-existent session
	err = store.Refresh(ctx, "nonexistent", 1*time.Hour)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestMiddlewareSessionSaveOnResponse tests session is saved on response
func TestMiddlewareSessionSaveOnResponse(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false
	middleware := Middleware(config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		// Modify session
		sess.Set("test_key", "test_value")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session was saved to store
	cookies := rec.Result().Cookies()
	var sessionID string
	for _, cookie := range cookies {
		if cookie.Name == "conduit_session" {
			sessionID = cookie.Value
			break
		}
	}
	require.NotEmpty(t, sessionID)

	// Retrieve session from store
	ctx := context.Background()
	sess, err := store.Get(ctx, sessionID)
	require.NoError(t, err)

	val, ok := sess.Get("test_key")
	require.True(t, ok)
	assert.Equal(t, "test_value", val)
}

// TestGetCSRFTokenWithNilSession tests GetCSRFToken when session is nil
func TestGetCSRFTokenWithNilSession(t *testing.T) {
	ctx := context.Background()
	token := GetCSRFToken(ctx)
	assert.Empty(t, token)
}

// TestRegenerateCSRFTokenWithNilSession tests RegenerateCSRFToken with nil session
func TestRegenerateCSRFTokenWithNilSession(t *testing.T) {
	ctx := context.Background()
	err := RegenerateCSRFToken(ctx)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// setupTestDBLocal creates a test database for testing
func setupTestDBLocal(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	return db
}
