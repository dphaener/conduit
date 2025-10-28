# CLI Reference

Complete reference for all introspection CLI commands, flags, and options.

## Table of Contents

- [Global Flags](#global-flags)
- [conduit introspect](#conduit-introspect)
- [conduit introspect resources](#conduit-introspect-resources)
- [conduit introspect resource](#conduit-introspect-resource)
- [conduit introspect routes](#conduit-introspect-routes)
- [conduit introspect deps](#conduit-introspect-deps)
- [conduit introspect patterns](#conduit-introspect-patterns)

## Global Flags

These flags apply to all introspect commands:

### --format

Output format: `json` or `table` (default: `table`)

```bash
# Human-readable table format
conduit introspect resources --format table

# Machine-readable JSON format
conduit introspect resources --format json
```

**When to use**:
- Use `table` for interactive exploration
- Use `json` for tooling and scripts

### --verbose

Show all details (default: `false`)

```bash
conduit introspect resources --verbose
```

**Effect**:
- Shows additional fields and metadata
- Includes documentation strings
- Displays source code for hooks
- Shows all validation rules

### --no-color

Disable colored output (default: `false`)

```bash
conduit introspect resources --no-color
```

**When to use**:
- In scripts and automation
- When redirecting to files
- In environments without color support

---

## conduit introspect

Root command for all introspection operations.

### Usage

```bash
conduit introspect [command]
```

### Description

The introspect command provides access to the runtime registry, allowing you to explore resources, routes, patterns, and dependencies in your application.

This is useful for:
- Understanding the structure of your application
- Debugging resource relationships
- Discovering common patterns
- Generating documentation
- Building tooling and integrations

The introspection system reads metadata from your compiled binary to provide accurate, up-to-date information about your application's structure.

### Available Commands

- `resources` - List all resources in the application
- `resource` - Show detailed information about a specific resource
- `routes` - List all HTTP routes
- `deps` - Show dependencies of a resource
- `patterns` - Show discovered patterns

### Examples

```bash
# List all resources in the application
conduit introspect resources

# View detailed information about a specific resource
conduit introspect resource Post

# List all HTTP routes
conduit introspect routes

# Show dependencies of a resource
conduit introspect deps Post

# Discover common patterns
conduit introspect patterns

# Output in JSON format for tooling
conduit introspect resources --format json

# Verbose output with all details
conduit introspect resource Post --verbose
```

---

## conduit introspect resources

List all resources in the application.

### Usage

```bash
conduit introspect resources [flags]
```

### Description

Shows a summary of all resources including their fields, relationships, and hooks. Use the `introspect resource <name>` command to view detailed information about a specific resource.

### Flags

All [global flags](#global-flags) are supported:
- `--format` (json, table)
- `--verbose`
- `--no-color`

### Output Format

**Table format (default)**:

Resources are grouped by category (Core Resources, Administrative, System) and displayed with:
- Resource name
- Field count
- Relationship count
- Hook count
- Flags (auth required, cached, nested)

**Verbose table format** (`--verbose`):

Shows detailed information for each resource:
- Field count with breakdown
- Relationship count
- Hook count
- All flags with descriptions

**JSON format** (`--format json`):

Returns a JSON object with:
```json
{
  "total_count": 10,
  "resources": [
    {
      "name": "Post",
      "field_count": 9,
      "relationship_count": 3,
      "hook_count": 4,
      "validation_count": 5,
      "constraint_count": 2,
      "middleware": {
        "create": ["auth", "rate_limit"],
        "list": ["cache"]
      },
      "category": "Core Resources",
      "flags": ["auth_required", "cached"]
    }
  ]
}
```

### Examples

```bash
# List all resources
conduit introspect resources

# List resources in JSON format
conduit introspect resources --format json

# Show verbose output with all details
conduit introspect resources --verbose

# For scripting (no color, JSON output)
conduit introspect resources --no-color --format json
```

### Common Use Cases

- **Quick overview**: See all resources at a glance
- **Resource discovery**: Find resources by name or category
- **Count resources**: Get total resource count
- **Tooling integration**: Export resource list as JSON

---

## conduit introspect resource

Show detailed information about a specific resource.

### Usage

```bash
conduit introspect resource <name> [flags]
```

### Arguments

- `<name>` (required) - Name of the resource to inspect (e.g., "Post", "User")

### Description

Displays all fields, relationships, hooks, constraints, and middleware associated with the resource.

### Flags

All [global flags](#global-flags) are supported:
- `--format` (json, table)
- `--verbose`
- `--no-color`

### Output Format

**Table format (default)**:

Shows organized sections:
1. **Header**: Resource name, file path, documentation
2. **Schema**: Fields (required/optional), relationships
3. **Behavior**: Lifecycle hooks, constraints, validations
4. **API Endpoints**: Auto-generated routes

**Verbose table format** (`--verbose`):

Includes additional details:
- Field documentation strings
- Hook source code
- Complete constraint conditions
- All validation rules
- Middleware by operation

**JSON format** (`--format json`):

Returns the complete `ResourceMetadata` structure.

### Examples

```bash
# View details of the Post resource
conduit introspect resource Post

# View details in JSON format
conduit introspect resource Post --format json

# Verbose output with hook source code
conduit introspect resource Post --verbose

# For tooling (JSON, no color)
conduit introspect resource User --format json --no-color
```

### Error Handling

If the resource is not found, suggestions are provided:

```bash
$ conduit introspect resource Pst

Error: resource not found: Pst

Did you mean?
  - Post
  - Category

Available resources:
  - User
  - Post
  - Comment
  - Category
```

### Common Use Cases

- **Understand schema**: See all fields and types
- **Check relationships**: View belongs_to, has_many relationships
- **Review hooks**: Understand lifecycle behavior
- **Verify constraints**: See business rules
- **API documentation**: View auto-generated endpoints

---

## conduit introspect routes

List all HTTP routes in the application.

### Usage

```bash
conduit introspect routes [flags]
```

### Description

Shows the HTTP method, path, handler, and middleware for each route.

### Flags

All [global flags](#global-flags) plus:

#### --method

Filter by HTTP method (GET, POST, PUT, DELETE)

```bash
conduit introspect routes --method GET
```

#### --middleware

Filter by middleware name (substring match)

```bash
conduit introspect routes --middleware auth
```

#### --resource

Filter by resource name (exact match)

```bash
conduit introspect routes --resource Post
```

### Output Format

**Table format (default)**:

Routes displayed with color-coded HTTP methods:
- GET (green)
- POST (blue)
- PUT (yellow)
- DELETE (red)

Format: `METHOD PATH -> HANDLER [MIDDLEWARE]`

**JSON format** (`--format json`):

```json
{
  "total_count": 20,
  "routes": [
    {
      "method": "GET",
      "path": "/posts",
      "handler": "ListPosts",
      "resource": "Post",
      "operation": "list",
      "middleware": ["cache"],
      "request_body": "",
      "response_body": "[]Post"
    }
  ]
}
```

### Examples

```bash
# List all routes
conduit introspect routes

# Filter by HTTP method
conduit introspect routes --method GET
conduit introspect routes --method POST

# Filter by middleware
conduit introspect routes --middleware auth
conduit introspect routes --middleware cache

# Filter by resource
conduit introspect routes --resource Post

# Combine filters (all GET routes for Post)
conduit introspect routes --method GET --resource Post

# Combine filters (all authenticated routes)
conduit introspect routes --middleware auth

# JSON output for tooling
conduit introspect routes --format json

# Find all unauthenticated routes (using grep)
conduit introspect routes | grep -v "\[auth"
```

### Common Use Cases

- **API overview**: See all endpoints at a glance
- **Security audit**: Find routes without auth middleware
- **Performance review**: Find routes without caching
- **Documentation**: Generate API reference docs
- **Route discovery**: Find endpoints for a resource

---

## conduit introspect deps

Show dependencies of a resource.

### Usage

```bash
conduit introspect deps <resource> [flags]
```

### Arguments

- `<resource>` (required) - Name of the resource to analyze

### Description

Displays both direct dependencies (what the resource uses) and reverse dependencies (what uses the resource). This includes relationships to other resources, middleware, and routes.

### Flags

All [global flags](#global-flags) plus:

#### --depth

Traversal depth for dependency tree (default: 1, max: 5)

```bash
conduit introspect deps Post --depth 2
```

Depth levels:
- `1`: Direct dependencies only (default)
- `2`: Dependencies + their dependencies
- `3-5`: Deeper traversal (slower)

#### --reverse

Show only reverse dependencies (default: false)

```bash
conduit introspect deps Post --reverse
```

When false (default): Shows both direct and reverse dependencies
When true: Shows only reverse dependencies (what uses this resource)

#### --type

Filter by dependency type: resource, middleware, or function

```bash
conduit introspect deps Post --type resource
conduit introspect deps Post --type middleware
conduit introspect deps Post --type function
```

### Output Format

**Table format (default)**:

Shows two sections:

1. **Direct Dependencies** (what Post uses):
   - Resources (with relationship type)
   - Middleware
   - Functions
   - Impact descriptions

2. **Reverse Dependencies** (what uses Post):
   - Resources (with relationship type)
   - Routes
   - Impact descriptions

Impact descriptions explain what happens on deletion:
- "Deleting X cascades to Y"
- "Cannot delete X with existing Y"
- "Deleting X nullifies Y.field"

**JSON format** (`--format json`):

Returns complete `DependencyGraph` structure:
```json
{
  "nodes": {
    "post": {
      "id": "post",
      "type": "resource",
      "name": "Post",
      "file_path": "/app/resources/post.cdt"
    },
    "user": {
      "id": "user",
      "type": "resource",
      "name": "User",
      "file_path": "/app/resources/user.cdt"
    }
  },
  "edges": [
    {
      "from": "post",
      "to": "user",
      "relationship": "belongs_to",
      "weight": 1
    }
  ]
}
```

### Examples

```bash
# Show dependencies of Post resource
conduit introspect deps Post

# Show reverse dependencies only
conduit introspect deps Post --reverse

# Traverse deeper dependency tree
conduit introspect deps Post --depth 2
conduit introspect deps Post --depth 3

# Filter by dependency type
conduit introspect deps Post --type resource
conduit introspect deps Post --type middleware
conduit introspect deps Post --type function

# Combine filters (deep resource dependencies)
conduit introspect deps Post --depth 2 --type resource

# JSON output for tooling
conduit introspect deps Post --format json

# Impact analysis before deletion
conduit introspect deps User --reverse
```

### Error Handling

**Resource not found**:
```bash
$ conduit introspect deps Pst

Error: resource not found: Pst

Did you mean?
  - Post
```

**Invalid depth**:
```bash
$ conduit introspect deps Post --depth 10

Error: depth must be between 1 and 5, got: 10
```

**Invalid type filter**:
```bash
$ conduit introspect deps Post --type invalid

Error: invalid type filter: invalid (valid: resource, middleware, function)
```

### Common Use Cases

- **Impact analysis**: What breaks if I delete this resource?
- **Dependency mapping**: What does this resource depend on?
- **Circular dependency detection**: Find dependency cycles
- **Refactoring planning**: Understand change ripple effects
- **Documentation**: Generate dependency diagrams

---

## conduit introspect patterns

Show discovered patterns in the application.

### Usage

```bash
conduit introspect patterns [category] [flags]
```

### Arguments

- `[category]` (optional) - Pattern category to filter by

Common categories:
- `authentication`
- `caching`
- `rate_limiting`
- `hook`
- `validation`
- `middleware`

### Description

The pattern discovery system analyzes your codebase to identify common patterns and conventions. This helps with:
- Understanding coding standards
- Maintaining consistency
- Generating documentation
- Training LLMs on project-specific patterns

### Flags

All [global flags](#global-flags) plus:

#### --min-frequency

Minimum number of occurrences for a pattern (default: 1)

```bash
conduit introspect patterns --min-frequency 3
```

Filters out patterns that appear less than the specified number of times.

### Output Format

**Table format (default)**:

Groups patterns by category and shows:
- Pattern name
- Frequency (number of uses)
- Confidence score (0.0-1.0)
- Template (code structure)
- Examples (up to 5)
- "When to use" guidance

Example output:
```
PATTERNS

Authentication (3 patterns, 85% coverage):

  1. authenticated_handler (12 uses, confidence: 1.0)

     Template:
     @on <operation>: [auth]

     Used by:
     • Post  /app/resources/post.cdt
     • User  /app/resources/user.cdt
     [... and 10 more]

     When to use:
     Endpoints requiring user authentication
```

**JSON format** (`--format json`):

```json
{
  "total_count": 15,
  "patterns": [
    {
      "id": "uuid-here",
      "name": "authenticated_handler",
      "category": "authentication",
      "description": "Handler with auth middleware",
      "template": "@on <operation>: [auth]",
      "examples": [
        {
          "resource": "Post",
          "file_path": "/app/resources/post.cdt",
          "line_number": 12,
          "code": "@on create: [auth]"
        }
      ],
      "frequency": 12,
      "confidence": 1.0
    }
  ]
}
```

### Examples

```bash
# Show all patterns
conduit introspect patterns

# Show patterns for a specific category
conduit introspect patterns authentication
conduit introspect patterns caching
conduit introspect patterns hook

# Filter by minimum frequency (common patterns only)
conduit introspect patterns --min-frequency 3
conduit introspect patterns --min-frequency 5

# Combine filters (common auth patterns)
conduit introspect patterns authentication --min-frequency 3

# JSON output for tooling
conduit introspect patterns --format json

# Very common patterns across all categories
conduit introspect patterns --min-frequency 10
```

### Pattern Categories

| Category | Description | Example |
|----------|-------------|---------|
| `authentication` | Auth-related patterns | `[auth]`, `[auth, admin]` |
| `authorization` | Permission checks | `[owner_or_admin]` |
| `caching` | Cache middleware | `[cache]`, `[cache(300)]` |
| `rate_limiting` | Rate limit patterns | `[rate_limit]` |
| `hook` | Lifecycle hook patterns | Slug generation, timestamps |
| `validation` | Validation patterns | Email, phone, custom rules |
| `middleware` | General middleware chains | Multiple middleware combos |
| `general` | Other patterns | Miscellaneous |

### Confidence Scores

Confidence is calculated based on frequency:
- `frequency / 10.0`, capped at 1.0
- `0.3-0.5`: Emerging pattern
- `0.5-0.8`: Common pattern
- `0.8-1.0`: Very common, high confidence

### Coverage Percentage

Shows what percentage of resources use patterns in each category:
- Calculated as: `(unique resources using patterns) / (total resources) * 100`
- High coverage (70%+) indicates consistent usage
- Low coverage (<30%) may indicate optional patterns

### Common Use Cases

- **Code generation**: Use templates for new code
- **Consistency checking**: Verify code follows patterns
- **Documentation**: Understand project conventions
- **Onboarding**: Learn project patterns quickly
- **LLM training**: Teach AI project-specific patterns

---

## Exit Codes

All introspect commands use standard exit codes:

- `0` - Success
- `1` - Error (with message to stderr)

## Output Redirection

### Save to File

```bash
# Save table output
conduit introspect resources > resources.txt

# Save JSON output
conduit introspect resources --format json > resources.json
```

### Pipe to Other Commands

```bash
# Use jq to process JSON
conduit introspect resources --format json | jq '.resources[] | .name'

# Count resources
conduit introspect resources --format json | jq '.total_count'

# Filter routes with grep
conduit introspect routes | grep GET
```

## Performance Notes

- **Registry initialization**: <1ms for typical applications
- **Simple queries** (resources, routes): Sub-microsecond
- **Complex queries** (deps with depth 3): <10µs cold, <1µs cached
- **Pattern extraction**: Cached after first query

Large applications (100+ resources) remain fast due to pre-computed indexes.

## Error Messages

### Registry Not Initialized

```
Error: registry not initialized - run 'conduit build' first to generate metadata
```

**Solution**: Run `conduit build` to compile your application and generate metadata.

### Resource Not Found

```
Error: resource not found: Pst

Did you mean?
  - Post
```

**Solution**: Check spelling and use suggested names.

### Invalid Flags

```
Error: depth must be between 1 and 5, got: 10
```

**Solution**: Adjust flag values to be within valid ranges.

## See Also

- [User Guide](user-guide.md) - Common workflows and practical examples
- [API Reference](api-reference.md) - Go API for programmatic access
- [Tutorial](tutorial/01-basic-queries.md) - Step-by-step walkthrough
- [Examples](../../examples/introspection/) - Working code samples
