# Introspection User Guide

This guide covers common workflows and practical examples for using the Conduit introspection system.

## Table of Contents

- [Getting Started](#getting-started)
- [Exploring Resources](#exploring-resources)
- [Understanding Routes](#understanding-routes)
- [Analyzing Dependencies](#analyzing-dependencies)
- [Discovering Patterns](#discovering-patterns)
- [Common Workflows](#common-workflows)
- [Programmatic Access](#programmatic-access)

## Getting Started

### Prerequisites

Before using introspection, you need a compiled Conduit application:

```bash
# Build your application to generate metadata
conduit build

# Now you can introspect
conduit introspect resources
```

The build process generates metadata that the introspection system uses. Without a build, you'll see:

```
Error: registry not initialized - run 'conduit build' first to generate metadata
```

### Basic Commands

Start with these fundamental commands:

```bash
# List all resources
conduit introspect resources

# Get details about a specific resource
conduit introspect resource Post

# List all routes
conduit introspect routes

# Show dependencies
conduit introspect deps Post

# Discover patterns
conduit introspect patterns
```

## Exploring Resources

### List All Resources

Get an overview of all resources in your application:

```bash
conduit introspect resources
```

**Example output:**

```
RESOURCES (4 total)

Core Resources:
  User          7 fields  2 relationships  3 hooks  ✓ auth required
  Post          9 fields  3 relationships  4 hooks  ✓ auth required  ✓ cached
  Comment       5 fields  2 relationships  1 hook   ✓ auth required  ✓ nested
  Category      3 fields  1 relationship   0 hooks
```

### Verbose Listing

Show more details about each resource:

```bash
conduit introspect resources --verbose
```

**Example output:**

```
RESOURCES (4 total)

Core Resources:
  User
    Fields: 7
    Relationships: 2
    Hooks: 3
    Flags: auth required

  Post
    Fields: 9
    Relationships: 3
    Hooks: 4
    Flags: auth required, cached
```

### View Resource Details

Get complete information about a specific resource:

```bash
conduit introspect resource Post
```

**Example output:**

```
RESOURCE: Post
File: /app/resources/post.cdt
Docs: Blog post with title, content, and metadata

━━━ SCHEMA ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

FIELDS (9):
Required (6):
  id  uuid  @primary @auto
  title  string  @min(5) @max(200)
  slug  string  @unique
  content  text  @min(100)
  author_id  uuid
  created_at  timestamp  @auto

Optional (3):
  published_at  timestamp?
  updated_at  timestamp?
  deleted_at  timestamp?

RELATIONSHIPS (3):
  → author (belongs_to User)
    Foreign key: author_id
    On delete: restrict

  → comments (has_many Comment)
    Foreign key: post_id
    On delete: cascade

  → categories (has_many_through Category)
    Through: post_categories

━━━ BEHAVIOR ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

LIFECYCLE HOOKS:
  @before_create [transaction]:
    self.slug = String.slugify(self.title)

  @after_create [async]:
    Notifications.notify(self.author, "post_created", self)

CONSTRAINTS (2):
  ✓ published_requires_content
    Operations: create, update
    Condition: self.status == "published"
    Error: Published posts need 500+ characters

  ✓ unique_slug_per_user
    Operations: create, update
    Condition: !Post.exists?(slug: self.slug, author_id: self.author_id)
    Error: You already have a post with this slug

━━━ API ENDPOINTS ━━━━━━━━━━━━━━━━━━━━━━━━

GET /posts → list [auth, cache]
POST /posts → create [auth, rate_limit]
GET /posts/:id → show [auth, cache]
PUT /posts/:id → update [auth]
DELETE /posts/:id → delete [auth]
```

### JSON Output

For tooling integration, use JSON format:

```bash
conduit introspect resource Post --format json
```

This outputs the complete ResourceMetadata structure for programmatic processing.

## Understanding Routes

### List All Routes

View all HTTP endpoints in your application:

```bash
conduit introspect routes
```

**Example output:**

```
GET    /users                 -> ListUsers            [auth, cache]
POST   /users                 -> CreateUser           [rate_limit]
GET    /users/:id             -> ShowUser             [auth]
PUT    /users/:id             -> UpdateUser           [auth, owner_or_admin]
DELETE /users/:id             -> DeleteUser           [auth, admin]
GET    /posts                 -> ListPosts            [cache]
POST   /posts                 -> CreatePost           [auth, rate_limit]
GET    /posts/:id             -> ShowPost             [cache]
PUT    /posts/:id             -> UpdatePost           [auth, owner]
DELETE /posts/:id             -> DeletePost           [auth, owner]
```

### Filter by HTTP Method

Find all endpoints for a specific HTTP method:

```bash
# All GET endpoints
conduit introspect routes --method GET

# All POST endpoints
conduit introspect routes --method POST
```

### Filter by Resource

View all routes for a specific resource:

```bash
conduit introspect routes --resource Post
```

**Example output:**

```
GET    /posts                 -> ListPosts            [cache]
POST   /posts                 -> CreatePost           [auth, rate_limit]
GET    /posts/:id             -> ShowPost             [cache]
PUT    /posts/:id             -> UpdatePost           [auth, owner]
DELETE /posts/:id             -> DeletePost           [auth, owner]
```

### Filter by Middleware

Find all routes using specific middleware:

```bash
# All routes requiring authentication
conduit introspect routes --middleware auth

# All cached routes
conduit introspect routes --middleware cache
```

### Combine Filters

You can combine filters to narrow down results:

```bash
# All GET routes for Post resource
conduit introspect routes --method GET --resource Post

# All authenticated GET routes
conduit introspect routes --method GET --middleware auth
```

## Analyzing Dependencies

### Direct Dependencies

See what a resource depends on:

```bash
conduit introspect deps Post
```

**Example output:**

```
DEPENDENCIES: Post

━━━ DIRECT DEPENDENCIES (what Post uses) ━━━━━━

Resources:
└─ User (belongs_to)
   Impact: Cannot delete User with existing Post

Middleware:
└─ auth
└─ rate_limit
└─ cache

Functions:
└─ String.slugify
└─ Notifications.notify

━━━ REVERSE DEPENDENCIES (what uses Post) ━━━━━━

Resources:
└─ Comment (via belongs_to to Post)
   Impact: Deleting Post cascades to Comment

Routes:
└─ GET /posts
└─ POST /posts
└─ GET /posts/:id
└─ PUT /posts/:id
└─ DELETE /posts/:id
```

### Reverse Dependencies Only

Focus on what depends on a resource:

```bash
conduit introspect deps Post --reverse
```

This shows only reverse dependencies (what uses Post).

### Deeper Dependency Trees

Traverse multiple levels of dependencies:

```bash
# Depth 2: Dependencies of dependencies
conduit introspect deps Post --depth 2

# Depth 3: Even deeper
conduit introspect deps Post --depth 3
```

**Note**: Maximum depth is 5 to prevent performance issues.

### Filter by Dependency Type

Focus on specific types of dependencies:

```bash
# Only resource relationships
conduit introspect deps Post --type resource

# Only middleware usage
conduit introspect deps Post --type middleware

# Only function calls
conduit introspect deps Post --type function
```

### Impact Analysis

Before deleting or modifying a resource, check its dependencies:

```bash
# What breaks if I delete User?
conduit introspect deps User --reverse

# What does Post depend on?
conduit introspect deps Post
```

Look for:
- **cascade**: Deletion will cascade to dependent resources
- **restrict**: Cannot delete with existing dependencies
- **set_null**: Deletion will null out foreign keys

## Discovering Patterns

### List All Patterns

See all discovered patterns in your codebase:

```bash
conduit introspect patterns
```

**Example output:**

```
PATTERNS

Authentication (3 patterns, 85% coverage):

  1. authenticated_handler (12 uses, confidence: 1.0)

     Template:
     @on <operation>: [auth]

     Used by:
     • Post  /app/resources/post.cdt
     • User  /app/resources/user.cdt
     • Comment  /app/resources/comment.cdt
     [... and 9 more]

     When to use:
     Endpoints requiring user authentication

  2. authenticated_rate_limited_handler (5 uses, confidence: 0.5)

     Template:
     @on <operation>: [auth, rate_limit]

     Used by:
     • Post  /app/resources/post.cdt:12
     • Comment  /app/resources/comment.cdt:8
     • User  /app/resources/user.cdt:15

     When to use:
     User-generated content creation needing spam protection

Caching (2 patterns, 60% coverage):

  1. cached_handler (8 uses, confidence: 0.8)

     Template:
     @on <operation>: [cache]

     Used by:
     • Post  /app/resources/post.cdt
     • Category  /app/resources/category.cdt
     [... and 6 more]

     When to use:
     Frequently accessed read-only data
```

### Filter by Category

Focus on patterns in a specific category:

```bash
# Authentication patterns
conduit introspect patterns authentication

# Caching patterns
conduit introspect patterns caching

# Hook patterns
conduit introspect patterns hook
```

### Filter by Frequency

Only show patterns that appear frequently:

```bash
# Patterns appearing at least 5 times
conduit introspect patterns --min-frequency 5

# Very common patterns (10+ uses)
conduit introspect patterns --min-frequency 10
```

### Using Patterns for Code Generation

Patterns help LLMs generate consistent code:

1. Query patterns before generating code
2. Use templates to match project conventions
3. Reference examples to understand context
4. Validate generated code against patterns

## Common Workflows

### Workflow 1: Onboarding to a New Codebase

**Goal**: Quickly understand an unfamiliar Conduit application

```bash
# Step 1: Get overview of all resources
conduit introspect resources

# Step 2: Understand the main resources
conduit introspect resource User
conduit introspect resource Post

# Step 3: See the API surface
conduit introspect routes

# Step 4: Discover coding conventions
conduit introspect patterns

# Step 5: Understand dependencies
conduit introspect deps Post
conduit introspect deps User
```

### Workflow 2: Adding a New Feature

**Goal**: Add a new resource that follows existing patterns

```bash
# Step 1: Find similar resources
conduit introspect resources

# Step 2: Study a similar resource
conduit introspect resource Post

# Step 3: Check middleware patterns
conduit introspect patterns middleware

# Step 4: Understand auth patterns
conduit introspect patterns authentication

# Step 5: Create new resource following patterns
# (use discovered patterns as templates)
```

### Workflow 3: Refactoring a Resource

**Goal**: Safely modify a resource without breaking dependents

```bash
# Step 1: Check what depends on this resource
conduit introspect deps Post --reverse

# Step 2: Understand impact of changes
# Look for cascade/restrict relationships

# Step 3: Check routes that use this resource
conduit introspect routes --resource Post

# Step 4: Make changes carefully
# (knowing all dependencies)

# Step 5: Verify nothing breaks
# (re-check dependencies after changes)
```

### Workflow 4: Performance Optimization

**Goal**: Find routes to optimize with caching

```bash
# Step 1: Find all GET routes (read-heavy)
conduit introspect routes --method GET

# Step 2: See which routes already use cache
conduit introspect routes --middleware cache

# Step 3: Find caching patterns
conduit introspect patterns caching

# Step 4: Apply caching to uncached GET routes
# (following discovered patterns)
```

### Workflow 5: Security Audit

**Goal**: Ensure all routes are properly protected

```bash
# Step 1: Find all routes
conduit introspect routes

# Step 2: Filter for unauthenticated routes
# (look for routes without [auth] middleware)

# Step 3: Check authentication patterns
conduit introspect patterns authentication

# Step 4: Verify auth is applied correctly
# (ensure no sensitive endpoints are unprotected)
```

### Workflow 6: Debugging Middleware Chains

**Goal**: Understand why a request behaves unexpectedly

```bash
# Step 1: Find the route in question
conduit introspect routes --resource Post

# Step 2: Check the middleware chain
# (look at middleware list for each route)

# Step 3: Compare with patterns
conduit introspect patterns middleware

# Step 4: Verify middleware order
# (order matters: auth before cache)
```

## Programmatic Access

For building tools, use the Go API:

### Basic Queries

```go
package main

import (
    "fmt"
    "log"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    // Get the registry
    registry := metadata.GetRegistry()

    // Query all resources
    resources := registry.Resources()
    fmt.Printf("Found %d resources\n", len(resources))

    // Query specific resource
    post, err := registry.Resource("Post")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Post has %d fields\n", len(post.Fields))
    fmt.Printf("Post has %d relationships\n", len(post.Relationships))
}
```

### Route Queries

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

// Get all authenticated routes
authRoutes := []metadata.RouteMetadata{}
for _, route := range allRoutes {
    for _, mw := range route.Middleware {
        if mw == "auth" {
            authRoutes = append(authRoutes, route)
            break
        }
    }
}
```

### Dependency Analysis

```go
// Get direct dependencies
deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
    Depth:   1,
    Reverse: false,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Post depends on %d resources\n", len(deps.Nodes)-1)

// Get reverse dependencies
reverseDeps, err := registry.Dependencies("User", metadata.DependencyOptions{
    Depth:   0, // Unlimited
    Reverse: true,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("%d resources depend on User\n", len(reverseDeps.Nodes)-1)
```

### Pattern Discovery

```go
// Get all patterns
allPatterns := registry.Patterns("")

// Get authentication patterns
authPatterns := registry.Patterns("authentication")

// Filter by frequency
frequentPatterns := []metadata.PatternMetadata{}
for _, pattern := range allPatterns {
    if pattern.Frequency >= 5 {
        frequentPatterns = append(frequentPatterns, pattern)
    }
}
```

### Complete Schema Access

```go
// Get the complete metadata structure
schema := registry.GetSchema()

if schema == nil {
    log.Fatal("Registry not initialized")
}

fmt.Printf("Schema version: %s\n", schema.Version)
fmt.Printf("Generated: %s\n", schema.Generated)
fmt.Printf("Source hash: %s\n", schema.SourceHash)
fmt.Printf("Total resources: %d\n", len(schema.Resources))
fmt.Printf("Total routes: %d\n", len(schema.Routes))
fmt.Printf("Total patterns: %d\n", len(schema.Patterns))
```

## Tips and Tricks

### Use Verbose Mode for Details

When you need more information, add `--verbose`:

```bash
conduit introspect resources --verbose
conduit introspect resource Post --verbose
```

### Use JSON for Processing

When building tools, use `--format json`:

```bash
conduit introspect resources --format json | jq '.resources[] | .name'
conduit introspect routes --format json | jq '.routes[] | select(.method == "GET")'
```

### Disable Color for Scripts

When using introspection in scripts:

```bash
conduit introspect resources --no-color
```

### Check Dependencies Before Changes

Always check reverse dependencies before modifying a resource:

```bash
conduit introspect deps Post --reverse
```

### Use Patterns for Consistency

When adding new code, check patterns first:

```bash
conduit introspect patterns authentication
conduit introspect patterns hook
```

## Next Steps

- Read the [CLI Reference](cli-reference.md) for complete command documentation
- Check out the [API Reference](api-reference.md) for Go API details
- Try the [Tutorial](tutorial/01-basic-queries.md) for hands-on learning
- Explore [Example Programs](../../examples/introspection/) for working code
- Review [Best Practices](best-practices.md) for effective usage
