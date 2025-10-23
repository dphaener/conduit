package session

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	return db
}

func TestDatabaseStoreConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)

	assert.NotNil(t, config.DB)
	assert.Equal(t, "sessions", config.TableName)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
}

func TestNewDatabaseStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0 // Disable cleanup for testing

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Verify table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "sessions", tableName)

	store.Close()
}

func TestDatabaseStoreGetSet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-session"
	ttl := 1 * time.Hour

	// Create session
	sess := NewSession(sessionID, ttl)
	sess.Set("key1", "value1")
	sess.Set("key2", 42)
	sess.UserID = "user-123"
	sess.CSRFToken = "csrf-token-123"
	sess.AddFlash(FlashSuccess, "Test message")

	// Set
	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Get
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrieved.ID)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, "csrf-token-123", retrieved.CSRFToken)

	val, ok := retrieved.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)

	val, ok = retrieved.Get("key2")
	require.True(t, ok)
	// JSON unmarshals numbers as float64
	assert.Equal(t, float64(42), val)

	assert.Len(t, retrieved.FlashMessages, 1)
	assert.Equal(t, FlashSuccess, retrieved.FlashMessages[0].Type)
	assert.Equal(t, "Test message", retrieved.FlashMessages[0].Message)
}

func TestDatabaseStoreUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-update"
	ttl := 1 * time.Hour

	// Create session
	sess := NewSession(sessionID, ttl)
	sess.Set("key1", "value1")
	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Update session
	sess.Set("key1", "updated-value")
	sess.Set("key2", "new-value")
	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)

	val, ok := retrieved.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "updated-value", val)

	val, ok = retrieved.Get("key2")
	require.True(t, ok)
	assert.Equal(t, "new-value", val)
}

func TestDatabaseStoreDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-delete"
	ttl := 1 * time.Hour

	// Create session
	sess := NewSession(sessionID, ttl)
	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Delete
	err = store.Delete(ctx, sessionID)
	require.NoError(t, err)

	// Verify deleted
	_, err = store.Get(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestDatabaseStoreRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-refresh"
	shortTTL := 100 * time.Millisecond

	// Create session with short TTL
	sess := NewSession(sessionID, shortTTL)
	err = store.Set(ctx, sessionID, sess, shortTTL)
	require.NoError(t, err)

	// Refresh with longer TTL
	err = store.Refresh(ctx, sessionID, 1*time.Hour)
	require.NoError(t, err)

	// Wait past original TTL
	time.Sleep(200 * time.Millisecond)

	// Session should still exist
	_, err = store.Get(ctx, sessionID)
	assert.NoError(t, err)
}

func TestDatabaseStoreGetNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	_, err = store.Get(ctx, "nonexistent-session")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestDatabaseStoreRefreshNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	err = store.Refresh(ctx, "nonexistent-session", 1*time.Hour)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestDatabaseStoreExpiredSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-expired"

	// Create session with very short TTL
	sess := NewSession(sessionID, 10*time.Millisecond)
	err = store.Set(ctx, sessionID, sess, 10*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	// Should not be retrievable
	_, err = store.Get(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestDatabaseStoreEmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	sessionID := "db-test-empty"
	ttl := 1 * time.Hour

	// Create session with no data
	sess := NewSession(sessionID, ttl)

	err = store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Get
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Data)
	assert.Empty(t, retrieved.Data)
	assert.Empty(t, retrieved.FlashMessages)
	assert.Empty(t, retrieved.UserID)
	assert.Empty(t, retrieved.CSRFToken)
}

func TestDatabaseStoreClose(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultDatabaseConfig(db)
	config.CleanupInterval = 0

	store, err := NewDatabaseStore(config)
	require.NoError(t, err)

	// Close should not error
	err = store.Close()
	assert.NoError(t, err)
}
