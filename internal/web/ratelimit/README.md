# Rate Limiting Package

This package provides flexible, high-performance rate limiting for the Conduit web framework.

## Features

- **Multiple Strategies**: Token bucket (in-memory) and sliding window (Redis-backed)
- **High Performance**: <1μs for token bucket, <1ms for Redis
- **Thread-Safe**: All implementations are safe for concurrent use
- **Flexible Key Functions**: Rate limit by IP, user, endpoint, or custom keys
- **Standard Headers**: Compliant with RFC 6585 and industry standards
- **Bypass Logic**: Skip rate limiting for admins or internal requests
- **Production-Ready**: 92.4% test coverage, comprehensive error handling

## Quick Start

### Token Bucket (In-Memory)

```go
package main

import (
    "context"
    "time"

    "github.com/conduit-lang/conduit/internal/web/ratelimit"
)

func main() {
    // Create a limiter: 100 requests per minute
    limiter := ratelimit.NewTokenBucket()
    defer limiter.Close()

    ctx := context.Background()

    // Check if request is allowed
    info, err := limiter.Allow(ctx, "user123")
    if err != nil {
        // Handle error
    }

    if info.Allowed {
        // Process request
        println("Request allowed, remaining:", info.Remaining)
    } else {
        // Rate limited
        println("Rate limited, retry at:", info.ResetAt)
    }
}
```

### Redis-Based (Distributed)

```go
package main

import (
    "context"
    "time"

    "github.com/conduit-lang/conduit/internal/web/ratelimit"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Create Redis rate limiter
    limiter, err := ratelimit.NewRedisRateLimiter(ratelimit.RedisRateLimiterConfig{
        Client: client,
        Limit:  1000,
        Window: time.Hour,
        Prefix: "api:",
    })
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // Check if request is allowed
    info, err := limiter.Allow(ctx, "user456")
    if err != nil {
        // Handle error
    }

    println("Allowed:", info.Allowed)
}
```

## Middleware Usage

```go
package main

import (
    "net/http"

    "github.com/conduit-lang/conduit/internal/web/middleware"
    "github.com/conduit-lang/conduit/internal/web/ratelimit"
)

func main() {
    // Create rate limiter
    limiter := ratelimit.NewTokenBucket()
    defer limiter.Close()

    // Apply to all routes
    http.Handle("/", middleware.RateLimit(limiter)(yourHandler))

    // Or with custom configuration
    http.Handle("/api/", middleware.RateLimitWithConfig(middleware.RateLimitConfig{
        Limiter:    limiter,
        KeyFunc:    middleware.UserEndpointKeyFunc,  // Rate limit per user per endpoint
        BypassFunc: middleware.AdminBypassFunc,       // Skip admins
        FailOpen:   true,                             // Allow on error
    })(yourAPIHandler))
}
```

## Configuration

### Token Bucket Config

```go
config := ratelimit.TokenBucketConfig{
    Capacity:        100,              // Max tokens (requests)
    RefillRate:      time.Minute,      // How often to refill
    CleanupInterval: 5 * time.Minute,  // How often to cleanup old buckets
}

limiter := ratelimit.NewTokenBucketWithConfig(config)
```

### Redis Rate Limiter Config

```go
config := ratelimit.RedisRateLimiterConfig{
    Client: redisClient,
    Limit:  1000,                      // Max requests
    Window: time.Hour,                 // Time window
    Prefix: "ratelimit:",              // Key prefix in Redis
}

limiter, err := ratelimit.NewRedisRateLimiter(config)
```

## Key Functions

Rate limit requests using different strategies:

```go
// By client IP
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    KeyFunc: middleware.IPKeyFunc,
})

// By authenticated user
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    KeyFunc: middleware.UserKeyFunc,
})

// By endpoint
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    KeyFunc: middleware.EndpointKeyFunc,
})

// By user + endpoint (most precise)
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    KeyFunc: middleware.UserEndpointKeyFunc,
})

// Custom key function
customKeyFunc := func(r *http.Request) string {
    return r.Header.Get("X-API-Key")
}
```

