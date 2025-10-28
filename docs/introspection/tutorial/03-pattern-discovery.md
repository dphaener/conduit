# Tutorial 3: Pattern Discovery

In this tutorial, you'll learn how Conduit's pattern discovery system identifies common patterns in your codebase and how to use these patterns to maintain consistency and generate code.

## Why Pattern Discovery?

Pattern discovery helps you:
- **Maintain consistency**: Follow established patterns automatically
- **Onboard faster**: New developers see how things are done
- **Generate code**: Use discovered patterns as templates
- **Train LLMs**: Patterns teach AI about your conventions
- **Document conventions**: Automatic documentation of coding standards

## How Pattern Discovery Works

Conduit analyzes your compiled application and extracts patterns from:
- Middleware chains (authentication, caching, rate limiting)
- Lifecycle hooks (before_create, after_update, etc.)
- Validation rules
- Relationship configurations

Each pattern has:
- **Name**: Descriptive identifier
- **Category**: Type of pattern (authentication, caching, etc.)
- **Template**: Code template for the pattern
- **Examples**: Real usage from your codebase
- **Frequency**: How often it appears
- **Confidence**: How reliable the pattern is (0.0-1.0)

## Step 1: View All Patterns

List all discovered patterns:

```bash
conduit introspect patterns
```

**Expected Output:**

```
PATTERNS

authentication (3 patterns, 60% coverage):

  1. authenticated_handler (5 uses, confidence: 0.5)

     Template:
     @on <operation>: [auth_required]

     Used by:
     • User  resources/user.cdt:15
     • Comment  resources/comment.cdt:22
     • Post  resources/post.cdt:18
     [... and 2 more]

     When to use:
     Endpoints requiring user authentication

  2. authenticated_rate_limited_handler (2 uses, confidence: 0.2)

     Template:
     @on <operation>: [auth_required, rate_limit]

     Used by:
     • Comment  resources/comment.cdt:22

     When to use:
     User-generated content creation needing spam protection

caching (1 patterns, 20% coverage):

  1. cached_handler (3 uses, confidence: 0.3)

     Template:
     @on <operation>: [cache]

     Used by:
     • Post  resources/post.cdt:12
     • Category  resources/category.cdt:8

     When to use:
     Frequently accessed read-only data
```

**What you're seeing:**
- **Category**: Grouping of related patterns
- **Coverage**: Percentage of resources using patterns in this category
- **Frequency**: Number of times pattern appears
- **Confidence**: How reliable/established the pattern is
- **Template**: Code structure for the pattern
- **Used by**: Resources implementing this pattern
- **When to use**: Guidance on when to apply the pattern

## Step 2: Filter by Category

Focus on specific pattern categories:

```bash
# Only authentication patterns
conduit introspect patterns authentication

# Only caching patterns
conduit introspect patterns caching

# Only hook patterns
conduit introspect patterns hook
```

Common categories:
- `authentication` - Auth middleware patterns
- `authorization` - Permission patterns
- `caching` - Cache middleware patterns
- `rate_limiting` - Rate limit patterns
- `hook` - Lifecycle hook patterns
- `validation` - Validation patterns
- `constraint` - Constraint patterns

## Step 3: Filter by Frequency

Show only patterns that appear frequently:

```bash
# Patterns appearing at least 3 times
conduit introspect patterns --min-frequency 3

# Patterns appearing at least 5 times
conduit introspect patterns --min-frequency 5
```

This helps you focus on truly established patterns and ignore one-offs.

## Step 4: JSON Output for Tooling

Export patterns as JSON for code generation or analysis:

```bash
conduit introspect patterns --format json > patterns.json
```

**Example JSON Output:**

```json
{
  "total_count": 5,
  "patterns": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "authenticated_handler",
      "category": "authentication",
      "description": "Handler with auth_required middleware",
      "template": "@on <operation>: [auth_required]",
      "examples": [
        {
          "resource": "User",
          "file_path": "resources/user.cdt",
          "line_number": 15,
          "code": "@on create: [auth_required]"
        }
      ],
      "frequency": 5,
      "confidence": 0.5
    }
  ]
}
```

