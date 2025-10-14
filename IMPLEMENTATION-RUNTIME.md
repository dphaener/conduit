# Conduit Runtime & Introspection System Implementation Guide

**Component:** Runtime & Introspection System
**Last Updated:** 2025-10-13
**Status:** Implementation Ready
**Priority:** CRITICAL - Core Differentiator

---

## Overview

The Runtime & Introspection System is **the killer feature** that transforms Conduit from a typical programming language into an LLM-first development platform. It makes codebases fully queryable and self-documenting, enabling LLMs to discover patterns, understand dependencies, and generate code deterministically rather than probabilistically.

### Core Innovation

Traditional languages provide runtime reflection for debugging. Conduit's introspection system is designed **specifically for LLM consumption**, providing:

1. **Pattern Discovery** - Extract and query canonical patterns from existing code
2. **Architectural Queries** - Answer "How do I..." questions with concrete examples
3. **Dependency Mapping** - Understand impact of changes before making them
4. **Template Generation** - Provide scaffolding for new features

### The Problem We Solve

**Current State:** LLMs guess at patterns from training data
- 60% pattern adherence in generated code
- 3-5 iterations to get code right
- Patterns drift and become inconsistent

**With Introspection:** LLMs query exact patterns in THIS codebase
- 95%+ pattern adherence
- 1-2 iterations (self-correcting through queries)
- Patterns are enforced and verifiable

---

## Architecture

### Hybrid Approach (Recommended)

We use a **hybrid compile-time + runtime approach** for optimal performance:

```
┌─────────────────────────────────────────────────────────┐
│                   Compile Time                          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Source Code → Parser → AST → Metadata Collector       │
│                                      ↓                  │
│                            Structured Metadata          │
│                                      ↓                  │
│                            Code Generator               │
│                                      ↓                  │
│                      Go Code + Embedded Metadata        │
│                                                         │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                   Runtime                               │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Binary Execution → Load Metadata → Build Registry     │
│                                          ↓              │
│                                   Pre-compute Indexes   │
│                                          ↓              │
│                          CLI/API Queries ← Registry     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Why Hybrid?

| Approach | Pros | Cons | Performance |
|----------|------|------|-------------|
| **Pure Compile-Time** | Fast, no reflection | Metadata can be stale | ✅✅✅ Excellent |
| **Pure Runtime Reflection** | Always accurate | 5-29x slower | ❌ Poor |
| **Hybrid (Our Choice)** | Fast + accurate | More complex | ✅✅ Very Good |

Go's reflection is 5-29x slower than direct access. The hybrid approach gives us:
- Fast queries for structural data (resources, fields, types)
- Accurate data (no stale metadata files)
- Runtime flexibility for dynamic queries when needed

---

## Component 1: Metadata Collection (Compile-Time)

### Responsibility

During compilation, extract all introspectable information from the AST and serialize it for runtime use.

### Metadata Schema

```go
package metadata

type Metadata struct {
    Version      string              `json:"version"`
    Generated    time.Time           `json:"generated"`
    SourceHash   string              `json:"source_hash"`
    Resources    []ResourceMetadata  `json:"resources"`
    Routes       []RouteMetadata     `json:"routes"`
    Patterns     []PatternMetadata   `json:"patterns"`
    Dependencies DependencyGraph     `json:"dependencies"`
}

type ResourceMetadata struct {
    Name          string                   `json:"name"`
    Documentation string                   `json:"documentation"`
    FilePath      string                   `json:"file_path"`
    Fields        []FieldMetadata          `json:"fields"`
    Relationships []RelationshipMetadata   `json:"relationships"`
    Hooks         []HookMetadata           `json:"hooks"`
    Validations   []ValidationMetadata     `json:"validations"`
    Constraints   []ConstraintMetadata     `json:"constraints"`
    Middleware    map[string][]string      `json:"middleware"`
    Scopes        []ScopeMetadata          `json:"scopes"`
}

type FieldMetadata struct {
    Name          string   `json:"name"`
    Type          string   `json:"type"`
    Nullable      bool     `json:"nullable"`
    Required      bool     `json:"required"`
    DefaultValue  string   `json:"default_value,omitempty"`
    Constraints   []string `json:"constraints,omitempty"`
    Documentation string   `json:"documentation,omitempty"`
}

type RelationshipMetadata struct {
    Name           string `json:"name"`
    Type           string `json:"type"` // "belongs_to", "has_many", "has_many_through"
    TargetResource string `json:"target_resource"`
    ForeignKey     string `json:"foreign_key,omitempty"`
    OnDelete       string `json:"on_delete,omitempty"`
}

