# Conduit Roadmap

**Last Updated:** 2025-11-02

This document tracks features that are documented in LANGUAGE-SPEC.md but not yet implemented. This roadmap exists to prevent user confusion and set clear expectations about what works today vs. what's planned for future releases.

---

## Context

The LANGUAGE-SPEC.md was written as an aspirational design document, but implementation is still in progress. Many features in that spec are not yet functional. This roadmap clarifies what's missing and when we plan to implement it.

---

## ✅ What Works Today (v0.1.0)

### Core Language Features
- ✅ Resource definitions with fields
- ✅ Explicit nullability (`!` vs `?`)
- ✅ Primitive types (string, int, float, bool, uuid, timestamp, etc.)
- ✅ Structural types (array, hash, enum, inline structs)
- ✅ Field constraints (@min, @max, @unique, @primary, @auto, @default, etc.)
- ✅ Relationships (belongs_to with foreign_key metadata)
- ✅ Comments (both leading and trailing)

### Lifecycle Hooks
- ✅ `@before create/update/delete/save` hooks
- ✅ `@after create/update/delete/save` hooks
- ✅ Hook body parsing and preservation

### Constraints
- ✅ `@constraint` blocks (parsed and stored in AST)
- ⚠️ **Note:** Constraint execution/validation is NOT yet implemented

### Standard Library (MVP - 15 functions)
#### String Namespace
- ✅ `String.length(s: string) -> int`
- ✅ `String.slugify(s: string) -> string`
- ✅ `String.upcase(s: string) -> string`
- ✅ `String.downcase(s: string) -> string`
- ✅ `String.trim(s: string) -> string`
- ✅ `String.contains(s: string, substr: string) -> bool`
- ✅ `String.replace(s: string, old: string, new: string) -> string`

#### Time Namespace
- ✅ `Time.now() -> timestamp`
- ✅ `Time.format(t: timestamp, format: string) -> string`
- ✅ `Time.parse(s: string) -> timestamp`

#### Array Namespace
- ✅ `Array.length(arr: array<T>) -> int`
- ✅ `Array.contains(arr: array<T>, item: T) -> bool`

#### UUID Namespace
- ✅ `UUID.generate() -> uuid`
- ✅ `UUID.validate(s: string) -> bool`

#### Random Namespace
- ✅ `Random.int(min: int, max: int) -> int`

### Compilation & Code Generation
- ✅ Lexer (tokenization)
- ✅ Parser (AST generation)
- ✅ Type checker (basic type safety)
- ✅ Code generator (Go source output)
- ✅ REST API generation (CRUD endpoints)
- ✅ Database migrations (PostgreSQL)

---

## ❌ Not Yet Implemented

### Resource-Level Annotations

#### `@has_many` - One-to-Many Relationships
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 328-358
**Planned:** v0.2.0

**Current Workaround:**
```conduit
# Instead of:
# @has_many Post as "posts"

# Use:
# Query posts manually via foreign key
# Post.where(author_id: self.id)
```

**What's Missing:**
- Parser support for `@has_many` annotation
- Relationship tracking in AST
- Code generation for collection accessors
- Eager loading support

---

#### `@belongs_to` - Annotation Form
**Status:** Partially Implemented
**Documentation:** LANGUAGE-SPEC.md lines 308-324
**Planned:** v0.2.0

**What Works:**
```conduit
# Inline relationship metadata (works)
author: User! {
  foreign_key: "author_id"
  on_delete: restrict
}
```

**What Doesn't Work:**
```conduit
# Annotation form (not implemented)
# author_id: uuid!
# @belongs_to User
```

**Workaround:** Use inline metadata form shown above

---

#### `@scope` - Query Scopes
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 1046-1079
**Planned:** v0.3.0

**Example (Not Functional):**
```conduit
# This will be ignored by the compiler
@scope published {
  where: { status: "published" }
  order_by: "created_at DESC"
}
```

**Current Workaround:**
Define scopes in application code or use raw queries.

**What's Missing:**
- Parser support for `@scope` blocks
- Scope DSL for where/order_by/limit
- Code generation for scope methods
- Scope chaining support

