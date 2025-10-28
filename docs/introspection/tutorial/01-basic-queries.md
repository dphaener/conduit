# Tutorial 1: Basic Queries

Welcome to the Conduit introspection tutorial series! In this first tutorial, you'll learn how to query basic information about your application's resources, routes, and patterns.

## Prerequisites

- Conduit CLI installed
- A Conduit application (we'll use a blog application as our example)
- Application compiled with `conduit build` (this generates the metadata)

## Tutorial Application: Simple Blog

Throughout this tutorial series, we'll work with a simple blog application. Here's the structure:

```conduit
/// Blog post with title and content
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  published: boolean! @default(false)

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }
}

/// User account
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  bio: text?

  @on create: [auth_required]
  @on update: [auth_required, owner_only]
}

/// Comment on a blog post
resource Comment {
  id: uuid! @primary @auto
  body: text! @min(10) @max(1000)

  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
  }

  @on create: [auth_required, rate_limit(5/hour)]
}
```

## Step 1: List All Resources

The most basic query is listing all resources in your application:

```bash
conduit introspect resources
```

**Expected Output:**

```
RESOURCES (3 total)

Core Resources:
  Post          5 fields  1 relationship   1 hook
  User          4 fields  -                 -        ✓ auth required
  Comment       2 fields  2 relationships  -        ✓ auth required  ✓ rate_limited

```

**What you're seeing:**
- **Field count**: Number of fields in each resource
- **Relationship count**: Number of relationships (belongs_to, has_many)
- **Hook count**: Number of lifecycle hooks
- **Flags**: Special features like authentication, caching, or nesting

## Step 2: Inspect a Specific Resource

Get detailed information about a single resource:

```bash
conduit introspect resource Post
```

**Expected Output:**

```
RESOURCE: Post
File: resources/post.cdt

━━━ SCHEMA ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

FIELDS (5):
Required (5):
  id  uuid  @primary @auto
  title  string  @min(5) @max(200)
  slug  string  @unique
  content  text  @min(100)
  published  boolean  (default: false)

RELATIONSHIPS (1):
  → author (belongs_to User)
    Foreign key: author_id
    On delete: restrict

━━━ BEHAVIOR ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

LIFECYCLE HOOKS:
  @before_create:
    self.slug = String.slugify(self.title)

━━━ API ENDPOINTS ━━━━━━━━━━━━━━━━━━━━━━━━━

GET /posts → list
POST /posts → create
GET /posts/:id → show
PUT /posts/:id → update
DELETE /posts/:id → delete
```

**What you're seeing:**
- **Schema**: All fields with their types and constraints
- **Relationships**: Foreign key relationships to other resources
- **Behavior**: Hooks that run during lifecycle events
- **API Endpoints**: Auto-generated REST routes

## Step 3: Query Routes

List all HTTP routes in your application:

```bash
conduit introspect routes
```

**Expected Output:**

```
GET    /posts                       -> PostHandler.List
POST   /posts                       -> PostHandler.Create
GET    /posts/:id                   -> PostHandler.Show
PUT    /posts/:id                   -> PostHandler.Update
DELETE /posts/:id                   -> PostHandler.Delete
GET    /users                       -> UserHandler.List      [auth_required]
POST   /users                       -> UserHandler.Create    [auth_required]
PUT    /users/:id                   -> UserHandler.Update    [auth_required, owner_only]
POST   /comments                    -> CommentHandler.Create [auth_required, rate_limit]
```

**Color coding:**
- **GET** routes in green
- **POST** routes in blue
- **PUT** routes in yellow
- **DELETE** routes in red

## Step 4: Filter Routes

You can filter routes by method, resource, or middleware:

```bash
# Only GET routes
conduit introspect routes --method GET

# Only routes for Post resource
conduit introspect routes --resource Post

# Only routes with auth middleware
conduit introspect routes --middleware auth
```

## Step 5: Using JSON Output

All introspect commands support JSON output for tooling:

```bash
conduit introspect resources --format json
```

**Example JSON Output:**

```json
{
  "total_count": 3,
  "resources": [
    {
      "name": "Post",
      "field_count": 5,
      "relationship_count": 1,
      "hook_count": 1,
      "validation_count": 3,
      "constraint_count": 0,
      "category": "Core Resources",
      "flags": []
    },
    {
      "name": "User",
      "field_count": 4,
      "relationship_count": 0,
      "hook_count": 0,
      "validation_count": 1,
      "constraint_count": 0,
      "middleware": {
        "create": ["auth_required"],
        "update": ["auth_required", "owner_only"]
      },
      "category": "Core Resources",
      "flags": ["auth_required"]
    }
  ]
}
```

## Step 6: Using the Go API

You can also query introspection data programmatically from Go code:

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    // Get the registry
    registry := metadata.GetRegistry()

    // List all resources
    resources := registry.Resources()
    fmt.Printf("Found %d resources:\n", len(resources))

    for _, res := range resources {
        fmt.Printf("  - %s (%d fields)\n", res.Name, len(res.Fields))
    }

    // Query a specific resource
    post, err := registry.Resource("Post")
    if err != nil {
        panic(err)
    }

    fmt.Printf("\nPost resource has:\n")
    fmt.Printf("  - %d fields\n", len(post.Fields))
    fmt.Printf("  - %d relationships\n", len(post.Relationships))
    fmt.Printf("  - %d hooks\n", len(post.Hooks))

    // Query routes for Post
    routes := registry.Routes(metadata.RouteFilter{
        Resource: "Post",
    })

    fmt.Printf("  - %d routes\n", len(routes))
}
```

## Common Use Cases

### 1. Verify Resource Schema

Check that your resource has the expected fields:

```bash
conduit introspect resource User | grep "FIELDS"
```

### 2. Find All Authenticated Routes

```bash
conduit introspect routes --middleware auth
```

### 3. Generate Resource Summary

```bash
conduit introspect resources --verbose
```

### 4. Export Schema as JSON

```bash
conduit introspect resources --format json > schema.json
```

## Understanding the Output

### Field Types

- `string!` - Required string
- `string?` - Optional string
- `uuid!` - Required UUID (often for IDs)
- `text!` - Required text (longer than string)
- `integer!` - Required integer
- `boolean!` - Required boolean

### Constraints

- `@primary` - Primary key
- `@auto` - Auto-generated value
- `@unique` - Must be unique
- `@min(N)` - Minimum value/length
- `@max(N)` - Maximum value/length
- `@default(value)` - Default value

### Relationship Types

- `belongs_to` - Many-to-one relationship
- `has_many` - One-to-many relationship
- `has_many_through` - Many-to-many via join table

## Exercises

Try these exercises to reinforce what you've learned:

1. **List all resources** in your application
2. **Inspect the User resource** and count how many fields it has
3. **Find all POST routes** using the --method filter
4. **Export all routes as JSON** and save to a file
5. **Query a resource programmatically** using the Go API

## Next Steps

In [Tutorial 2: Dependency Analysis](02-dependency-analysis.md), you'll learn how to:
- Discover resource dependencies
- Analyze impact of changes
- Detect circular dependencies
- Visualize dependency graphs

## Troubleshooting

**"Registry not initialized"**
- Run `conduit build` first to generate metadata
- Make sure you're in a Conduit project directory

**"Resource not found"**
- Check spelling (resource names are case-sensitive)
- Run `conduit introspect resources` to see all available resources

**Empty output**
- Verify your application has resources defined
- Check that `conduit build` completed successfully
- Look for `build/app.meta.json` file

## Key Takeaways

- Use `conduit introspect resources` to list all resources
- Use `conduit introspect resource <name>` for detailed info
- Use `conduit introspect routes` to see all HTTP endpoints
- Add `--format json` for machine-readable output
- Filter results with flags like `--method`, `--resource`, `--middleware`
- Query programmatically using the Go API

**Time to complete:** ~15 minutes

---

[← Back to Overview](../README.md) | [Next: Dependency Analysis →](02-dependency-analysis.md)