type HookMetadata struct {
    Type        string `json:"type"` // "before_create", "after_update", etc.
    Transaction bool   `json:"transaction"`
    Async       bool   `json:"async"`
    SourceCode  string `json:"source_code,omitempty"`
    LineNumber  int    `json:"line_number"`
}

type PatternMetadata struct {
    ID          string           `json:"id"`
    Name        string           `json:"name"`
    Category    string           `json:"category"`
    Description string           `json:"description"`
    Template    string           `json:"template"`
    Examples    []PatternExample `json:"examples"`
    Frequency   int              `json:"frequency"`
    Confidence  float64          `json:"confidence"`
}

type DependencyGraph struct {
    Nodes map[string]*DependencyNode `json:"nodes"`
    Edges []DependencyEdge           `json:"edges"`
}

type DependencyNode struct {
    ID       string `json:"id"`
    Type     string `json:"type"` // "resource", "function", "middleware"
    Name     string `json:"name"`
    FilePath string `json:"file_path"`
}

type DependencyEdge struct {
    From         string `json:"from"`
    To           string `json:"to"`
    Relationship string `json:"relationship"` // "uses", "calls", "belongs_to"
    Weight       int    `json:"weight"`
}
```

### Collection Implementation

```go
package compiler

type MetadataCollector struct {
    metadata *Metadata
    resources map[string]*ResourceMetadata
    functions map[string]*FunctionMetadata
}

func NewMetadataCollector() *MetadataCollector {
    return &MetadataCollector{
        metadata: &Metadata{
            Version: "1.0",
            Generated: time.Now(),
        },
        resources: make(map[string]*ResourceMetadata),
        functions: make(map[string]*FunctionMetadata),
    }
}

func (mc *MetadataCollector) Collect(ast *AST) (*Metadata, error) {
    // Step 1: Collect resources
    for _, resource := range ast.Resources {
        rm := mc.collectResource(resource)
        mc.metadata.Resources = append(mc.metadata.Resources, rm)
        mc.resources[rm.Name] = &rm
    }

    // Step 2: Build dependency graph
    mc.buildDependencyGraph()

    // Step 3: Extract patterns
    mc.extractPatterns()

    // Step 4: Generate routes
    mc.generateRoutes()

    // Step 5: Compute source hash
    mc.metadata.SourceHash = mc.computeSourceHash(ast)

    return mc.metadata, nil
}

func (mc *MetadataCollector) collectResource(node *ResourceNode) ResourceMetadata {
    rm := ResourceMetadata{
        Name:          node.Name,
        Documentation: node.Documentation,
        FilePath:      node.Location.File,
        Middleware:    make(map[string][]string),
    }

    // Collect fields
    for _, field := range node.Fields {
        rm.Fields = append(rm.Fields, mc.collectField(field))
    }

    // Collect relationships
    for _, rel := range node.Relationships {
        rm.Relationships = append(rm.Relationships, mc.collectRelationship(rel))
    }

    // Collect hooks
    for _, hook := range node.Hooks {
        rm.Hooks = append(rm.Hooks, mc.collectHook(hook))
    }

    // Collect middleware (from CRUD annotations)
    mc.collectMiddleware(node, &rm)

    return rm
}

func (mc *MetadataCollector) collectField(node *FieldNode) FieldMetadata {
    return FieldMetadata{
        Name:          node.Name,
        Type:          node.Type.String(),
        Nullable:      node.Nullable,
        Required:      !node.Nullable,
        DefaultValue:  mc.exprToString(node.Default),
        Constraints:   mc.collectConstraints(node),
        Documentation: node.Documentation,
    }
}

func (mc *MetadataCollector) collectHook(node *HookNode) HookMetadata {
    return HookMetadata{
        Type:        fmt.Sprintf("%s_%s", node.Timing, node.Event),
        Transaction: node.IsTransaction,
        Async:       node.IsAsync,
        SourceCode:  mc.statementsToString(node.Body),
        LineNumber:  node.Location.Line,
    }
}
```

---

## Component 2: Metadata Embedding

### Responsibility

Embed collected metadata into the generated Go binary for runtime access.

### Embedding Strategy

```go
package compiler

func (cg *CodeGenerator) EmbedMetadata(metadata *Metadata) string {
    // Compress metadata to reduce binary size
    jsonData, _ := json.Marshal(metadata)
    compressed := compress(jsonData)

    // Generate Go code with embedded metadata
    code := `
package main

import (
    "compress/gzip"
    "encoding/json"
    "github.com/conduit-lang/conduit/runtime"
)

// Embedded metadata (generated at compile time)
var embeddedMetadata = []byte{` + bytesToGoArray(compressed) + `}

func init() {
    // Decompress and register metadata
    metadata := decompressMetadata(embeddedMetadata)
    runtime.RegisterMetadata(metadata)
}

func decompressMetadata(data []byte) *runtime.Metadata {
    reader, _ := gzip.NewReader(bytes.NewReader(data))
    jsonData, _ := io.ReadAll(reader)

    var metadata runtime.Metadata
    json.Unmarshal(jsonData, &metadata)

    return &metadata
}
`
    return code
}