## Bypass Functions

Skip rate limiting for certain requests:

```go
// Skip admins
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    BypassFunc: middleware.AdminBypassFunc,
})

// Skip internal requests
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    BypassFunc: middleware.InternalBypassFunc,
})

// Combine multiple bypass conditions
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    BypassFunc: middleware.CombinedBypassFunc(
        middleware.AdminBypassFunc,
        middleware.InternalBypassFunc,
    ),
})
```

## Response Headers

When rate limiting is applied, the following headers are set:

- `X-RateLimit-Limit`: Maximum number of requests allowed
- `X-RateLimit-Remaining`: Number of requests remaining in current window
- `X-RateLimit-Reset`: Unix timestamp when the rate limit resets
- `Retry-After`: Seconds until the client can retry (when rate limited)

## HTTP Status Codes

- `200 OK`: Request allowed
- `429 Too Many Requests`: Rate limit exceeded

## Error Handling

### Fail Open (Default)

When the rate limiter encounters an error, the request is allowed:

```go
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    FailOpen: true,  // Allow requests on error
})
```

### Fail Closed

When the rate limiter encounters an error, the request is denied:

```go
middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    FailOpen: false,  // Deny requests on error
    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
        http.Error(w, "Rate limiting unavailable", http.StatusServiceUnavailable)
    },
})
```

## Performance

Benchmarks on Apple M1:

- **Token Bucket**: ~60-192 ns/op
- **Redis**: ~112-599 μs/op
- **Middleware Overhead**: ~880 ns/op

All well under the <1ms (token bucket) and <5ms (Redis) targets.

## Testing

Run tests:

```bash
go test ./internal/web/ratelimit/...
```

Run with race detector:

```bash
go test -race ./internal/web/ratelimit/...
```

Run benchmarks:

```bash
go test -bench=. ./internal/web/ratelimit/...
```

## Thread Safety

All rate limiter implementations are thread-safe and can be used concurrently from multiple goroutines. The token bucket uses mutex-based synchronization, while the Redis implementation relies on Lua scripts for atomic operations.

## Best Practices

1. **Choose the right strategy**:
   - Use token bucket for single-server deployments
   - Use Redis for distributed systems

2. **Select appropriate key functions**:
   - IP-based: Simple but can be circumvented
   - User-based: Requires authentication
   - User+Endpoint: Most precise, prevents per-endpoint abuse

3. **Configure bypass logic**:
   - Always allow health checks
   - Consider exempting admins
   - Skip rate limiting for internal services

4. **Set reasonable limits**:
   - Start conservative and adjust based on monitoring
   - Different limits for different endpoint types
   - Higher limits for authenticated users

5. **Monitor and alert**:
   - Track rate limit hits
   - Alert on unusual patterns
   - Log denied requests for analysis

## Architecture

```
┌─────────────────┐
│   HTTP Request  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Middleware    │──► Extract key (IP/User/Endpoint)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Rate Limiter   │──► Check bypass function
│   (Interface)   │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌──────┐  ┌──────┐
│Token │  │Redis │
│Bucket│  │ SWin │
└──────┘  └──────┘
    │         │
    └────┬────┘
         │
         ▼
  ┌──────────────┐
  │ RateLimitInfo│
  │ - Limit      │
  │ - Remaining  │
  │ - ResetAt    │
  │ - Allowed    │
  └──────────────┘
         │
         ▼
┌─────────────────┐
│ Set Headers     │
│ Return 200/429  │
└─────────────────┘
```

## Code Generation

Conduit supports automatic rate limiting via annotations:

```conduit
resource Post {
  id: uuid! @primary @auto
  title: string!

  @rate_limit(limit: 100, window: 60, strategy: "token_bucket", key: "user")
}
```

This generates middleware configuration automatically.

## Contributing

When adding new rate limiting strategies:

1. Implement the `RateLimiter` interface
2. Add comprehensive tests (>90% coverage)
3. Include benchmarks
4. Document performance characteristics
5. Update this README

## License

Part of the Conduit framework.