---

#### `@validate` - Procedural Validation
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 910-926
**Planned:** v0.2.0

**Example (Not Functional):**
```conduit
@validate {
  if self.status == "published" && String.length(self.content) < 500 {
    error("Published posts need 500+ characters")
  }
}
```

**Current Workaround:**
Use `@constraint` blocks (partially implemented) or validation in application code.

**What's Missing:**
- Parser support for `@validate` blocks
- Validation execution in runtime
- Error aggregation and reporting
- Integration with CRUD handlers

---

#### `@invariant` - Runtime Invariants
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 957-972
**Planned:** v0.3.0

**Example (Not Functional):**
```conduit
@invariant metrics_non_negative {
  condition: self.view_count >= 0 && self.like_count >= 0
  error: "Metrics cannot be negative"
}
```

**Current Workaround:**
Implement invariant checks in application code or database constraints.

**What's Missing:**
- Parser support for `@invariant` blocks
- Runtime checking on all mutations
- Performance optimization (caching, batching)

---

#### `@computed` - Computed Fields
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 1278-1291
**Planned:** v0.4.0

**Example (Not Functional):**
```conduit
@computed is_published: bool! {
  return self.status == "published" && self.published_at != nil
}
```

**Current Workaround:**
Implement computed values in application code as helper methods.

**What's Missing:**
- Parser support for `@computed` blocks
- Expression evaluation in Go
- Caching/memoization
- Inclusion in API responses

---

#### `@function` - Custom Functions
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 794-810
**Planned:** v0.4.0

**Example (Not Functional):**
```conduit
@function generate_slug(title: string) -> string {
  let cleaned = String.downcase(title)
  return String.replace(cleaned, " ", "-")
}
```

**Current Workaround:**
Define logic inline in hooks or use stdlib functions.

**What's Missing:**
- Parser support for `@function` definitions
- Function signature validation
- Code generation for custom functions
- Function call resolution in expressions

---

#### `@on` - Middleware Annotations
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 1294-1298
**Planned:** v0.5.0

**Example (Not Functional):**
```conduit
@on list: [cache(300), auth]
@on create: [auth, rate_limit(5, per: "hour")]
```

**Current Workaround:**
Implement middleware in generated handler code manually.

---

#### `@nested` - Nested Resources
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 362-367
**Planned:** v0.3.0

**Example (Not Functional):**
```conduit
@nested under Post as "comments"
```

**Current Workaround:**
Use flat resource structure with foreign keys.

---

#### `@has_many through` - Many-to-Many Relationships
**Status:** Not Implemented
**Documentation:** LANGUAGE-SPEC.md lines 340-347
**Planned:** v0.3.0

**Example (Not Functional):**
```conduit
@has_many Tag through PostTag as "tags"
```

**Current Workaround:**
Manage join tables manually in application code.

---

### Standard Library - Missing Functions

The following namespaces/functions are documented but not implemented:

#### String Namespace (Missing)
- ❌ `String.capitalize`
- ❌ `String.truncate`
- ❌ `String.split`
- ❌ `String.join`
- ❌ `String.starts_with?`
- ❌ `String.ends_with?`
- ❌ `String.includes?`

#### Text Namespace (All Missing)
- ❌ `Text.calculate_reading_time`
- ❌ `Text.word_count`
- ❌ `Text.character_count`
- ❌ `Text.excerpt`

#### Number Namespace (All Missing)
- ❌ `Number.format`
- ❌ `Number.round`
- ❌ `Number.abs`
- ❌ `Number.ceil`
- ❌ `Number.floor`
- ❌ `Number.min`
- ❌ `Number.max`

#### Array Namespace (Missing)
- ❌ `Array.first`
- ❌ `Array.last`
- ❌ `Array.empty?`
- ❌ `Array.unique`
- ❌ `Array.sort`
- ❌ `Array.reverse`
- ❌ `Array.push`
- ❌ `Array.concat`

