# Conduit Future Vision

**Last Updated:** 2025-11-05
**Current Version:** v0.1.0

⚠️ **IMPORTANT: All features in this document are NOT YET IMPLEMENTED**

This document describes planned features for future versions of Conduit. These features are documented in design documents but are not functional in the current release.

**For features that work today, see [LANGUAGE-SPEC.md](LANGUAGE-SPEC.md)**

---

## Table of Contents

1. [Version 0.2.0 - The Essential 10](#version-020---the-essential-10)
2. [Version 0.3.0 - Expression Language](#version-030---expression-language)
3. [Version 0.4.0 - Advanced Features](#version-040---advanced-features)
4. [Version 1.0.0 - Production Ready](#version-100---production-ready)

---

## Version 0.2.0 - The Essential 10

**Status:** 🚧 In Planning
**Target:** Q1 2026
**Focus:** Ten essential features needed for production use

### Named Enum Support

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-106
**Target:** v0.2.0

**What This Enables:**
Type-safe status fields, priorities, roles, and other categorical data.

**Planned Syntax:**
```conduit
enum PostStatus { draft, published, archived }
enum Priority { low, medium, high, urgent }

resource Post {
  status: PostStatus! @default(draft)
  priority: Priority! @default(medium)
}
```

**Current Workaround:**
```conduit
// Use unvalidated strings (not type-safe)
status: string! @default("draft")  // ❌ Nothing prevents invalid values
```

**Design Notes:**
- Generate Go enum types with constants
- Generate PostgreSQL ENUM types in migrations
- Dot notation access (PostStatus.published)
- Validation in API handlers

---

### Named Indexes with Unique Constraints

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-107
**Target:** v0.2.0

**What This Enables:**
Composite unique constraints for data integrity in join tables and multi-column uniqueness.

**Planned Syntax:**
```conduit
resource PostTag {
  post: Post!
  tag: Tag!

  @index {
    columns: [post_id, tag_id]
    unique: true
  }
}
```

**Current Workaround:**
```conduit
// Add unique constraints manually in migrations
// No language-level support
```

---

### Join Table Helper (@join_table)

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-108
**Target:** v0.2.0

**What This Enables:**
Shorthand for many-to-many relationships without boilerplate join tables.

**Planned Syntax:**
```conduit
resource Post {
  @join_table(Tag) tags
}

// Auto-generates:
// resource PostTag {
//   post: Post!
//   tag: Tag!
//   @index { columns: [post_id, tag_id], unique: true }
// }
```

**Current Workaround:**
```conduit
// Manually create join table resource
resource PostTag {
  id: uuid! @primary @auto
  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }
  tag: Tag! {
    foreign_key: "tag_id"
    on_delete: cascade
  }
  created_at: timestamp! @auto
}
```

---

### Query Filtering

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-110
**Target:** v0.2.0

**What This Enables:**
Filter list endpoints by field values via query parameters.

**Planned Syntax:**
```bash
GET /posts?status=published&author_id=123
GET /comments?post_id=456&deleted_at=null
```

**Current Workaround:**
No filtering available - all list endpoints return all records.

---

### Relationship Loading (@include)

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-111
**Target:** v0.2.0

**What This Enables:**
Eager load related resources without N+1 queries.

**Planned Syntax:**
```bash
GET /posts?include=author,comments
GET /users/123?include=posts.comments
```

**Current Workaround:**
Make separate API calls for each relationship (causes N+1 problem).

---

### Hot Reload Dev Mode

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-112
**Target:** v0.2.0

**What This Enables:**
Auto-rebuild and restart on file changes during development.

**Planned Command:**
```bash
$ conduit dev --watch
Watching for changes...
✓ Initial build complete (1.2s)
Server running on :3000

[File changed: app/resources/post.cdt]
Rebuilding... ✓ Done (0.8s)
Server restarted
```

**Current Workaround:**
Manually run `conduit build` and restart server after each change.

---

### Routes Introspection

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-113
**Target:** v0.2.0

**What This Enables:**
List all generated routes for debugging 404 errors.

**Planned Command:**
```bash
$ conduit routes
GET    /posts           List posts
POST   /posts           Create post
GET    /posts/:id       Get post
PUT    /posts/:id       Update post
DELETE /posts/:id       Delete post

Total: 15 routes
```

**Current Workaround:**
Manually inspect generated Go code to see routes.

---

### Seed Data System

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-114
**Target:** v0.2.0

**What This Enables:**
Populate database with test data for development.

**Planned Syntax:**
```yaml
# seeds/dev.yaml
users:
  - id: user-1
    email: alice@example.com
    name: Alice Developer

posts:
  - author_id: user-1
    title: First Post
    status: published
```

**Planned Command:**
```bash
$ conduit seed
Loading seeds/dev.yaml...
✓ Created 2 users
✓ Created 2 posts
Done
```

**Current Workaround:**
Manually insert data via SQL or API calls.

---

### Standard Library Expansion

**Status:** ⚠️ Partially Implemented (15 functions work)
**Tracking Issue:** CON-115
**Target:** v0.2.0

**What This Enables:**
~30 essential functions for common operations.

**Currently Implemented (15 functions):**
- String: `length`, `slugify`, `upcase`, `downcase`, `trim`, `contains`, `replace`
- Time: `now`, `format`, `parse`
- Array: `length`, `contains`
- UUID: `generate`, `validate`
- Random: `int`

**Planned Additions:**

#### String Namespace
- `String.split(s: string, delimiter: string) -> array<string>`
- `String.join(arr: array<string>, separator: string) -> string`
- `String.startsWith(s: string, prefix: string) -> bool`
- `String.endsWith(s: string, suffix: string) -> bool`

#### Time Namespace
- `Time.add(t: timestamp, duration: string) -> timestamp`
- `Time.sub(t: timestamp, duration: string) -> timestamp`
- `Time.isBefore(t1: timestamp, t2: timestamp) -> bool`
- `Time.isAfter(t1: timestamp, t2: timestamp) -> bool`

#### Array Namespace (NEW)
- `Array.isEmpty(arr: array<T>) -> bool`
- `Array.indexOf(arr: array<T>, item: T) -> int`
- `Array.first(arr: array<T>) -> T?`
- `Array.last(arr: array<T>) -> T?`

#### Math Namespace (NEW)
- `Math.min(a: int, b: int) -> int`
- `Math.max(a: int, b: int) -> int`
- `Math.abs(n: int) -> int`
- `Math.round(f: float) -> int`

**Current Workaround:**
Implement custom functions or use raw SQL for missing functions.

---

### Performance Optimization

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-116
**Target:** v0.2.0

**What This Enables:**
Benchmarking and optimization for production workloads.

**Planned Features:**
- Compile time benchmarks (<2s for typical project)
- Runtime API benchmarks (p95 <100ms)
- Database query optimization
- Memory usage optimization (<100MB for typical app)

**Current State:**
No performance targets or benchmarking infrastructure.

---

### Comprehensive Examples

**Status:** ❌ Not Implemented
**Tracking Issue:** CON-117
**Target:** v0.2.0

**What This Enables:**
Production-ready example applications showcasing all Essential 10 features.

**Planned Examples:**
- Blog v2 (named enums, composite constraints, query filtering, relationships)
- Todo v2 (named enums, join tables, query filtering, seed data)
- E-commerce Prototype (multiple enums, complex relationships, all Essential 10)

**Current State:**
Basic examples exist (minimal, blog, todo, api-with-auth) but don't showcase v0.2.0 features.

---

## Version 0.3.0 - Expression Language

**Status:** 📋 Designed
**Target:** Q2 2026
**Focus:** Full expression language for hooks, validations, and computed fields

### Expression Language Features

**Status:** ❌ Mostly Not Implemented
**Reference:** LANGUAGE-SPEC.md (Expression Language section)

The expression language is extensively documented in LANGUAGE-SPEC.md but is not functional. Key features planned:

#### Literals
- String interpolation: `"Hello, #{user.name}!"`
- Array literals: `[1, 2, 3]`
- Hash literals: `{key: "value"}`
- Numeric underscores: `1_000_000`

#### Operators
- Arithmetic: `+`, `-`, `*`, `/`, `%`, `**`
- Comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Logical: `&&`, `||`, `!`
- String: `+` (concatenation)
- Pipeline: `|>`
- Elvis: `?:`
- Safe navigation: `?.`

#### Control Flow
```conduit
// If/else expressions
let status = if self.published_at < Time.now() {
  "published"
} else {
  "draft"
}

// Pattern matching
let message = match self.status {
  "draft" => "Still editing",
  "published" => "Live!",
  "archived" => "No longer available",
  _ => "Unknown status"
}
```

#### Function Calls
- Method chaining: `self.title.slugify().downcase()`
- Pipeline operator: `self.title |> String.slugify() |> String.downcase()`

**Current Workaround:**
Very limited expression support in hooks. Most complex logic must be implemented in custom Go code.

---

### Query Language

**Status:** ❌ Not Implemented
**Reference:** LANGUAGE-SPEC.md (Query Language section)
**Target:** v0.3.0

**Planned Features:**

#### Basic Queries
```conduit
// Find by ID
Post.find(id)

// Find by attribute
Post.find_by(slug: "hello-world")

// Where conditions
Post.where(status: "published")
Post.where(status: "published", featured: true)

// Comparison operators
Post.where(view_count > 1000)
Post.where(created_at >= Time.now() - 7.days)
```

#### Query Scopes
```conduit
@scope published {
  where: { status: "published" }
  order_by: "published_at DESC"
}

@scope featured {
  where: { featured: true, status: "published" }
}

@scope by_category(category_id: uuid) {
  where: { category_id: category_id }
}
```

#### Aggregations
```conduit
Post.count()
Post.where(status: "published").count()
Post.sum("view_count")
Post.average("rating")
Post.maximum("view_count")
Post.minimum("created_at")
```

**Current Workaround:**
No query builder - use raw SQL or custom Go code.

---

## Version 0.4.0 - Advanced Features

**Status:** 📋 Designed
**Target:** Q3 2026
**Focus:** Advanced resource features and developer experience

### @has_many Relationships

**Status:** ❌ Not Implemented
**Reference:** ROADMAP.md
**Target:** v0.4.0

**Planned Syntax:**
```conduit
resource User {
  id: uuid! @primary @auto

  @has_many Post as "posts"
  @has_many Comment as "comments"
}

// Enables:
// user.posts           // Get all posts
// user.posts.count     // Count posts
// user.posts.where(...)// Filter posts
```

**Current Workaround:**
```conduit
// Query posts manually via foreign key
Post.where(author_id: user.id)
```

---

### @computed Fields

**Status:** ❌ Not Implemented
**Reference:** ROADMAP.md
**Target:** v0.4.0

**Planned Syntax:**
```conduit
resource Post {
  title: string!
  content: text!

  @computed word_count: int {
    self.content.split(" ").length
  }

  @computed reading_time: string {
    let words = self.word_count
    let minutes = words / 200
    "#{minutes} min read"
  }
}
```

**Current Workaround:**
Compute values in application code, not in resource definition.

---

### @validate Blocks

**Status:** ❌ Not Implemented
**Reference:** ROADMAP.md
**Target:** v0.4.0

**Planned Syntax:**
```conduit
@validate {
  // Complex validation logic
  if self.discount_code {
    let discount = Discount.find_by(code: self.discount_code)
    if !discount || discount.expired? {
      error("Invalid or expired discount code")
    }
  }

  if self.start_date >= self.end_date {
    error("End date must be after start date")
  }
}
```

**Current Workaround:**
Use declarative constraints (@min, @max, etc.) for simple validation. Complex validation requires custom Go code.

---

### @function Custom Functions

**Status:** ❌ Not Implemented
**Reference:** ROADMAP.md
**Target:** v0.4.0

**Planned Syntax:**
```conduit
resource Post {
  title: string!

  @function publish() -> bool {
    if self.title.length < 5 {
      return false
    }

    self.status = "published"
    self.published_at = Time.now()
    self.save()
    return true
  }

  @function archive() {
    self.status = "archived"
    self.save()
  }
}
```

**Current Workaround:**
Implement custom methods in generated Go code (requires editing generated files).

---

### Error Handling

**Status:** ❌ Not Implemented
**Reference:** LANGUAGE-SPEC.md (Error Handling section)
**Target:** v0.4.0

#### Rescue Blocks
```conduit
// Basic rescue
Email.send(user, "welcome") rescue |err| {
  Logger.error("Email failed", error: err)
}

// Rescue with fallback
let result = risky_operation() rescue |err| {
  Logger.warn("Fallback used", error: err)
  return default_value
}
```

#### Unwrap Operator
```conduit
// ! unwraps or panics (use for invariants)
let user = Context.current_user!()  // Panics if nil
let value = self.required_field!    // Panics if nil
```

**Current Workaround:**
Error handling must be implemented in custom Go code.

---

## Version 1.0.0 - Production Ready

**Status:** 🎯 Vision
**Target:** Q4 2026
**Focus:** Production stability, tooling, and ecosystem

### Testing Framework

**Status:** ❌ Not Implemented
**Target:** v1.0.0

Planned features:
- Built-in test syntax: `test ResourceName { @test "description" { } }`
- Factories for test data generation
- HTTP testing utilities
- Test database management
- Coverage reporting

---

### Advanced Resource Features

Features planned for v1.0.0:
- Soft deletes (`@soft_delete`)
- Polymorphic relationships
- Single Table Inheritance (STI)
- Multi-tenancy support
- Nested resources (`@nested under Parent`)
- Middleware system (`@middleware [auth, rate_limit]`)

---

### Developer Tools

Planned tooling:
- LSP (Language Server Protocol) support for editors
- Syntax highlighting for VSCode, Vim, Emacs
- Interactive REPL for Conduit expressions
- Database console (`conduit console`)
- Schema visualization
- Performance profiling tools

---

## Open Questions & Design Decisions

### Type System Evolution
- Should we support generics? When?
- Union types vs enum types - which pattern?
- Nominal vs structural typing for inline structs?

### Query Language
- SQL-like syntax vs method chaining?
- How to handle complex joins?
- GraphQL-style nested queries?

### Async/Await
- Should we support async/await for hooks?
- How to handle background jobs?
- Transaction boundaries with async operations?

---

## Contributing to Future Features

Want to help implement these features? See:
- **Linear:** https://linear.app/haener-dev/team/CON (Conduit team)
- **GitHub:** https://github.com/dphaener/conduit
- **Docs:** Check ticket descriptions in Linear for detailed implementation plans

---

## Versioning Policy

**Semantic Versioning:**
- **Major (1.0.0 → 2.0.0):** Breaking changes
- **Minor (0.1.0 → 0.2.0):** New features, backward compatible
- **Patch (0.1.0 → 0.1.1):** Bug fixes, no new features

**Feature Stability:**
- **Alpha:** May change without notice
- **Beta:** Stable API, may have bugs
- **Stable:** Production-ready, semantic versioning applies

---

**Last Updated:** 2025-11-05
