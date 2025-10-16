# Introspection Metadata Package

This package provides introspection metadata generation for Conduit applications, enabling runtime queries about the application schema, patterns, and structure.

## Overview

The metadata system extracts comprehensive information from the Abstract Syntax Tree (AST) and generates:
- **JSON metadata file** (`app.meta.json`) with complete schema information
- **Go accessor code** embedded in the compiled binary for runtime queries
- **Pattern detection** identifying common conventions across resources

## Key Components

### 1. Metadata Schema (`schema.go`)

Defines the structure of introspection metadata:

```go
type Metadata struct {
    Version   string
    Resources []ResourceMetadata
    Patterns  []PatternMetadata
    Routes    []RouteMetadata
}
```

Each resource includes:
- Fields with types and constraints
- Relationships (belongs-to, has-many, etc.)
- Lifecycle hooks with timing and middleware
- Validations and custom constraints
- Scopes and computed fields

### 2. Metadata Extractor (`extractor.go`)

Analyzes the AST and extracts metadata:

```go
extractor := metadata.NewExtractor("1.0.0")
meta, err := extractor.Extract(program)
jsonStr, err := meta.ToJSON()
```

**Features:**
- Complete resource schema extraction
- Relationship mapping
- Hook analysis (transactions, async, middleware)
- Pattern detection across codebase
- REST route generation

### 3. Code Generator Integration

The code generator automatically:
1. Generates `introspection/metadata.json` with complete schema
2. Creates `introspection/introspection.go` with runtime query API
3. Embeds metadata in the compiled binary

## Usage

### Extracting Metadata

```go
import "github.com/conduit-lang/conduit/internal/compiler/metadata"

// Create extractor
extractor := metadata.NewExtractor("1.0.0")

// Extract from AST
meta, err := extractor.Extract(program)
if err != nil {
    return err
}

// Generate JSON
jsonStr, err := meta.ToJSON()
```

### Runtime Queries

Generated accessor functions in the compiled binary:

```go
import "your-app/introspection"

// Get all metadata
meta, err := introspection.GetMetadata()

// Query specific information
resources, err := introspection.QueryResources()
patterns, err := introspection.QueryPatterns()
routes, err := introspection.QueryRoutes()

// Find specific resource
user, err := introspection.FindResource("User")
```

## Generated Metadata Structure

Example `metadata.json`:

```json
{
  "version": "1.0.0",
  "resources": [
    {
      "name": "User",
      "documentation": "User account",
      "fields": [
        {
          "name": "username",
          "type": "string!",
          "nullable": false,
          "constraints": ["unique"]
        }
      ],
      "hooks": [
        {
          "timing": "after",
          "event": "create",
          "has_transaction": true,
          "middleware": ["auth"]
        }
      ],
      "relationships": [],
      "validations": []
    }
  ],
  "patterns": [
    {
      "name": "authenticated_handler",
      "template": "@after <event>: [auth]",
      "occurrences": 3
    }
  ],
  "routes": [
    {
      "method": "GET",
      "path": "/users",
      "handler": "Index",
      "resource": "User"
    }
  ]
}
```

## Pattern Detection

The system automatically detects common patterns:

1. **Authenticated Handlers** - Hooks with auth middleware
2. **Transactional Hooks** - Hooks running in transactions
3. **Async Operations** - Asynchronous hook blocks
4. **Unique Fields** - Fields with unique constraints

Each pattern includes:
- Template showing the pattern structure
- Number of occurrences
- Optional description

## Performance

**Benchmarks** (Apple M3 Pro):
- **67.5µs per operation** (10 resources)
- **70.9KB memory allocated**
- **231 allocations**

**Requirements:**
- ✅ < 50ms generation time (exceeded by 740x)
- ✅ Minimal memory footprint
- ✅ No runtime overhead (embedded at compile time)

## Use Cases

### 1. LLM Pattern Discovery

LLMs can query metadata to discover patterns:
```
"How do I add authentication?"
→ Query patterns for "authenticated_handler"
→ Show template and examples
```

### 2. Documentation Generation

Auto-generate API documentation from metadata:
```go
routes, _ := introspection.QueryRoutes()
// Generate OpenAPI/Swagger spec
```

### 3. Schema Migration Tools

Compare metadata between versions for migrations:
```go
oldMeta := loadMetadata("v1.0.0")
newMeta := loadMetadata("v2.0.0")
diff := compareSchemas(oldMeta, newMeta)
```

### 4. IDE Integration

Power IDE features with schema information:
- Field autocomplete
- Type hints
- Relationship navigation

## Testing

Run tests:
```bash
go test ./internal/compiler/metadata/...
```

Run benchmarks:
```bash
go test -bench=. ./internal/compiler/codegen/...
```

## Implementation Details

### Type Formatting

Types are formatted with explicit nullability:
- `string!` - Required string
- `string?` - Optional string
- `array<string!>!` - Required array of required strings
- `hash<string!,int!>?` - Optional hash

### Relationship Kinds

- `belongs_to` - Many-to-one relationship
- `has_many` - One-to-many relationship
- `has_one` - One-to-one relationship
- `has_many_through` - Many-to-many through join table

### Hook Metadata

Captures:
- Timing: `before` or `after`
- Event: `create`, `update`, `delete`, `save`
- Transactions: Boolean flag
- Async: Boolean flag
- Middleware: Array of middleware names

## Future Enhancements

Potential improvements:
- [ ] Expression tree serialization (currently placeholder)
- [ ] Custom pattern definitions
- [ ] Metadata versioning and migration
- [ ] Incremental metadata updates
- [ ] Schema diffing utilities
