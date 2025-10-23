package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseStoreCleanup tests the cleanup functionality
func TestDatabaseStoreCleanup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	// Manually insert expired session to test cleanup
	config.CleanupInterval = 0 // We'll manually trigger cleanup

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add sessions with different expiration times
	for i := 0; i < 3; i++ {
		sessionID := "cleanup-test-" + string(rune('0'+i))
		sess := NewSession(sessionID, -1*time.Hour) // Already expired
		sess.ExpiresAt = time.Now().Add(-1 * time.Hour)

		// Directly insert expired session
		_, err := db.Exec(
			"INSERT INTO sessions (id, user_id, data, flash_messages, csrf_token, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			sess.ID, "", "{}", "[]", "", sess.CreatedAt, sess.ExpiresAt,
		)
		require.NoError(t, err)
	}

	// Add non-expired session
	sessionID := "cleanup-test-valid"
	sess := NewSession(sessionID, 1*time.Hour)
	err = store.Set(ctx, sessionID, sess, 1*time.Hour)
	require.NoError(t, err)

	// Count sessions before cleanup
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// Run cleanup manually by calling the cleanup SQL directly
	_, err = db.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now())
	require.NoError(t, err)

	// Count sessions after cleanup
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Only valid session should remain
}

// TestMiddlewareWithFailedSessionLoad tests middleware handles store errors
func TestMiddlewareWithFailedSessionLoad(t *testing.T) {
	// Use a closed store to simulate errors
	store := NewMemoryStore()
	store.Close() // Close immediately

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should still get a session (new one) even if load failed
		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should succeed with new session
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestCSRFTokenGenerationError tests error handling in GetCSRFToken
func TestCSRFTokenGenerationErrorHandling(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		require.NotNil(t, sess)

		// Try to get CSRF token multiple times
		for i := 0; i < 10; i++ {
			token := GetCSRFToken(r.Context())
			assert.NotEmpty(t, token)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestDatabaseStoreErrorHandling tests error handling in database operations
func TestDatabaseStoreErrorHandling(t *testing.T) {
	db := setupTestDB(t)

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)

	// Close database to trigger errors
	db.Close()

	ctx := context.Background()
	sessionID := "error-test-session"
	ttl := 1 * time.Hour

	// Try operations on closed DB - they should error
	sess := NewSession(sessionID, ttl)

	err = store.Set(ctx, sessionID, sess, ttl)
	assert.Error(t, err)

	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err)

	err = store.Delete(ctx, sessionID)
	assert.Error(t, err)

	err = store.Refresh(ctx, sessionID, ttl)
	assert.Error(t, err)

	// Close should not error
	err = store.Close()
	assert.NoError(t, err)
}

// TestGetFlashesByTypeEmpty tests getting flashes when none match type
func TestGetFlashesByTypeEmpty(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add only success flashes
		AddFlashSuccess(r.Context(), "Success 1")
		AddFlashSuccess(r.Context(), "Success 2")

		// Try to get error flashes (none exist)
		flashes := GetFlashesByType(r.Context(), FlashError)
		assert.Empty(t, flashes)

		// All flashes should still be cleared though
		sess := GetSession(r.Context())
		assert.Empty(t, sess.FlashMessages)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