## Step 5: Using Patterns Programmatically

Query patterns from Go code:

```go
package main

import (
    "fmt"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func main() {
    registry := metadata.GetRegistry()

    // Get all patterns
    patterns := registry.Patterns("")

    fmt.Printf("Found %d patterns:\n", len(patterns))

    // Group by category
    byCategory := make(map[string][]metadata.PatternMetadata)
    for _, p := range patterns {
        byCategory[p.Category] = append(byCategory[p.Category], p)
    }

    // Show categories with most patterns
    for category, categoryPatterns := range byCategory {
        fmt.Printf("\n%s: %d patterns\n", category, len(categoryPatterns))

        for _, p := range categoryPatterns {
            fmt.Printf("  - %s (used %d times, confidence %.1f)\n",
                p.Name, p.Frequency, p.Confidence)
        }
    }
}
```

## Common Use Cases

### 1. Generate Code from Patterns

Use discovered patterns as templates for new resources:

```go
package main

import (
    "fmt"
    "strings"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func generateResourceFromPattern(resourceName string, patternName string) string {
    registry := metadata.GetRegistry()
    patterns := registry.Patterns("")

    // Find pattern
    var pattern *metadata.PatternMetadata
    for _, p := range patterns {
        if p.Name == patternName {
            pattern = &p
            break
        }
    }

    if pattern == nil {
        return ""
    }

    // Use pattern template
    template := pattern.Template
    code := fmt.Sprintf("resource %s {\n", resourceName)
    code += "  // Fields go here\n\n"

    // Add pattern
    code += "  " + strings.ReplaceAll(template, "<operation>", "create") + "\n"
    code += "}\n"

    return code
}

func main() {
    code := generateResourceFromPattern("Product", "authenticated_handler")
    fmt.Println(code)
}
```

**Output:**
```conduit
resource Product {
  // Fields go here

  @on create: [auth_required]
}
```

### 2. Validate Pattern Consistency

Check that resources follow established patterns:

```go
package main

import (
    "fmt"
    "strings"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func validatePatternUsage() {
    registry := metadata.GetRegistry()
    resources := registry.Resources()

    // Find "authenticated create" pattern
    patterns := registry.Patterns("authentication")
    var authPattern *metadata.PatternMetadata
    for _, p := range patterns {
        if strings.Contains(p.Name, "authenticated") {
            authPattern = &p
            break
        }
    }

    if authPattern == nil {
        return
    }

    // Check each resource
    for _, res := range resources {
        createMW, hasCreate := res.Middleware["create"]

        if hasCreate {
            hasAuth := false
            for _, mw := range createMW {
                if strings.Contains(mw, "auth") {
                    hasAuth = true
                    break
                }
            }

            if !hasAuth {
                fmt.Printf("⚠️  %s: create operation missing auth (pattern: %s)\n",
                    res.Name, authPattern.Name)
            } else {
                fmt.Printf("✓ %s: follows %s pattern\n",
                    res.Name, authPattern.Name)
            }
        }
    }
}
```

### 3. Document Coding Standards

Generate documentation from discovered patterns:

```go
package main

import (
    "fmt"
    "sort"
    "github.com/conduit-lang/conduit/runtime/metadata"
)

func generateStyleGuide() {
    registry := metadata.GetRegistry()
    patterns := registry.Patterns("")

    // Sort by confidence (most reliable first)
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].Confidence > patterns[j].Confidence
    })

    fmt.Println("# Coding Standards\n")
    fmt.Println("Auto-generated from codebase patterns.\n")

    currentCategory := ""
    for _, p := range patterns {
        // High confidence patterns only
        if p.Confidence < 0.5 {
            continue
        }

        if p.Category != currentCategory {
            currentCategory = p.Category
            fmt.Printf("\n## %s\n\n", currentCategory)
        }

        fmt.Printf("### %s\n\n", p.Name)
        fmt.Printf("**Usage:** %d resources (%.0f%% confidence)\n\n",
            p.Frequency, p.Confidence*100)
        fmt.Printf("**Template:**\n```conduit\n%s\n```\n\n",
            p.Template)

        if p.Description != "" {
            fmt.Printf("**Description:** %s\n\n", p.Description)
        }

        // Show first example
        if len(p.Examples) > 0 {
            ex := p.Examples[0]
            fmt.Printf("**Example:** %s\n```conduit\n%s\n```\n\n",
                ex.Resource, ex.Code)
        }
    }
}
```

