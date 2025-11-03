# API Prefix Configuration

This document describes how to configure API route prefixes in Conduit for versioned APIs.

## Overview

The API prefix feature allows you to add a path prefix to all your resource routes, following industry best practices for API versioning. This provides:

- **API Versioning**: Support for versioned routes like `/api/v1/users`
- **Clear Separation**: Distinguish API routes from static assets
- **Namespace Prevention**: Avoid collisions with other routes

## Configuration

Add the `api_prefix` field to your `conduit.yaml` file under the `server` section:

```yaml
server:
  port: 3000
  host: localhost
  api_prefix: "/api/v1"
```

### Default Behavior

If `api_prefix` is not specified or is empty, routes will have no prefix (backward compatible):

```yaml
server:
  port: 3000
  host: localhost
  api_prefix: ""  # or omit this line entirely
```

### Validation Rules

The API prefix must follow these rules:

1. **Must start with `/`** - Invalid: `"api/v1"`, Valid: `"/api/v1"`
2. **Must not end with `/`** - Invalid: `"/api/v1/"`, Valid: `"/api/v1"`
3. **Empty string is valid** - No prefix will be applied

Invalid configurations will fail during `conduit build` with a clear error message.

## Examples

### Example 1: Versioned API

```yaml
# conduit.yaml
server:
  api_prefix: "/api/v1"
```

**Generated routes:**
```
GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/users/:id
GET  /api/v1/posts
POST /api/v1/posts
```

### Example 2: Simple Versioning

```yaml
# conduit.yaml
server:
  api_prefix: "/v1"
```

**Generated routes:**
```
GET  /v1/users
POST /v1/users
GET  /v1/posts
POST /v1/posts
```

### Example 3: No Prefix (Default)

```yaml
# conduit.yaml
server:
  # No api_prefix configured
```

**Generated routes:**
```
GET  /users
POST /users
GET  /posts
POST /posts
```

## Health Check Endpoint

The `/health` endpoint is **always outside the API prefix** to ensure load balancers and monitoring tools can access it regardless of versioning:

```yaml
server:
  api_prefix: "/api/v1"
```

**Routes:**
```
GET /health              # Always accessible here
GET /api/v1/users        # Prefixed routes
POST /api/v1/users
```

## Build Output

When you run `conduit build`, the configured API prefix is shown in the output:

```bash
$ conduit build

✓ Build successful in 2.34s
  Binary: build/app
  API Prefix: /api/v1
```

## Introspection

The `conduit introspect` commands show the full route paths including the prefix:

### List Routes

```bash
$ conduit introspect routes
GET    /api/v1/users              -> handlers.UsersList
POST   /api/v1/users              -> handlers.UsersCreate
GET    /api/v1/users/:id          -> handlers.UsersShow
```

### View Resource Details

```bash
$ conduit introspect resource User

━━━ API ENDPOINTS ━━━━━━━━━━━━━━━━━━━━━━━━━

GET /api/v1/users → list
POST /api/v1/users → create
GET /api/v1/users/:id → show
```

### JSON Output

The JSON output also includes the API prefix:

```bash
$ conduit introspect routes --format json
{
  "total_count": 3,
  "api_prefix": "/api/v1",
  "routes": [
    {
      "method": "GET",
      "path": "/api/v1/users",
      "handler": "handlers.UsersList",
      "resource": "User",
      "operation": "list"
    }
  ]
}
```

## Migration Guide

### Migrating from No Prefix to Versioned Routes

1. **Update configuration:**
   ```yaml
   server:
     api_prefix: "/api/v1"
   ```

2. **Rebuild your application:**
   ```bash
   conduit build
   ```

3. **Update client applications:**
   - Change API calls from `/users` to `/api/v1/users`
   - Update any hardcoded paths

4. **Optional: Add redirect middleware** (if supporting both old and new routes temporarily)

### API Versioning Strategy

When releasing a new API version:

1. Create a new prefix in configuration (e.g., `/api/v2`)
2. Build and deploy the new version
3. Maintain backward compatibility by running multiple versions if needed
4. Deprecate old versions gradually

## Technical Details

### Implementation

- Routes are wrapped in `r.Route(prefix, func(r chi.Router) { ... })` when a prefix is configured
- The health check endpoint is registered before the prefixed routes
- All introspection commands respect the configured prefix
- Validation occurs at config load time, preventing invalid builds

### Generated Code

With `api_prefix: "/api/v1"`, the generated `main.go` includes:

```go
// Health check endpoint (outside API prefix)
r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})

// Register resource routes with API prefix: /api/v1
r.Route("/api/v1", func(r chi.Router) {
    handlers.RegisterUserRoutes(r, db)
    handlers.RegisterPostRoutes(r, db)
})
```

## Best Practices

1. **Use semantic versioning**: `/api/v1`, `/api/v2`, etc.
2. **Keep the prefix simple**: Avoid deeply nested prefixes
3. **Document your API versions**: Maintain changelog for each version
4. **Don't version everything**: Static assets and health checks stay unversioned
5. **Plan for deprecation**: Set clear timelines for old version support

## Troubleshooting

### Error: "api_prefix must start with '/'"

Your configuration is missing the leading slash:

```yaml
# ❌ Wrong
server:
  api_prefix: "api/v1"

# ✓ Correct
server:
  api_prefix: "/api/v1"
```

### Error: "api_prefix must not end with '/'"

Your configuration has a trailing slash:

```yaml
# ❌ Wrong
server:
  api_prefix: "/api/v1/"

# ✓ Correct
server:
  api_prefix: "/api/v1"
```

### Routes not showing prefix

Make sure you've rebuilt the application after changing the configuration:

```bash
conduit build
```

## Related Documentation

- [Configuration Reference](configuration.md)
- [Introspection Guide](introspection/user-guide.md)
- [Deployment Best Practices](deployment.md)
