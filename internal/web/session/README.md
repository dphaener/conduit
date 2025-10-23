# Session Storage Component

This package provides comprehensive session management for the Conduit web framework.

## Features

- Multiple storage backends (memory, Redis, database)
- Secure cookie configuration (HttpOnly, Secure, SameSite)
- CSRF protection middleware
- Flash messages
- Automatic session expiration and cleanup
- Type-safe session access through context

## Storage Backends

### Memory Store
In-memory session storage suitable for development and testing. Sessions are lost on application restart.

```go
store := session.NewMemoryStore()
defer store.Close()
```

### Redis Store
Production-ready session storage using Redis for persistence and horizontal scaling.

```go
config := session.DefaultRedisConfig("localhost:6379")
store := session.NewRedisStore(config)
defer store.Close()
```

### Database Store
SQL database-backed session storage with automatic cleanup.

```go
config := session.DefaultDatabaseConfig(db)
store, err := session.NewDatabaseStore(config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

## Usage

### Basic Setup

```go
// Create store
store := session.NewMemoryStore()
defer store.Close()

// Configure session
config := session.DefaultConfig(store)
config.MaxAge = 86400 * 7 // 7 days

// Add middleware
router.Use(session.Middleware(config))
```

### Working with Sessions

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get session
    sess := session.GetSession(r.Context())

    // Store data
    sess.Set("user_id", "123")

    // Retrieve data
    userID, ok := sess.Get("user_id")

    // Delete data
    sess.Delete("user_id")
}
```

### Authentication

```go
// Login - set authenticated user and regenerate session ID
session.SetAuthenticatedUser(r.Context(), "user-123")

// IMPORTANT: Regenerate session ID after authentication to prevent session fixation
if err := session.RegenerateSessionID(r.Context(), store, config, w); err != nil {
    http.Error(w, "Session regeneration failed", http.StatusInternalServerError)
    return
}

// Get authenticated user
userID := session.GetAuthenticatedUser(r.Context())

// Logout
session.ClearAuthenticatedUser(r.Context())
session.DestroySession(r.Context(), store, "conduit_session", w)
```

### CSRF Protection

```go
// Add CSRF middleware after session middleware
csrfConfig := session.DefaultCSRFConfig()
router.Use(session.CSRFMiddleware(csrfConfig))

// Get CSRF token in handler
token := session.GetCSRFToken(r.Context())

// In template: <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
```

### Flash Messages

```go
// Add flash message
session.AddFlashSuccess(r.Context(), "Operation successful")
session.AddFlashError(r.Context(), "An error occurred")

// Get and clear flashes (typically in next request)
flashes := session.GetFlashes(r.Context())
for _, flash := range flashes {
    fmt.Printf("%s: %s\n", flash.Type, flash.Message)
}
```

## Test Coverage

Current test coverage: **80.3%**

### Coverage Breakdown

- **Core session functionality**: 100%
- **Memory store**: 95%
- **Database store**: 85%
- **Middleware**: 85%
- **CSRF protection**: 90%
- **Flash messages**: 95%
- **Redis store**: 0% (requires Redis infrastructure)

### Uncovered Code

The following code is not covered by unit tests:

1. **Redis Store Methods** (0%): Requires Redis server for integration tests
   - Get, Set, Delete, Refresh operations
   - These are tested in `redis_store_test.go` but skipped without Redis

2. **Background Cleanup Goroutines** (~50%):
   - Memory store cleanup (runs every minute)
   - Database store cleanup (configurable interval)
   - These are difficult to test deterministically

3. **Error Paths**:
   - Rare error conditions (e.g., crypto.rand failures)
   - Database connection failures
   - JSON marshal/unmarshal errors

### Running Tests

```bash
# Run all tests
go test ./internal/web/session/...

# Run with coverage
go test -cover ./internal/web/session/...

# Generate coverage report
go test -coverprofile=coverage.out ./internal/web/session/...
go tool cover -html=coverage.out
```

### Integration Tests

To run Redis integration tests:

```bash
# Start Redis
docker run -d -p 6379:6379 redis:latest

# Run tests (integration tests are skipped in short mode)
go test -v ./internal/web/session/...
```

## Security Considerations

1. **Secure Cookies**: Always set `Secure: true` in production (requires HTTPS)
2. **SameSite**: Use `SameSiteLaxMode` or `SameSiteStrictMode` for CSRF protection
3. **HttpOnly**: Prevent JavaScript access to session cookies
4. **CSRF Tokens**: Regenerate after authentication to prevent session fixation
5. **Session Expiration**: Set appropriate `MaxAge` based on security requirements
6. **Cleanup**: Enable automatic cleanup to remove expired sessions

## Performance Targets

Based on IMPLEMENTATION-WEB.md specifications:

- **Session Read**:
  - Memory: <1ms
  - Redis: <5ms
  - Database: <10ms

- **Session Write**:
  - Memory: <2ms
  - Redis: <10ms
  - Database: <20ms

## Architecture

The session component follows a clean architecture:

1. **Store Interface**: Defines session storage operations
2. **Middleware**: Manages session lifecycle per request
3. **Context Integration**: Type-safe session access
4. **CSRF Middleware**: Optional security layer
5. **Flash Messages**: One-time message storage

## Future Enhancements

- [ ] Session fingerprinting (IP, User-Agent validation)
- [ ] Session activity tracking
- [ ] Configurable encryption for sensitive data
- [ ] Redis cluster support
- [ ] Session statistics and monitoring
