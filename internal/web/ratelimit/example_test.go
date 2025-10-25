package ratelimit_test

import (
	"context"
	"fmt"
	"time"

	"github.com/conduit-lang/conduit/internal/web/ratelimit"
)

// Example demonstrates basic token bucket rate limiting
func Example_tokenBucket() {
	// Create a token bucket limiter: 5 requests per second
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        5,
		RefillRate:      time.Second,
		CleanupInterval: time.Minute,
	})
	defer limiter.Close()

	ctx := context.Background()

	// Make 6 requests
	for i := 1; i <= 6; i++ {
		info, err := limiter.Allow(ctx, "user123")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if info.Allowed {
			fmt.Printf("Request %d: Allowed (remaining: %d)\n", i, info.Remaining)
		} else {
			fmt.Printf("Request %d: Rate limited (limit: %d)\n", i, info.Limit)
		}
	}

	// Output:
	// Request 1: Allowed (remaining: 4)
	// Request 2: Allowed (remaining: 3)
	// Request 3: Allowed (remaining: 2)
	// Request 4: Allowed (remaining: 1)
	// Request 5: Allowed (remaining: 0)
	// Request 6: Rate limited (limit: 5)
}

// Example demonstrates per-key isolation
func Example_perKeyIsolation() {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	ctx := context.Background()

	// User1 makes requests
	for i := 1; i <= 3; i++ {
		info, _ := limiter.Allow(ctx, "user1")
		if info.Allowed {
			fmt.Printf("User1 request %d: Allowed\n", i)
		} else {
			fmt.Printf("User1 request %d: Rate limited\n", i)
		}
	}

	// User2 has separate limit
	info, _ := limiter.Allow(ctx, "user2")
	if info.Allowed {
		fmt.Println("User2 request 1: Allowed")
	}

	// Output:
	// User1 request 1: Allowed
	// User1 request 2: Allowed
	// User1 request 3: Rate limited
	// User2 request 1: Allowed
}
