# IMPLEMENTATION-WEB.md

**Component:** Web Framework Primitives
**Status:** Implementation Ready
**Last Updated:** 2025-10-13
**Estimated Effort:** 18-24 weeks (90-120 person-days)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Component 1: Router Implementation](#component-1-router-implementation)
4. [Component 2: Middleware Chain](#component-2-middleware-chain)
5. [Component 3: Request/Response Lifecycle](#component-3-requestresponse-lifecycle)
6. [Component 4: Authentication & Authorization](#component-4-authentication--authorization)
7. [Component 5: WebSocket Support](#component-5-websocket-support)
8. [Component 6: Session Storage](#component-6-session-storage)
9. [Component 7: Background Jobs](#component-7-background-jobs)
10. [Component 8: Caching Layer](#component-8-caching-layer)
11. [Component 9: Rate Limiting](#component-9-rate-limiting)
12. [Component 10: Performance Optimization](#component-10-performance-optimization)
13. [Development Phases](#development-phases)
14. [Testing Strategy](#testing-strategy)
15. [Integration Points](#integration-points)
16. [Performance Targets](#performance-targets)
17. [Risk Mitigation](#risk-mitigation)
18. [Success Criteria](#success-criteria)

---

## Overview

### Purpose

The Web Framework Primitives component provides the **HTTP layer** built on top of Go's standard library and the Resource System, delivering:

1. **Auto-generated REST APIs** from resource definitions
2. **Explicit middleware chains** with clear execution order
3. **Type-safe request/response handling** with validation
4. **Built-in security** (CSRF, rate limiting, authentication)
5. **WebSocket support** for real-time communication
6. **Background job processing** without external dependencies

### Design Philosophy

**Built on Go's Standard Library**
- Use `net/http` as foundation (battle-tested, production-ready)
- Go 1.22+ pattern matching for routing
- Minimal external dependencies
- Simple, auditable codebase

**Resource-Oriented Design**
- Routes automatically generated from Resource definitions
- RESTful conventions by default
- Explicit overrides when needed
- Introspection-friendly structure

**Explicitness over Magic**
- All routing, middleware, and request handling is declarative
- No hidden behavior or conventions
- Clear error messages
- Zero ambiguity for LLM code generation

### Key Innovations

1. **Auto-Generated Routes**: REST APIs generated automatically from resource definitions
2. **Declarative Middleware**: Middleware chains specified per operation with clear order
3. **Built-in Security**: CSRF, rate limiting, and auth primitives out-of-box
4. **Integrated Background Jobs**: Async task execution without external queue
5. **Zero-Config Development**: Sensible defaults, production-ready from the start

### Compilation Target

**Primary:** Go (net/http)
**Dependencies:**
- gorilla/websocket (WebSocket)
- go-redis (sessions, cache, rate limiting, jobs)
- golang-jwt/jwt (JWT authentication)

**Generated Code:** HTTP handlers + middleware + route registration

---

## Architecture

### High-Level Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                    HTTP Server (net/http)                         │
│                  Connection Pool & TLS Management                 │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Router Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Pattern     │  │  Parameter   │  │   Method     │          │
│  │  Matching    │  │  Extraction  │  │   Routing    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                   │
│  Resource Routes (Auto-generated):                               │
│  GET    /posts           → List                                  │
│  POST   /posts           → Create                                │
│  GET    /posts/{id}      → Read                                  │
│  PUT    /posts/{id}      → Update                                │
│  DELETE /posts/{id}      → Delete                                │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Middleware Chain                                │
│  Global → Resource → Operation → Handler                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   CORS       │  │     Auth     │  │  Rate Limit  │          │
│  │   Logging    │  │   Context    │  │     CSRF     │          │
│  │  Recovery    │  │  Timeout     │  │   Caching    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                Request/Response Lifecycle                         │
│  ┌──────────────────────────────────────────────────────┐       │
│  │  1. Parse Request (JSON/Form/Multipart)              │       │
│  │  2. Validate Input (Schema Validation)               │       │
│  │  3. Execute Handler (Resource CRUD / Custom)         │       │
│  │  4. Format Response (JSON with metadata)             │       │
│  │  5. Error Handling (Structured errors)               │       │
│  └──────────────────────────────────────────────────────┘       │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                ┌────────────┴────────────┐
                ▼                         ▼
┌───────────────────────────┐  ┌────────────────────────────┐
│  Resource CRUD Handlers   │  │   Custom Route Handlers    │
│  (Auto-generated)         │  │   (User-defined)           │
│                           │  │                            │
│  - List (GET /)           │  │  @route POST "/publish"   │
│  - Create (POST /)        │  │  @handler custom_logic     │
│  - Read (GET /:id)        │  │                            │
│  - Update (PUT /:id)      │  │                            │
│  - Delete (DELETE /:id)   │  │                            │
└───────────────────────────┘  └────────────────────────────┘
                │                         │
                └────────────┬────────────┘
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                  Special Features                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  WebSocket   │  │   Session    │  │  Background  │          │
│  │   Upgrade    │  │   Storage    │  │     Jobs     │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└──────────────────────────────────────────────────────────────────┘
```

### Layered Architecture

**Layer 1: HTTP Server**
- Go's net/http server
- Connection pooling
- TLS management
- Graceful shutdown

**Layer 2: Routing**
- Pattern matching (Go 1.22+ ServeMux)
- Auto-generated resource routes
- Custom route registration
- Parameter extraction

**Layer 3: Middleware**
- Chain builder
- Built-in middleware (CORS, auth, logging, etc.)
- Custom middleware support
- Context propagation

**Layer 4: Request/Response**
- Content negotiation
- Parsing (JSON, form, multipart)
- Validation
- Serialization
- Error formatting

**Layer 5: Handlers**
- Auto-generated CRUD handlers
- Custom handler execution
- Integration with ORM
- Background job dispatch

---

## Component 1: Router Implementation

### Responsibility

Pattern matching, parameter extraction, method routing, and auto-generated resource routes.

### Key Data Structures

```go
type Router struct {
    mux            *http.ServeMux
    routes         map[string]*Route
    middleware     []MiddlewareFunc

    // For introspection
    registeredRoutes []*RouteInfo
}

type Route struct {
    Pattern     string           // /posts/{id}
    Method      string           // GET, POST, etc.
    Handler     http.HandlerFunc
    Middleware  []MiddlewareFunc

    // Resource metadata
    Resource    *ResourceSchema  // If auto-generated
    Operation   CRUDOperation    // create, read, update, delete, list

    // Validation
    RequestSchema  *Schema
    ResponseSchema *Schema
}

type RouteInfo struct {
    Pattern        string
    Method         string
    ResourceName   string
    Operation      string
    Middleware     []string
    Parameters     []RouteParameter
    Documentation  string
}

type RouteParameter struct {
    Name     string
    Type     string
    Required bool
    Source   ParameterSource  // path, query, header, body
}

type ParameterSource int
const (
    PathParam ParameterSource = iota
    QueryParam
    HeaderParam
    BodyParam
)

type CRUDOperation int
const (
    OpList CRUDOperation = iota
    OpCreate
    OpRead
    OpUpdate
    OpDelete
)
```

### Auto-Generated Resource Routes

```go
func (r *Router) RegisterResource(schema *ResourceSchema) error {
    basePath := "/" + toPlural(toSnakeCase(schema.Name))

    // Standard RESTful routes
    routes := []struct {
        method    string
        pattern   string
        operation CRUDOperation
        handler   http.HandlerFunc
    }{
        {"GET",    basePath,           OpList,   r.generateListHandler(schema)},
        {"POST",   basePath,           OpCreate, r.generateCreateHandler(schema)},
        {"GET",    basePath + "/{id}", OpRead,   r.generateReadHandler(schema)},
        {"PUT",    basePath + "/{id}", OpUpdate, r.generateUpdateHandler(schema)},
        {"PATCH",  basePath + "/{id}", OpUpdate, r.generatePatchHandler(schema)},
        {"DELETE", basePath + "/{id}", OpDelete, r.generateDeleteHandler(schema)},
    }

    for _, route := range routes {
        // Apply resource-level middleware
        handler := r.applyMiddleware(
            route.handler,
            schema.Middleware[route.operation],
        )

        // Register with ServeMux (Go 1.22+ syntax)
        pattern := route.method + " " + route.pattern
        r.mux.HandleFunc(pattern, handler)

        // Store for introspection
        r.registeredRoutes = append(r.registeredRoutes, &RouteInfo{
            Pattern:      route.pattern,
            Method:       route.method,
            ResourceName: schema.Name,
            Operation:    route.operation.String(),
            Middleware:   getMiddlewareNames(schema.Middleware[route.operation]),
        })
    }

    return nil
}
```

### Handler Generation

```go
func (r *Router) generateListHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Parse pagination parameters
        page := getIntParam(req, "page", 1)
        perPage := getIntParam(req, "per_page", 50)
        offset := (page - 1) * perPage

        // Get CRUD operations from ORM
        crud := r.orm.GetCRUD(schema.Name)

        // Build query with pagination
        qb := crud.Where(map[string]interface{}{}).
            Limit(perPage).
            Offset(offset)

        // Execute query
        records, err := qb.Execute(ctx)
        if err != nil {
            writeError(w, http.StatusInternalServerError, err)
            return
        }

        // Count total
        total, err := qb.Count(ctx)
        if err != nil {
            writeError(w, http.StatusInternalServerError, err)
            return
        }

        // Return paginated response
        writeJSON(w, http.StatusOK, map[string]interface{}{
            "data": records,
            "meta": map[string]interface{}{
                "page":     page,
                "per_page": perPage,
                "total":    total,
            },
        })
    }
}

func (r *Router) generateCreateHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Parse request body
        var data map[string]interface{}
        if err := parseRequest(req, &data); err != nil {
            writeError(w, http.StatusBadRequest, err)
            return
        }

        // Get CRUD operations
        crud := r.orm.GetCRUD(schema.Name)

        // Create record (includes validation and hooks)
        record, err := crud.Create(ctx, data)
        if err != nil {
            if isValidationError(err) {
                writeError(w, http.StatusUnprocessableEntity, err)
            } else {
                writeError(w, http.StatusInternalServerError, err)
            }
            return
        }

        // Return created resource
        writeJSON(w, http.StatusCreated, record)
    }
}

func (r *Router) generateReadHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Extract ID from path
        id := req.PathValue("id")
        if id == "" {
            writeError(w, http.StatusBadRequest, fmt.Errorf("missing id"))
            return
        }

        // Get CRUD operations
        crud := r.orm.GetCRUD(schema.Name)

        // Find record
        record, err := crud.Find(ctx, id)
        if err != nil {
            if err == sql.ErrNoRows {
                writeError(w, http.StatusNotFound, fmt.Errorf("not found"))
            } else {
                writeError(w, http.StatusInternalServerError, err)
            }
            return
        }

        writeJSON(w, http.StatusOK, record)
    }
}

func (r *Router) generateUpdateHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Extract ID
        id := req.PathValue("id")
        if id == "" {
            writeError(w, http.StatusBadRequest, fmt.Errorf("missing id"))
            return
        }

        // Parse request body
        var data map[string]interface{}
        if err := parseRequest(req, &data); err != nil {
            writeError(w, http.StatusBadRequest, err)
            return
        }

        // Get CRUD operations
        crud := r.orm.GetCRUD(schema.Name)

        // Update record (includes validation and hooks)
        record, err := crud.Update(ctx, id, data)
        if err != nil {
            if isValidationError(err) {
                writeError(w, http.StatusUnprocessableEntity, err)
            } else if err == sql.ErrNoRows {
                writeError(w, http.StatusNotFound, fmt.Errorf("not found"))
            } else {
                writeError(w, http.StatusInternalServerError, err)
            }
            return
        }

        writeJSON(w, http.StatusOK, record)
    }
}

func (r *Router) generateDeleteHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Extract ID
        id := req.PathValue("id")
        if id == "" {
            writeError(w, http.StatusBadRequest, fmt.Errorf("missing id"))
            return
        }

        // Get CRUD operations
        crud := r.orm.GetCRUD(schema.Name)

        // Delete record (includes hooks)
        if err := crud.Delete(ctx, id); err != nil {
            if err == sql.ErrNoRows {
                writeError(w, http.StatusNotFound, fmt.Errorf("not found"))
            } else {
                writeError(w, http.StatusInternalServerError, err)
            }
            return
        }

        w.WriteHeader(http.StatusNoContent)
    }
}
```

### Nested Resource Routes

```go
// @nested under Post as "comments"
// Generates:
// GET    /posts/{post_id}/comments
// POST   /posts/{post_id}/comments
// GET    /posts/{post_id}/comments/{id}
// PUT    /posts/{post_id}/comments/{id}
// DELETE /posts/{post_id}/comments/{id}

func (r *Router) RegisterNestedResource(
    parent *ResourceSchema,
    child *ResourceSchema,
    nestConfig *NestedConfig,
) error {
    parentPath := "/" + toPlural(toSnakeCase(parent.Name))
    childPath := parentPath + "/{" + toSnakeCase(parent.Name) + "_id}/" + nestConfig.CollectionName

    // Handler automatically scopes queries by parent ID
    r.generateNestedListHandler(parent, child, childPath)
    r.generateNestedCreateHandler(parent, child, childPath)
    // ... other handlers

    return nil
}

func (r *Router) generateNestedListHandler(
    parent, child *ResourceSchema,
    basePath string,
) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()

        // Extract parent ID
        parentIDKey := toSnakeCase(parent.Name) + "_id"
        parentID := req.PathValue(parentIDKey)
        if parentID == "" {
            writeError(w, http.StatusBadRequest, fmt.Errorf("missing %s", parentIDKey))
            return
        }

        // Get CRUD operations
        crud := r.orm.GetCRUD(child.Name)

        // Query scoped by parent ID
        records, err := crud.Where(map[string]interface{}{
            parentIDKey: parentID,
        }).Execute(ctx)
        if err != nil {
            writeError(w, http.StatusInternalServerError, err)
            return
        }

        writeJSON(w, http.StatusOK, map[string]interface{}{
            "data": records,
        })
    }
}
```

### Custom Route Registration

```go
// Allow custom routes alongside auto-generated ones
func (r *Router) RegisterCustomRoute(
    method string,
    pattern string,
    handler http.HandlerFunc,
    middleware ...MiddlewareFunc,
) {
    fullPattern := method + " " + pattern
    wrappedHandler := r.applyMiddleware(handler, middleware)
    r.mux.HandleFunc(fullPattern, wrappedHandler)

    // Store for introspection
    r.registeredRoutes = append(r.registeredRoutes, &RouteInfo{
        Pattern: pattern,
        Method:  method,
    })
}

// Usage from resource:
// @route POST "/posts/:id/publish" -> publish_post
// Generates:
// r.RegisterCustomRoute("POST", "/posts/{id}/publish", publishPostHandler, authMiddleware, adminMiddleware)
```

### Testing Strategy

**Unit Tests:**
- Test route registration
- Test pattern matching
- Test parameter extraction
- Test handler generation

**Integration Tests:**
- Test auto-generated routes with ORM
- Test nested resources
- Test custom routes
- Test introspection

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Low

---

## Component 2: Middleware Chain

### Responsibility

Execute middleware in order, propagate context, handle errors.

### Key Data Structures

```go
type MiddlewareFunc func(http.Handler) http.Handler

// Standard signature matches net/http conventions
// Enables compatibility with existing ecosystem

type MiddlewareChain struct {
    middlewares []MiddlewareFunc
}
```

### Middleware Chain Builder

```go
func NewChain(middlewares ...MiddlewareFunc) *MiddlewareChain {
    return &MiddlewareChain{middlewares: middlewares}
}

func (c *MiddlewareChain) Then(h http.Handler) http.Handler {
    // Apply middleware in reverse order so they execute in declaration order
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        h = c.middlewares[i](h)
    }
    return h
}

func (c *MiddlewareChain) ThenFunc(fn http.HandlerFunc) http.Handler {
    return c.Then(fn)
}

// Usage:
// chain := NewChain(LoggingMiddleware, AuthMiddleware, RateLimitMiddleware)
// handler := chain.ThenFunc(myHandler)
```

### Built-in Middleware

#### Recovery Middleware

```go
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Log stack trace
                stack := debug.Stack()
                log.Printf("PANIC: %v\n%s", err, stack)

                // Return 500 error
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": "Internal server error",
                })
            }
        }()

        next.ServeHTTP(w, r)
    })
}
```

#### Logging Middleware

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        // Log after request completes
        duration := time.Since(start)
        log.Printf(
            "%s %s %d %v",
            r.Method,
            r.URL.Path,
            wrapped.statusCode,
            duration,
        )
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

#### Timeout Middleware

```go
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), timeout)
            defer cancel()

            r = r.WithContext(ctx)

            done := make(chan struct{})
            go func() {
                defer close(done)
                next.ServeHTTP(w, r)
            }()

            select {
            case <-done:
                return
            case <-ctx.Done():
                w.WriteHeader(http.StatusGatewayTimeout)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": "Request timeout",
                })
            }
        })
    }
}
```

#### CORS Middleware

```go
type CORSConfig struct {
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
}

