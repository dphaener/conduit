package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHasFlashesByTypeWithNilSession tests HasFlashesByType when session is nil
func TestHasFlashesByTypeWithNilSession(t *testing.T) {
	ctx := context.Background()
	result := HasFlashesByType(ctx, "success")
	assert.False(t, result)
}
