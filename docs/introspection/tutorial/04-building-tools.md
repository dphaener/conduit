# Tutorial 4: Building Tools with Introspection

In this final tutorial, you'll learn how to build practical tools using the introspection API. We'll create several useful utilities that demonstrate the power of runtime introspection.

## Why Build Tools?

The introspection API enables you to build:
- **Documentation generators**: Auto-generate API docs, ER diagrams, etc.
- **Code generators**: Scaffold new resources from templates
- **Linters**: Enforce coding standards and best practices
- **Analyzers**: Detect issues like circular dependencies
- **Exporters**: Convert schema to other formats (OpenAPI, GraphQL, etc.)

## Tool Architecture

All introspection tools follow a common pattern:

```go
package main

import (
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    // 1. Get the registry
    registry := metadata.GetRegistry()

    // 2. Query metadata
    resources := registry.Resources()
    routes := registry.Routes(metadata.RouteFilter{})
    patterns := registry.Patterns("")

    // 3. Process and analyze
    // Your tool logic here

    // 4. Generate output
    // Write files, print reports, etc.
}
```

## Tool 1: API Documentation Generator

Let's build a tool that generates API documentation from routes and resources.

### Create the Tool

Create `tools/apidoc/main.go`:

```go
package main

import (
    "fmt"
    "os"
    "sort"
    "strings"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()

    // Get all routes
    routes := registry.Routes(metadata.RouteFilter{})

    // Sort by path
    sort.Slice(routes, func(i, j int) bool {
        return routes[i].Path < routes[j].Path
    })

    // Group by resource
    byResource := make(map[string][]metadata.RouteMetadata)
    for _, route := range routes {
        byResource[route.Resource] = append(byResource[route.Resource], route)
    }

    // Generate markdown
    fmt.Println("# API Documentation")
    fmt.Println("\nAuto-generated from Conduit introspection.\n")

    // Sort resource names
    resourceNames := make([]string, 0, len(byResource))
    for name := range byResource {
        resourceNames = append(resourceNames, name)
    }
    sort.Strings(resourceNames)

    for _, resourceName := range resourceNames {
        routes := byResource[resourceName]

        fmt.Printf("## %s\n\n", resourceName)

        // Get resource metadata
        resource, err := registry.Resource(resourceName)
        if err == nil && resource.Documentation != "" {
            fmt.Printf("%s\n\n", resource.Documentation)
        }

        // Document each route
        for _, route := range routes {
            fmt.Printf("### %s %s\n\n", route.Method, route.Path)
            fmt.Printf("**Operation:** %s\n\n", route.Operation)

            if len(route.Middleware) > 0 {
                fmt.Printf("**Middleware:** %s\n\n",
                    strings.Join(route.Middleware, ", "))
            }

            // Describe operation
            switch route.Operation {
            case "list":
                fmt.Println("Lists all records.\n")
                fmt.Printf("**Response:** Array of %s objects\n\n", resourceName)
            case "show":
                fmt.Println("Shows a single record by ID.\n")
                fmt.Printf("**Response:** Single %s object\n\n", resourceName)
            case "create":
                fmt.Println("Creates a new record.\n")
                fmt.Printf("**Request:** %s object\n", resourceName)
                fmt.Printf("**Response:** Created %s object\n\n", resourceName)
            case "update":
                fmt.Println("Updates an existing record.\n")
                fmt.Printf("**Request:** Partial %s object\n", resourceName)
                fmt.Printf("**Response:** Updated %s object\n\n", resourceName)
            case "delete":
                fmt.Println("Deletes a record.\n")
                fmt.Println("**Response:** 204 No Content\n\n")
            }

            // Show request/response schema if available
            if resource != nil {
                fmt.Println("**Schema:**")
                fmt.Println("```json")
                fmt.Println("{")
                for i, field := range resource.Fields {
                    comma := ","
                    if i == len(resource.Fields)-1 {
                        comma = ""
                    }
                    fmt.Printf("  \"%s\": \"%s\"%s\n",
                        field.Name, field.Type, comma)
                }
                fmt.Println("}")
                fmt.Println("```\n")
            }
        }
    }
}
```

### Run the Tool

```bash
cd tools/apidoc
go run main.go > ../../docs/API.md
```

### Example Output

```markdown
# API Documentation

Auto-generated from Conduit introspection.

## Post

Blog post with title and content

### GET /posts

**Operation:** list

Lists all records.

**Response:** Array of Post objects

**Schema:**
```json
{
  "id": "uuid",
  "title": "string",
  "slug": "string",
  "content": "text",
  "published": "boolean"
}
```

