# Conduit Introspection Examples

This directory contains practical example programs demonstrating how to use the Conduit introspection API.

## Overview

Each example is a complete, runnable Go program that showcases specific introspection capabilities. All examples follow best practices and include comprehensive error handling.

## Quick Start

```bash
# Navigate to any example directory
cd list-resources

# Build and run
go build -o list-resources
./list-resources
```

## Available Examples

### 1. List Resources ([list-resources/](list-resources/))

**Difficulty:** Beginner
**Time:** 5 minutes

Lists all resources in your application with summary information.

**Features:**
- Lists all resources with counts
- Categorizes resources
- Supports JSON and table output
- Filtering by category

**Usage:**
```bash
./list-resources
./list-resources --format json
./list-resources --verbose
```

**Learn:**
- Basic registry access
- Querying resources
- Output formatting
- Resource categorization

---

### 2. Dependency Analyzer ([dependency-analyzer/](dependency-analyzer/))

**Difficulty:** Intermediate
**Time:** 10 minutes

Analyzes dependencies between resources and detects issues.

**Features:**
- Circular dependency detection
- Dependency depth analysis
- Impact analysis (what breaks when you change something)
- Complexity metrics

**Usage:**
```bash
./dependency-analyzer --check-cycles
./dependency-analyzer --report
./dependency-analyzer Post
```

**Learn:**
- Dependency graph traversal
- Cycle detection algorithms
- Impact analysis
- Complexity metrics

---

### 3. API Documentation Generator ([api-doc-generator/](api-doc-generator/))

**Difficulty:** Beginner
**Time:** 5 minutes

Generates API documentation from routes and resources.

**Features:**
- Auto-generates REST API docs
- Markdown and HTML output
- Request/response schemas
- Middleware documentation

**Usage:**
```bash
./api-doc-generator > API.md
./api-doc-generator --format html > API.html
```

**Learn:**
- Querying routes
- Combining resource and route metadata
- Document generation
- Multiple output formats

---

### 4. Pattern Validator ([pattern-validator/](pattern-validator/))

**Difficulty:** Intermediate
**Time:** 10 minutes

Validates that resources follow discovered patterns and coding standards.

**Features:**
- Validates authentication patterns
- Checks rate limiting
- Enforces slug generation patterns
- Configurable rules

**Usage:**
```bash
./pattern-validator
./pattern-validator --strict
```

**Learn:**
- Pattern-based validation
- Rule-based analysis
- Custom validation logic
- Code quality enforcement

---

### 5. Schema Explorer ([schema-explorer/](schema-explorer/))

**Difficulty:** Beginner
**Time:** 5 minutes

Interactive terminal UI for exploring the schema.

**Features:**
- REPL interface
- Browse resources interactively
- View dependencies
- Query routes
- Show patterns

**Usage:**
```bash
./schema-explorer
```

**Commands:**
- `list` - List all resources
- `show <resource>` - Show resource details
- `routes` - List routes
- `deps <resource>` - Show dependencies
- `patterns` - Show patterns
- `help` - Show help
- `exit` - Exit

**Learn:**
- Building interactive tools
- User-friendly interfaces
- Command parsing
- REPL pattern

---

## Common Patterns

All examples follow these patterns:

### 1. Initialize Registry

```go
registry := metadata.GetRegistry()

// Always check if initialized
if registry.GetSchema() == nil {
    fmt.Fprintln(os.Stderr, "Error: Registry not initialized")
    os.Exit(1)
}
```

### 2. Query Metadata

```go
// Get all resources
resources := registry.Resources()

// Get specific resource
resource, err := registry.Resource("Post")
if err != nil {
    // Handle error
}

// Query routes
routes := registry.Routes(metadata.RouteFilter{
    Resource: "Post",
})

// Get dependencies
deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth: 2,
})
```

### 3. Handle Errors

```go
resource, err := registry.Resource("Post")
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### 4. Format Output

```go
// JSON output
if format == "json" {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encoder.Encode(data)
} else {
    // Table output
    formatAsTable(data)
}
```

## Building Your Own Tools

Use these examples as templates for your own tools:

1. **Start with an example**: Pick the closest match to what you want
2. **Copy the structure**: Follow the same initialization and error handling
3. **Add your logic**: Implement your specific functionality
4. **Test thoroughly**: Use the Go API, errors are your friends

### Example: Custom Linter

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    for _, res := range resources {
        // Your custom validation logic
        if len(res.Fields) < 3 {
            fmt.Printf("⚠️  %s: Too few fields\n", res.Name)
        }
    }
}
```

## Prerequisites

All examples require:
- Go 1.23+
- Conduit application compiled with `conduit build`
- Metadata generated (`build/app.meta.json`)

## Running Examples

```bash
# Option 1: Build and run
cd list-resources
go build
./list-resources

# Option 2: Run directly
cd list-resources
go run main.go

# Option 3: Install globally
cd list-resources
go install
list-resources  # Now available anywhere
```

## Tips

1. **Always check registry initialization**: Don't assume it's ready
2. **Handle errors properly**: Introspection can fail
3. **Use filters for performance**: Don't fetch everything if you need specific data
4. **Cache results**: Metadata doesn't change at runtime
5. **Format for humans**: Make output readable
6. **Support JSON**: Enable tool composition

## Common Issues

**"Registry not initialized"**
- Run `conduit build` first
- Make sure you're in a Conduit project directory

**"Resource not found"**
- Check resource name spelling (case-sensitive)
- Run `conduit introspect resources` to see available resources

**Build errors**
- Make sure `go.mod` includes Conduit dependencies
- Run `go mod tidy` to fetch missing packages

## Next Steps

1. **Read the tutorials**: See [docs/introspection/tutorial/](../../docs/introspection/tutorial/)
2. **Read the API reference**: See [docs/introspection/api-reference.md](../../docs/introspection/api-reference.md)
3. **Build your own tool**: Use these examples as templates
4. **Share with community**: Submit your tools as examples!

## Contributing

Found a bug or want to add an example?

1. Fork the repository
2. Create a new example directory
3. Add README and working code
4. Submit a pull request

## Resources

- [Introspection Documentation](../../docs/introspection/)
- [User Guide](../../docs/introspection/user-guide.md)
- [API Reference](../../docs/introspection/api-reference.md)
- [Tutorial Series](../../docs/introspection/tutorial/)
- [Best Practices](../../docs/introspection/best-practices.md)

---

**Need Help?**

- Check the [Troubleshooting Guide](../../docs/introspection/troubleshooting.md)
- File an issue on GitHub
- Ask in the community Discord
