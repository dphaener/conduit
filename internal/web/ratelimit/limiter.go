package ratelimit

import (
	"context"
	"time"
)

// RateLimiter defines the interface for rate limiting implementations
type RateLimiter interface {
	// Allow checks if a request should be allowed for the given key
	// Returns RateLimitInfo with current state and error if any
	Allow(ctx context.Context, key string) (*RateLimitInfo, error)
}

// RateLimitInfo contains information about the current rate limit state
type RateLimitInfo struct {
	// Limit is the maximum number of requests allowed in the window
	Limit int
	// Remaining is the number of requests remaining in the current window
	Remaining int
	// ResetAt is when the rate limit window resets
	ResetAt time.Time
	// Allowed indicates whether the request should be allowed
	Allowed bool
}