### POST /posts

**Operation:** create

**Middleware:** auth_required

Creates a new record.

**Request:** Post object
**Response:** Created Post object
```

## Tool 2: Dependency Visualizer

Build a tool that generates a dependency graph in DOT format (for Graphviz).

### Create the Tool

Create `tools/depgraph/main.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    // Build a complete dependency graph by collecting all resource dependencies
    allNodes := make(map[string]bool)
    allEdges := []struct {
        From string
        To   string
        Rel  string
    }{}

    // Collect all dependencies across all resources
    for _, res := range resources {
        allNodes[res.Name] = true

        graph, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
            Depth:   1, // Direct dependencies only for visualization
            Reverse: false,
        })
        if err != nil {
            continue
        }

        for _, edge := range graph.Edges {
            allNodes[edge.From] = true
            allNodes[edge.To] = true
            allEdges = append(allEdges, struct {
                From string
                To   string
                Rel  string
            }{edge.From, edge.To, edge.Relationship})
        }
    }

    // Generate DOT format
    fmt.Println("digraph Dependencies {")
    fmt.Println("  rankdir=LR;")
    fmt.Println("  node [shape=box];")
    fmt.Println()

    // Add nodes with colors (all resources for now)
    for nodeName := range allNodes {
        color := "lightgreen" // Default to resource color
        fmt.Printf("  \"%s\" [style=filled, fillcolor=%s];\n",
            nodeName, color)
    }

    fmt.Println()

    // Add edges with labels
    for _, edge := range allEdges {
        label := edge.Rel
        style := "solid"

        // Style by relationship type
        switch edge.Rel {
        case "belongs_to":
            style = "solid"
        case "has_many":
            style = "dashed"
        case "uses":
            style = "dotted"
        case "calls":
            style = "dotted"
        }

        fmt.Printf("  \"%s\" -> \"%s\" [label=\"%s\", style=%s];\n",
            edge.From, edge.To, label, style)
    }

    fmt.Println("}")
}
```

### Generate Graph

```bash
cd tools/depgraph
go run main.go > deps.dot
dot -Tpng deps.dot -o deps.png
```

This creates a visual dependency graph you can include in documentation!

## Tool 3: Pattern-Based Linter

Build a linter that checks if resources follow established patterns.

### Create the Tool

Create `tools/linter/main.go`:

```go
package main

import (
    "fmt"
    "os"
    "strings"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

type Rule struct {
    Name        string
    Description string
    Check       func(res metadata.ResourceMetadata) []string
}

var rules = []Rule{
    {
        Name:        "auth_on_mutations",
        Description: "Create/update/delete operations should require auth",
        Check: func(res metadata.ResourceMetadata) []string {
            var issues []string
            mutationOps := []string{"create", "update", "delete"}

            for _, op := range mutationOps {
                mw, exists := res.Middleware[op]
                if !exists {
                    continue
                }

                hasAuth := false
                for _, m := range mw {
                    if strings.Contains(m, "auth") {
                        hasAuth = true
                        break
                    }
                }

                if !hasAuth {
                    issues = append(issues,
                        fmt.Sprintf("  - %s operation missing auth", op))
                }
            }

            return issues
        },
    },
    {
        Name:        "rate_limit_on_creates",
        Description: "Create operations should have rate limiting",
        Check: func(res metadata.ResourceMetadata) []string {
            var issues []string
            createMW, exists := res.Middleware["create"]

            if !exists {
                return issues
            }

            hasRateLimit := false
            for _, mw := range createMW {
                if strings.Contains(mw, "rate_limit") {
                    hasRateLimit = true
                    break
                }
            }

            if !hasRateLimit {
                issues = append(issues,
                    "  - create operation missing rate_limit")
            }

            return issues
        },
    },
    {
        Name:        "slugs_on_titled_resources",
        Description: "Resources with 'title' field should have 'slug' field",
        Check: func(res metadata.ResourceMetadata) []string {
            var issues []string

            hasTitle := false
            hasSlug := false

            for _, field := range res.Fields {
                if field.Name == "title" {
                    hasTitle = true
                }
                if field.Name == "slug" {
                    hasSlug = true
                }
            }

            if hasTitle && !hasSlug {
                issues = append(issues,
                    "  - has 'title' but missing 'slug' field")
            }

            return issues
        },
    },
}

func main() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    fmt.Println("LINTING RESULTS")
    fmt.Println()

    totalIssues := 0

    for _, res := range resources {
        resourceIssues := []string{}

        // Run all rules
        for _, rule := range rules {
            issues := rule.Check(res)
            resourceIssues = append(resourceIssues, issues...)
        }

        if len(resourceIssues) > 0 {
            fmt.Printf("⚠️  %s:\n", res.Name)
            for _, issue := range resourceIssues {
                fmt.Println(issue)
            }
            fmt.Println()
            totalIssues += len(resourceIssues)
        }
    }

    if totalIssues == 0 {
        fmt.Println("✓ No issues found!")
        os.Exit(0)
    } else {
        fmt.Printf("Found %d issues across %d resources\n",
            totalIssues, len(resources))
        os.Exit(1)
    }
}
```

### Run the Linter

```bash
cd tools/linter
go run main.go
```

**Example Output:**

```
LINTING RESULTS