func CORSMiddleware(config CORSConfig) MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Check if origin is allowed
            if isOriginAllowed(origin, config.AllowedOrigins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
            }

            if config.AllowCredentials {
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }

            if len(config.ExposedHeaders) > 0 {
                w.Header().Set("Access-Control-Expose-Headers",
                    strings.Join(config.ExposedHeaders, ", "))
            }

            // Handle preflight
            if r.Method == "OPTIONS" {
                w.Header().Set("Access-Control-Allow-Methods",
                    strings.Join(config.AllowedMethods, ", "))
                w.Header().Set("Access-Control-Allow-Headers",
                    strings.Join(config.AllowedHeaders, ", "))
                w.Header().Set("Access-Control-Max-Age",
                    strconv.Itoa(config.MaxAge))
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

#### Request ID Middleware

```go
type contextKey string

const (
    RequestIDKey  contextKey = "request_id"
    CurrentUserKey contextKey = "current_user"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        // Add to context
        ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
        r = r.WithContext(ctx)

        // Add to response headers
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r)
    })
}
```

### Middleware Execution Order

```
Request Flow:
  1. RecoveryMiddleware (catch panics)
  2. LoggingMiddleware (log requests)
  3. RequestIDMiddleware (track requests)
  4. CORSMiddleware (handle CORS)
  5. TimeoutMiddleware (enforce timeouts)
  6. AuthMiddleware (authenticate user)
  7. AuthorizationMiddleware (check permissions)
  8. RateLimitMiddleware (rate limiting)
  9. CacheMiddleware (check cache)
  10. Handler (business logic)
  11. [Response flows back through middleware]
```

### Testing Strategy

**Unit Tests:**
- Test each middleware individually
- Test middleware composition
- Test error handling

**Integration Tests:**
- Test middleware chains
- Test execution order
- Test context propagation

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Component 3: Request/Response Lifecycle

### Responsibility

Parse requests, validate input, format responses, handle errors.

### Request Parsing

```go
type RequestParser struct {
    maxBodySize int64
}

func (p *RequestParser) Parse(r *http.Request, target interface{}) error {
    contentType := r.Header.Get("Content-Type")

    switch {
    case strings.HasPrefix(contentType, "application/json"):
        return p.parseJSON(r, target)
    case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
        return p.parseForm(r, target)
    case strings.HasPrefix(contentType, "multipart/form-data"):
        return p.parseMultipart(r, target)
    default:
        return fmt.Errorf("unsupported content type: %s", contentType)
    }
}

func (p *RequestParser) parseJSON(r *http.Request, target interface{}) error {
    // Limit body size to prevent DoS
    r.Body = http.MaxBytesReader(nil, r.Body, p.maxBodySize)

    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()  // Strict parsing

    if err := decoder.Decode(target); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }

    return nil
}

func (p *RequestParser) parseForm(r *http.Request, target interface{}) error {
    if err := r.ParseForm(); err != nil {
        return fmt.Errorf("invalid form data: %w", err)
    }

    // Convert form values to struct/map
    return formToStruct(r.Form, target)
}

func (p *RequestParser) parseMultipart(r *http.Request, target interface{}) error {
    if err := r.ParseMultipartForm(p.maxBodySize); err != nil {
        return fmt.Errorf("invalid multipart form: %w", err)
    }

    // Convert form and files to struct/map
    return multipartToStruct(r.MultipartForm, target)
}
```

### Input Validation

```go
type RequestValidator struct {
    schemas map[string]*ValidationSchema
}

type ValidationSchema struct {
    Fields map[string]*FieldValidation
}

type FieldValidation struct {
    Type     string
    Required bool
    Min      *int
    Max      *int
    Pattern  *regexp.Regexp
}

func (v *RequestValidator) Validate(
    resourceName string,
    operation CRUDOperation,
    data map[string]interface{},
) error {
    schema := v.schemas[resourceName]
    if schema == nil {
        return nil  // No schema, skip validation
    }

    var errors []error

    for fieldName, validation := range schema.Fields {
        value, exists := data[fieldName]

        // Required check
        if validation.Required && !exists {
            errors = append(errors,
                fmt.Errorf("%s is required", fieldName))
            continue
        }

        if !exists {
            continue
        }

        // Type check
        if err := v.validateType(value, validation.Type); err != nil {
            errors = append(errors,
                fmt.Errorf("%s: %w", fieldName, err))
        }

        // Min/Max check
        if validation.Min != nil || validation.Max != nil {
            if err := v.validateRange(value, validation.Min, validation.Max); err != nil {
                errors = append(errors,
                    fmt.Errorf("%s: %w", fieldName, err))
            }
        }

        // Pattern check
        if validation.Pattern != nil {
            if err := v.validatePattern(value, validation.Pattern); err != nil {
                errors = append(errors,
                    fmt.Errorf("%s: %w", fieldName, err))
            }
        }
    }

    if len(errors) > 0 {
        return &ValidationError{Errors: errors}
    }

    return nil
}
```

### Response Formatting

```go
type ResponseFormatter struct {
    prettyPrint bool
}

type APIResponse struct {
    Data   interface{}            `json:"data,omitempty"`
    Error  *APIError              `json:"error,omitempty"`
    Meta   map[string]interface{} `json:"meta,omitempty"`
}

type APIError struct {
    Code    string   `json:"code"`
    Message string   `json:"message"`
    Details []string `json:"details,omitempty"`
}

func (f *ResponseFormatter) WriteJSON(
    w http.ResponseWriter,
    statusCode int,
    data interface{},
) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    encoder := json.NewEncoder(w)
    if f.prettyPrint {
        encoder.SetIndent("", "  ")
    }

    return encoder.Encode(APIResponse{Data: data})
}

func (f *ResponseFormatter) WriteError(
    w http.ResponseWriter,
    statusCode int,
    err error,
) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    apiError := &APIError{
        Code:    errorCodeFromStatus(statusCode),
        Message: err.Error(),
    }

    // Add validation details if applicable
    if validationErr, ok := err.(*ValidationError); ok {
        for _, e := range validationErr.Errors {
            apiError.Details = append(apiError.Details, e.Error())
        }
    }

    encoder := json.NewEncoder(w)
    if f.prettyPrint {
        encoder.SetIndent("", "  ")
    }

    return encoder.Encode(APIResponse{Error: apiError})
}

func errorCodeFromStatus(status int) string {
    switch status {
    case http.StatusBadRequest:
        return "bad_request"
    case http.StatusUnauthorized:
        return "unauthorized"
    case http.StatusForbidden:
        return "forbidden"
    case http.StatusNotFound:
        return "not_found"
    case http.StatusUnprocessableEntity:
        return "validation_failed"
    case http.StatusInternalServerError:
        return "internal_error"
    default:
        return "error"
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test each parser
- Test validation rules
- Test response formatting
- Test error handling

**Integration Tests:**
- Test full request/response cycle
- Test content negotiation
- Test error responses

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Component 4: Authentication & Authorization

### Responsibility

User authentication (JWT/session), permission checking, resource-level authorization.

### Authentication Strategies

#### JWT-based Authentication

```go
type JWTAuthConfig struct {
    SecretKey      []byte
    SigningMethod  jwt.SigningMethod
    TokenDuration  time.Duration
    RefreshEnabled bool
}

type JWTAuthenticator struct {
    config *JWTAuthConfig
}

func (a *JWTAuthenticator) GenerateToken(userID string, claims map[string]interface{}) (string, error) {
    token := jwt.NewWithClaims(a.config.SigningMethod, jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(a.config.TokenDuration).Unix(),
        "iat": time.Now().Unix(),
    })

    // Add custom claims
    for key, value := range claims {
        token.Claims.(jwt.MapClaims)[key] = value
    }

    return token.SignedString(a.config.SecretKey)
}