func compress(data []byte) []byte {
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    writer.Write(data)
    writer.Close()
    return buf.Bytes()
}

func bytesToGoArray(data []byte) string {
    parts := []string{}
    for _, b := range data {
        parts = append(parts, fmt.Sprintf("0x%02x", b))
    }
    return strings.Join(parts, ", ")
}
```

### Size Optimization

| Format | Size per Resource | 50 Resources |
|--------|------------------|--------------|
| Raw JSON | ~10 KB | ~500 KB |
| Compressed JSON | ~2-3 KB | ~100-150 KB |
| Binary (Protocol Buffers) | ~1-2 KB | ~50-100 KB |

**Recommendation:** Use compressed JSON for MVP (simple, debuggable). Consider binary format later if size becomes an issue.

---

## Component 3: Runtime Registry

### Responsibility

Load embedded metadata at startup and provide fast query APIs.

### Registry Implementation

```go
package runtime

var globalRegistry *Registry

type Registry struct {
    metadata *Metadata

    // Pre-computed indexes for fast queries
    resourcesByName map[string]*ResourceMetadata
    routesByPath    map[string][]*RouteMetadata
    patternsByCategory map[string][]*PatternMetadata
    dependencyGraph *DependencyGraph

    // Cache
    cache       map[string]interface{}
    cacheMutex  sync.RWMutex

    // Lazy initialization
    initialized atomic.Bool
    initMutex   sync.Mutex
}

func init() {
    globalRegistry = &Registry{
        resourcesByName: make(map[string]*ResourceMetadata),
        routesByPath: make(map[string][]*RouteMetadata),
        patternsByCategory: make(map[string][]*PatternMetadata),
        cache: make(map[string]interface{}),
    }
}

func RegisterMetadata(metadata *Metadata) {
    globalRegistry.metadata = metadata
    globalRegistry.buildIndexes()
}

func GetRegistry() *Registry {
    return globalRegistry
}

func (r *Registry) buildIndexes() {
    // Build resource index
    for i := range r.metadata.Resources {
        res := &r.metadata.Resources[i]
        r.resourcesByName[res.Name] = res
    }

    // Build route index
    for i := range r.metadata.Routes {
        route := &r.metadata.Routes[i]
        r.routesByPath[route.Path] = append(r.routesByPath[route.Path], route)
    }

    // Build pattern index
    for i := range r.metadata.Patterns {
        pattern := &r.metadata.Patterns[i]
        r.patternsByCategory[pattern.Category] = append(
            r.patternsByCategory[pattern.Category],
            pattern,
        )
    }

    r.initialized.Store(true)
}

// Query APIs
func (r *Registry) Resources() []ResourceMetadata {
    r.ensureInitialized()
    return r.metadata.Resources
}

func (r *Registry) Resource(name string) (*ResourceMetadata, error) {
    r.ensureInitialized()

    if resource, ok := r.resourcesByName[name]; ok {
        return resource, nil
    }

    return nil, fmt.Errorf("resource not found: %s", name)
}

func (r *Registry) Routes(filter RouteFilter) []RouteMetadata {
    r.ensureInitialized()

    routes := []RouteMetadata{}
    for _, route := range r.metadata.Routes {
        if filter.Matches(route) {
            routes = append(routes, *route)
        }
    }

    return routes
}

func (r *Registry) Patterns(category string) []PatternMetadata {
    r.ensureInitialized()

    if category == "" {
        return r.metadata.Patterns
    }

    if patterns, ok := r.patternsByCategory[category]; ok {
        result := make([]PatternMetadata, len(patterns))
        for i, p := range patterns {
            result[i] = *p
        }
        return result
    }

    return []PatternMetadata{}
}

func (r *Registry) Dependencies(resource string, opts DependencyOptions) (*DependencyGraph, error) {
    r.ensureInitialized()

    // Use cached if available
    cacheKey := fmt.Sprintf("deps:%s:%d:%v", resource, opts.Depth, opts.Reverse)
    if cached := r.getCache(cacheKey); cached != nil {
        return cached.(*DependencyGraph), nil
    }

    // Query dependency graph
    deps := r.queryDependencies(resource, opts)

    // Cache result
    r.setCache(cacheKey, deps)

    return deps, nil
}