## Understanding Confidence Scores

Pattern confidence is calculated as `frequency / 10.0`, capped at 1.0:

- **0.3-0.4**: Emerging pattern (3-4 uses)
- **0.5-0.7**: Established pattern (5-7 uses)
- **0.8-1.0**: Standard pattern (8+ uses)

Higher confidence = more reliable and widespread pattern.

## Pattern Categories Explained

### Authentication Patterns
Patterns for user authentication:
```conduit
@on create: [auth_required]
@on update: [auth_required]
@on delete: [auth_required]
```

### Authorization Patterns
Patterns for permission checks:
```conduit
@on update: [auth_required, owner_only]
@on delete: [auth_required, admin_only]
```

### Caching Patterns
Patterns for caching responses:
```conduit
@on list: [cache(300)]
@on show: [cache(600)]
```

### Rate Limiting Patterns
Patterns for abuse prevention:
```conduit
@on create: [auth_required, rate_limit(5/hour)]
```

### Hook Patterns
Common lifecycle hook patterns:
```conduit
@before create {
  self.slug = String.slugify(self.title)
}

@after create @transaction {
  // Send notification
}
```

## Real-World Example: New Resource Template

Let's create a new resource following discovered patterns:

### Step 1: Analyze Existing Patterns

```bash
conduit introspect patterns --min-frequency 3
```

You discover these common patterns:
- `authenticated_handler` (5 uses)
- `before_create_slug_generation` (4 uses)
- `rate_limited_create` (3 uses)

### Step 2: Apply Patterns to New Resource

```conduit
/// Article resource following discovered patterns
resource Article {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  author_id: uuid!

  // Pattern: before_create_slug_generation
  @before create {
    self.slug = String.slugify(self.title)
  }

  // Pattern: authenticated_handler
  @on create: [auth_required]
  @on update: [auth_required, owner_only]
  @on delete: [auth_required, admin_only]

  // Pattern: rate_limited_create
  @on create: [rate_limit(10/hour)]
}
```

### Step 3: Verify Pattern Compliance

```bash
conduit build
conduit introspect resource Article
```

Your new resource now follows established patterns!

## Exercises

1. **Find most common pattern**: Which pattern has the highest frequency?
2. **Category analysis**: Which category has the most patterns?
3. **Generate template**: Use a pattern to generate a new resource
4. **Validate consistency**: Check if all resources follow auth patterns
5. **Export patterns**: Save all patterns as JSON for tooling

## Next Steps

In [Tutorial 4: Building Tools](04-building-tools.md), you'll learn how to:
- Build custom introspection tools
- Create code generators
- Write linters and validators
- Generate documentation automatically

## Troubleshooting

**"No patterns found"**
- Build your application first: `conduit build`
- Patterns require at least 3 occurrences by default
- Use `--min-frequency 1` to see all patterns

**"Pattern has low confidence"**
- Pattern appears infrequently (< 5 times)
- May be emerging or experimental
- Consider if it should be a standard pattern

**Patterns seem wrong**
- Pattern extraction is heuristic-based
- Review examples to understand pattern usage
- Submit feedback if patterns are incorrect

## Key Takeaways

- Use `conduit introspect patterns` to see discovered patterns
- Filter by category with `conduit introspect patterns <category>`
- Use `--min-frequency` to focus on established patterns
- Patterns have templates you can use for code generation
- Confidence scores indicate how reliable a pattern is
- Use patterns to maintain consistency across your codebase
- Export patterns as JSON for tooling and code generation

**Time to complete:** ~25 minutes

---

[← Previous: Dependency Analysis](02-dependency-analysis.md) | [Back to Overview](../README.md) | [Next: Building Tools →](04-building-tools.md)