⚠️  Post:
  - create operation missing rate_limit

⚠️  Comment:
  - has 'title' but missing 'slug' field

Found 2 issues across 2 resources
```

## Tool 4: Schema Exporter (OpenAPI)

Build a tool that exports schema as OpenAPI 3.0 spec.

### Create the Tool

Create `tools/openapi/main.go`:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

type OpenAPI struct {
    OpenAPI    string                `json:"openapi"`
    Info       Info                  `json:"info"`
    Paths      map[string]PathItem   `json:"paths"`
    Components Components            `json:"components"`
}

type Info struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Version     string `json:"version"`
}

type PathItem map[string]Operation

type Operation struct {
    Summary     string              `json:"summary"`
    OperationID string              `json:"operationId"`
    Tags        []string            `json:"tags,omitempty"`
    Responses   map[string]Response `json:"responses"`
}

type Response struct {
    Description string `json:"description"`
}

type Components struct {
    Schemas map[string]Schema `json:"schemas"`
}

type Schema struct {
    Type       string            `json:"type"`
    Properties map[string]Property `json:"properties,omitempty"`
}

type Property struct {
    Type string `json:"type"`
}

func main() {
    registry := metadata.GetRegistry()

    // Initialize OpenAPI structure
    spec := OpenAPI{
        OpenAPI: "3.0.0",
        Info: Info{
            Title:       "Conduit API",
            Description: "Auto-generated from introspection",
            Version:     "1.0.0",
        },
        Paths:      make(map[string]PathItem),
        Components: Components{
            Schemas: make(map[string]Schema),
        },
    }

    // Get all routes
    routes := registry.Routes(metadata.RouteFilter{})

    // Process routes
    for _, route := range routes {
        if spec.Paths[route.Path] == nil {
            spec.Paths[route.Path] = make(PathItem)
        }

        method := strings.ToLower(route.Method)
        spec.Paths[route.Path][method] = Operation{
            Summary:     fmt.Sprintf("%s %s", route.Operation, route.Resource),
            OperationID: fmt.Sprintf("%s%s", route.Operation, route.Resource),
            Tags:        []string{route.Resource},
            Responses: map[string]Response{
                "200": {Description: "Successful response"},
            },
        }
    }

    // Process resources to generate schemas
    resources := registry.Resources()
    for _, res := range resources {
        schema := Schema{
            Type:       "object",
            Properties: make(map[string]Property),
        }

        for _, field := range res.Fields {
            propType := mapConduitTypeToJSON(field.Type)
            schema.Properties[field.Name] = Property{Type: propType}
        }

        spec.Components.Schemas[res.Name] = schema
    }

    // Output as JSON
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encoder.Encode(spec)
}

func mapConduitTypeToJSON(conduitType string) string {
    switch conduitType {
    case "uuid", "string", "text":
        return "string"
    case "integer":
        return "integer"
    case "boolean":
        return "boolean"
    default:
        return "string"
    }
}
```

### Export OpenAPI Spec

```bash
cd tools/openapi
go run main.go > ../../api-spec.json
```

You can now import this into Swagger UI, Postman, or other API tools!

## Tool 5: Interactive Schema Explorer

Build a terminal UI for exploring the schema interactively.

### Create the Tool