func (r *Registry) ensureInitialized() {
    if r.initialized.Load() {
        return
    }

    r.initMutex.Lock()
    defer r.initMutex.Unlock()

    if r.initialized.Load() {
        return
    }

    r.buildIndexes()
}

func (r *Registry) getCache(key string) interface{} {
    r.cacheMutex.RLock()
    defer r.cacheMutex.RUnlock()
    return r.cache[key]
}

func (r *Registry) setCache(key string, value interface{}) {
    r.cacheMutex.Lock()
    defer r.cacheMutex.Unlock()
    r.cache[key] = value
}
```

---

## Component 4: Pattern Discovery

### Responsibility

Automatically extract common patterns from codebase metadata.

### Pattern Extraction Algorithms

#### Algorithm 1: Extract Middleware Patterns

```go
package analysis

type PatternExtractor struct {
    minFrequency int  // Minimum occurrences to be considered a pattern
}

func NewPatternExtractor() *PatternExtractor {
    return &PatternExtractor{
        minFrequency: 3, // Pattern must appear at least 3 times
    }
}

func (pe *PatternExtractor) ExtractMiddlewarePatterns(resources []ResourceMetadata) []PatternMetadata {
    // Step 1: Collect all middleware chains
    chains := make(map[string]*middlewareChain)

    for _, resource := range resources {
        for operation, middleware := range resource.Middleware {
            // Create canonical key for this middleware chain
            key := strings.Join(middleware, "|")

            if _, exists := chains[key]; !exists {
                chains[key] = &middlewareChain{
                    middleware: middleware,
                    usages: []patternUsage{},
                }
            }

            chains[key].usages = append(chains[key].usages, patternUsage{
                resource:  resource.Name,
                operation: operation,
                filePath:  resource.FilePath,
            })
        }
    }

    // Step 2: Filter by frequency
    patterns := []PatternMetadata{}

    for _, chain := range chains {
        if len(chain.usages) >= pe.minFrequency {
            pattern := pe.generateMiddlewarePattern(chain)
            patterns = append(patterns, pattern)
        }
    }

    // Step 3: Sort by frequency (most common first)
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].Frequency > patterns[j].Frequency
    })

    return patterns
}

func (pe *PatternExtractor) generateMiddlewarePattern(chain *middlewareChain) PatternMetadata {
    // Generate pattern name
    name := pe.generatePatternName(chain.middleware)

    // Generate template
    template := fmt.Sprintf("@on <operation>: [%s]", strings.Join(chain.middleware, ", "))

    // Generate examples
    examples := []PatternExample{}
    for _, usage := range chain.usages {
        examples = append(examples, PatternExample{
            Resource:   usage.resource,
            FilePath:   usage.filePath,
            LineNumber: 0, // TODO: track line numbers
            Code:       fmt.Sprintf("@on %s: [%s]", usage.operation, strings.Join(chain.middleware, ", ")),
        })
    }

    // Infer category
    category := pe.inferCategory(chain.middleware)

    return PatternMetadata{
        ID:          uuid.New().String(),
        Name:        name,
        Category:    category,
        Description: fmt.Sprintf("Handler with %s middleware", strings.Join(chain.middleware, " + ")),
        Template:    template,
        Examples:    examples,
        Frequency:   len(chain.usages),
        Confidence:  pe.calculateConfidence(len(chain.usages)),
    }
}

func (pe *PatternExtractor) generatePatternName(middleware []string) string {
    parts := []string{}

    for _, m := range middleware {
        // Extract base name (ignore parameters)
        baseName := strings.Split(m, "(")[0]

        switch {
        case strings.Contains(baseName, "auth"):
            parts = append(parts, "authenticated")
        case strings.Contains(baseName, "cache"):
            parts = append(parts, "cached")
        case strings.Contains(baseName, "rate_limit"):
            parts = append(parts, "rate_limited")
        // Add more pattern recognition...
        }
    }

    parts = append(parts, "handler")
    return strings.Join(parts, "_")
}

func (pe *PatternExtractor) inferCategory(middleware []string) string {
    for _, m := range middleware {
        if strings.Contains(m, "auth") {
            return "authentication"
        }
        if strings.Contains(m, "cache") {
            return "caching"
        }
        if strings.Contains(m, "rate_limit") {
            return "rate_limiting"
        }
    }
    return "general"
}

func (pe *PatternExtractor) calculateConfidence(frequency int) float64 {
    // Confidence increases with frequency, capped at 1.0
    confidence := float64(frequency) / 10.0
    if confidence > 1.0 {
        confidence = 1.0
    }
    return confidence
}

type middlewareChain struct {
    middleware []string
    usages     []patternUsage
}

