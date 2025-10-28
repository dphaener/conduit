# Tutorial 2: Dependency Analysis

In this tutorial, you'll learn how to analyze dependencies between resources, understand the impact of changes, and detect potential issues like circular dependencies.

## Why Dependency Analysis Matters

Understanding dependencies is crucial for:
- **Impact analysis**: Know what breaks when you change a resource
- **Refactoring**: Safely restructure your application
- **Onboarding**: Understand how resources relate to each other
- **Debugging**: Trace issues through relationship chains

## Step 1: View Direct Dependencies

Show what a resource depends on (forward dependencies):

```bash
conduit introspect deps Post
```

**Expected Output:**

```
DEPENDENCIES: Post

━━━ DIRECT DEPENDENCIES (what Post uses) ━━━━━━

Resources:
└─ User (belongs_to)
   Impact: Post requires User

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

**What you're seeing:**
- **Direct dependencies**: Resources that Post depends on (User)
- **Reverse dependencies**: Resources that depend on Post (Comment)
- **Impact**: What happens when you modify or delete records
- **Routes**: HTTP endpoints that use this resource

## Step 2: Understanding Impact Analysis

The impact description tells you what happens when you make changes:

```bash
conduit introspect deps Comment
```

**Expected Output:**

```
DEPENDENCIES: Comment

━━━ DIRECT DEPENDENCIES (what Comment uses) ━━━━━━

Resources:
└─ Post (belongs_to)
   Impact: Comment requires Post

└─ User (belongs_to)
   Impact: Comment requires User

Middleware:
└─ auth_required
└─ rate_limit

━━━ REVERSE DEPENDENCIES (what uses Comment) ━━━━━━

Routes:
└─ POST /comments
```

**Key insights:**
- Comment requires both Post and User (can't create without them)
- Comment uses authentication and rate limiting middleware
- Only one route (create) exists for comments

## Step 3: Reverse Dependencies Only

See only what depends on a resource:

```bash
conduit introspect deps User --reverse
```

**Expected Output:**

```
DEPENDENCIES: User

━━━ REVERSE DEPENDENCIES (what uses User) ━━━━━━

Resources:
└─ Post (via belongs_to to User)
   Impact: Cannot delete User with existing Post

└─ Comment (via belongs_to to User)
   Impact: Deleting User cascades to Comment

Routes:
└─ GET /users
└─ POST /users
└─ PUT /users/:id
```

**Key insights:**
- Deleting a User is restricted if they have Posts (on_delete: restrict)
- Deleting a User cascades to their Comments (on_delete: cascade)
- This helps you understand data integrity constraints

## Step 4: Traversal Depth

Control how deep to traverse the dependency tree:

```bash
# Only direct dependencies (depth 1)
conduit introspect deps Post --depth 1

# Go 2 levels deep
conduit introspect deps Post --depth 2
```

**Example with depth 2:**

```
DEPENDENCIES: Post

━━━ DIRECT DEPENDENCIES (what Post uses) ━━━━━━

Resources:
└─ User (belongs_to)
   └─ Session (belongs_to)  # Depth 2
   └─ Profile (has_one)     # Depth 2
```

Maximum depth is 5 to prevent excessive traversal.

## Step 5: Filter by Dependency Type

Focus on specific types of dependencies:

```bash
# Only resource relationships
conduit introspect deps Post --type resource

# Only middleware dependencies
conduit introspect deps Post --type middleware

# Only function calls
conduit introspect deps Post --type function
```

**Resource dependencies:**
```
Resources:
└─ User (belongs_to)
```

**Middleware dependencies:**
```
Middleware:
└─ auth_required
└─ owner_only
```

**Function dependencies:**
```
Functions:
└─ String.slugify
```

## Step 6: Using the Go API

Query dependencies programmatically:

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()

    // Query dependencies with options
    graph, err := registry.Dependencies("Post", metadata.DependencyOptions{
        Depth:   2,
        Reverse: false,
        Types:   []string{"belongs_to", "has_many"},
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Dependency graph for Post:\n")
    fmt.Printf("  Nodes: %d\n", len(graph.Nodes))
    fmt.Printf("  Edges: %d\n", len(graph.Edges))

    // Iterate over edges
    for _, edge := range graph.Edges {
        fromNode := graph.Nodes[edge.From]
        toNode := graph.Nodes[edge.To]
        fmt.Printf("  %s --%s--> %s\n",
            fromNode.Name, edge.Relationship, toNode.Name)
    }
}
```

## Common Patterns

### 1. Pre-Deletion Impact Analysis

Before deleting a resource, check what will be affected:

```bash
# What depends on User?
conduit introspect deps User --reverse
```

If you see:
- `on_delete: cascade` - Records will be automatically deleted
- `on_delete: restrict` - Deletion will fail if records exist
- `on_delete: set_null` - Foreign key will be set to null

### 2. Refactoring Safety Check

Before changing a resource, understand its blast radius:

```bash
# Full dependency tree (forward and reverse)
conduit introspect deps Post

# Export as JSON for analysis
conduit introspect deps Post --format json > post-deps.json
```

### 3. Find Circular Dependencies