#### Hash Namespace (All Missing)
- ❌ `Hash.keys`
- ❌ `Hash.values`
- ❌ `Hash.merge`
- ❌ `Hash.has_key?`
- ❌ `Hash.get`

#### Time Namespace (Missing)
- ❌ `Time.today`
- ❌ `Time.add`
- ❌ `Time.subtract`
- ❌ `Time.diff`
- ❌ `Time.year`
- ❌ `Time.month`
- ❌ `Time.day`
- ❌ `Time.hour`
- ❌ `Time.minute`
- ❌ `Time.second`

#### UUID Namespace (Missing)
- ❌ `UUID.parse`

#### Random Namespace (Missing)
- ❌ `Random.float`
- ❌ `Random.uuid`
- ❌ `Random.hex`
- ❌ `Random.alphanumeric`

#### Crypto Namespace (All Missing)
- ❌ `Crypto.hash`
- ❌ `Crypto.compare`
- ❌ `Crypto.encrypt`
- ❌ `Crypto.decrypt`

#### HTML Namespace (All Missing)
- ❌ `HTML.strip_tags`
- ❌ `HTML.escape`
- ❌ `HTML.unescape`

#### JSON Namespace (All Missing)
- ❌ `JSON.parse`
- ❌ `JSON.stringify`
- ❌ `JSON.validate`

#### Regex Namespace (All Missing)
- ❌ `Regex.match`
- ❌ `Regex.replace`
- ❌ `Regex.test`
- ❌ `Regex.split`

#### Logger Namespace (All Missing)
- ❌ `Logger.debug`
- ❌ `Logger.info`
- ❌ `Logger.warn`
- ❌ `Logger.error`

#### Cache Namespace (All Missing)
- ❌ `Cache.get`
- ❌ `Cache.set`
- ❌ `Cache.invalidate`
- ❌ `Cache.clear`

#### Context Namespace (All Missing)
- ❌ `Context.current_user`
- ❌ `Context.current_user!`
- ❌ `Context.authenticated?`
- ❌ `Context.current_request`

#### Env Namespace (All Missing)
- ❌ `Env.get`
- ❌ `Env.set`
- ❌ `Env.has?`

---

### Query Language Features

#### Basic Queries (Missing)
- ❌ `Post.find(id)`
- ❌ `Post.find_by(slug: "...")`
- ❌ `Post.where(...)`
- ❌ `Post.order_by(...)`
- ❌ `Post.limit(...)`
- ❌ `Post.offset(...)`

#### Aggregations (All Missing)
- ❌ `count(...)`
- ❌ `sum(...)`
- ❌ `avg(...)`
- ❌ `min(...)`
- ❌ `max(...)`

#### Advanced Queries (All Missing)
- ❌ `Post.joins(...)`
- ❌ `Post.includes(...)`
- ❌ `Post.pluck(...)`
- ❌ `Post.exists?(...)`

**Current Workaround:** Use raw SQL queries or implement ORM methods manually in Go.

---

### Expression Language Features

Most expression language features are documented but not implemented:

#### Control Flow
- ❌ `if/elsif/else` statements
- ❌ `unless` statements
- ❌ `match` expressions
- ❌ Ternary operator (`condition ? true_val : false_val`)

#### Operators
- ❌ Null coalescing (`??`)
- ❌ Safe navigation (`?.`)
- ❌ `in` operator (membership testing)

#### Error Handling
- ❌ `rescue |err| { }` blocks
- ❌ Unwrap operator (`!`)

#### Async Execution
- ✅ `@async { }` blocks in hooks - generates Go goroutines for background execution
- ❌ Job queue integration (Sidekiq, etc.)
- ❌ Retry policies
- ❌ Job scheduling

**Current Status:** Basic async execution works via goroutines. For production job queues, integrate manually in Go code.

#### Change Tracking
- ✅ Generated Go methods: `TitleChanged()`, `PreviousTitle()`, `ChangedFields()`, `HasChanges()`
- ❌ Hook DSL syntax: `self.field_changed?`, `self.previous_value(:field)` (not yet accessible in @before/@after blocks)
- ❌ `self.field_changed_to?(value)`
- ❌ `self.field_changed_from?(value)`