type patternUsage struct {
    resource  string
    operation string
    filePath  string
}
```

#### Algorithm 2: Build Dependency Graph

```go
package analysis

func BuildDependencyGraph(resources []ResourceMetadata) *DependencyGraph {
    graph := &DependencyGraph{
        Nodes: make(map[string]*DependencyNode),
        Edges: []DependencyEdge{},
    }

    // Step 1: Add all resources as nodes
    for _, resource := range resources {
        node := &DependencyNode{
            ID:       resource.Name,
            Type:     "resource",
            Name:     resource.Name,
            FilePath: resource.FilePath,
        }
        graph.Nodes[resource.Name] = node
    }

    // Step 2: Add edges for relationships
    for _, resource := range resources {
        for _, rel := range resource.Relationships {
            edge := DependencyEdge{
                From:         resource.Name,
                To:           rel.TargetResource,
                Relationship: rel.Type,
                Weight:       1,
            }
            graph.Edges = append(graph.Edges, edge)
        }
    }

    // Step 3: Add edges for middleware usage
    for _, resource := range resources {
        for _, middlewareList := range resource.Middleware {
            for _, middleware := range middlewareList {
                middlewareName := extractMiddlewareName(middleware)

                // Add middleware node if not exists
                if _, exists := graph.Nodes[middlewareName]; !exists {
                    node := &DependencyNode{
                        ID:   middlewareName,
                        Type: "middleware",
                        Name: middlewareName,
                    }
                    graph.Nodes[middlewareName] = node
                }

                // Add edge
                edge := DependencyEdge{
                    From:         resource.Name,
                    To:           middlewareName,
                    Relationship: "uses",
                    Weight:       1,
                }
                graph.Edges = append(graph.Edges, edge)
            }
        }
    }

    return graph
}

func (r *Registry) queryDependencies(resource string, opts DependencyOptions) *DependencyGraph {
    result := &DependencyGraph{
        Nodes: make(map[string]*DependencyNode),
        Edges: []DependencyEdge{},
    }

    visited := make(map[string]bool)

    // BFS traversal
    var traverse func(nodeID string, depth int)
    traverse = func(nodeID string, depth int) {
        if depth > opts.Depth || visited[nodeID] {
            return
        }

        visited[nodeID] = true

        // Add node to result
        if node, ok := r.metadata.Dependencies.Nodes[nodeID]; ok {
            result.Nodes[nodeID] = node
        }

        // Find edges
        var edges []DependencyEdge
        if opts.Reverse {
            edges = findIncomingEdges(r.metadata.Dependencies, nodeID)
        } else {
            edges = findOutgoingEdges(r.metadata.Dependencies, nodeID)
        }

        // Filter by type if specified
        if len(opts.Types) > 0 {
            edges = filterEdgesByType(edges, opts.Types)
        }

        // Add edges and traverse targets
        for _, edge := range edges {
            result.Edges = append(result.Edges, edge)

            target := edge.To
            if opts.Reverse {
                target = edge.From
            }

            traverse(target, depth+1)
        }
    }

    traverse(resource, 1)
    return result
}
```

---

## Component 5: CLI Tool

### CLI Commands

#### `conduit introspect resources`

List all resources in the application.

**Output:**
```
Resources:
  Post (15 fields, 3 relationships, 2 hooks)
    Fields: id, title, slug, content, status, ...
    Relationships: author (User!), category (Category?), comments (has_many)
    Hooks: @before create, @after update

  User (8 fields, 1 relationship, 1 hook)
    Fields: id, username, email, password_hash, ...
    Relationships: posts (has_many)
    Hooks: @before create
```

**Flags:**
- `--format json|table` - Output format
- `--verbose` - Show all details

#### `conduit introspect resource <name>`

Show detailed information about a specific resource.

**Output:**
```
Resource: Post
File: resources/post.cdt
Documentation: Blog post with content and categorization

Fields (15):
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  status: enum ["draft", "published", "archived"] @default("draft")

Relationships (3):
  author: User! (belongs_to)
    Foreign Key: author_id
    On Delete: restrict

  category: Category? (belongs_to)
    Foreign Key: category_id
    On Delete: set_null

  comments: Comment[] (has_many)
    Foreign Key: post_id (in Comment)
    On Delete: cascade

Middleware:
  list: [cache(300)]
  get: [cache(600)]
  create: [auth, rate_limit(5, per: "hour")]
  update: [auth, author_or_editor]
  delete: [auth, author_or_admin]