Circular dependencies can cause issues. Check your dependency graph:

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    fmt.Println("Checking for circular dependencies...")

    // Check each resource for circular paths
    for _, res := range resources {
        // Get dependency graph with unlimited depth
        graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
            Depth:   0, // 0 means unlimited depth
            Reverse: false,
        })
        if err != nil {
            continue
        }

        // Look for cycles by checking if any dependency path loops back
        for _, edge := range graph.Edges {
            if edge.To == res.Name && edge.From != res.Name {
                fmt.Printf("  ⚠️  Circular dependency detected involving %s\n", res.Name)
                break
            }
        }
    }

    fmt.Println("Circular dependency check complete.")
}
```

### 4. Calculate Dependency Depth

Find resources with deep dependency chains (potential complexity):

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    fmt.Println("Resource complexity analysis:")
    for _, res := range resources {
        // Get full dependency graph
        graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
            Depth:   0, // 0 means unlimited depth
            Reverse: false,
        })
        if err != nil {
            continue
        }

        // Calculate depth by counting unique levels in the graph
        depth := calculateGraphDepth(graph, res.Name)

        if depth > 3 {
            fmt.Printf("  ⚠️  %s (depth: %d)\n", res.Name, depth)
        } else {
            fmt.Printf("  ✓ %s (depth: %d)\n", res.Name, depth)
        }
    }
}

func calculateGraphDepth(graph *metadata.DependencyGraph, startNode string) int {
    if len(graph.Edges) == 0 {
        return 0
    }

    // Simple depth calculation: count maximum edge chain length
    maxDepth := 0
    visited := make(map[string]int)

    var dfs func(node string, depth int)
    dfs = func(node string, depth int) {
        if depth > maxDepth {
            maxDepth = depth
        }
        visited[node] = depth

        for _, edge := range graph.Edges {
            if edge.From == node {
                if _, seen := visited[edge.To]; !seen {
                    dfs(edge.To, depth+1)
                }
            }
        }
    }

    dfs(startNode, 0)
    return maxDepth
}
```

## Real-World Example: Blog Refactoring

Let's say you want to make User optional on Post (allow anonymous posts):

### Step 1: Check Current Dependencies

```bash
conduit introspect deps Post
```

Output shows:
```
Resources:
└─ User (belongs_to)
   Impact: Post requires User
```

### Step 2: Check Reverse Dependencies

```bash
conduit introspect deps User --reverse
```

Output shows:
```
Resources:
└─ Post (via belongs_to to User)
   Impact: Cannot delete User with existing Post
```

### Step 3: Analyze Impact

From the analysis, you know:
- **Current**: Post requires User (belongs_to User!)
- **Change**: You want to make it optional (belongs_to User?)
- **Impact**:
  - Existing Posts will still have authors
  - New Posts can be created without authors
  - Deletion behavior needs review (what happens to anonymous posts?)

### Step 4: Make the Change Safely

1. Update the relationship: `author: User?` (make optional)
2. Update on_delete behavior if needed
3. Rebuild: `conduit build`
4. Verify: `conduit introspect resource Post`

## Understanding On-Delete Behaviors

### CASCADE
```
Impact: Deleting User cascades to Comment
```
When you delete a User, all their Comments are automatically deleted.

### RESTRICT
```
Impact: Cannot delete User with existing Post
```
You cannot delete a User if they have any Posts. Delete Posts first.

### SET_NULL
```
Impact: Deleting User nullifies Post.author_id
```
When you delete a User, their Posts remain but author_id is set to NULL.

## Exercises

1. **Analyze the User resource**: What depends on it?
2. **Check Comment dependencies**: What resources does it require?
3. **Find cascade deletions**: Which relationships use `on_delete: cascade`?
4. **Calculate complexity**: Use the Go API to find resources with depth > 2
5. **Simulate deletion**: Check what would happen if you deleted User

## Next Steps

In [Tutorial 3: Pattern Discovery](03-pattern-discovery.md), you'll learn how to:
- Discover common patterns in your codebase
- Understand pattern categories
- Use patterns for code generation
- Apply patterns consistently

## Troubleshooting

**"Resource not found"**
- Verify the resource name is correct (case-sensitive)
- Run `conduit introspect resources` to list all resources

**"Depth must be between 1 and 5"**
- Use `--depth 1` to `--depth 5`
- Omit `--depth` to use default (1)

**Empty dependencies**
- Resource may not have relationships
- Check with `conduit introspect resource <name>`

**"Invalid type filter"**
- Valid types: `resource`, `middleware`, `function`
- Type is singular, not plural

## Key Takeaways

- Use `conduit introspect deps <resource>` to see dependencies
- Add `--reverse` to see what depends on a resource
- Use `--depth N` to control traversal depth
- Use `--type` to filter by dependency type
- Understand on_delete behaviors: cascade, restrict, set_null
- Check dependencies before refactoring or deleting
- Use the Go API for programmatic dependency analysis

**Time to complete:** ~20 minutes

---

[← Previous: Basic Queries](01-basic-queries.md) | [Back to Overview](../README.md) | [Next: Pattern Discovery →](03-pattern-discovery.md)
