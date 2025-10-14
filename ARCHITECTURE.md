# Conduit System Architecture

**Version:** 1.0
**Status:** Reference Architecture
**Updated:** 2025-10-13

This document provides a comprehensive overview of the Conduit language system architecture, describing how all components work together to deliver an LLM-first programming experience.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Overview](#system-overview)
3. [Component Architecture](#component-architecture)
4. [Compilation Pipeline](#compilation-pipeline)
5. [Runtime Architecture](#runtime-architecture)
6. [Data Flow](#data-flow)
7. [Deployment Architecture](#deployment-architecture)
8. [Technology Stack](#technology-stack)
9. [Integration Points](#integration-points)
10. [Design Decisions](#design-decisions)
11. [Performance Characteristics](#performance-characteristics)
12. [Security Architecture](#security-architecture)
13. [Future Evolution](#future-evolution)

---

## Executive Summary

### What is Conduit?

Conduit is an LLM-first programming language designed to make web application development seamless for both AI tools and human developers. It provides:

- **Explicit, unambiguous syntax** optimized for LLM code generation
- **Compile-to-Go** approach for native performance and simple deployment
- **Built-in ORM and web framework** eliminating boilerplate
- **Runtime introspection** enabling pattern discovery and documentation
- **Developer tooling** with LSP, debugger, and hot reload

### Architecture Philosophy

**Single Binary Deployment**
- Compile-to-Go produces self-contained executables
- No runtime dependencies or virtual machines
- Simple deployment: copy binary and run

**Progressive Disclosure**
- Simple applications require minimal code
- Advanced features available when needed
- Complexity doesn't leak into simple use cases

**LLM-Optimized Design**
- Explicit nullability prevents ambiguity
- Namespaced standard library eliminates hallucination
- Structured error messages enable self-correction
- Pattern extraction enables learning from existing code

**Convention over Configuration**
- Sensible defaults for common use cases
- Compile-time enforcement of conventions
- Zero-config for standard scenarios

---

## System Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Developer Experience                        │
├──────────────┬──────────────┬──────────────┬───────────────────┤
│  IDE/Editor  │   CLI Tool   │  Web Browser │  LLM (Claude/GPT) │
│  (with LSP)  │   (conduit)  │  (Hot Reload)│  (Code Generator) │
└──────┬───────┴──────┬───────┴──────┬───────┴──────┬────────────┘
       │              │              │              │
       └──────────────┴──────────────┴──────────────┘
                            │
       ┌────────────────────┴────────────────────┐
       │           Development Tools              │
       ├──────────────────┬──────────────────────┤
       │  Language Server │  Watch Mode & Hot    │
       │  Protocol (LSP)  │  Reload Manager      │
       ├──────────────────┼──────────────────────┤
       │  Debug Adapter   │  Code Formatter      │
       │  Protocol (DAP)  │  & Project Templates │
       └────────┬─────────┴──────────────────────┘
                │
       ┌────────┴─────────────────────────────────┐
       │             Compiler                     │
       ├───────────┬──────────────┬───────────────┤
       │   Lexer   │    Parser    │  Type Checker │
       │  (Tokens) │  (AST Build) │  (Validation) │
       ├───────────┴──────────────┴───────────────┤
       │      Go Code Generator                   │
       │  (AST → Go Source Transformation)        │
       └────────┬─────────────────────────────────┘
                │
       ┌────────┴─────────────────────────────────┐
       │          Go Toolchain                    │
       │      (go build, go fmt, go test)         │
       └────────┬─────────────────────────────────┘
                │
       ┌────────┴─────────────────────────────────┐
       │         Compiled Application             │
       ├──────────────────────────────────────────┤
       │              Runtime                     │
       ├───────────┬───────────┬──────────────────┤
       │    ORM    │    Web    │  Introspection   │
       │  (Query)  │ Framework │   (Pattern API)  │
       ├───────────┼───────────┼──────────────────┤
       │  Resource │  Routing  │  Metadata Store  │
       │ Operations│ Middleware│  Schema Registry │
       ├───────────┴───────────┴──────────────────┤
       │        Standard Library                  │
       │  (String, Time, Array, Hash, etc.)       │
       └────────┬─────────────────────────────────┘
                │
       ┌────────┴─────────────────────────────────┐
       │       External Dependencies              │
       ├──────────────────┬───────────────────────┤
       │    Database      │    HTTP Server        │
       │  (PostgreSQL)    │    (net/http)         │
       └──────────────────┴───────────────────────┘
```

### System Boundaries

**Compile Time**
- Source code parsing and validation
- Type checking and constraint verification
- Go code generation
- Introspection metadata generation

**Runtime**
- HTTP request handling
- Database operations
- Lifecycle hook execution
- Validation and constraint enforcement
- Introspection queries

**Development Time**
- File watching and hot reload
- IDE integration via LSP
- Debugging via DAP
- Pattern extraction and documentation

---

## Component Architecture

### 1. Compiler Subsystem

**Purpose:** Transform Conduit source files (`.cdt`) into executable Go code.

**Components:**

```
Compiler
├── Lexer/Tokenizer
│   ├── Character stream → Token stream
│   ├── Source location tracking
│   └── Comment preservation
│
├── Parser
│   ├── Token stream → Abstract Syntax Tree (AST)
│   ├── Error recovery for partial parsing
│   └── Source mapping for errors
│
├── Type System
│   ├── Type representation (primitives, structs, arrays, hashes)
│   ├── Nullability checking (! vs ?)
│   ├── Type inference for local variables
│   └── Constraint validation
│
├── Semantic Analyzer
│   ├── Symbol resolution
│   ├── Scope checking
│   ├── Relationship validation
│   └── Lifecycle hook verification
│
├── Code Generator
│   ├── AST → Go code transformation
│   ├── Resource → Go struct generation
│   ├── Hook → Go function generation
│   ├── Query → Go ORM code generation
│   └── Import management
│
└── Introspection Generator
    ├── Metadata extraction from AST
    ├── Pattern recognition
    └── Documentation generation
```

**Key Data Structures:**

```go
// Core AST nodes
type AST struct {
    Resources   []*ResourceNode
    Imports     []*ImportNode
    SourceFiles []*SourceFile
}

type ResourceNode struct {
    Name         string
    Documentation string
    Fields       []*FieldNode
    Relationships []*RelationshipNode
    Hooks        []*HookNode
    Validations  []*ValidationNode
    Computed     []*ComputedNode
    Functions    []*FunctionNode
    Middleware   []*MiddlewareNode
    Scopes       []*ScopeNode
    Location     SourceLocation
}

// Type system
type Type interface {
    IsNullable() bool
    Equals(Type) bool
    String() string
}

type PrimitiveType struct {
    Kind     PrimitiveKind // string, int, float, bool, etc.
    Nullable bool
    Constraints []*Constraint
}
```

**File:** See `IMPLEMENTATION-COMPILER.md` for detailed implementation.

---

### 2. Runtime Subsystem

**Purpose:** Execute compiled applications with support for introspection, lifecycle management, and dynamic queries.

**Components:**

```
Runtime
├── Core Runtime
│   ├── Application Bootstrap
│   ├── Configuration Management
│   ├── Database Connection Pool
│   └── Middleware Chain
│
├── Resource Lifecycle Manager
│   ├── Hook Execution Engine
│   ├── Transaction Management
│   ├── Async Task Queue
│   └── Change Tracking
│
├── Validation Engine
│   ├── Field Constraint Validation
│   ├── Declarative Constraint Checking
│   ├── Runtime Invariant Verification
│   └── Custom Validation Execution
│
├── Introspection API
│   ├── Schema Registry (all resources, fields, types)
│   ├── Pattern Database (extracted patterns)
│   ├── Relationship Graph
│   ├── Query Interface
│   └── Documentation Server
│
└── Standard Library
    ├── String namespace
    ├── Time namespace
    ├── Array namespace
    ├── Hash namespace
    ├── Crypto namespace
    └── ... (all stdlib namespaces)
```

**Key Interfaces:**

```go
// Runtime interface
type Runtime interface {
    Bootstrap(config *Config) error
    RegisterResource(resource *ResourceMeta) error
    RegisterMiddleware(name string, handler MiddlewareFunc)
    GetIntrospection() *IntrospectionAPI
    Shutdown(ctx context.Context) error
}

// Introspection API
type IntrospectionAPI interface {
    GetSchema() *Schema
    GetResource(name string) (*ResourceMeta, error)
    QueryPatterns(query PatternQuery) ([]*Pattern, error)
    GetRelationships(resourceName string) ([]*Relationship, error)
}

// Hook execution
type HookExecutor interface {
    ExecuteBefore(ctx context.Context, op Operation, resource interface{}) error
    ExecuteAfter(ctx context.Context, op Operation, resource interface{}) error
}
```

**File:** See `IMPLEMENTATION-RUNTIME.md` for detailed implementation.

---

### 3. ORM Subsystem

**Purpose:** Provide database abstraction, query building, and resource operations.

**Components:**

```
ORM
├── Query Builder
│   ├── Where clause construction
│   ├── Join management
│   ├── Order/limit/offset
│   ├── Aggregation (count, sum, avg)
│   └── SQL generation
│
├── Resource Operations
│   ├── Create/Find/Update/Delete
│   ├── Bulk operations
│   ├── Increment/Decrement
│   └── Transaction wrapping
│
├── Relationship Manager
│   ├── Belongs-to resolution
│   ├── Has-many loading
│   ├── Has-many-through joins
│   ├── Eager loading (N+1 prevention)
│   └── Nested resource operations
│
├── Migration System
│   ├── Schema diff generation
│   ├── Migration file creation
│   ├── Migration execution
│   └── Rollback support
│
└── Query Scopes
    ├── Named scope management
    ├── Scope composition
    └── Dynamic scope building
```

**Query Architecture:**

```go
// Query builder
type QueryBuilder struct {
    resource   *ResourceMeta
    conditions []*Condition
    joins      []*Join
    includes   []string
    orderBy    []string
    limit      *int
    offset     *int
}

// Fluent API
func (qb *QueryBuilder) Where(conditions ...Condition) *QueryBuilder
func (qb *QueryBuilder) OrderBy(fields ...string) *QueryBuilder
func (qb *QueryBuilder) Limit(n int) *QueryBuilder
func (qb *QueryBuilder) Includes(relations ...string) *QueryBuilder
func (qb *QueryBuilder) ToSQL() (string, []interface{}, error)
func (qb *QueryBuilder) Execute(ctx context.Context) ([]*Resource, error)
```

**File:** See `IMPLEMENTATION-ORM.md` for detailed implementation.

---

### 4. Web Framework Subsystem

**Purpose:** Handle HTTP requests, routing, middleware, and REST API generation.

**Components:**

```
Web Framework
├── Router
│   ├── Resource route registration
│   ├── Custom route support
│   ├── Nested resource routing
│   ├── Path parameter extraction
│   └── Method routing (GET/POST/PUT/DELETE)
│
├── Handler Generator
│   ├── CRUD handler generation
│   ├── Request parsing (JSON/form/query)
│   ├── Response formatting (JSON)
│   ├── Error handling
│   └── Pagination support
│
├── Middleware System
│   ├── Middleware chain execution
│   ├── Built-in middleware (auth, CORS, rate limit)
│   ├── Custom middleware registration
│   └── Per-resource middleware
│
├── Request/Response
│   ├── Request context
│   ├── Parameter binding
│   ├── Validation
│   └── Response builders
│
└── WebSocket Support
    ├── Connection management
    ├── Hot reload notifications
    └── Real-time updates
```

**Request Flow:**

```
HTTP Request
    ↓
Router (match path, method)
    ↓
Global Middleware Chain
    ↓
Resource-Specific Middleware
    ↓
Handler (@before hooks)
    ↓
Handler (operation: create/read/update/delete)
    ↓
Handler (@after hooks)
    ↓
Response Formatter
    ↓
HTTP Response
```

**File:** See `IMPLEMENTATION-WEB.md` for detailed implementation.

---

### 5. Tooling Subsystem

**Purpose:** Provide developer experience tools for editing, debugging, and iterating.

**Components:**

```
Tooling
├── CLI Tool
│   ├── new (project scaffolding)
│   ├── build (compilation)
│   ├── run (dev server)
│   ├── format (code formatting)
│   ├── introspect (query metadata)
│   ├── docs (documentation generation)
│   └── migrate (database migrations)
│
├── Language Server Protocol (LSP)
│   ├── Hover (type information)
│   ├── Completion (code completion)
│   ├── Go to Definition
│   ├── Find References
│   ├── Diagnostics (real-time errors)
│   └── Symbol search
│
├── Watch Mode & Hot Reload
│   ├── File system watching (fsnotify)
│   ├── Change detection & debouncing
│   ├── Incremental compilation
│   ├── Browser refresh (WebSocket)
│   └── State preservation
│
├── Debug Adapter Protocol (DAP)
│   ├── Delve integration (Go debugger)
│   ├── Source map translation
│   ├── Breakpoint management
│   ├── Variable inspection
│   └── Stack trace mapping
│
├── Code Formatter
│   ├── AST-based formatting
│   ├── Idempotent output
│   ├── Style enforcement
│   └── CLI/LSP integration
│
├── Project Templates
│   ├── Template engine
│   ├── Built-in templates (minimal, blog, SaaS)
│   ├── Variable substitution
│   └── Custom template support
│
├── Build System
│   ├── Incremental compilation
│   ├── Build cache (per-file hashing)
│   ├── Dependency graph
│   └── Parallel builds
│
└── Documentation Generator
    ├── Introspection-based generation
    ├── Markdown output
    ├── API reference
    └── Pattern examples
```

**File:** See `IMPLEMENTATION-TOOLING.md` for detailed implementation.

---

## Compilation Pipeline

### Source to Executable Flow

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 1: Source Processing                                       │
└──────────────────────────────────────────────────────────────────┘

Source Files (*.cdt)
    ↓
[Lexer] Tokenization
    ↓
Token Stream
    ↓
[Parser] Syntax Analysis
    ↓
Abstract Syntax Tree (AST)
    ↓
[Symbol Resolution] Build symbol table
    ↓
Annotated AST

┌──────────────────────────────────────────────────────────────────┐
│ Phase 2: Semantic Analysis                                       │
└──────────────────────────────────────────────────────────────────┘

Annotated AST
    ↓
[Type Checker] Verify types, nullability
    ↓
[Constraint Validator] Check field constraints
    ↓
[Relationship Validator] Verify foreign keys, relationships
    ↓
[Hook Validator] Validate lifecycle hooks
    ↓
Validated AST

┌──────────────────────────────────────────────────────────────────┐
│ Phase 3: Code Generation                                         │
└──────────────────────────────────────────────────────────────────┘

Validated AST
    ↓
[Resource Generator] Generate Go structs
    ↓
[Hook Generator] Generate lifecycle functions
    ↓
[Query Generator] Generate query methods
    ↓
[Handler Generator] Generate HTTP handlers
    ↓
[Main Generator] Generate main.go with bootstrap
    ↓
Generated Go Code (*.go files)

┌──────────────────────────────────────────────────────────────────┐
│ Phase 4: Introspection Metadata                                  │
└──────────────────────────────────────────────────────────────────┘

Validated AST
    ↓
[Metadata Extractor] Extract schema, types, relationships
    ↓
[Pattern Analyzer] Identify patterns
    ↓
[Documentation Generator] Create API docs
    ↓
Introspection Data (JSON)

┌──────────────────────────────────────────────────────────────────┐
│ Phase 5: Go Compilation                                          │
└──────────────────────────────────────────────────────────────────┘

Generated Go Code
    ↓
[go fmt] Format code
    ↓
[go build] Compile to binary
    ↓
Executable Binary

┌──────────────────────────────────────────────────────────────────┐
│ Output                                                            │
└──────────────────────────────────────────────────────────────────┘

- Binary: ./build/app
- Metadata: ./build/app.meta.json
- Source Map: ./build/app.map
```

### Example Transformation

**Input (post.cdt):**

```conduit
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  content: text!
  author: User!

  @before create {
    self.slug = String.slugify(self.title)
  }
}
```

**Output (post.go):**

```go
package resources

import (
    "github.com/google/uuid"
    "conduit/runtime"
    "conduit/stdlib"
)

// Post represents a blog post
type Post struct {
    ID      uuid.UUID `db:"id" json:"id"`
    Title   string    `db:"title" json:"title"`
    Content string    `db:"content" json:"content"`
    AuthorID uuid.UUID `db:"author_id" json:"author_id"`
    Slug    string    `db:"slug" json:"slug"`
}

// Validate performs field validation
func (p *Post) Validate() error {
    if len(p.Title) < 5 || len(p.Title) > 200 {
        return runtime.NewValidationError("title", "length must be between 5 and 200")
    }
    return nil
}

// BeforeCreate hook
func (p *Post) BeforeCreate(ctx runtime.Context) error {
    p.Slug = stdlib.String.Slugify(p.Title)
    return nil
}

// Generated repository methods
func (r *PostRepository) Create(ctx context.Context, post *Post) error {
    if err := post.Validate(); err != nil {
        return err
    }
    if err := post.BeforeCreate(ctx); err != nil {
        return err
    }
    // Insert into database...
}
```

---

## Runtime Architecture

### Application Lifecycle

```
┌──────────────────────────────────────────────────────────────────┐
│ Startup                                                           │
└──────────────────────────────────────────────────────────────────┘

main()
    ↓
[Load Configuration] Read config file, environment vars
    ↓
[Initialize Runtime] Create runtime instance
    ↓
[Load Introspection Metadata] Parse .meta.json
    ↓
[Register Resources] Register all resource types
    ↓
[Connect Database] Establish connection pool
    ↓
[Register Middleware] Global + resource middleware
    ↓
[Build Routes] Generate REST API routes
    ↓
[Start HTTP Server] Listen on port
    ↓
Ready (serve requests)

┌──────────────────────────────────────────────────────────────────┐
│ Request Handling                                                  │
└──────────────────────────────────────────────────────────────────┘

HTTP Request
    ↓
[Router] Match route
    ↓
[Context Builder] Build request context (user, params, etc.)
    ↓
[Global Middleware] Auth, CORS, rate limit, etc.
    ↓
[Resource Middleware] Resource-specific middleware
    ↓
[Handler] Parse request, validate input
    ↓
[@before hooks] Execute before hooks in transaction
    ↓
[Operation] Create/Read/Update/Delete
    ↓
[@after hooks] Execute after hooks
    ↓
[Async Jobs] Queue async tasks (notifications, indexing)
    ↓
[Response Formatter] Build JSON response
    ↓
HTTP Response

┌──────────────────────────────────────────────────────────────────┐
│ Shutdown                                                          │
└──────────────────────────────────────────────────────────────────┘

SIGTERM/SIGINT
    ↓
[Graceful Shutdown] Stop accepting new requests
    ↓
[Wait for In-Flight] Wait for active requests (timeout: 30s)
    ↓
[Close Database] Close connection pool
    ↓
[Flush Logs] Flush buffered logs
    ↓
Exit
```

### Memory Layout

```
┌─────────────────────────────────────────────────────────────────┐
│                     Process Memory                               │
├─────────────────────────────────────────────────────────────────┤
│  Text Segment (Code)                                            │
│    - Compiled Go binary                                         │
│    - Generated resource operations                              │
│    - Standard library functions                                 │
├─────────────────────────────────────────────────────────────────┤
│  Data Segment (Static Data)                                     │
│    - Introspection metadata                                     │
│    - Compiled templates                                         │
│    - Configuration constants                                    │
├─────────────────────────────────────────────────────────────────┤
│  Heap (Dynamic Allocations)                                     │
│    - HTTP request contexts                                      │
│    - Database query results                                     │
│    - Resource instances                                         │
│    - Middleware state                                           │
│    - Go runtime (GC heap)                                       │
├─────────────────────────────────────────────────────────────────┤
│  Stack (Per-Goroutine)                                          │
│    - Function call frames                                       │
│    - Local variables                                            │
│    - Handler execution                                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Flow

### Write Operation (Create/Update)

```
Client
  │
  │ POST /posts
  │ { "title": "Hello", "content": "..." }
  ↓
Router
  │
  ↓
Middleware Chain (auth, rate_limit, ...)
  │
  ↓
Handler (PostsController.Create)
  │
  ├─→ Parse Request Body
  │     ↓
  ├─→ Validate Input (field constraints)
  │     ↓
  ├─→ Begin Transaction
  │     ↓
  ├─→ Execute @before hooks
  │     │
  │     ├─→ self.slug = String.slugify(self.title)
  │     ├─→ self.published_at = Time.now()
  │     └─→ (other before logic)
  │     ↓
  ├─→ Validate (constraints, invariants)
  │     ↓
  ├─→ Insert into Database
  │     │
  │     └─→ INSERT INTO posts (...)
  │     ↓
  ├─→ Execute @after hooks
  │     │
  │     ├─→ Sync operations (in transaction)
  │     │     └─→ Revision.create!(...)
  │     │
  │     └─→ @async operations (queued)
  │           ├─→ Send notifications
  │           └─→ Update search index
  │     ↓
  ├─→ Commit Transaction
  │     ↓
  └─→ Queue Async Jobs
        ↓
Response
  │
  │ 201 Created
  │ { "id": "...", "title": "Hello", ... }
  ↓
Client

(Async Jobs Execute in Background)
```

### Read Operation (List/Get)

```
Client
  │
  │ GET /posts?status=published&page=1
  ↓
Router
  │
  ↓
Middleware Chain (cache, auth, ...)
  │
  ↓
Handler (PostsController.List)
  │
  ├─→ Parse Query Parameters
  │     │
  │     ├─→ filters: { status: "published" }
  │     ├─→ page: 1
  │     └─→ per_page: 20
  │     ↓
  ├─→ Apply Query Scopes
  │     │
  │     └─→ Post.published.where(...)
  │     ↓
  ├─→ Build SQL Query
  │     │
  │     └─→ SELECT * FROM posts WHERE status = 'published'
  │           ORDER BY published_at DESC
  │           LIMIT 20 OFFSET 0
  │     ↓
  ├─→ Execute Query
  │     ↓
  ├─→ Load Relationships (if includes specified)
  │     │
  │     └─→ SELECT * FROM users WHERE id IN (...)
  │     ↓
  ├─→ Serialize to JSON
  │     ↓
  └─→ Cache Response (if cache middleware enabled)
        ↓
Response
  │
  │ 200 OK
  │ { "data": [...], "meta": { "total": 150, "page": 1 } }
  ↓
Client
```

### Introspection Query

```
Client/LLM
  │
  │ POST /introspect
  │ { "query": "what resources exist?" }
  ↓
Introspection API
  │
  ├─→ Parse Query
  │     ↓
  ├─→ Query Schema Registry
  │     │
  │     └─→ In-memory schema data (loaded from .meta.json)
  │     ↓
  ├─→ Format Response (JSON for LLMs, Markdown for humans)
  │     ↓
  └─→ Return Results
        ↓
Response
  │
  │ 200 OK
  │ { "resources": ["Post", "User", "Comment"], ... }
  ↓
Client/LLM
```

---

## Deployment Architecture

### Development Environment

```
┌─────────────────────────────────────────────────────────────────┐
│                     Developer Machine                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────┐        ┌─────────────────┐                 │
│  │   IDE/Editor    │◄──────►│   LSP Server    │                 │
│  │  (VS Code)      │  JSON  │  (conduit lsp)  │                 │
│  └─────────────────┘  RPC   └─────────────────┘                 │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │   conduit run --watch                                   │    │
│  │                                                          │    │
│  │   ┌─────────────┐   ┌──────────────┐  ┌─────────────┐  │    │
│  │   │ File Watcher│──→│  Compiler    │─→│  Dev Server │  │    │
│  │   │  (fsnotify) │   │ (incremental)│  │  (port 3000)│  │    │
│  │   └─────────────┘   └──────────────┘  └──────┬──────┘  │    │
│  │                                                │         │    │
│  │   ┌────────────────────────────────────────────┼──────┐  │    │
│  │   │ Hot Reload Manager                         │      │  │    │
│  │   │  - WebSocket Server (port 3001) ◄──────────┘      │  │    │
│  │   │  - Change detection & browser refresh             │  │    │
│  │   └────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌─────────────────┐                                             │
│  │   PostgreSQL    │ (Docker or local)                           │
│  │  (port 5432)    │                                             │
│  └─────────────────┘                                             │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     Web Browser                                  │
├─────────────────────────────────────────────────────────────────┤
│  Application (http://localhost:3000)                             │
│     ↕                                                             │
│  WebSocket (ws://localhost:3001) ← Hot reload notifications      │
└─────────────────────────────────────────────────────────────────┘
```

### Production Environment (Single Server)

```
┌─────────────────────────────────────────────────────────────────┐
│                        Production Server                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │   Reverse Proxy (Nginx/Caddy)                          │    │
│  │   - SSL termination                                     │    │
│  │   - Static file serving                                 │    │
│  │   - Rate limiting                                       │    │
│  └──────────────────────┬──────────────────────────────────┘    │
│                         │                                         │
│  ┌──────────────────────┴──────────────────────────────────┐    │
│  │   Application Binary (./app)                            │    │
│  │   - Single executable                                   │    │
│  │   - No runtime dependencies                             │    │
│  │   - Systemd service                                     │    │
│  └──────────────────────┬──────────────────────────────────┘    │
│                         │                                         │
│  ┌──────────────────────┴──────────────────────────────────┐    │
│  │   PostgreSQL                                            │    │
│  │   - Connection pool (max 100)                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Production Environment (Multi-Server with Load Balancer)

```
                    ┌──────────────────────────┐
                    │    Load Balancer         │
                    │    (AWS ALB/HAProxy)     │
                    └──────────┬───────────────┘
                               │
             ┌─────────────────┴─────────────────┐
             │                                   │
   ┌─────────┴─────────┐               ┌────────┴──────────┐
   │   App Server 1    │               │   App Server 2    │
   │   (./app)         │               │   (./app)         │
   └─────────┬─────────┘               └────────┬──────────┘
             │                                   │
             └─────────────────┬─────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │  PostgreSQL (Primary)       │
                │  + Read Replicas (optional) │
                └─────────────────────────────┘

   ┌───────────────────────────────────────────────────────────┐
   │  Shared Services                                          │
   ├───────────────────────────────────────────────────────────┤
   │  - Redis (sessions, cache)                                │
   │  - S3 (file uploads)                                      │
   │  - CloudWatch/Prometheus (metrics)                        │
   └───────────────────────────────────────────────────────────┘
```

### Container Deployment (Docker/Kubernetes)

**Dockerfile:**

```dockerfile
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN conduit build --release

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/build/app .
COPY --from=builder /app/build/app.meta.json .
EXPOSE 3000
CMD ["./app"]
```

**Kubernetes Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: conduit-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: conduit-app
  template:
    metadata:
      labels:
        app: conduit-app
    spec:
      containers:
      - name: app
        image: myapp:latest
        ports:
        - containerPort: 3000
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## Technology Stack

### Language and Compilation

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Source Language | Conduit | 2.0 | Application code |
| Target Language | Go | 1.23+ | Compilation target |
| Compiler Implementation | Go | 1.23+ | Self-hosted compiler |
| Parser | Recursive Descent | - | Hand-written parser |
| Build Tool | go build | - | Final compilation |

### Runtime Dependencies

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Database | PostgreSQL | 15+ | Primary data store |
| HTTP Server | net/http | stdlib | HTTP handling |
| Database Driver | pgx | v5 | PostgreSQL driver |
| Router | chi | v5 | HTTP routing |
| JSON | encoding/json | stdlib | Serialization |
| UUID | google/uuid | v1 | UUID generation |

### Development Tools

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| CLI Framework | Cobra | v1.8 | Command-line interface |
| File Watching | fsnotify | v1.7 | File change detection |
| LSP | gopls libs | - | Language server |
| Debugger | Delve | v1.22 | Debug adapter |
| Formatter | AST-based | custom | Code formatting |
| Testing | go test | stdlib | Unit testing |

### External Services (Optional)

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Cache | Redis | Session/query cache |
| File Storage | S3/MinIO | File uploads |
| Search | Elasticsearch | Full-text search |
| Monitoring | Prometheus | Metrics collection |
| Logging | Loki | Log aggregation |
| Tracing | Jaeger | Distributed tracing |

---

## Integration Points

### 1. Compiler ↔ Runtime

**Interface:** Generated Go code imports runtime package

```go
import (
    "conduit/runtime"
    "conduit/orm"
    "conduit/web"
)

// Resource struct uses runtime tags
type Post struct {
    ID uuid.UUID `runtime:"primary" db:"id" json:"id"`
}

// Hooks call runtime functions
func (p *Post) BeforeCreate(ctx runtime.Context) error {
    // Use runtime context for user, transaction, etc.
}

// Register with runtime at startup
func init() {
    runtime.RegisterResource(&runtime.ResourceMeta{
        Name: "Post",
        Table: "posts",
        Fields: [...],
    })
}
```

### 2. Compiler ↔ Introspection

**Interface:** Compiler generates `.meta.json` file

```json
{
  "version": "1.0",
  "resources": [
    {
      "name": "Post",
      "table": "posts",
      "fields": [
        {
          "name": "title",
          "type": "string",
          "nullable": false,
          "constraints": [
            { "type": "min", "value": 5 },
            { "type": "max", "value": 200 }
          ]
        }
      ],
      "relationships": [...],
      "hooks": [...],
      "patterns": [...]
    }
  ]
}
```

Runtime loads this file at startup:

```go
func (rt *Runtime) LoadIntrospection(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &rt.schema)
}
```

### 3. Runtime ↔ ORM

**Interface:** Runtime provides transaction and context, ORM executes queries

```go
// Runtime starts transaction
tx, err := runtime.BeginTransaction(ctx)
if err != nil {
    return err
}

// ORM executes within transaction
post := &Post{Title: "Hello"}
err = orm.Create(tx, post)
if err != nil {
    tx.Rollback()
    return err
}

// Runtime commits
return tx.Commit()
```

### 4. Runtime ↔ Web Framework

**Interface:** Web framework registers routes with runtime, calls handlers

```go
// Runtime initialization
func (rt *Runtime) Init() {
    router := web.NewRouter()

    // Register resource routes
    for _, resource := range rt.GetResources() {
        web.RegisterRESTRoutes(router, resource)
    }

    // Start server
    http.ListenAndServe(":3000", router)
}

// Handler execution
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := runtime.BuildContext(r)

    // Parse request
    var post Post
    json.NewDecoder(r.Body).Decode(&post)

    // Execute with lifecycle
    err := runtime.ExecuteCreate(ctx, &post)

    // Return response
    web.JSON(w, 201, post)
}
```

### 5. Tooling ↔ Compiler

**Interface:** LSP and other tools use compiler API

```go
// LSP server
type LanguageServer struct {
    compiler *Compiler
}

func (ls *LanguageServer) Hover(params *HoverParams) (*Hover, error) {
    // Parse file
    ast, err := ls.compiler.Parse(params.File)
    if err != nil {
        return nil, err
    }

    // Find symbol at position
    symbol := ast.FindSymbolAt(params.Position)

    // Get type information
    typeInfo := ls.compiler.TypeOf(symbol)

    // Format hover
    return &Hover{
        Contents: formatTypeInfo(typeInfo),
    }, nil
}
```

---

## Design Decisions

### 1. Why Compile to Go?

**Decision:** Compile Conduit source to Go source, then use Go toolchain.

**Rationale:**
- **Fast compilation:** Go compiles in seconds, enabling rapid iteration
- **Simple deployment:** Single binary, no runtime dependencies
- **Mature ecosystem:** Leverage existing Go libraries and tools
- **Easy debugging:** Generated code is readable Go
- **Portability:** Go's cross-compilation works out of the box

**Trade-offs:**
- Not maximum performance (Rust would be 20-40% faster)
- WebAssembly requires TinyGo (adds complexity)
- Debugging shows generated code, not source (mitigated by source maps)

**Alternatives Considered:**
- LLVM: More complex, slower compilation
- Rust: Slower compilation, steeper learning curve
- JVM/.NET: Larger binaries, slower startup
- Interpreted: Too slow for web servers

**File:** See `docs/research/compilation-target-research.md`

---

### 2. Why AST-Based Code Generation?

**Decision:** Transform AST to Go code, not direct bytecode generation.

**Rationale:**
- **Leverage Go toolchain:** Use `go build` for optimization and linking
- **Readable output:** Developers can inspect generated code
- **Easier debugging:** Go debugger works natively
- **Simpler implementation:** Don't need to write optimizer

**Trade-offs:**
- Extra compilation step (Conduit → Go → binary)
- Generated code can be verbose
- Compilation time includes both Conduit and Go compilation

---

### 3. Why Runtime Introspection?

**Decision:** Maintain schema metadata at runtime for queries.

**Rationale:**
- **LLM pattern discovery:** LLMs can query "how do I..." and get examples
- **Documentation generation:** Auto-generate API docs from running app
- **Migration generation:** Compare runtime schema to database
- **API exploration:** Developers can query available resources

**Trade-offs:**
- Memory overhead (schema data loaded at startup)
- Binary size increase (metadata embedded)
- Potential security risk (exposes schema)

**Mitigation:**
- Make introspection API optional in production
- Only expose to authenticated requests
- Strip introspection in release builds (optional flag)

---

### 4. Why Explicit Nullability?

**Decision:** Require `!` or `?` on every type.

**Rationale:**
- **Zero ambiguity:** LLMs know exactly when nil is allowed
- **Compile-time safety:** Catch null reference errors before runtime
- **Clear intent:** Readers immediately see optionality

**Trade-offs:**
- More verbose than implicit nullability
- Requires migration from dynamically typed languages

**Alternatives Considered:**
- Implicit nullability (like Go): Too ambiguous for LLMs
- Optional types (like Rust Option<T>): More complex syntax
- Null by default: Unsafe, causes runtime errors

---

### 5. Why Namespaced Standard Library?

**Decision:** All built-in functions use namespaces (`String.slugify()` not `slugify()`).

**Rationale:**
- **Prevent hallucination:** LLMs can't invent functions; they must use namespaces
- **Clear provenance:** Obvious what's built-in vs custom
- **Avoid name collisions:** Custom functions can use simple names
- **Better autocomplete:** IDE shows all String functions under `String.`

**Trade-offs:**
- More verbose than global functions
- Less familiar to developers from Ruby/Python

**Alternatives Considered:**
- Global functions: LLMs hallucinate names
- Module imports: Requires import management, still ambiguous
- Prefix convention: Not enforced by compiler

---

### 6. Why Transaction Boundaries Explicit?

**Decision:** Require `@transaction` and `@async` annotations.

**Rationale:**
- **No surprises:** Developers know exactly what runs in transaction
- **Performance control:** Avoid accidentally blocking async work
- **Error handling clarity:** Transactional errors roll back, async errors log

**Trade-offs:**
- More annotations required
- Developers must understand transaction semantics

**Alternatives Considered:**
- Implicit transactions: Too magical, performance issues
- No transactions: Unsafe, data integrity problems
- Transaction-per-request: Doesn't handle complex workflows

---

## Performance Characteristics

### Compilation Speed

| Metric | Target | Notes |
|--------|--------|-------|
| Lexer/Parser | < 10ms per 1000 LOC | In-memory parsing |
| Type Checking | < 20ms per 1000 LOC | Single-pass |
| Code Generation | < 50ms per 1000 LOC | Template-based |
| Go Compilation | < 2s for typical app | Depends on Go toolchain |
| **Total (Conduit)** | **< 100ms for 1000 LOC** | Excluding Go build |
| **Total (Full)** | **< 3s for typical app** | Including Go build |

**Optimization Strategies:**
- Incremental compilation (only recompile changed files)
- Parallel parsing (multiple files simultaneously)
- Build cache (per-file hash-based invalidation)
- Watch mode optimization (keep AST in memory)

---

### Runtime Performance

| Metric | Target | Notes |
|--------|--------|-------|
| HTTP Handler Latency | < 5ms (p50) | Excluding DB |
| Database Query | < 10ms (p50) | Simple indexed query |
| Full Request (CRUD) | < 20ms (p50) | Including all hooks |
| Throughput | 10,000+ req/s | Single server, simple endpoint |
| Memory per Request | < 10 KB | Request context allocation |
| Concurrent Connections | 10,000+ | Go goroutines |

**Benchmarks (Expected):**

```
# Simple GET /posts/123
Requests/sec: 15,000
Latency p50: 4ms
Latency p99: 15ms

# POST /posts (create with hooks)
Requests/sec: 5,000
Latency p50: 18ms
Latency p99: 50ms

# GET /posts (list with pagination)
Requests/sec: 8,000
Latency p50: 10ms
Latency p99: 30ms
```

---

### Resource Usage

| Metric | Typical | Notes |
|--------|---------|-------|
| Binary Size | 10-30 MB | Depends on resource count |
| Memory (Startup) | 20-50 MB | Including introspection data |
| Memory (Per Request) | 10-50 KB | Request context |
| Memory (Steady State) | 100-500 MB | With connection pool, caches |
| CPU (Idle) | < 1% | Minimal background work |
| CPU (Load) | 40-60% | At 10K req/s |
| Database Connections | 10-100 | Pooled connections |

---

## Security Architecture

### 1. Compile-Time Security

**Type Safety**
- Nullability checking prevents nil dereferences
- Type mismatches caught at compile time
- Foreign key types must match

**SQL Injection Prevention**
- All queries use parameterized statements
- No string concatenation for SQL
- Query builder validates inputs

**Validation Enforcement**
- Field constraints enforced at compile time
- Validation blocks required for operations
- Custom validations type-checked

---

### 2. Runtime Security

**Authentication & Authorization**
- Middleware-based auth (JWT, session, API key)
- Per-resource authorization checks
- Context-based user access

```go
// Middleware checks authentication
@on update: [auth, owner_or_admin]

// Access current user in hooks
@before update {
    if Context.current_user!().role != "admin" {
        error("Not authorized")
    }
}
```

**Rate Limiting**
- Per-route rate limiting
- Token bucket algorithm
- Configurable limits per user/IP

```conduit
@on create: [rate_limit(5, per: "hour")]
```

**CORS Protection**
- Configurable CORS middleware
- Whitelist origins
- Credential handling

---

### 3. Database Security

**Connection Security**
- TLS connections to database
- Connection pooling with limits
- Prepared statements only

**Row-Level Security**
- Tenant isolation via filters
- Automatic scope filtering
- User-specific queries

```conduit
@scope user_owned {
    where: { user_id: Context.current_user!().id }
}
```

**Audit Logging**
- Automatic change tracking
- `created_by`, `updated_by` fields
- Revision history

---

### 4. Application Security

**Secret Management**
- Environment variable configuration
- No secrets in source code
- External secret stores (AWS Secrets Manager, Vault)

**Input Validation**
- Field constraints validated
- Type coercion strict
- Length limits enforced

**Output Encoding**
- JSON encoding automatic
- HTML escaping when rendering
- Content-Type headers set

**Error Handling**
- Production errors sanitized
- Stack traces hidden
- Detailed logs server-side only

---

## Future Evolution

### Phase 1: Current (v1.0)
- ✅ Compile-to-Go
- ✅ PostgreSQL ORM
- ✅ REST API generation
- ✅ Basic introspection
- ✅ CLI tooling
- ✅ LSP support

### Phase 2: v1.1 (6 months)
- 🔄 Advanced introspection (pattern templates)
- 🔄 Migration system automation
- 🔄 Multiple database support (MySQL, SQLite)
- 🔄 GraphQL API generation
- 🔄 Real-time subscriptions (WebSockets)
- 🔄 File upload handling

### Phase 3: v1.2 (12 months)
- 🔮 Background job system
- 🔮 Caching layer (Redis)
- 🔮 Full-text search integration
- 🔮 Email/SMS notifications
- 🔮 Multi-tenancy support
- 🔮 Admin panel generation

### Phase 4: v2.0 (18 months)
- 🔮 WebAssembly compilation (TinyGo)
- 🔮 Frontend framework (Conduit UI)
- 🔮 Mobile compilation (React Native bridge)
- 🔮 Microservices support
- 🔮 gRPC API generation
- 🔮 Event sourcing primitives

### Long-Term Vision (24+ months)
- 🔮 Visual editor for LLM collaboration
- 🔮 Distributed deployment (Kubernetes native)
- 🔮 Multi-language compilation targets (Rust, Zig)
- 🔮 LLM-assisted refactoring tools
- 🔮 Automatic performance optimization
- 🔮 Formal verification support

---

## Appendix: File Structure

### Project Layout

```
conduit-project/
├── src/                          # Conduit source files
│   ├── resources/
│   │   ├── user.cdt
│   │   ├── post.cdt
│   │   └── comment.cdt
│   ├── middleware/
│   │   ├── auth.cdt
│   │   └── rate_limit.cdt
│   └── config/
│       └── app.yaml
│
├── build/                        # Build output (gitignored)
│   ├── app                       # Compiled binary
│   ├── app.meta.json            # Introspection metadata
│   ├── app.map                  # Source maps
│   └── generated/               # Generated Go code
│       ├── resources/
│       │   ├── user.go
│       │   ├── post.go
│       │   └── comment.go
│       ├── handlers/
│       │   └── rest.go
│       └── main.go
│
├── migrations/                   # Database migrations
│   ├── 001_create_users.sql
│   ├── 002_create_posts.sql
│   └── 003_create_comments.sql
│
├── tests/                        # Test files
│   ├── user_test.cdt
│   └── post_test.cdt
│
├── docs/                         # Generated documentation
│   ├── api.md
│   └── resources.md
│
├── conduit.yaml                  # Project configuration
└── README.md
```

### Configuration File (conduit.yaml)

```yaml
project:
  name: blog
  version: 1.0.0

database:
  driver: postgres
  host: localhost
  port: 5432
  name: blog_dev
  user: postgres
  password: ${DB_PASSWORD}
  pool:
    min: 5
    max: 100

server:
  host: 0.0.0.0
  port: 3000
  cors:
    origins: ["http://localhost:3001"]
    credentials: true

introspection:
  enabled: true
  auth_required: true

watch:
  enabled: true
  patterns: ["src/**/*.cdt"]
  ignored: ["build/", "node_modules/"]

build:
  output: build/app
  cache: true
  parallel: true
```

---

## Summary

Conduit's architecture is designed around three core principles:

1. **LLM-First:** Every design decision optimizes for AI code generation and understanding
2. **Progressive Disclosure:** Simple things stay simple, complexity is available when needed
3. **Compile-to-Native:** Fast compilation, single binary deployment, no runtime overhead

The system consists of five major subsystems (Compiler, Runtime, ORM, Web Framework, Tooling) that work together to provide a seamless development experience from source code to deployed application.

By compiling to Go and leveraging its ecosystem, Conduit achieves production-ready performance while maintaining the rapid development velocity that makes LLM-assisted development practical.

---

**Related Documents:**
- `LANGUAGE-SPEC.md` - Language syntax and semantics
- `IMPLEMENTATION-COMPILER.md` - Compiler implementation details
- `IMPLEMENTATION-RUNTIME.md` - Runtime system details
- `IMPLEMENTATION-ORM.md` - ORM implementation
- `IMPLEMENTATION-WEB.md` - Web framework details
- `IMPLEMENTATION-TOOLING.md` - Developer tooling
- `GETTING-STARTED.md` - Quick start guide

**Status:** Complete
**Next:** Create GETTING-STARTED.md for onboarding
