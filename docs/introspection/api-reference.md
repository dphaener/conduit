# Go API Reference

Complete reference for programmatic introspection access using the Go API.

## Table of Contents

- [Package Overview](#package-overview)
- [Getting Started](#getting-started)
- [RegistryAPI](#registryapi)
- [Query Functions](#query-functions)
- [Data Structures](#data-structures)
- [Performance](#performance)
- [Error Handling](#error-handling)
- [Thread Safety](#thread-safety)

## Package Overview

**Import path**: `github.com/conduit-lang/conduit/runtime/metadata`

The `metadata` package provides an ergonomic public API for runtime introspection. It exposes complete information about compiled Conduit applications through a simple, type-safe interface.

### Key Features

- **Fast queries**: Sub-microsecond for simple queries, <10µs for complex queries
- **Zero-overhead**: Compile-time metadata generation, no runtime reflection
- **Type-safe**: Strongly typed API with explicit error handling
- **Thread-safe**: All methods are safe for concurrent access
- **Cached**: Automatic caching for complex queries (LRU with 1000 entry limit)

### Package Documentation

The package defines the schema for the introspection system. All metadata is:
- JSON-serializable
- Optimized for size (<2KB per resource uncompressed, ~700 bytes compressed)
- Immutable at runtime (defensive copies provided)
- Pre-indexed for fast lookups

## Getting Started

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    // Get the registry singleton
    registry := metadata.GetRegistry()

    // Query all resources
    resources := registry.Resources()
    fmt.Printf("Found %d resources\n", len(resources))

    // Query specific resource
    post, err := registry.Resource("Post")
    if err != nil {
        log.Fatalf("Resource not found: %v", err)
    }

    fmt.Printf("Post has %d fields\n", len(post.Fields))
}
```

### Installation

The metadata package is part of the Conduit runtime. Include it in your application:

```go
import "github.com/conduit-lang/conduit/runtime/metadata"
```

## RegistryAPI

### GetRegistry

```go
func GetRegistry() *RegistryAPI
```

Returns the global registry singleton. This is the primary entry point for runtime introspection.

The registry is initialized at application startup via `RegisterMetadata` and provides fast indexed access to metadata with sub-millisecond query times.

**Example**:

```go
registry := metadata.GetRegistry()
```

**Performance**: O(1), sub-microsecond

**Thread safety**: Safe for concurrent access

---

### Resources

```go
func (r *RegistryAPI) Resources() []ResourceMetadata
```

Returns all registered resources.

Returns a copy of the resource metadata to prevent external mutation. This is a fast operation (<1ms) that reads from pre-computed indexes.

**Example**:

```go
registry := metadata.GetRegistry()
resources := registry.Resources()
for _, res := range resources {
    fmt.Printf("Resource: %s (%d fields)\n", res.Name, len(res.Fields))
}
```

**Returns**: Slice of `ResourceMetadata` (defensive copy)

**Performance**: O(n) where n = number of resources, typically <1ms

**Thread safety**: Safe for concurrent access

---

### Resource

```go
func (r *RegistryAPI) Resource(name string) (*ResourceMetadata, error)
```

Returns metadata for a single resource by name.

This is an O(1) lookup using pre-computed indexes. Returns an error if the resource is not found or if the registry is not initialized.

**Parameters**:
- `name` (string): Resource name (e.g., "Post", "User")

**Returns**:
- `*ResourceMetadata`: Resource metadata (pointer to defensive copy)
- `error`: Error if resource not found or registry not initialized

**Errors**:
- `"resource not found: <name>"`: Resource doesn't exist
- `"registry not initialized"`: Registry hasn't been loaded

**Example**:

```go
post, err := registry.Resource("Post")
if err != nil {
    log.Fatalf("Resource not found: %v", err)
}
fmt.Printf("Post has %d fields\n", len(post.Fields))
fmt.Printf("Post file: %s\n", post.FilePath)
```

**Performance**: O(1) hash map lookup, sub-microsecond

**Thread safety**: Safe for concurrent access

---

### Routes

```go
func (r *RegistryAPI) Routes(filter RouteFilter) []RouteMetadata
```

Returns routes filtered by the provided criteria.

If filter is empty (all fields are empty strings), returns all routes. Multiple filter criteria are combined with AND logic.

**Parameters**:
- `filter` (RouteFilter): Filter criteria (see [RouteFilter](#routefilter))

**Returns**: Slice of `RouteMetadata`

**Performance**:
- Method filtering: O(1) lookup
- Path filtering: O(1) lookup
- Resource filtering: O(n) scan
- No filtering: O(1)

**Thread safety**: Safe for concurrent access

**Example**:

```go
// Get all routes
allRoutes := registry.Routes(metadata.RouteFilter{})

// Get all GET routes
getRoutes := registry.Routes(metadata.RouteFilter{
    Method: "GET",
})

// Get all routes for Post resource
postRoutes := registry.Routes(metadata.RouteFilter{
    Resource: "Post",
})

// Combine filters
authPostRoutes := registry.Routes(metadata.RouteFilter{
    Method:   "GET",
    Resource: "Post",
})
```

---

### Patterns

```go
func (r *RegistryAPI) Patterns(category string) []PatternMetadata
```

Returns patterns filtered by category.

If category is an empty string, returns all patterns. Category matching is case-sensitive and exact.

**Parameters**:
- `category` (string): Pattern category (e.g., "hook", "authentication", "")

**Common categories**:
- `"authentication"` - Auth patterns
- `"caching"` - Cache patterns
- `"rate_limiting"` - Rate limit patterns
- `"hook"` - Lifecycle hook patterns
- `"validation"` - Validation patterns
- `""` - All patterns

**Returns**: Slice of `PatternMetadata`

**Performance**: O(n) where n = number of patterns

**Thread safety**: Safe for concurrent access

**Example**:

```go
// Get all hook patterns
hookPatterns := registry.Patterns("hook")
for _, p := range hookPatterns {
    fmt.Printf("Pattern: %s (used %d times)\n", p.Name, p.Frequency)
    fmt.Printf("Template: %s\n", p.Template)
}

// Get all patterns
allPatterns := registry.Patterns("")
```

---

### Dependencies

```go
func (r *RegistryAPI) Dependencies(resource string, opts DependencyOptions) (*DependencyGraph, error)
```

Returns a dependency graph starting from the specified resource.

The graph includes all nodes and edges reachable from the starting resource according to the provided options. Results are cached for performance since metadata never changes at runtime.

**Parameters**:
- `resource` (string): Starting resource name
- `opts` (DependencyOptions): Query options (see [DependencyOptions](#dependencyoptions))

**Returns**:
- `*DependencyGraph`: Dependency graph
- `error`: Error if resource not found or registry not initialized

**Errors**:
- `"resource not found: <name>"`: Resource doesn't exist
- `"registry not initialized"`: Registry hasn't been loaded

**Performance**:
- Cold cache: ~8µs for depth 3 traversal
- Warm cache: ~112ns (70x speedup)
- Automatic LRU caching with 1000 entry limit

**Thread safety**: Safe for concurrent access (cached results are immutable)

**Example**:

```go
// Get all forward dependencies (what Post depends on)
deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth:   0, // Unlimited
    Reverse: false,
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Post depends on %d resources\n", len(deps.Nodes)-1)

// Get reverse dependencies (what depends on User)
reverseDeps, err := registry.Dependencies("User", metadata.DependencyOptions{
    Depth:   0,
    Reverse: true,
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d resources depend on User\n", len(reverseDeps.Nodes)-1)

// Get direct dependencies only (depth 1)
directDeps, err := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth:   1,
    Reverse: false,
})
```

---

### GetSchema

```go
func (r *RegistryAPI) GetSchema() *Metadata
```

Returns the complete metadata schema.

This returns the entire `Metadata` structure containing all resources, routes, patterns, and the full dependency graph. Returns nil if the registry has not been initialized.

Use this when you need the complete schema for serialization or comprehensive analysis. For targeted queries, prefer the specific query methods (Resources, Resource, Routes, etc.) which are faster.

**Returns**: `*Metadata` or `nil` if not initialized

**Performance**: O(1), sub-microsecond

**Thread safety**: Safe for concurrent access (returns reference to immutable data)

**Example**:

```go
schema := registry.GetSchema()
if schema == nil {
    log.Fatal("Registry not initialized")
}

fmt.Printf("Schema version: %s\n", schema.Version)
fmt.Printf("Generated: %s\n", schema.Generated)
fmt.Printf("Total resources: %d\n", len(schema.Resources))
fmt.Printf("Total routes: %d\n", len(schema.Routes))
fmt.Printf("Total patterns: %d\n", len(schema.Patterns))

// Serialize to JSON
jsonData, err := json.Marshal(schema)
if err != nil {
    log.Fatal(err)
}
```

## Query Functions

In addition to the `RegistryAPI` methods, the package provides standalone query functions:

### QueryResources

```go
func QueryResources() []ResourceMetadata
```

Returns all resources. Equivalent to `registry.Resources()`.

### QueryResource

```go
func QueryResource(name string) (*ResourceMetadata, error)
```

Returns a specific resource. Equivalent to `registry.Resource(name)`.

### QueryRoutes

```go
func QueryRoutes() []RouteMetadata
```

Returns all routes.

### QueryRoutesByMethod

```go
func QueryRoutesByMethod(method string) []RouteMetadata
```

Returns routes filtered by HTTP method.

**Parameters**:
- `method` (string): HTTP method (GET, POST, PUT, DELETE, etc.)

**Performance**: O(1) using pre-computed index

### QueryRoutesByPath

```go
func QueryRoutesByPath(path string) []RouteMetadata
```

Returns routes filtered by exact path match.

**Parameters**:
- `path` (string): Exact path (e.g., "/posts/:id")

**Performance**: O(1) using pre-computed index

### QueryPatterns

```go
func QueryPatterns() []PatternMetadata
```

Returns all discovered patterns.

### QueryDependencies

```go
func QueryDependencies(resourceName string, opts DependencyOptions) (*DependencyGraph, error)
```

Returns dependency graph for a resource. Equivalent to `registry.Dependencies(resourceName, opts)`.

### QueryRelationshipsFrom

```go
func QueryRelationshipsFrom(resourceName string) ([]RelationshipMetadata, error)
```

Returns all relationships from a resource (what it depends on).

**Parameters**:
- `resourceName` (string): Resource name

**Returns**: Slice of relationships

### QueryRelationshipsTo

```go
func QueryRelationshipsTo(resourceName string) []RelationshipMetadata
```

Returns all relationships to a resource (what depends on it).

**Parameters**:
- `resourceName` (string): Resource name

**Returns**: Slice of relationships

### GetMetadata

```go
func GetMetadata() *Metadata
```

Returns the complete metadata structure. Equivalent to `registry.GetSchema()`.

## Data Structures

### RouteFilter

```go
type RouteFilter struct {
    Method   string // Optional: filter by HTTP method (GET, POST, etc.)
    Path     string // Optional: filter by exact path pattern
    Resource string // Optional: filter by resource name
}
```

Provides optional filters for route queries. All fields are optional - empty string means no filtering on that field.

**Example**:

```go
// Filter by HTTP method
routes := registry.Routes(metadata.RouteFilter{Method: "GET"})

// Filter by resource
routes := registry.Routes(metadata.RouteFilter{Resource: "Post"})

// Combine filters
routes := registry.Routes(metadata.RouteFilter{
    Method:   "GET",
    Resource: "Post",
})

// Get all routes (no filtering)
routes := registry.Routes(metadata.RouteFilter{})
```

---

### DependencyOptions

```go
type DependencyOptions struct {
    Depth   int      // Maximum traversal depth (0 = unlimited)
    Reverse bool     // If true, finds what depends on this resource
    Types   []string // Filter edges by relationship type
}
```

Configures dependency graph queries.

**Fields**:
- `Depth`: Maximum traversal depth
  - `0`: Unlimited (traverse entire graph)
  - `1`: Direct dependencies only
  - `2-5`: Multi-level traversal (max 5)
- `Reverse`: Direction of traversal
  - `false`: Forward (what resource uses)
  - `true`: Reverse (what uses resource)
- `Types`: Filter by edge relationship type
  - `[]string{"belongs_to", "has_many"}` for resource relationships
  - `[]string{"uses"}` for middleware
  - `[]string{"calls"}` for function calls
  - `nil` or empty for all types

**Example**:

```go
// All forward dependencies
opts := metadata.DependencyOptions{
    Depth:   0,
    Reverse: false,
}

// Direct reverse dependencies only
opts := metadata.DependencyOptions{
    Depth:   1,
    Reverse: true,
}

// Only resource relationships, depth 2
opts := metadata.DependencyOptions{
    Depth:   2,
    Reverse: false,
    Types:   []string{"belongs_to", "has_many", "has_many_through"},
}
```

---

### ResourceMetadata

```go
type ResourceMetadata struct {
    Name           string
    Documentation  string
    FilePath       string
    Fields         []FieldMetadata
    Relationships  []RelationshipMetadata
    Hooks          []HookMetadata
    Validations    []ValidationMetadata
    Constraints    []ConstraintMetadata
    Middleware     map[string][]string
    Scopes         []ScopeMetadata
    ComputedFields []ComputedFieldMetadata
}
```

Complete metadata about a Conduit resource.

**Example**:

```go
post, _ := registry.Resource("Post")

fmt.Printf("Name: %s\n", post.Name)
fmt.Printf("File: %s\n", post.FilePath)
fmt.Printf("Docs: %s\n", post.Documentation)
fmt.Printf("Fields: %d\n", len(post.Fields))
fmt.Printf("Relationships: %d\n", len(post.Relationships))
fmt.Printf("Hooks: %d\n", len(post.Hooks))

// Iterate fields
for _, field := range post.Fields {
    fmt.Printf("  %s: %s (required: %v)\n",
        field.Name, field.Type, field.Required)
}

// Check middleware
if middleware, ok := post.Middleware["create"]; ok {
    fmt.Printf("Create middleware: %v\n", middleware)
}
```

---

### FieldMetadata

```go
type FieldMetadata struct {
    Name          string
    Type          string
    Nullable      bool
    Required      bool
    DefaultValue  string
    Constraints   []string
    Documentation string
    Tags          []string
}
```

Metadata about a single field in a resource.

**Example**:

```go
for _, field := range post.Fields {
    fmt.Printf("Field: %s\n", field.Name)
    fmt.Printf("  Type: %s\n", field.Type)
    fmt.Printf("  Required: %v\n", field.Required)
    fmt.Printf("  Nullable: %v\n", field.Nullable)

    if field.DefaultValue != "" {
        fmt.Printf("  Default: %s\n", field.DefaultValue)
    }

    if len(field.Constraints) > 0 {
        fmt.Printf("  Constraints: %v\n", field.Constraints)
    }
}
```

---

### RelationshipMetadata

```go
type RelationshipMetadata struct {
    Name           string
    Type           string // "belongs_to", "has_many", "has_many_through"
    TargetResource string
    ForeignKey     string
    ThroughTable   string
    OnDelete       string // "cascade", "restrict", "set_null"
    OnUpdate       string
}
```

Metadata about relationships between resources.

**Example**:

```go
for _, rel := range post.Relationships {
    fmt.Printf("Relationship: %s\n", rel.Name)
    fmt.Printf("  Type: %s\n", rel.Type)
    fmt.Printf("  Target: %s\n", rel.TargetResource)
    fmt.Printf("  On delete: %s\n", rel.OnDelete)
}
```

---

### RouteMetadata

```go
type RouteMetadata struct {
    Method       string
    Path         string
    Handler      string
    Resource     string
    Operation    string
    Middleware   []string
    RequestBody  string
    ResponseBody string
}
```

Information about auto-generated HTTP routes.

**Example**:

```go
routes := registry.Routes(metadata.RouteFilter{Resource: "Post"})
for _, route := range routes {
    fmt.Printf("%s %s -> %s\n", route.Method, route.Path, route.Handler)
    fmt.Printf("  Operation: %s\n", route.Operation)
    fmt.Printf("  Middleware: %v\n", route.Middleware)
}
```

---

### PatternMetadata

```go
type PatternMetadata struct {
    ID          string
    Name        string
    Category    string
    Description string
    Template    string
    Examples    []PatternExample
    Frequency   int
    Confidence  float64
}
```

Discovered usage patterns for LLM learning.

**Example**:

```go
patterns := registry.Patterns("authentication")
for _, pattern := range patterns {
    fmt.Printf("Pattern: %s\n", pattern.Name)
    fmt.Printf("  Category: %s\n", pattern.Category)
    fmt.Printf("  Frequency: %d\n", pattern.Frequency)
    fmt.Printf("  Confidence: %.2f\n", pattern.Confidence)
    fmt.Printf("  Template: %s\n", pattern.Template)

    for i, example := range pattern.Examples {
        if i >= 3 break // Limit to 3 examples
        fmt.Printf("  Example: %s\n", example.Resource)
    }
}
```

---

### DependencyGraph

```go
type DependencyGraph struct {
    Nodes map[string]*DependencyNode
    Edges []DependencyEdge
}
```

Dependency relationships between resources.

**Example**:

```go
deps, _ := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 2,
})

fmt.Printf("Nodes: %d\n", len(deps.Nodes))
fmt.Printf("Edges: %d\n", len(deps.Edges))

// Iterate nodes
for id, node := range deps.Nodes {
    fmt.Printf("Node: %s (type: %s)\n", node.Name, node.Type)
}

// Iterate edges
for _, edge := range deps.Edges {
    fromNode := deps.Nodes[edge.From]
    toNode := deps.Nodes[edge.To]
    fmt.Printf("%s -> %s (%s)\n",
        fromNode.Name, toNode.Name, edge.Relationship)
}
```

## Performance

### Query Performance

| Operation | Cold | Warm (Cached) | Complexity |
|-----------|------|---------------|------------|
| Registry init | 0.34ms | N/A | O(n) |
| Resources() | <1µs | N/A | O(n) |
| Resource(name) | <1µs | N/A | O(1) |
| Routes() | <1µs | N/A | O(n) |
| Routes(filter) | <1µs | N/A | O(1) or O(n) |
| Patterns() | <1µs | N/A | O(n) |
| Dependencies(depth 1) | ~3µs | ~100ns | O(e) |
| Dependencies(depth 3) | ~8µs | ~112ns | O(e²) |
| GetSchema() | <1µs | N/A | O(1) |

Where:
- n = number of resources, routes, or patterns
- e = number of edges in dependency graph

### Memory Footprint

- Registry initialization: ~207KB for 50 resources
- Per resource: ~4KB average
- Defensive copies: <2KB per query
- Cache overhead: ~40 bytes per cached entry

### Caching

Complex queries (dependency traversal) are automatically cached:
- **LRU eviction**: 1000 entry limit
- **Cache key**: Includes resource name, options, depth
- **Speedup**: 70x faster for warm cache vs cold
- **Thread-safe**: Concurrent cache access is safe

### Scaling Characteristics

- **100 resources**: ~1MB memory, <1ms init
- **500 resources**: ~5MB memory, <3ms init
- **1000 resources**: ~10MB memory, <5ms init

Performance scales linearly with schema size.

## Error Handling

### Error Types

The API uses standard Go error values:

```go
// Resource not found
_, err := registry.Resource("NonExistent")
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle not found
    }
}

// Registry not initialized
_, err := registry.Resource("Post")
if err != nil {
    if strings.Contains(err.Error(), "not initialized") {
        // Registry hasn't been loaded yet
    }
}
```

### Best Practices

1. **Always check errors** for Resource() and Dependencies()
2. **Check nil** for GetSchema() return value
3. **Use defensive programming** when accessing slices/maps
4. **Log errors** for debugging

**Example**:

```go
func getResourceInfo(name string) error {
    registry := metadata.GetRegistry()

    resource, err := registry.Resource(name)
    if err != nil {
        return fmt.Errorf("failed to get resource %s: %w", name, err)
    }

    // Safe to use resource
    fmt.Printf("Found resource: %s\n", resource.Name)
    return nil
}
```

## Thread Safety

### Concurrent Access

All registry methods are safe for concurrent access:

```go
var wg sync.WaitGroup

// Safe: concurrent reads
for _, name := range []string{"Post", "User", "Comment"} {
    wg.Add(1)
    go func(n string) {
        defer wg.Done()
        resource, _ := registry.Resource(n)
        fmt.Printf("Got %s\n", resource.Name)
    }(name)
}
wg.Wait()
```

### Why It's Safe

- **Immutable metadata**: Never changes after initialization
- **Defensive copies**: Returns copies, not references to internal data
- **Read-only access**: No mutation methods
- **Thread-safe caching**: Internal cache uses sync.RWMutex

### Performance Notes

- No lock contention for reads (uses sync.RWMutex)
- Scales well with concurrent access
- No performance penalty for single-threaded use

## See Also

- [User Guide](user-guide.md) - Practical workflows and examples
- [CLI Reference](cli-reference.md) - Command-line introspection
- [Architecture](architecture.md) - How it works internally
- [Tutorial](tutorial/01-basic-queries.md) - Step-by-step walkthrough
- [Examples](../../examples/introspection/) - Working code samples
