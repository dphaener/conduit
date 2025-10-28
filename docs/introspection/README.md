# Conduit Introspection System

The Conduit introspection system provides powerful runtime querying capabilities that enable LLMs, developers, and tools to explore and understand your application's structure, dependencies, and patterns.

## Quick Start

### 5-Minute Introduction

After building your Conduit application, you can immediately start exploring it:

```bash
# List all resources in your application
conduit introspect resources

# View detailed information about a specific resource
conduit introspect resource Post

# List all HTTP routes
conduit introspect routes

# Show dependencies of a resource
conduit introspect deps Post

# Discover common patterns in your codebase
conduit introspect patterns
```

### What Can You Do With Introspection?

The introspection system enables:

- **Schema Exploration**: Understand resource structure, fields, relationships, and constraints
- **Dependency Analysis**: Map dependencies between resources and detect circular references
- **Pattern Discovery**: Extract common patterns for LLM learning and code generation
- **Route Inspection**: View all HTTP endpoints and their middleware chains
- **Tooling Integration**: Build custom tools that query your application's metadata
- **Documentation Generation**: Auto-generate API docs and schema diagrams

## Core Concepts

### Resources

Resources are the fundamental building blocks of Conduit applications. The introspection system captures complete metadata about each resource:

- Fields (types, nullability, constraints)
- Relationships (belongs_to, has_many, has_many_through)
- Lifecycle hooks (before/after create, update, delete)
- Validations and constraints
- Middleware configuration
- Auto-generated routes

### Routes

Every Conduit resource automatically gets RESTful HTTP routes. Introspection lets you:

- View all routes (method, path, handler)
- See middleware chains applied to each route
- Filter routes by method, resource, or middleware
- Understand the API surface of your application

### Patterns

Patterns are recurring code structures discovered by analyzing your codebase:

- Middleware combinations (auth + rate_limit)
- Hook patterns (slug generation, timestamps)
- Validation patterns (email, phone, custom)
- Query patterns (common scopes and filters)

Patterns help LLMs generate consistent code that follows your project's conventions.

### Dependencies

The dependency graph maps relationships between resources, middleware, and functions:

- **Direct dependencies**: What a resource uses
- **Reverse dependencies**: What depends on a resource
- **Circular dependency detection**: Identify potential issues
- **Impact analysis**: Understand change ripple effects

## CLI Commands Overview

### `conduit introspect resources`

List all resources with summary information.

```bash
# Basic list
conduit introspect resources

# Verbose output with all details
conduit introspect resources --verbose

# JSON output for tooling
conduit introspect resources --format json
```

### `conduit introspect resource <name>`

Show detailed information about a specific resource.

```bash
# View Post resource details
conduit introspect resource Post

# JSON format
conduit introspect resource Post --format json
```

### `conduit introspect routes`

List all HTTP routes in your application.

```bash
# All routes
conduit introspect routes

# Filter by HTTP method
conduit introspect routes --method GET

# Filter by resource
conduit introspect routes --resource Post

# Filter by middleware
conduit introspect routes --middleware auth
```

### `conduit introspect deps <resource>`

Show dependencies of a resource.

```bash
# Direct dependencies (what Post uses)
conduit introspect deps Post

# Reverse dependencies (what uses Post)
conduit introspect deps Post --reverse

# Deeper dependency tree
conduit introspect deps Post --depth 2

# Filter by dependency type
conduit introspect deps Post --type resource
```

### `conduit introspect patterns`

Discover common patterns in your codebase.

```bash
# All patterns
conduit introspect patterns

# Patterns in specific category
conduit introspect patterns authentication

# Filter by minimum frequency
conduit introspect patterns --min-frequency 3
```

## Go API Overview

You can also use introspection programmatically from Go:

```go
import "github.com/conduit-lang/conduit/runtime/metadata"

// Get the registry
registry := metadata.GetRegistry()

// Query all resources
resources := registry.Resources()

// Query single resource
post, err := registry.Resource("Post")
if err != nil {
    log.Fatal(err)
}

// Query routes with filters
routes := registry.Routes(metadata.RouteFilter{
    Method: "GET",
    Resource: "Post",
})

// Query patterns
patterns := registry.Patterns("hook")

// Query dependencies
deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 2,
    Reverse: false,
})
```

## Use Cases

### For Developers

- **Explore unfamiliar codebases**: Quickly understand structure and relationships
- **Impact analysis**: See what breaks before making changes
- **Code review**: Verify patterns are followed consistently
- **Debugging**: Understand middleware chains and hook execution order

### For LLMs

- **Pattern learning**: Discover and replicate project-specific conventions
- **Context gathering**: Get accurate schema information for code generation
- **Validation**: Verify generated code follows discovered patterns
- **Self-correction**: Query to fix errors without human intervention

### For Tools

- **Documentation generators**: Auto-generate API docs from routes
- **Schema visualizers**: Create ER diagrams from dependency graphs
- **Linters**: Enforce project-specific patterns
- **Migration tools**: Analyze impact of schema changes

## Performance Characteristics

The introspection system is optimized for performance:

- **Registry initialization**: <1ms for 50-100 resources
- **Simple queries** (resource, routes): Sub-microsecond
- **Complex queries** (dependency traversal): <10µs cold, <1µs cached
- **Memory footprint**: ~200KB for 50 resources
- **Query caching**: Automatic LRU caching for complex queries

See [architecture.md](architecture.md) for detailed performance analysis.

## Documentation Structure

- **[User Guide](user-guide.md)**: Common workflows and practical examples
- **[CLI Reference](cli-reference.md)**: Complete command reference with all flags
- **[API Reference](api-reference.md)**: Go API documentation for programmatic access
- **[Architecture](architecture.md)**: How introspection works internally
- **[Tutorial](tutorial/01-basic-queries.md)**: Step-by-step walkthrough with examples
- **[Best Practices](best-practices.md)**: When and how to use introspection
- **[Troubleshooting](troubleshooting.md)**: Common issues and solutions

## Example Programs

Check out the working examples in `examples/introspection/`:

- **list-resources**: Simple query tool to explore resources
- **dependency-analyzer**: Detect circular dependencies and show impact
- **api-doc-generator**: Generate API documentation from routes
- **pattern-validator**: Verify code follows discovered patterns
- **schema-explorer**: Interactive tool to explore your application schema

## Next Steps

1. **Read the [User Guide](user-guide.md)** for common workflows
2. **Try the [Tutorial](tutorial/01-basic-queries.md)** for hands-on learning
3. **Explore the [Examples](../../examples/introspection/)** for practical code samples
4. **Check the [API Reference](api-reference.md)** for programmatic usage

## Getting Help

If you run into issues:

1. Check the [Troubleshooting Guide](troubleshooting.md)
2. Review the [Best Practices](best-practices.md)
3. See example programs for working code
4. File an issue on GitHub with your use case

## Design Philosophy

The introspection system is built on three principles:

1. **LLM-First**: Designed specifically for AI consumption and pattern learning
2. **Zero-Overhead**: Compile-time metadata generation, runtime indexing
3. **Self-Documenting**: Your code is the source of truth, documentation auto-generated

This makes Conduit applications inherently more maintainable, understandable, and AI-friendly than traditional codebases.
