# Dependency Analyzer Example

A command-line tool that analyzes dependencies between resources, detects circular dependencies, and calculates complexity metrics.

## What It Does

- Analyzes resource dependencies (forward and reverse)
- Detects circular dependencies
- Calculates dependency depth (complexity metric)
- Shows impact analysis for deletions
- Generates dependency reports

## Usage

```bash
# Build the tool
go build -o dependency-analyzer

# Analyze a specific resource
./dependency-analyzer Post

# Check for circular dependencies
./dependency-analyzer --check-cycles

# Generate full report
./dependency-analyzer --report

# Export as JSON
./dependency-analyzer --report --format json > deps-report.json
```

## Example Output

```
DEPENDENCY ANALYSIS

Analyzing resource dependencies...

=== CIRCULAR DEPENDENCIES ===
⚠️  Found 1 circular dependency:
  Cycle 1: User -> Profile -> User

=== DEPENDENCY COMPLEXITY ===

High Complexity (depth > 3):
  ⚠️  Order (depth: 4)
      Order -> User -> Profile -> Setting -> Config

Medium Complexity (depth 2-3):
  ➜ Post (depth: 2)
      Post -> User -> Profile
  ➜ Comment (depth: 3)
      Comment -> Post -> User -> Profile

Low Complexity (depth 0-1):
  ✓ User (depth: 1)
  ✓ Category (depth: 0)

=== DEPENDENCY IMPACT ===

Most Depended On (high impact if changed):
  1. User (5 resources depend on it)
     - Post (on_delete: restrict)
     - Comment (on_delete: cascade)
     - Order (on_delete: restrict)
     - Review (on_delete: cascade)
     - Session (on_delete: cascade)

  2. Post (2 resources depend on it)
     - Comment (on_delete: cascade)
     - Favorite (on_delete: cascade)
```

## Features

### 1. Circular Dependency Detection

Finds cycles in the dependency graph:

```bash
./dependency-analyzer --check-cycles
```

### 2. Complexity Analysis

Calculates dependency depth for each resource:

```bash
./dependency-analyzer --complexity
```

### 3. Impact Analysis

Shows what depends on each resource:

```bash
./dependency-analyzer --impact
```

### 4. Resource-Specific Analysis

Analyze a single resource:

```bash
./dependency-analyzer Post
```

Output:
```
DEPENDENCIES: Post

Direct Dependencies (what Post uses):
  → User (belongs_to)
    Impact: Post requires User
    On delete: restrict (cannot delete User with existing Posts)

Reverse Dependencies (what uses Post):
  ← Comment (belongs_to)
    Impact: Deleting Post cascades to Comments

  Routes using Post:
    GET    /posts
    POST   /posts
    GET    /posts/:id
    PUT    /posts/:id
    DELETE /posts/:id
```

## Code Overview

The tool demonstrates:
- Building complete dependency graphs
- Cycle detection using DFS
- Calculating dependency metrics
- Impact analysis for deletions
- Graph traversal algorithms

## Learning Points

1. **Graph Algorithms**: DFS for cycle detection
2. **Dependency Analysis**: Understanding relationship impact
3. **Metrics Calculation**: Measuring code complexity
4. **Report Generation**: Presenting analysis results

## Implementation Details

### Cycle Detection Algorithm

Uses depth-first search (DFS) with a recursion stack:

```go
func detectCycles(graph *metadata.DependencyGraph) [][]string {
    visited := make(map[string]bool)
    recStack := make(map[string]bool)
    var cycles [][]string

    for nodeID := range graph.Nodes {
        if !visited[nodeID] {
            findCycles(graph, nodeID, visited, recStack, []string{}, &cycles)
        }
    }

    return cycles
}
```

### Complexity Calculation

Uses BFS to calculate maximum depth:

```go
func calculateDepth(resource string) int {
    maxDepth := 0
    queue := []depthNode{{id: resource, depth: 0}}

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        if current.depth > maxDepth {
            maxDepth = current.depth
        }

        // Add children to queue
        // ...
    }

    return maxDepth
}
```

## Next Steps

- See [pattern-validator](../pattern-validator/) for pattern validation
- See [api-doc-generator](../api-doc-generator/) for documentation generation
- See [schema-explorer](../schema-explorer/) for interactive exploration