```

#### `conduit introspect routes`

List all HTTP routes.

**Output:**
```
GET    /api/posts              -> Post.list    [cache(300)]
GET    /api/posts/:id          -> Post.get     [cache(600)]
POST   /api/posts              -> Post.create  [auth, rate_limit(5, per: "hour")]
PUT    /api/posts/:id          -> Post.update  [auth, author_or_editor]
DELETE /api/posts/:id          -> Post.delete  [auth, author_or_admin]
```

**Flags:**
- `--method GET|POST|PUT|DELETE` - Filter by method
- `--middleware <name>` - Filter by middleware
- `--format json|table` - Output format

#### `conduit introspect deps <resource>`

Show dependencies of a resource.

**Output:**
```
Dependencies of Post:

Direct Dependencies (what Post uses):
  Resources:
    - User (author relationship)
    - Category (category relationship)

  Middleware:
    - auth
    - rate_limit
    - cache

Reverse Dependencies (what uses Post):
  Resources:
    - Comment.post (belongs_to)

  Routes:
    - GET /api/posts
    - POST /api/posts
    - PUT /api/posts/:id
    - DELETE /api/posts/:id
```

**Flags:**
- `--depth <n>` - Traversal depth (default: 1)
- `--reverse` - Show reverse dependencies
- `--type resource|middleware|function` - Filter by type

#### `conduit introspect patterns [category]`

Show discovered patterns.

**Output:**
```
Patterns:

Authentication (3 patterns, 87% coverage):

  1. authenticated_handler (used 12 times)
     Description: Handler requiring authentication
     Template:
       @on create: [auth, rate_limit(5, per: "hour")]

     Examples:
       - Post.create (resources/post.cdt:45)
       - Comment.create (resources/comment.cdt:23)

  2. authenticated_route_with_ownership (used 8 times)
     Description: Route requiring auth and ownership check
     Template:
       @on update: [auth, author_or_editor]
       @on delete: [auth, author_or_admin]

     Examples:
       - Post.update (resources/post.cdt:47)
       - Comment.update (resources/comment.cdt:25)
```

**Flags:**
- `--category <name>` - Filter by category
- `--min-frequency <n>` - Minimum occurrences
- `--format json|table` - Output format

### CLI Implementation

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/conduit-lang/conduit/runtime"
)

func main() {
    // Load registry from compiled binary
    registry := runtime.GetRegistry()

    rootCmd := &cobra.Command{
        Use:   "conduit",
        Short: "Conduit CLI Tool",
    }

    introspectCmd := &cobra.Command{
        Use:   "introspect",
        Short: "Introspect the application",
    }

    // Add subcommands
    introspectCmd.AddCommand(
        resourcesCommand(registry),
        resourceCommand(registry),
        routesCommand(registry),
        depsCommand(registry),
        patternsCommand(registry),
    )

    rootCmd.AddCommand(introspectCmd)
    rootCmd.Execute()
}

func resourcesCommand(registry *runtime.Registry) *cobra.Command {
    return &cobra.Command{
        Use:   "resources",
        Short: "List all resources",
        Run: func(cmd *cobra.Command, args []string) {
            format, _ := cmd.Flags().GetString("format")

            resources := registry.Resources()

            if format == "json" {
                printJSON(resources)
            } else {
                printResourcesTable(resources)
            }
        },
    }
}

func depsCommand(registry *runtime.Registry) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "deps <resource>",
        Short: "Show dependencies",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            resourceName := args[0]

            depth, _ := cmd.Flags().GetInt("depth")
            reverse, _ := cmd.Flags().GetBool("reverse")

            opts := runtime.DependencyOptions{
                Depth:   depth,
                Reverse: reverse,
            }

            deps, err := registry.Dependencies(resourceName, opts)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: %v\n", err)
                os.Exit(1)
            }

            printDependencies(deps)
        },
    }

    cmd.Flags().Int("depth", 1, "Traversal depth")
    cmd.Flags().Bool("reverse", false, "Show reverse dependencies")

    return cmd
}
```

---

## Development Phases

### Phase 1: Foundation (Weeks 1-4)

**Goal:** Basic metadata collection and CLI introspection

**Week 1: Metadata Collection Infrastructure**
- [ ] Define metadata schema structs
- [ ] Implement AST visitor for metadata collection
- [ ] Collect resource definitions (fields, types, relationships)
- [ ] Unit tests with 80%+ coverage
- **Success:** Extract all fields and types from resource AST

**Week 2: Metadata Serialization & Embedding**
- [ ] Implement metadata serialization
- [ ] Generate code to embed metadata in binary
- [ ] Implement JSON export for debugging
- [ ] Create metadata loader for runtime
- **Success:** Metadata embedded in compiled binary

**Week 3: CLI Tool Foundation**
- [ ] Create CLI tool structure (cobra)
- [ ] Implement `introspect resources` command
- [ ] Implement `introspect resource <name>` command
- [ ] Add table and JSON output formats
- **Success:** CLI can list and show resources