Create `tools/explorer/main.go`:

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()
    scanner := bufio.NewScanner(os.Stdin)

    fmt.Println("Conduit Schema Explorer")
    fmt.Println("Commands: list, show <resource>, routes, deps <resource>, exit")
    fmt.Println()

    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }

        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        parts := strings.Split(line, " ")
        command := parts[0]

        switch command {
        case "list":
            resources := registry.Resources()
            fmt.Printf("Found %d resources:\n", len(resources))
            for _, res := range resources {
                fmt.Printf("  - %s (%d fields)\n",
                    res.Name, len(res.Fields))
            }

        case "show":
            if len(parts) < 2 {
                fmt.Println("Usage: show <resource>")
                continue
            }

            resourceName := parts[1]
            res, err := registry.Resource(resourceName)
            if err != nil {
                fmt.Printf("Error: %v\n", err)
                continue
            }

            fmt.Printf("\nResource: %s\n", res.Name)
            fmt.Printf("Fields: %d\n", len(res.Fields))
            for _, field := range res.Fields {
                fmt.Printf("  - %s: %s\n", field.Name, field.Type)
            }

        case "routes":
            routes := registry.Routes(metadata.RouteFilter{})
            fmt.Printf("Found %d routes:\n", len(routes))
            for _, route := range routes {
                fmt.Printf("  %s %s -> %s\n",
                    route.Method, route.Path, route.Operation)
            }

        case "deps":
            if len(parts) < 2 {
                fmt.Println("Usage: deps <resource>")
                continue
            }

            resourceName := parts[1]
            graph, err := registry.Dependencies(resourceName,
                metadata.DependencyOptions{Depth: 1})
            if err != nil {
                fmt.Printf("Error: %v\n", err)
                continue
            }

            fmt.Printf("\nDependencies for %s:\n", resourceName)
            fmt.Printf("  Nodes: %d\n", len(graph.Nodes))
            fmt.Printf("  Edges: %d\n", len(graph.Edges))

        case "exit", "quit":
            fmt.Println("Goodbye!")
            return

        default:
            fmt.Printf("Unknown command: %s\n", command)
        }

        fmt.Println()
    }
}
```

### Run the Explorer

```bash
cd tools/explorer
go run main.go
```

**Example Session:**

```
Conduit Schema Explorer
Commands: list, show <resource>, routes, deps <resource>, exit

> list
Found 3 resources:
  - Post (5 fields)
  - User (4 fields)
  - Comment (2 fields)

> show Post
Resource: Post
Fields: 5
  - id: uuid
  - title: string
  - slug: string
  - content: text
  - published: boolean

> deps Post
Dependencies for Post:
  Nodes: 2
  Edges: 1

> exit
Goodbye!
```

## Packaging Tools

To make tools easy to use, create a `Makefile`:

```makefile
.PHONY: all apidoc depgraph lint openapi

all: apidoc depgraph lint openapi

apidoc:
	@cd tools/apidoc && go run main.go > ../../docs/API.md
	@echo "Generated docs/API.md"

depgraph:
	@cd tools/depgraph && go run main.go > deps.dot
	@dot -Tpng deps.dot -o ../../docs/dependencies.png
	@echo "Generated docs/dependencies.png"

lint:
	@cd tools/linter && go run main.go

openapi:
	@cd tools/openapi && go run main.go > ../../api-spec.json
	@echo "Generated api-spec.json"
```

Now you can run:

```bash
make apidoc     # Generate API docs
make depgraph   # Generate dependency graph
make lint       # Run linter
make openapi    # Export OpenAPI spec
make all        # Run everything
```

## Exercises

1. **Build a migration generator**: Create SQL migrations from resource schemas
2. **Build a test generator**: Generate test cases from patterns
3. **Build a route tester**: Verify all routes are tested
4. **Build a changelog generator**: Detect schema changes between versions
5. **Build a metrics collector**: Track resource complexity metrics

## Best Practices

1. **Cache metadata**: The registry is read-only, cache it
2. **Handle errors gracefully**: Registry may not be initialized
3. **Use type-safe queries**: Leverage the Go API, not string parsing
4. **Generate idiomatic output**: Match conventions of target format
5. **Add CLI flags**: Make tools configurable
6. **Write tests**: Test your tools like any other code
7. **Document usage**: Add --help flags and README files

## Key Takeaways

- Introspection enables powerful tooling
- All tools follow the same pattern: query → process → output
- Use the registry API, not CLI parsing
- JSON output enables tool composition
- Build incrementally and test thoroughly
- Package tools for easy reuse

**Time to complete:** ~35 minutes

---

[← Previous: Pattern Discovery](03-pattern-discovery.md) | [Back to Overview](../README.md)

## Congratulations!

You've completed the Conduit introspection tutorial series. You now know how to:

- Query resources, routes, and patterns (Tutorial 1)
- Analyze dependencies and impact (Tutorial 2)
- Discover and apply patterns (Tutorial 3)
- Build powerful tools with the introspection API (Tutorial 4)

## Next Steps

- Read the [Best Practices Guide](../best-practices.md)
- Explore the [API Reference](../api-reference.md)
- Check out the [Example Programs](../../../examples/introspection/)
- Build your own introspection tools!

Happy coding!
