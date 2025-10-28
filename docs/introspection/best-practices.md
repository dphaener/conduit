# Best Practices

Guidelines for effective use of the introspection system.

## Table of Contents

- [When to Use Introspection](#when-to-use-introspection)
- [Performance Best Practices](#performance-best-practices)
- [Caching Strategies](#caching-strategies)
- [Error Handling](#error-handling)
- [Pattern Discovery](#pattern-discovery)
- [Dependency Analysis](#dependency-analysis)
- [Tooling Integration](#tooling-integration)
- [Antipatterns to Avoid](#antipatterns-to-avoid)

## When to Use Introspection

### ✅ Good Use Cases

**1. Build-Time Code Generation**

Use introspection to generate documentation, API clients, or test scaffolding:

```go
// Generate API documentation at build time
func generateAPIDocs() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    for _, res := range resources {
        generateResourceDocs(res)
    }
}
```

**2. Development Tools**

Build CLI tools, IDE plugins, or debuggers:

```bash
# CLI tool to explore schema
conduit introspect resources
conduit introspect resource Post
```

**3. LLM Context Gathering**

Query patterns before code generation:

```go
// Get authentication patterns for LLM prompt
patterns := registry.Patterns("authentication")
context := fmt.Sprintf("Common auth patterns in this project:\n")
for _, p := range patterns {
    context += fmt.Sprintf("- %s: %s\n", p.Name, p.Template)
}
// Include context in LLM prompt
```

**4. Dependency Impact Analysis**

Check dependencies before refactoring:

```bash
# What breaks if I delete User?
conduit introspect deps User --reverse

# What does Post depend on?
conduit introspect deps Post
```

**5. Schema Validation**

Verify schema constraints programmatically:

```go
// Ensure all resources have auth middleware
for _, res := range registry.Resources() {
    hasAuth := false
    for _, mw := range res.Middleware["create"] {
        if strings.Contains(mw, "auth") {
            hasAuth = true
            break
        }
    }
    if !hasAuth {
        log.Printf("WARNING: %s has no auth on create\n", res.Name)
    }
}
```

### ❌ When Not to Use

**1. Hot Path Request Handling**

Don't query introspection in request handlers:

```go
// BAD: Queries on every request
func handleRequest(w http.ResponseWriter, r *http.Request) {
    resource := metadata.QueryResource("Post") // Too slow!
    // ... use resource data
}

// GOOD: Query once at startup
var postSchema *metadata.ResourceMetadata

func init() {
    postSchema = metadata.QueryResource("Post")
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Use cached schema
}
```

**2. Dynamic Schema Modification**

Metadata is immutable at runtime:

```go
// BAD: Trying to modify metadata
resource.Fields = append(resource.Fields, newField) // Won't affect registry!

// GOOD: Metadata is read-only by design
// Schema changes require recompilation
```

**3. As a Database**

Don't use introspection to store application data:

```go
// BAD: Storing data in metadata
// (metadata is for schema, not data)

// GOOD: Use a database for application data
```

## Performance Best Practices

### 1. Query Once, Use Many Times

Cache query results that won't change:

```go
// BAD: Repeated queries
for i := 0; i < 1000; i++ {
    resource := registry.Resource("Post") // Wasteful!
}

// GOOD: Query once, reuse
resource := registry.Resource("Post")
for i := 0; i < 1000; i++ {
    // Use cached resource
}
```

### 2. Use Appropriate Query Depth

Limit dependency traversal depth to what you need:

```go
// BAD: Unlimited depth when you only need depth 1
deps := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 0, // Traverses entire graph!
})

// GOOD: Use minimum necessary depth
deps := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 1, // Only direct dependencies
})
```

**Performance impact**:
- Depth 1: ~3µs
- Depth 2: ~5µs
- Depth 3: ~8µs
- Unlimited: Can be 10-100x slower

### 3. Filter Early

Apply filters at query time, not after:

```go
// BAD: Fetch all, filter later
allRoutes := registry.Routes(metadata.RouteFilter{})
var getRoutes []metadata.RouteMetadata
for _, r := range allRoutes {
    if r.Method == "GET" {
        getRoutes = append(getRoutes, r)
    }
}

// GOOD: Filter at query time
getRoutes := registry.Routes(metadata.RouteFilter{
    Method: "GET",
})
```

### 4. Use JSON Format for Large Exports

For bulk data export, use JSON format:

```bash
# BAD: Table format is slow for large outputs
conduit introspect resources > resources.txt

# GOOD: JSON format is faster
conduit introspect resources --format json > resources.json
```

### 5. Leverage Automatic Caching

Complex queries are automatically cached:

```go
// First call: ~8µs (cold)
deps1 := registry.Dependencies("Post", metadata.DependencyOptions{Depth: 3})

// Subsequent calls: ~112ns (warm, 70x faster!)
deps2 := registry.Dependencies("Post", metadata.DependencyOptions{Depth: 3})
```

**Cache behavior**:
- LRU eviction (1000 entries)
- Thread-safe
- Automatic, no manual management

## Caching Strategies

### Application-Level Caching

For frequently accessed data, cache at application startup:

```go
var (
    allResources []metadata.ResourceMetadata
    postSchema   *metadata.ResourceMetadata
    userSchema   *metadata.ResourceMetadata
)

func init() {
    registry := metadata.GetRegistry()

    allResources = registry.Resources()
    postSchema, _ = registry.Resource("Post")
    userSchema, _ = registry.Resource("User")
}

// Now access cached data directly
func getPostSchema() *metadata.ResourceMetadata {
    return postSchema
}
```

### When to Cache

**Always cache**:
- Resources queried in hot paths
- Pattern data for LLM prompts
- Route lists for middleware

**Don't cache**:
- One-off development queries
- CLI tool output (query each time)

### Cache Invalidation

Metadata cache is valid until recompilation:

```bash
# After code changes, rebuild to regenerate metadata
conduit build

# Old cached metadata is now stale
# Application restart loads new metadata
```

## Error Handling

### Check Errors Properly

Always check errors for fallible operations:

```go
// BAD: Ignoring errors
resource, _ := registry.Resource("Post")
// What if Post doesn't exist?

// GOOD: Proper error handling
resource, err := registry.Resource("Post")
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        log.Printf("Resource not found: Post")
        // Handle gracefully
    } else {
        log.Fatalf("Registry error: %v", err)
    }
}
```

### Handle Registry Not Initialized

Check if registry is initialized before queries:

```go
schema := registry.GetSchema()
if schema == nil {
    log.Fatal("Registry not initialized - run 'conduit build' first")
}
```

### Provide Helpful Error Messages

When building tools, provide context:

```go
resource, err := registry.Resource(name)
if err != nil {
    // BAD: Generic error
    return err

    // GOOD: Helpful context
    return fmt.Errorf("failed to get resource %s: %w\nHint: Run 'conduit introspect resources' to see available resources", name, err)
}
```

## Pattern Discovery

### Set Appropriate Frequency Thresholds

Filter patterns by frequency to focus on common patterns:

```bash
# Development: See all patterns (min 1)
conduit introspect patterns --min-frequency 1

# Production docs: Only common patterns (min 3-5)
conduit introspect patterns --min-frequency 3
```

### Use Patterns for Consistency

Reference patterns when generating code:

```go
// Get auth patterns
authPatterns := registry.Patterns("authentication")

// Find most common pattern
var mostCommon *metadata.PatternMetadata
maxFrequency := 0
for _, p := range authPatterns {
    if p.Frequency > maxFrequency {
        maxFrequency = p.Frequency
        mostCommon = &p
    }
}

// Use template for code generation
template := mostCommon.Template
// Generate code matching template
```

### Validate Generated Code

After generating code, verify it matches patterns:

```go
func validateAgainstPatterns(generatedCode string, category string) bool {
    patterns := registry.Patterns(category)

    for _, pattern := range patterns {
        if matchesPattern(generatedCode, pattern.Template) {
            return true
        }
    }

    return false
}
```

## Dependency Analysis

### Check Dependencies Before Changes

Always check reverse dependencies before modifying:

```bash
# Before deleting User resource
conduit introspect deps User --reverse

# Before changing Post schema
conduit introspect deps Post --reverse
```

### Understand Cascade Behavior

Pay attention to on_delete behavior:

```go
deps, _ := registry.Dependencies("User", metadata.DependencyOptions{
    Reverse: true,
})

for _, edge := range deps.Edges {
    if fromNode := deps.Nodes[edge.From]; fromNode != nil {
        resource, _ := registry.Resource(fromNode.Name)
        for _, rel := range resource.Relationships {
            if rel.TargetResource == "User" {
                fmt.Printf("%s -> User: on_delete=%s\n",
                    fromNode.Name, rel.OnDelete)
            }
        }
    }
}
```

**Behaviors**:
- `cascade`: Deletion propagates (high impact!)
- `restrict`: Cannot delete with dependents
- `set_null`: Nullifies foreign keys

### Detect Circular Dependencies

Check for cycles in your dependency graph:

```go
registry := metadata.GetRegistry()
resources := registry.Resources()

// Check each resource for circular dependencies
var cycles [][]string
for _, res := range resources {
    graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
        Depth:   0, // unlimited depth to detect all cycles
        Reverse: false,
    })
    if err != nil {
        continue
    }

    // Check if any edge loops back to the starting resource
    for _, edge := range graph.Edges {
        if edge.To == res.Name && edge.From != res.Name {
            cycles = append(cycles, []string{edge.From, edge.To})
        }
    }
}

if len(cycles) > 0 {
    log.Printf("WARNING: %d circular dependencies detected\n", len(cycles))
    for i, cycle := range cycles {
        log.Printf("Cycle %d: %s -> %s\n", i+1, cycle[0], cycle[1])
    }
}
```

## Tooling Integration

### Use JSON for Piping

When building tool chains, use JSON format:

```bash
# Pipe to jq for processing
conduit introspect resources --format json | \
    jq '.resources[] | select(.field_count > 10) | .name'

# Pipe to other tools
conduit introspect routes --format json | \
    python generate-docs.py
```

### Disable Color for Scripts

In scripts and automation, disable color output:

```bash
#!/bin/bash
conduit introspect resources --no-color --format json > schema.json
```

### Handle Errors in Scripts

Check exit codes in scripts:

```bash
#!/bin/bash
if ! conduit introspect resource Post --format json > post.json; then
    echo "ERROR: Failed to introspect Post resource"
    exit 1
fi
```

### Build Idempotent Tools

Tools should be idempotent (safe to run multiple times):

```go
func generateDocs() error {
    registry := metadata.GetRegistry()

    // Check if already generated
    if exists("docs/api") {
        log.Println("Docs already exist, regenerating...")
        os.RemoveAll("docs/api")
    }

    // Generate
    os.MkdirAll("docs/api", 0755)
    // ...
}
```

## Antipatterns to Avoid

### ❌ Don't Query in Hot Paths

```go
// BAD: Query on every request
func handler(w http.ResponseWriter, r *http.Request) {
    routes := registry.Routes(metadata.RouteFilter{})
    // ...
}

// GOOD: Query once at startup
var cachedRoutes []metadata.RouteMetadata
func init() {
    cachedRoutes = registry.Routes(metadata.RouteFilter{})
}
```

### ❌ Don't Ignore Errors

```go
// BAD: Ignoring errors
resource, _ := registry.Resource("Post")

// GOOD: Check errors
resource, err := registry.Resource("Post")
if err != nil {
    return err
}
```

### ❌ Don't Use Excessive Depth

```go
// BAD: Unlimited depth for no reason
deps := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 0, // Traverses entire graph!
})

// GOOD: Use minimum necessary depth
deps := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 1,
})
```

### ❌ Don't Fetch All Then Filter

```go
// BAD: Fetch all, filter later
allRoutes := registry.Routes(metadata.RouteFilter{})
for _, r := range allRoutes {
    if r.Resource == "Post" {
        // ...
    }
}

// GOOD: Filter at query time
postRoutes := registry.Routes(metadata.RouteFilter{
    Resource: "Post",
})
```

### ❌ Don't Modify Returned Data

```go
// BAD: Modifying returned metadata
resource, _ := registry.Resource("Post")
resource.Fields = append(resource.Fields, newField) // Won't work!

// GOOD: Metadata is read-only
// Create new structures if you need modifications
```

### ❌ Don't Use as a Database

```go
// BAD: Storing application data in metadata
// (metadata is for schema, not data)

// GOOD: Use metadata for schema information only
// Store application data in a database
```

### ❌ Don't Skip Registry Initialization Check

```go
// BAD: Assuming registry is initialized
resources := registry.Resources() // May return nil!

// GOOD: Check initialization
schema := registry.GetSchema()
if schema == nil {
    log.Fatal("Registry not initialized")
}
resources := registry.Resources()
```

## Summary

**Key Takeaways**:

1. ✅ Cache query results for repeated access
2. ✅ Use minimum necessary depth for dependency queries
3. ✅ Filter at query time, not after
4. ✅ Always check errors properly
5. ✅ Leverage automatic caching for complex queries
6. ✅ Check reverse dependencies before schema changes
7. ❌ Don't query in hot paths
8. ❌ Don't use excessive depth
9. ❌ Don't ignore errors
10. ❌ Don't modify returned metadata

## See Also

- [User Guide](user-guide.md) - Common workflows
- [API Reference](api-reference.md) - Complete API docs
- [Architecture](architecture.md) - How it works
- [Troubleshooting](troubleshooting.md) - Common issues
- [Tutorial](tutorial/01-basic-queries.md) - Step-by-step walkthrough