**Week 4: Route Introspection**
- [ ] Collect route metadata from resources
- [ ] Generate route table
- [ ] Implement `introspect routes` command
- [ ] Add middleware chain display
- **Success:** Can list all routes with middleware

**Milestone:** Basic CLI introspection working on example project

### Phase 2: Dependency Analysis (Weeks 5-8)

**Goal:** Dependency graph construction and queries

**Week 5: Dependency Graph Construction**
- [ ] Implement dependency graph data structures
- [ ] Build graph from resource metadata
- [ ] Add relationship edges
- [ ] Add function call edges
- [ ] Add middleware edges
- **Success:** Can build complete dependency graph

**Week 6: Graph Queries**
- [ ] Implement graph traversal algorithms
- [ ] Depth-limited BFS
- [ ] Forward/reverse dependency queries
- [ ] Circular dependency detection
- **Success:** Queries work to arbitrary depth

**Week 7: Dependency CLI Commands**
- [ ] Implement `introspect deps <resource>` command
- [ ] Add depth and filter flags
- [ ] Tree and table output formats
- [ ] Impact analysis features
- **Success:** Shows direct and indirect dependencies

**Week 8: Performance Optimization**
- [ ] Profile graph construction
- [ ] Optimize hot paths
- [ ] Add caching for repeated queries
- [ ] Benchmark large codebases
- **Success:** Queries < 50ms P95

**Milestone:** Dependency analysis working efficiently

### Phase 3: Pattern Discovery (Weeks 9-12)

**Goal:** Automatic pattern extraction and templates

**Week 9: Pattern Extraction - Middleware**
- [ ] Implement middleware chain extraction
- [ ] Group similar chains
- [ ] Calculate frequency and confidence
- [ ] Generate pattern templates
- **Success:** Extracts common middleware patterns

**Week 10: Pattern Extraction - Validations**
- [ ] Implement constraint pattern extraction
- [ ] Normalize constraints
- [ ] Group similar validations
- [ ] Generate validation templates
- **Success:** Extracts validation patterns

**Week 11: Pattern CLI & Testing**
- [ ] Implement `introspect patterns` command
- [ ] Add category filtering
- [ ] Test with LLMs
- [ ] Iterate based on LLM feedback
- **Success:** LLMs can understand and use patterns (80%+ success)

**Week 12: Pattern Quality & Refinement**
- [ ] Implement pattern quality metrics
- [ ] Add manual pattern curation support
- [ ] Implement pattern versioning
- [ ] Add pattern usage tracking
- **Success:** Can measure and improve pattern quality

**Milestone:** Pattern discovery validated with LLMs

### Phase 4: Runtime API & Polish (Weeks 13-16)

**Goal:** Production-ready runtime API

**Week 13: Runtime API Implementation**
- [ ] Implement Go runtime API
- [ ] Add query interface
- [ ] Create example programs
- [ ] API documentation
- **Success:** API is intuitive and ergonomic

**Week 14: Performance & Caching**
- [ ] Aggressive caching for queries
- [ ] Lazy loading
- [ ] Profile and optimize
- [ ] Benchmark suite
- **Success:** Cached queries < 1ms, first query < 50ms

**Week 15: Developer Experience**
- [ ] Improve error messages
- [ ] Add helpful hints
- [ ] Implement watch mode
- [ ] Shell completions
- **Success:** Positive user feedback

**Week 16: Documentation & Release**
- [ ] Comprehensive documentation
- [ ] Tutorial and examples
- [ ] Demo videos
- [ ] Blog post
- **Success:** Ready for public release

**Milestone:** Production-ready introspection system

---

## Performance Targets

| Metric | Target | Acceptable | Unacceptable |
|--------|--------|------------|--------------|
| Metadata size per resource | <2KB | <5KB | >10KB |
| Registry initialization | <10ms | <50ms | >100ms |
| Simple query (get resource) | <1ms | <5ms | >10ms |
| Complex query (deps depth 3) | <20ms | <50ms | >100ms |
| Memory usage (50 resources) | <10MB | <50MB | >100MB |

### Optimization Strategies

**1. Compression**
```go
// Compress metadata before embedding (60-80% size reduction)
compressed := gzip.Compress(json.Marshal(metadata))
```

**2. Pre-computed Indexes**
```go
// Build indexes at startup, not on every query
resourcesByName map[string]*ResourceMetadata
routesByPath    map[string][]*RouteMetadata
```

**3. Aggressive Caching**
```go
// Cache query results indefinitely (metadata doesn't change at runtime)
cache map[string]interface{}
```

