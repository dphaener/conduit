package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	ttl := 1 * time.Hour
	sess := NewSession("test-id", ttl)

	assert.Equal(t, "test-id", sess.ID)
	assert.NotNil(t, sess.Data)
	assert.NotNil(t, sess.FlashMessages)
	assert.False(t, sess.CreatedAt.IsZero())
	assert.False(t, sess.ExpiresAt.IsZero())
	assert.WithinDuration(t, time.Now().Add(ttl), sess.ExpiresAt, 1*time.Second)
}

func TestSessionIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		expected bool
	}{
		{
			name:     "not expired",
			ttl:      1 * time.Hour,
			expected: false,
		},
		{
			name:     "expired",
			ttl:      -1 * time.Hour,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := NewSession("test-id", tt.ttl)
			assert.Equal(t, tt.expected, sess.IsExpired())
		})
	}
}

func TestSessionGetSet(t *testing.T) {
	sess := NewSession("test-id", 1*time.Hour)

	// Test Set and Get
	sess.Set("key1", "value1")
	sess.Set("key2", 42)
	sess.Set("key3", true)

	val1, ok1 := sess.Get("key1")
	require.True(t, ok1)
	assert.Equal(t, "value1", val1)

	val2, ok2 := sess.Get("key2")
	require.True(t, ok2)
	assert.Equal(t, 42, val2)

	val3, ok3 := sess.Get("key3")
	require.True(t, ok3)
	assert.Equal(t, true, val3)

	// Test non-existent key
	_, ok := sess.Get("nonexistent")
	assert.False(t, ok)
}

func TestSessionDelete(t *testing.T) {
	sess := NewSession("test-id", 1*time.Hour)

	sess.Set("key1", "value1")
	_, ok := sess.Get("key1")
	require.True(t, ok)

	sess.Delete("key1")
	_, ok = sess.Get("key1")
	assert.False(t, ok)
}

func TestSessionFlashMessages(t *testing.T) {
	sess := NewSession("test-id", 1*time.Hour)

	// Add flash messages
	sess.AddFlash(FlashSuccess, "Success message")
	sess.AddFlash(FlashError, "Error message")
	sess.AddFlash(FlashWarning, "Warning message")

	assert.Len(t, sess.FlashMessages, 3)

	// Get flashes (should clear them)
	flashes := sess.GetFlashes()
	assert.Len(t, flashes, 3)
	assert.Equal(t, FlashSuccess, flashes[0].Type)
	assert.Equal(t, "Success message", flashes[0].Message)

	// Flashes should be cleared
	assert.Len(t, sess.FlashMessages, 0)

	// Getting again should return empty
	flashes = sess.GetFlashes()
	assert.Len(t, flashes, 0)
}

func TestDefaultConfig(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultConfig(store)

	assert.Equal(t, "conduit_session", config.CookieName)
	assert.Equal(t, "/", config.CookiePath)
	assert.Equal(t, 86400*7, config.MaxAge)
	assert.True(t, config.HttpOnly)
	assert.True(t, config.Secure)
	assert.Equal(t, "Lax", config.SameSite)
	assert.NotNil(t, config.Store)
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Check uniqueness
		assert.False(t, ids[id], "Session ID should be unique")
		ids[id] = true

		// Check length (base64 encoded 32 bytes should be ~44 chars)
		assert.Greater(t, len(id), 40)
	}
}

func TestSameSiteFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"Strict", 3}, // http.SameSiteStrictMode
		{"Lax", 2},    // http.SameSiteLaxMode
		{"None", 4},   // http.SameSiteNoneMode
		{"invalid", 2}, // defaults to Lax
		{"", 2},        // defaults to Lax
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sameSiteFromString(tt.input)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}
