package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()
	assert.NotZero(t, config.DefaultTTL)
	assert.NotEmpty(t, config.Prefix)
	assert.Equal(t, "conduit:", config.Prefix)
}

func TestErrCacheMiss(t *testing.T) {
	err := ErrCacheMiss{Key: "test"}
	assert.Equal(t, "cache miss: test", err.Error())
	assert.True(t, IsCacheMiss(err))
}

func TestIsCacheMiss(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "cache miss error",
			err:      ErrCacheMiss{Key: "test"},
			expected: true,
		},
		{
			name:     "other error",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCacheMiss(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