**Current Workaround:** Use generated Go methods in application code, or use hook workarounds without change tracking.

---

### Tooling & Developer Experience

#### LSP (Language Server Protocol)
**Status:** Not Implemented
**Planned:** v0.6.0

**Missing Features:**
- Hover information
- Go to definition
- Find references
- Autocomplete
- Real-time diagnostics
- Code formatting

**Current Workaround:** Use generic text editor features.

---

#### Hot Reload / Watch Mode
**Status:** Not Implemented
**Planned:** v0.2.0

**Example (Not Functional):**
```bash
# This flag exists but doesn't work
conduit run --watch
```

**Current Workaround:** Manually rebuild after changes.

---

#### Testing Framework
**Status:** Not Implemented
**Planned:** v0.5.0

**Example (Not Functional):**
```conduit
@test "Creating a post" {
  let post = Post.create!({ title: "Test" })
  assert_not_nil(post.id)
}
```

**Current Workaround:** Write integration tests in Go.

---

#### Introspection API
**Status:** Partially Implemented
**Documentation:** GETTING-STARTED.md lines 822-841
**Planned:** Full implementation in v0.4.0

**What's Missing:**
- Pattern discovery
- Schema querying
- Runtime introspection

---

#### GraphQL Support
**Status:** Not Implemented
**Documentation:** GETTING-STARTED.md lines 843-850
**Planned:** v1.1.0

---

#### Background Jobs
**Status:** Not Implemented
**Documentation:** GETTING-STARTED.md lines 852-864
**Planned:** v1.2.0

---

## Release Timeline

### v0.2.0 (Q1 2026)
- [ ] `@has_many` relationships
- [ ] `@validate` blocks with execution
- [ ] Hot reload / watch mode
- [ ] Additional String namespace functions
- [ ] Query builder basics (find, find_by, where)

### v0.3.0 (Q2 2026)
- [ ] `@scope` query scopes
- [ ] `@invariant` runtime checks
- [ ] `@nested` resources
- [ ] Many-to-many relationships
- [ ] Time namespace functions
- [ ] Array/Hash namespace functions

### v0.4.0 (Q3 2026)
- [ ] `@computed` fields
- [ ] `@function` custom functions
- [ ] Full expression language support
- [ ] Introspection API
- [ ] Logger namespace
- [ ] Context namespace

### v0.5.0 (Q4 2026)
- [ ] `@on` middleware annotations
- [ ] Testing framework
- [ ] Cache namespace
- [ ] Crypto namespace
- [ ] Advanced query features

### v0.6.0 (Q1 2027)
- [ ] LSP implementation
- [ ] VS Code extension
- [ ] Debugger integration
- [ ] Performance optimizations

### v1.0.0 (Q2 2027)
- [ ] Production-ready release
- [ ] Comprehensive documentation
- [ ] Migration guides
- [ ] Stability guarantees

### v1.1.0+ (Beyond)
- [ ] GraphQL support
- [ ] Background jobs
- [ ] Multi-database support (MySQL, SQLite)
- [ ] Advanced caching strategies
- [ ] Real-time subscriptions

---

## Contributing

Want to help implement these features? We welcome contributions!

### High-Priority Items
1. **@validate blocks** - Validation execution engine
2. **Hot reload** - File watching and auto-rebuild
3. **Query builder** - Basic ORM methods (find, where, etc.)
4. **Standard library** - Missing String/Time/Array functions

### How to Contribute
1. Check the [GitHub Issues](https://github.com/conduit-lang/conduit/issues) for open tasks
2. Comment on an issue to claim it
3. Submit a PR referencing the issue
4. Update this ROADMAP.md when features are completed

---

## Reporting Documentation Bugs

If you find documentation that describes unimplemented features:
1. Open an issue with the "documentation" label
2. Include the file name and line numbers
3. Describe the actual vs. documented behavior

We're committed to keeping documentation accurate as implementation progresses.

---

**Last Updated:** 2025-11-02
**Version:** 0.1.0