func (a *JWTAuthenticator) ValidateToken(tokenString string) (*jwt.Token, error) {
    return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Validate signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return a.config.SecretKey, nil
    })
}

func (a *JWTAuthenticator) Middleware() MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token from Authorization header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", http.StatusUnauthorized)
                return
            }

            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
                return
            }

            // Validate token
            token, err := a.ValidateToken(parts[1])
            if err != nil || !token.Valid {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            // Extract user ID from claims
            claims := token.Claims.(jwt.MapClaims)
            userID := claims["sub"].(string)

            // Add to context
            ctx := context.WithValue(r.Context(), CurrentUserKey, userID)
            r = r.WithContext(ctx)

            next.ServeHTTP(w, r)
        })
    }
}
```

#### Session-based Authentication

```go
type SessionAuthenticator struct {
    store SessionStore
}

type SessionStore interface {
    Get(ctx context.Context, sessionID string) (*Session, error)
    Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error
    Delete(ctx context.Context, sessionID string) error
}

type Session struct {
    ID        string
    UserID    string
    Data      map[string]interface{}
    ExpiresAt time.Time
}

func (a *SessionAuthenticator) Middleware() MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get session ID from cookie
            cookie, err := r.Cookie("session_id")
            if err != nil {
                http.Error(w, "Missing session", http.StatusUnauthorized)
                return
            }

            // Retrieve session
            session, err := a.store.Get(r.Context(), cookie.Value)
            if err != nil || session.ExpiresAt.Before(time.Now()) {
                http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
                return
            }

            // Add to context
            ctx := context.WithValue(r.Context(), CurrentUserKey, session.UserID)
            r = r.WithContext(ctx)

            next.ServeHTTP(w, r)
        })
    }
}
```

### Authorization

#### Permission-based Authorization

```go
type Permission string