**4. Code Generation over Reflection**
```go
// Generate introspection methods at compile time
func (p *Post) GetFieldType(name string) string { ... }
// Rather than using reflect.ValueOf(p).FieldByName(name)
```

---

## Testing Strategy

### Unit Tests

```go
func TestMetadataCollector_ExtractFields(t *testing.T)
func TestMetadataCollector_ExtractRelationships(t *testing.T)
func TestDependencyGraph_QueryDependencies(t *testing.T)
func TestDependencyGraph_DetectCircular(t *testing.T)
func TestPatternExtractor_ExtractMiddleware(t *testing.T)
```

**Coverage Target:** 80%+ code coverage

### Integration Tests

```go
func TestIntegration_CompileAndIntrospect(t *testing.T)
func TestIntegration_CLICommands(t *testing.T)
func TestIntegration_PatternDiscovery(t *testing.T)
```

### LLM Validation Tests

**Test Protocol:**
1. Extract patterns from blog example
2. Prompt LLM: "Based on introspection, add auth to Comment.create"
3. Verify LLM generates: `@on create: [auth, rate_limit(10, per: "hour")]`
4. Success: Generated code matches pattern exactly

**Success Criteria:**
- 80%+ success rate for Claude Opus
- 70%+ success rate for GPT-4
- 60%+ success rate for GPT-3.5

### Performance Benchmarks

```go
func BenchmarkRegistryInit(b *testing.B)
func BenchmarkResourceQuery(b *testing.B)
func BenchmarkDependencyQuery(b *testing.B)
func BenchmarkPatternExtraction(b *testing.B)
```

---

## Integration Points

### With Compiler

```go
// Compiler calls metadata collector during compilation
func (c *Compiler) Compile(files []SourceFile, opts CompileOptions) (*CompileResult, error) {
    ast, _ := c.parser.Parse(files)

    // Collect metadata
    if opts.Introspection {
        collector := metadata.NewCollector()
        metadata := collector.Collect(ast)
        c.metadata = metadata
    }

    // Generate Go code
    goCode := c.generator.Generate(ast, opts)

    // Embed metadata
    if opts.Introspection {
        goCode = c.embedder.Embed(goCode, c.metadata)
    }

    return &CompileResult{Binary: binary, Metadata: c.metadata}, nil
}
```

### With Web Framework

```go
// Web framework uses introspection to generate routes
func (wf *WebFramework) RegisterRoutes() {
    registry := runtime.GetRegistry()

    for _, route := range registry.Routes(runtime.RouteFilter{}) {
        wf.router.Handle(route.Method, route.Path, route.Handler)
    }
}
```

### With LSP (Future)

```go
// LSP server uses registry for completions
func (s *LSPServer) Completion(params lsp.CompletionParams) (*lsp.CompletionList, error) {
    patterns := s.registry.Patterns("")

    items := []lsp.CompletionItem{}
    for _, p := range patterns {
        items = append(items, lsp.CompletionItem{
            Label:      p.Name,
            Detail:     p.Description,
            InsertText: p.Template,
        })
    }

    return &lsp.CompletionList{Items: items}, nil
}
```

---

## Critical Success Factors

✅ **Query Response Time:** < 50ms for typical introspection queries
✅ **Metadata Accuracy:** 100% accurate representation of code structure
✅ **Pattern Quality:** Extracted patterns reproducible by LLMs
✅ **LLM Success Rate:** 80%+ LLMs successfully use patterns
✅ **Developer Satisfaction:** 90% find patterns without documentation

---

## Major Risks & Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Metadata size bloat | MEDIUM | Compression, selective collection |
| Reflection performance | HIGH | Hybrid approach, code generation |
| Pattern discovery accuracy | HIGH | LLM validation, manual curation |
| Stale metadata in development | MEDIUM | Watch mode, fast rebuilds |

---

## Next Steps

### Immediate Actions
1. Define metadata schema in Go
2. Implement AST visitor for metadata collection
3. Create basic CLI tool structure
4. Write unit tests for metadata collection
5. Profile and benchmark metadata size

### Research Needed
1. Go reflection performance characteristics
2. Pattern extraction algorithms from research
3. Compression trade-offs (size vs speed)
4. LLM prompt engineering for pattern usage

### Prototypes to Build
1. Simple metadata collector for subset of features
2. CLI tool with basic commands
3. Pattern extractor for middleware chains
4. Dependency graph builder

---

## References

- See `LANGUAGE-SPEC.md` for complete syntax specification
- See `IMPLEMENTATION-COMPILER.md` for compiler integration
- See `docs/open-questions.md` for unresolved design decisions
- See `docs/research/` for historical analysis documents

---

**End of Runtime & Introspection Implementation Guide**
