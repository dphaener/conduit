package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreGetSet(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "test-session-id"
	ttl := 1 * time.Hour

	// Create a session
	sess := NewSession(sessionID, ttl)
	sess.Set("user_id", "123")
	sess.Set("email", "test@example.com")

	// Store it
	err := store.Set(ctx, sessionID, sess, ttl)
	require.NoError(t, err)

	// Retrieve it
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrieved.ID)

	val, ok := retrieved.Get("user_id")
	require.True(t, ok)
	assert.Equal(t, "123", val)

	val, ok = retrieved.Get("email")
	require.True(t, ok)
	assert.Equal(t, "test@example.com", val)
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryStoreGetExpired(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "test-session-id"

	// Create a session with very short TTL
	sess := NewSession(sessionID, 1*time.Millisecond)
	err := store.Set(ctx, sessionID, sess, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Try to retrieve
	_, err = store.Get(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionExpired)

	// Session should be deleted
	assert.Equal(t, 0, store.Count())
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "test-session-id"

	// Create and store a session
	sess := NewSession(sessionID, 1*time.Hour)
	err := store.Set(ctx, sessionID, sess, 1*time.Hour)
	require.NoError(t, err)

	// Verify it exists
	assert.Equal(t, 1, store.Count())

	// Delete it
	err = store.Delete(ctx, sessionID)
	require.NoError(t, err)

	// Verify it's gone
	assert.Equal(t, 0, store.Count())
	_, err = store.Get(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryStoreRefresh(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "test-session-id"

	// Create a session with short TTL
	sess := NewSession(sessionID, 100*time.Millisecond)
	err := store.Set(ctx, sessionID, sess, 100*time.Millisecond)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Refresh with longer TTL
	err = store.Refresh(ctx, sessionID, 1*time.Hour)
	require.NoError(t, err)

	// Wait past original expiration
	time.Sleep(100 * time.Millisecond)

	// Session should still be valid
	_, err = store.Get(ctx, sessionID)
	assert.NoError(t, err)
}

func TestMemoryStoreRefreshNotFound(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	err := store.Refresh(ctx, "nonexistent", 1*time.Hour)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryStoreCleanup(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	// Create multiple sessions with different TTLs
	for i := 0; i < 5; i++ {
		sessionID := "session-" + string(rune('0'+i))
		sess := NewSession(sessionID, 100*time.Millisecond)
		err := store.Set(ctx, sessionID, sess, 100*time.Millisecond)
		require.NoError(t, err)
	}

	// Create sessions that won't expire
	for i := 5; i < 10; i++ {
		sessionID := "session-" + string(rune('0'+i))
		sess := NewSession(sessionID, 1*time.Hour)
		err := store.Set(ctx, sessionID, sess, 1*time.Hour)
		require.NoError(t, err)
	}

	assert.Equal(t, 10, store.Count())

	// Wait for first batch to expire
	time.Sleep(200 * time.Millisecond)

	// Trigger cleanup by waiting for the cleanup interval
	// The cleanup runs every minute, so we'll manually verify expired sessions are gone
	// by trying to access them
	for i := 0; i < 5; i++ {
		sessionID := "session-" + string(rune('0'+i))
		_, err := store.Get(ctx, sessionID)
		assert.Error(t, err) // Should be expired
	}

	// Non-expired sessions should still be there
	for i := 5; i < 10; i++ {
		sessionID := "session-" + string(rune('0'+i))
		_, err := store.Get(ctx, sessionID)
		assert.NoError(t, err)
	}
}

func TestMemoryStoreClose(t *testing.T) {
	store := NewMemoryStore()

	ctx := context.Background()

	// Add some sessions
	for i := 0; i < 5; i++ {
		sessionID := "session-" + string(rune('0'+i))
		sess := NewSession(sessionID, 1*time.Hour)
		err := store.Set(ctx, sessionID, sess, 1*time.Hour)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, store.Count())

	// Close the store
	err := store.Close()
	require.NoError(t, err)

	// All sessions should be cleared
	assert.Equal(t, 0, store.Count())
}

func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	ttl := 1 * time.Hour

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			sessionID := "session-" + string(rune('0'+id))
			sess := NewSession(sessionID, ttl)
			sess.Set("value", id)

			// Set
			err := store.Set(ctx, sessionID, sess, ttl)
			assert.NoError(t, err)

			// Get
			retrieved, err := store.Get(ctx, sessionID)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved)

			// Delete
			err = store.Delete(ctx, sessionID)
			assert.NoError(t, err)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