const (
    PermissionCreate Permission = "create"
    PermissionRead   Permission = "read"
    PermissionUpdate Permission = "update"
    PermissionDelete Permission = "delete"
)

type Authorizer struct {
    permissionChecker PermissionChecker
}

type PermissionChecker interface {
    HasPermission(ctx context.Context, userID string, resource string, permission Permission) (bool, error)
}

func (a *Authorizer) Middleware(resource string, permission Permission) MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user from context
            userID, ok := r.Context().Value(CurrentUserKey).(string)
            if !ok {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            // Check permission
            hasPermission, err := a.permissionChecker.HasPermission(
                r.Context(),
                userID,
                resource,
                permission,
            )
            if err != nil {
                http.Error(w, "Error checking permissions", http.StatusInternalServerError)
                return
            }

            if !hasPermission {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

#### Resource-level Authorization

```go
// Integrate with Resource System annotations:
// @on create: [auth, can_create_post]
// @on update: [auth, is_author_or_editor]
// @on delete: [auth, is_author_or_admin]

// Middleware runs before handlers, ensuring authorization checks
// are enforced automatically for all resource operations
```

### Testing Strategy

**Unit Tests:**
- Test JWT generation/validation
- Test session storage/retrieval
- Test permission checks
- Test token expiry

**Security Tests:**
- Test token tampering
- Test session hijacking
- Test permission bypass
- Test expired tokens

**Coverage Target:** >95% (security critical)

### Estimated Effort

**Time:** 3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium-High
**Risk:** High (security critical)

---

## Component 5: WebSocket Support

### Responsibility

Upgrade HTTP connections to WebSocket, manage connections, broadcast messages.

### WebSocket Upgrader

```go
import "github.com/gorilla/websocket"

type WebSocketConfig struct {
    ReadBufferSize  int
    WriteBufferSize int
    CheckOrigin     func(r *http.Request) bool
}

type WebSocketManager struct {
    upgrader    *websocket.Upgrader
    connections sync.Map  // map[string]*websocket.Conn
    handlers    map[string]MessageHandler
}

type MessageHandler func(ctx context.Context, conn *websocket.Conn, message []byte) error

func NewWebSocketManager(config WebSocketConfig) *WebSocketManager {
    upgrader := &websocket.Upgrader{
        ReadBufferSize:  config.ReadBufferSize,
        WriteBufferSize: config.WriteBufferSize,
        CheckOrigin:     config.CheckOrigin,
    }

    return &WebSocketManager{
        upgrader: upgrader,
        handlers: make(map[string]MessageHandler),
    }
}

func (m *WebSocketManager) RegisterHandler(messageType string, handler MessageHandler) {
    m.handlers[messageType] = handler
}

func (m *WebSocketManager) UpgradeHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := m.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }

    // Generate connection ID
    connID := uuid.New().String()
    m.connections.Store(connID, conn)

    // Handle connection in goroutine
    go m.handleConnection(r.Context(), connID, conn)
}

func (m *WebSocketManager) handleConnection(ctx context.Context, connID string, conn *websocket.Conn) {
    defer func() {
        conn.Close()
        m.connections.Delete(connID)
    }()

    // Set deadlines
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    // Ping ticker
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // Message channel
    messages := make(chan []byte)
    errors := make(chan error)

    // Read messages in goroutine
    go func() {
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                errors <- err
                return
            }
            messages <- message
        }
    }()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        case message := <-messages:
            if err := m.processMessage(ctx, conn, message); err != nil {
                log.Printf("Message processing error: %v", err)
            }
        case err := <-errors:
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            return
        }
    }
}

func (m *WebSocketManager) processMessage(ctx context.Context, conn *websocket.Conn, message []byte) error {
    // Parse message type from payload
    var msg struct {
        Type string          `json:"type"`
        Data json.RawMessage `json:"data"`
    }

    if err := json.Unmarshal(message, &msg); err != nil {
        return err
    }

    // Find handler
    handler, ok := m.handlers[msg.Type]
    if !ok {
        return fmt.Errorf("unknown message type: %s", msg.Type)
    }

    // Execute handler
    return handler(ctx, conn, msg.Data)
}

func (m *WebSocketManager) Broadcast(message []byte) {
    m.connections.Range(func(key, value interface{}) bool {
        conn := value.(*websocket.Conn)
        if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
            log.Printf("Broadcast error: %v", err)
        }
        return true
    })
}

func (m *WebSocketManager) SendToConnection(connID string, message []byte) error {
    value, ok := m.connections.Load(connID)
    if !ok {
        return fmt.Errorf("connection not found: %s", connID)
    }

    conn := value.(*websocket.Conn)
    return conn.WriteMessage(websocket.TextMessage, message)
}
```

### WebSocket Route Registration

```go
// Register WebSocket endpoint
router.RegisterCustomRoute(
    "GET",
    "/ws",
    wsManager.UpgradeHandler,
    RequestIDMiddleware,
    // Auth middleware if needed
)
```

### Testing Strategy

**Unit Tests:**
- Test connection upgrade
- Test message handling
- Test connection cleanup

**Integration Tests:**
- Test multiple concurrent connections
- Test message broadcast
- Test connection lifecycle

**Load Tests:**
- Test 10K+ concurrent connections
- Test memory usage
- Test no connection leaks

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2 weeks
**Team:** 1 engineer
**Complexity:** Medium
**Risk:** Medium (connection management, memory leaks)

---

## Component 6: Session Storage

### Responsibility

Persistent session storage with Redis/memory backend, session management.

### Session Store Interface

```go
type SessionStore interface {
    Get(ctx context.Context, sessionID string) (*Session, error)
    Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error
    Delete(ctx context.Context, sessionID string) error
    Refresh(ctx context.Context, sessionID string, ttl time.Duration) error
}

type Session struct {
    ID        string
    UserID    string
    Data      map[string]interface{}
    ExpiresAt time.Time
}
```

### Redis Session Store

```go
import "github.com/redis/go-redis/v9"

type RedisSessionStore struct {
    client *redis.Client
}

func NewRedisSessionStore(addr string, password string, db int) *RedisSessionStore {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,

        // Connection pooling
        PoolSize:     100,
        MinIdleConns: 10,
        MaxIdleConns: 20,

        // Timeouts
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })

    return &RedisSessionStore{client: client}
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
    key := "session:" + sessionID

    data, err := s.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, fmt.Errorf("session not found")
    }
    if err != nil {
        return nil, err
    }

    var session Session
    if err := json.Unmarshal([]byte(data), &session); err != nil {
        return nil, err
    }

    return &session, nil
}

func (s *RedisSessionStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
    key := "session:" + sessionID

    data, err := json.Marshal(session)
    if err != nil {
        return err
    }

    return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
    key := "session:" + sessionID
    return s.client.Del(ctx, key).Err()
}

func (s *RedisSessionStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
    key := "session:" + sessionID
    return s.client.Expire(ctx, key, ttl).Err()
}
```

### Memory Session Store (Development)

```go
type MemorySessionStore struct {
    sessions sync.Map
}

type sessionEntry struct {
    session   *Session
    expiresAt time.Time
}

func NewMemorySessionStore() *MemorySessionStore {
    store := &MemorySessionStore{}

    // Start cleanup goroutine
    go store.cleanup()

    return store
}

func (s *MemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
    value, ok := s.sessions.Load(sessionID)
    if !ok {
        return nil, fmt.Errorf("session not found")
    }

    entry := value.(*sessionEntry)
    if entry.expiresAt.Before(time.Now()) {
        s.sessions.Delete(sessionID)
        return nil, fmt.Errorf("session expired")
    }

    return entry.session, nil
}

func (s *MemorySessionStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
    entry := &sessionEntry{
        session:   session,
        expiresAt: time.Now().Add(ttl),
    }
    s.sessions.Store(sessionID, entry)
    return nil
}

func (s *MemorySessionStore) Delete(ctx context.Context, sessionID string) error {
    s.sessions.Delete(sessionID)
    return nil
}

func (s *MemorySessionStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
    value, ok := s.sessions.Load(sessionID)
    if !ok {
        return fmt.Errorf("session not found")
    }

    entry := value.(*sessionEntry)
    entry.expiresAt = time.Now().Add(ttl)
    return nil
}

func (s *MemorySessionStore) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        now := time.Now()
        s.sessions.Range(func(key, value interface{}) bool {
            entry := value.(*sessionEntry)
            if entry.expiresAt.Before(now) {
                s.sessions.Delete(key)
            }
            return true
        })
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test session CRUD operations
- Test expiration
- Test cleanup

**Integration Tests:**
- Test Redis session store
- Test memory session store
- Test session refresh

**Coverage Target:** >90%

### Estimated Effort

**Time:** 1-2 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Medium (data persistence)

---

## Component 7: Background Jobs

### Responsibility

Asynchronous task execution, job scheduling, retry logic.

### Job Queue Interface

```go
type JobQueue interface {
    Enqueue(ctx context.Context, job *Job) error
    Dequeue(ctx context.Context) (*Job, error)
    Schedule(ctx context.Context, job *Job, runAt time.Time) error
}

type Job struct {
    ID         string
    Type       string
    Payload    map[string]interface{}
    MaxRetries int
    Retries    int
    CreatedAt  time.Time
    RunAt      time.Time
}
```

### Redis Job Queue

```go
type RedisJobQueue struct {
    client *redis.Client
}

func NewRedisJobQueue(client *redis.Client) *RedisJobQueue {
    return &RedisJobQueue{client: client}
}

func (q *RedisJobQueue) Enqueue(ctx context.Context, job *Job) error {
    data, err := json.Marshal(job)
    if err != nil {
        return err
    }

    // Add to queue (FIFO list)
    return q.client.RPush(ctx, "jobs:queue", data).Err()
}

func (q *RedisJobQueue) Dequeue(ctx context.Context) (*Job, error) {
    // BLPOP with 1 second timeout
    result, err := q.client.BLPop(ctx, 1*time.Second, "jobs:queue").Result()
    if err == redis.Nil {
        return nil, fmt.Errorf("no jobs available")
    }
    if err != nil {
        return nil, err
    }

    var job Job
    if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
        return nil, err
    }

    return &job, nil
}

func (q *RedisJobQueue) Schedule(ctx context.Context, job *Job, runAt time.Time) error {
    data, err := json.Marshal(job)
    if err != nil {
        return err
    }

    // Add to sorted set with score = unix timestamp
    score := float64(runAt.Unix())
    return q.client.ZAdd(ctx, "jobs:scheduled", redis.Z{
        Score:  score,
        Member: data,
    }).Err()
}
```

### Worker Pool

```go
type WorkerPool struct {
    queue      JobQueue
    handlers   map[string]JobHandler
    numWorkers int
    stopChan   chan struct{}
    wg         sync.WaitGroup
}

type JobHandler func(ctx context.Context, payload map[string]interface{}) error

func NewWorkerPool(queue JobQueue, numWorkers int) *WorkerPool {
    return &WorkerPool{
        queue:      queue,
        handlers:   make(map[string]JobHandler),
        numWorkers: numWorkers,
        stopChan:   make(chan struct{}),
    }
}

func (p *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
    p.handlers[jobType] = handler
}

func (p *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.numWorkers; i++ {
        p.wg.Add(1)
        go p.worker(ctx, i)
    }
}

func (p *WorkerPool) Stop() {
    close(p.stopChan)
    p.wg.Wait()
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
    defer p.wg.Done()

    log.Printf("Worker %d started", id)

    for {
        select {
        case <-p.stopChan:
            log.Printf("Worker %d stopped", id)
            return
        default:
            job, err := p.queue.Dequeue(ctx)
            if err != nil {
                time.Sleep(100 * time.Millisecond)
                continue
            }

            if err := p.processJob(ctx, job); err != nil {
                p.handleJobError(ctx, job, err)
            }
        }
    }
}

func (p *WorkerPool) processJob(ctx context.Context, job *Job) error {
    handler, ok := p.handlers[job.Type]
    if !ok {
        return fmt.Errorf("unknown job type: %s", job.Type)
    }

    log.Printf("Processing job %s (type: %s)", job.ID, job.Type)

    return handler(ctx, job.Payload)
}

func (p *WorkerPool) handleJobError(ctx context.Context, job *Job, err error) {
    log.Printf("Job %s failed: %v (retries: %d/%d)", job.ID, err, job.Retries, job.MaxRetries)

    if job.Retries < job.MaxRetries {
        job.Retries++
        // Exponential backoff
        backoff := time.Duration(job.Retries*job.Retries) * time.Minute
        job.RunAt = time.Now().Add(backoff)

        if err := p.queue.Schedule(ctx, job, job.RunAt); err != nil {
            log.Printf("Failed to reschedule job: %v", err)
        }
    } else {
        log.Printf("Job %s exceeded max retries, giving up", job.ID)
    }
}
```

### Integration with @async Blocks

```go
// From lifecycle hooks:
// @after create @transaction {
//   @async {
//     Email.send(user, "welcome")
//   }
// }

// Interpreter generates:
jobQueue.Enqueue(ctx, &Job{
    ID:         uuid.New().String(),
    Type:       "email.send",
    Payload: map[string]interface{}{
        "template": "welcome",
        "user_id":  user.ID,
    },
    MaxRetries: 3,
    CreatedAt:  time.Now(),
    RunAt:      time.Now(),
})
```

### Testing Strategy

**Unit Tests:**
- Test job enqueueing
- Test job dequeueing
- Test handler execution
- Test retry logic

**Integration Tests:**
- Test worker pool
- Test multiple workers
- Test job completion
- Test error handling

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Medium (job reliability)

---

## Component 8: Caching Layer

### Responsibility

HTTP response caching, cache invalidation.

### Cache Interface

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) error
}
```

### Redis Cache Implementation

```go
type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
    return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
    return c.client.Get(ctx, key).Bytes()
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
    return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
    iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
            return err
        }
    }
    return iter.Err()
}
```

### Cache Middleware

```go
func CacheMiddleware(cache Cache, ttl time.Duration) MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only cache GET requests
            if r.Method != "GET" {
                next.ServeHTTP(w, r)
                return
            }

            // Generate cache key
            cacheKey := "cache:" + r.URL.Path + "?" + r.URL.RawQuery

            // Try to get from cache
            cached, err := cache.Get(r.Context(), cacheKey)
            if err == nil {
                w.Header().Set("Content-Type", "application/json")
                w.Header().Set("X-Cache", "HIT")
                w.Write(cached)
                return
            }

            // Cache miss - capture response
            recorder := &responseCapturer{
                ResponseWriter: w,
                body:          new(bytes.Buffer),
                statusCode:    http.StatusOK,
            }

            next.ServeHTTP(recorder, r)

            // Cache successful responses
            if recorder.statusCode >= 200 && recorder.statusCode < 300 {
                cache.Set(r.Context(), cacheKey, recorder.body.Bytes(), ttl)
            }

            w.Header().Set("X-Cache", "MISS")
        })
    }
}

type responseCapturer struct {
    http.ResponseWriter
    body       *bytes.Buffer
    statusCode int
}

func (r *responseCapturer) Write(b []byte) (int, error) {
    r.body.Write(b)
    return r.ResponseWriter.Write(b)
}

func (r *responseCapturer) WriteHeader(statusCode int) {
    r.statusCode = statusCode
    r.ResponseWriter.WriteHeader(statusCode)
}
```

### Cache Invalidation

```go
// Integrate with ORM lifecycle hooks
// Automatically invalidate cache when resource updated

func (r *Router) generateUpdateHandler(schema *ResourceSchema) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        // ... update logic ...

        // Invalidate cache for this resource
        cachePattern := fmt.Sprintf("cache:/%s/*", toPlural(toSnakeCase(schema.Name)))
        if err := cache.DeletePattern(req.Context(), cachePattern); err != nil {
            log.Printf("Cache invalidation failed: %v", err)
        }

        // ... rest of handler ...
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test cache get/set
- Test cache deletion
- Test pattern deletion

**Integration Tests:**
- Test cache hit/miss
- Test invalidation
- Test TTL

**Performance Tests:**
- Measure cache speedup
- Test cache efficiency

**Coverage Target:** >90%

### Estimated Effort

**Time:** 1-2 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Component 9: Rate Limiting

### Responsibility

Request rate limiting per user/IP, distributed rate limiting.

### Rate Limiter Interface

```go
type RateLimiter interface {
    Allow(ctx context.Context, key string) (bool, error)
}
```

### Token Bucket Implementation

```go
type TokenBucketLimiter struct {
    redis  *redis.Client
    rate   int           // Requests per window
    window time.Duration
}

func NewTokenBucketLimiter(redis *redis.Client, rate int, window time.Duration) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        redis:  redis,
        rate:   rate,
        window: window,
    }
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
    now := time.Now().Unix()
    windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now/int64(l.window.Seconds()))

    // Increment counter
    count, err := l.redis.Incr(ctx, windowKey).Result()
    if err != nil {
        return false, err
    }

    // Set expiry on first request
    if count == 1 {
        l.redis.Expire(ctx, windowKey, l.window)
    }

    return count <= int64(l.rate), nil
}
```

### Rate Limit Middleware

```go
func RateLimitMiddleware(limiter RateLimiter) MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Use user ID if authenticated, otherwise IP
            key := r.RemoteAddr
            if userID, ok := r.Context().Value(CurrentUserKey).(string); ok {
                key = userID
            }

            allowed, err := limiter.Allow(r.Context(), key)
            if err != nil {
                http.Error(w, "Rate limit check failed", http.StatusInternalServerError)
                return
            }

            if !allowed {
                w.Header().Set("Retry-After", "60")
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test rate limit enforcement
- Test window sliding
- Test counter reset

**Integration Tests:**
- Test distributed rate limiting
- Test per-user limits
- Test per-IP limits

**Coverage Target:** >90%

### Estimated Effort

**Time:** 1 week
**Team:** 1 engineer
**Complexity:** Low
**Risk:** Low

---

## Component 10: Performance Optimization

### Responsibility

HTTP server tuning, connection pooling, zero-copy optimizations.

### HTTP Server Configuration

```go
type ServerConfig struct {
    Addr           string
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    IdleTimeout    time.Duration
    MaxHeaderBytes int
}

func NewHTTPServer(config ServerConfig, handler http.Handler) *http.Server {
    return &http.Server{
        Addr:           config.Addr,
        Handler:        handler,
        ReadTimeout:    config.ReadTimeout,
        WriteTimeout:   config.WriteTimeout,
        IdleTimeout:    config.IdleTimeout,
        MaxHeaderBytes: config.MaxHeaderBytes,

        // Enable HTTP/2
        // (automatically enabled if TLS configured)
    }
}

// Recommended production settings:
// ReadTimeout:    30 * time.Second
// WriteTimeout:   30 * time.Second
// IdleTimeout:    120 * time.Second
// MaxHeaderBytes: 1 << 20 (1MB)
```

### Connection Pool Configuration

#### Database Connection Pooling

```go
// From ORM component - ensure proper configuration
db.SetMaxOpenConns(100)                    // Max connections
db.SetMaxIdleConns(25)                     // Idle connections
db.SetConnMaxLifetime(5 * time.Minute)     // Connection lifetime
db.SetConnMaxIdleTime(1 * time.Minute)     // Idle timeout
```

#### Redis Connection Pooling

```go
redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,              // 10 * GOMAXPROCS
    MinIdleConns: 10,
    MaxIdleConns: 20,

    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

### Zero-Copy Optimizations

```go
// Use io.Copy for large responses
func streamFile(w http.ResponseWriter, file io.Reader) error {
    _, err := io.Copy(w, file)
    return err
}

// Reuse buffers with sync.Pool
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func getBuffer() *bytes.Buffer {
    return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
    buf.Reset()
    bufferPool.Put(buf)
}
```

### Graceful Shutdown

```go
func StartServerWithGracefulShutdown(srv *http.Server) error {
    // Start server
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown with 30s timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        return fmt.Errorf("server forced to shutdown: %w", err)
    }

    log.Println("Server stopped gracefully")
    return nil
}
```

### Testing Strategy

**Performance Tests:**
- Load testing with wrk/hey
- Measure throughput (req/s)
- Measure latency (p50, p95, p99)
- Measure memory usage
- Measure CPU usage

**Profiling:**
- CPU profiling (pprof)
- Memory profiling
- Goroutine profiling
- Mutex profiling

**Coverage Target:** Performance benchmarks meet targets

### Performance Targets

- **Throughput:** 30,000+ req/s (simple GET)
- **Latency:**
  - p50: <5ms
  - p95: <10ms
  - p99: <25ms
- **Memory:** <100MB for 10K concurrent connections
- **CPU:** <50% utilization at target throughput

### Estimated Effort

**Time:** 1-2 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Medium (performance targets)

---

## Development Phases

### Phase Summary

| Phase | Component | Duration | Team | Risk |
|-------|-----------|----------|------|------|
| 0 | Foundation | 1 week | 1 | Low |
| 1 | Router Implementation | 2 weeks | 1-2 | Low |
| 2 | Middleware Chain | 2 weeks | 1 | Low |
| 3 | Request/Response Lifecycle | 2 weeks | 1 | Low |
| 4 | Authentication & Authorization | 3 weeks | 1-2 | High |
| 5 | WebSocket Support | 2 weeks | 1 | Medium |
| 6 | Session Storage | 1-2 weeks | 1 | Medium |
| 7 | Background Jobs | 2-3 weeks | 1-2 | Medium |
| 8 | Caching Layer | 1-2 weeks | 1 | Low |
| 9 | Rate Limiting | 1 week | 1 | Low |
| 10 | Performance Optimization | 1-2 weeks | 1-2 | Medium |

**Total Duration:** 18-24 weeks
**Total Effort:** 90-120 person-days
**Recommended Team Size:** 1-2 engineers

### Critical Path

```
Foundation (Phase 0)
    ↓
Router (Phase 1) → Middleware (Phase 2) → Request/Response (Phase 3)
    ↓                                           ↓
Authentication (Phase 4) ← ← ← ← ← ← ← ← ← ← ←┘
    ↓
Session Storage (Phase 6)
    ↓
Background Jobs (Phase 7)
    ↓
WebSocket (Phase 5), Caching (Phase 8), Rate Limiting (Phase 9)
    ↓
Performance Optimization (Phase 10)
```

---

## Testing Strategy

### Testing Pyramid

```
        ┌─────────────────┐
        │   E2E Tests     │   10% - Full HTTP flows
        │    (10%)        │
        ├─────────────────┤
        │ Integration     │   30% - Middleware chains, auth flows
        │   Tests (30%)   │
        ├─────────────────┤
        │   Unit Tests    │   60% - Individual middleware, parsers
        │    (60%)        │
        └─────────────────┘
```

### Unit Tests (60%)

**Focus:** Individual functions, middleware components

**Examples:**
- Request parsing (JSON, form, multipart)
- Response formatting
- Middleware logic (auth, CORS, logging)
- Route matching and parameter extraction
- Token generation/validation

**Tools:**
- Go testing package
- httptest package
- Table-driven tests

**Target:** >90% code coverage

---

### Integration Tests (30%)

**Focus:** Component interactions

**Examples:**
- Middleware chain execution order
- Auth flow (login → token → protected endpoint)
- Request → validation → handler → response
- Cache middleware with Redis
- Job queue with workers
- WebSocket connection lifecycle

**Tools:**
- httptest.Server
- Real Redis instance (testcontainers)
- Mock ORM for resource handlers

**Target:** All major flows covered

---

### End-to-End Tests (10%)

**Focus:** Full HTTP scenarios

**Examples:**
- Create resource via POST
- List resources with pagination
- Update resource with authentication
- WebSocket real-time updates
- Background job execution

**Tools:**
- HTTP client tests
- Full server instance
- Real database and Redis

**Target:** Common use cases covered

---

### Performance Tests

**Focus:** Load and stress testing

**Scenarios:**
- Baseline throughput (simple GET)
- Latency under load
- Connection pool efficiency
- Memory usage under load
- WebSocket concurrent connections

**Tools:**
- wrk
- hey
- pprof (profiling)
- Apache Bench (ab)

**Targets:**
- 30K+ req/s
- <10ms p95 latency
- <100MB memory for 10K connections

---

### Security Tests

**Focus:** Vulnerability detection

**Scenarios:**
- JWT tampering attempts
- CSRF attack prevention
- Rate limit bypass attempts
- SQL injection (via parameter injection)
- Session hijacking
- XSS protection

**Tools:**
- Custom security test suite
- OWASP ZAP
- Manual penetration testing

**Target:** No critical vulnerabilities

---

## Integration Points

### 1. Resource System Integration

**Interface:**
```go
type ResourceRegistry interface {
    GetSchema(name string) (*ResourceSchema, error)
    GetCRUD(name string) (*CRUDOperations, error)
    ListResources() []string
}
```

**Integration:**
- Web framework generates routes from resource schemas
- Handlers delegate CRUD operations to ORM
- Middleware respects resource-level configuration
- Validation uses resource schema definitions

**Data Flow:**
```
Resource Definition → Schema → Router Registration → Handler Generation
```

---

### 2. Expression Interpreter Integration

**Interface:**
```go
type Interpreter interface {
    ExecuteMiddleware(ast *AST, ctx *ExecutionContext) error
    ExecuteHandler(ast *AST, ctx *ExecutionContext) (interface{}, error)
}
```

**Integration:**
- Execute custom middleware defined in language
- Provide HTTP context (request, response, params)
- Execute custom handler logic
- Handle @async blocks via job queue

**Data Flow:**
```
Custom Handler → AST → Interpreter → HTTP Context → Response
```

---

### 3. Runtime Introspection Integration

**Interface:**
```go
type IntrospectionAPI interface {
    GetRoutes() []*RouteInfo
    GetMiddleware() []string
    GetHandlers() map[string]*HandlerInfo
}
```

**Integration:**
- Web framework exposes route/middleware info
- Introspection available at runtime
- LLM uses for code generation
- Developer tools use for debugging

**Data Flow:**
```
Framework State → Introspection API → JSON Response → LLM/Tools
```

---

## Performance Targets

### Baseline Metrics

**Throughput:**
- Simple GET request: 30,000+ req/s
- POST with validation: 20,000+ req/s
- Authenticated request: 25,000+ req/s

**Latency (at 10K req/s):**
- p50: <5ms
- p95: <10ms
- p99: <25ms
- p99.9: <50ms

**Resource Usage:**
- Memory: <100MB for 10K concurrent connections
- CPU: <50% utilization at 30K req/s
- Database connections: <100 active
- Redis connections: <100 active

### Benchmark Scenarios

1. **Simple GET** (JSON response, no DB)
2. **POST with validation** (JSON body, validation rules)
3. **Authenticated request** (JWT verification)
4. **Cached request** (Redis cache hit)
5. **Database query** (single record fetch)
6. **WebSocket echo** (message round-trip)
7. **Background job** (enqueue latency)

---

## Risk Mitigation

### Risk 1: Performance Degradation Under Load

**Impact:** HIGH
**Probability:** MEDIUM

**Mitigation:**
1. Load testing from Phase 1 onward
2. Connection pooling for all resources
3. Caching strategy for frequently accessed data
4. Zero-copy optimizations in hot paths
5. Regular profiling with pprof

---

### Risk 2: Middleware Execution Order Bugs

**Impact:** HIGH (security vulnerabilities)
**Probability:** MEDIUM

**Mitigation:**
1. Explicit ordering rules (Global → Resource → Operation)
2. Integration tests for middleware chains
3. Linting for common mistakes
4. Introspection shows middleware order
5. Documentation with examples

---

### Risk 3: WebSocket Memory Leaks

**Impact:** HIGH (memory exhaustion)
**Probability:** MEDIUM

**Mitigation:**
1. Connection tracking with sync.Map
2. Automatic cleanup on disconnect
3. Read/write deadlines
4. Ping/pong health checks
5. Memory profiling during load tests

---

### Risk 4: Session Storage Failures

**Impact:** MEDIUM (users logged out)
**Probability:** LOW

**Mitigation:**
1. Fallback to memory store if Redis unavailable
2. Redis clustering with Sentinel
3. Session recovery with JWT fallback
4. Monitoring and alerting

---

### Risk 5: CSRF Vulnerabilities

**Impact:** CRITICAL (security)
**Probability:** HIGH

**Mitigation:**
1. Automatic CSRF middleware on all state-changing operations
2. Token management integrated with sessions
3. Security testing suite
4. Penetration testing

---

### Risk 6: Rate Limit Bypass

**Impact:** MEDIUM (DoS attacks)
**Probability:** MEDIUM

**Mitigation:**
1. Multiple rate limit dimensions (user, IP, endpoint, global)
2. Distributed rate limiting with Redis
3. Intelligent detection of suspicious patterns
4. Testing for bypass attempts

---

## Success Criteria

### Functional Requirements

- [ ] All auto-generated routes work correctly
- [ ] Custom routes integrate seamlessly
- [ ] Middleware chains execute in correct order
- [ ] Authentication (JWT and session) works
- [ ] Authorization checks enforce permissions
- [ ] WebSocket connections are stable
- [ ] Session storage persists correctly
- [ ] Background jobs execute reliably
- [ ] Caching improves performance measurably
- [ ] Rate limiting prevents abuse

### Performance Requirements

- [ ] 30K+ req/s baseline throughput
- [ ] <10ms p95 latency
- [ ] <100MB memory for 10K connections
- [ ] <50% CPU at target throughput
- [ ] Zero memory leaks under load

### Quality Requirements

- [ ] >90% test coverage (overall)
- [ ] >95% test coverage (security components)
- [ ] All integration tests pass
- [ ] All security tests pass
- [ ] No critical vulnerabilities (OWASP)
- [ ] Performance benchmarks meet targets

### Developer Experience

- [ ] Clear documentation
- [ ] Examples for all features
- [ ] Introspection API works
- [ ] Error messages are helpful
- [ ] Middleware is easy to write
- [ ] Route registration is intuitive

---

## Appendix: Configuration Example

```go
config := &WebFrameworkConfig{
    // Server
    Addr:              ":8080",
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1MB

    // TLS
    TLSEnabled:        true,
    CertFile:          "/etc/certs/server.crt",
    KeyFile:           "/etc/certs/server.key",

    // Database
    DatabaseURL:       "postgres://user:pass@localhost/db",
    MaxDBConnections:  100,

    // Redis
    RedisAddr:         "localhost:6379",
    RedisPassword:     "",
    RedisDB:           0,

    // Sessions
    SessionTTL:        24 * time.Hour,
    SessionStorageType: "redis",

    // Rate Limiting
    RateLimitEnabled:  true,
    RateLimitRate:     100,
    RateLimitWindow:   1 * time.Minute,

    // Caching
    CacheEnabled:      true,
    CacheTTL:          5 * time.Minute,

    // Background Jobs
    NumWorkers:        10,

    // Security
    CSRFEnabled:       true,
    JWTSecret:         []byte("your-secret-key"),
    JWTDuration:       1 * time.Hour,

    // Performance
    MaxBodySize:       10 << 20, // 10MB
    EnableGzip:        true,
}
```

---

**Document Status:** Complete
**Last Updated:** 2025-10-13
**Next Steps:** Begin Phase 0 - Foundation
