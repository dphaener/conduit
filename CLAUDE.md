# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Conduit is an **LLM-first programming language** for building web applications. It compiles to Go and provides explicit syntax optimized for AI-assisted development. The project is currently in the **design and planning phase** - all implementation guides are complete, but actual implementation has not yet begun.

**Key Characteristics:**
- Compiles Conduit source (`.cdt` files) to Go, then to native binaries
- Built-in ORM, web framework, and runtime introspection
- Explicit nullability (`type!` vs `type?`), namespaced stdlib to prevent LLM hallucination
- Single binary deployment with no runtime dependencies

## Repository Structure

```
conduit/
├── LANGUAGE-SPEC.md           # Complete language specification (30KB)
├── ARCHITECTURE.md            # System architecture overview (58KB)
├── IMPLEMENTATION-COMPILER.md # Compiler implementation guide (26KB)
├── IMPLEMENTATION-ORM.md      # ORM implementation guide (109KB)
├── IMPLEMENTATION-RUNTIME.md  # Runtime system guide (39KB)
├── IMPLEMENTATION-WEB.md      # Web framework guide (73KB)
├── IMPLEMENTATION-TOOLING.md  # Developer tooling guide (71KB)
├── GETTING-STARTED.md         # Quick start guide (22KB)
└── README.md                  # Project overview
```

## Architecture Overview

### Five Major Subsystems

1. **Compiler** (IMPLEMENTATION-COMPILER.md)
   - Lexer/tokenizer → Parser → Type checker → Go code generator
   - Transforms `.cdt` files to Go source code
   - Generates introspection metadata (`.meta.json`)

2. **Runtime** (IMPLEMENTATION-RUNTIME.md)
   - Lifecycle management (hooks, transactions, async tasks)
   - Validation engine (constraints, invariants)
   - Introspection API (schema registry, pattern database)

3. **ORM** (IMPLEMENTATION-ORM.md)
   - Query builder with fluent API
   - Relationship management (belongs-to, has-many, has-many-through)
   - Migration system (schema diff, rollback support)

4. **Web Framework** (IMPLEMENTATION-WEB.md)
   - Auto-generated REST API from resources
   - Middleware system (auth, CORS, rate limiting)
   - Request/response handling with JSON serialization

5. **Tooling** (IMPLEMENTATION-TOOLING.md)
   - CLI (`conduit new`, `build`, `run`, `migrate`, etc.)
   - Language Server Protocol (LSP) for IDE integration
   - Watch mode with hot reload
   - Debug Adapter Protocol (DAP) integration

### Compilation Pipeline

```
.cdt files → Lexer → Parser → AST → Type Checker →
Go Code Generator → .go files → go build → Binary
```

**Key Output Files:**
- `build/app` - Compiled binary
- `build/app.meta.json` - Introspection metadata
- `build/generated/` - Generated Go source code

## Language Design Principles

### Explicit Nullability
Every type must specify `!` (required) or `?` (optional):
```conduit
title: string!        // Required
bio: text?           // Optional
```

### Namespaced Standard Library
All built-in functions use namespaces to prevent LLM hallucination:
```conduit
String.slugify(text)     // ✓ Correct
Time.now()              // ✓ Correct
slugify(text)           // ✗ Won't compile
```

### Explicit Transaction Boundaries
```conduit
@after create @transaction {
  // Runs in database transaction

  @async {
    // Runs asynchronously after response
  }
}
```

### Progressive Disclosure
- **Simple:** 3 lines is a valid resource
- **Medium:** Add hooks, validations, relationships
- **Advanced:** Full transaction control, async operations, complex constraints

## Technology Stack

**Implementation Language:** Go 1.23+
**Target Database:** PostgreSQL 15+ (primary)
**Compilation:** Conduit → Go source → native binary
**Dependencies:**
- `pgx` (PostgreSQL driver)
- `chi` (HTTP router)
- `cobra` (CLI framework)
- `fsnotify` (file watching)
- Go standard library

## Key Files to Read First

1. **README.md** - Project overview and design philosophy (6KB)
2. **LANGUAGE-SPEC.md** - Complete syntax and semantics (30KB)
3. **ARCHITECTURE.md** - How all pieces fit together (58KB)
4. **GETTING-STARTED.md** - User-facing quick start guide (22KB)

For implementation details, refer to the specific `IMPLEMENTATION-*.md` files for each subsystem.

## Development Status

**Current Phase:** Design Complete, Implementation Starting Soon

- ✅ Language specification finalized
- ✅ Architecture documented
- ✅ Implementation guides written
- ⏳ Compiler implementation (not started)
- ⏳ Runtime implementation (not started)
- ⏳ ORM implementation (not started)
- ⏳ Web framework (not started)
- ⏳ Tooling (not started)

## Important Concepts for Implementation

### Resource Definition Example
```conduit
/// Blog post with title and content
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }

  @constraint published_requires_content {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts need 500+ characters"
  }
}
```

### Generated Go Output Pattern
The compiler generates Go structs, repository methods, validation functions, and hook execution:
```go
type Post struct {
    ID       uuid.UUID `db:"id" json:"id"`
    Title    string    `db:"title" json:"title"`
    Slug     string    `db:"slug" json:"slug"`
    Content  string    `db:"content" json:"content"`
    AuthorID uuid.UUID `db:"author_id" json:"author_id"`
}

func (p *Post) Validate() error { /* ... */ }
func (p *Post) BeforeCreate(ctx Context) error { /* ... */ }
```

### Introspection Metadata
Compiler generates `app.meta.json` with complete schema information for runtime queries:
```json
{
  "resources": [{
    "name": "Post",
    "fields": [...],
    "relationships": [...],
    "hooks": [...],
    "patterns": [...]
  }]
}
```

## Working with This Repository

### Reading Documentation
All specs are **complete and authoritative**. When implementing:
1. Start with the relevant `IMPLEMENTATION-*.md` guide
2. Reference `LANGUAGE-SPEC.md` for syntax details
3. Check `ARCHITECTURE.md` for integration points between subsystems

### Implementation Order (Planned)
1. **Compiler:** Lexer → Parser → Type checker → Code generator
2. **Runtime:** Bootstrap → Resource registration → Hook execution
3. **ORM:** Query builder → Resource operations → Relationships
4. **Web Framework:** Router → Handlers → Middleware
5. **Tooling:** CLI → LSP → Watch mode → Debugger

### File Naming Conventions
- Language files: `.cdt` extension
- Implementation guides: `IMPLEMENTATION-{SUBSYSTEM}.md`
- Generated Go code: `build/generated/{resources,handlers,main}.go`
- Migrations: `migrations/{number}_{description}.sql`

### Commit Message Conventions

**Always use Conventional Commits format** for consistency and automated tooling support.

**Format:**
```
<type>: <subject>

<body>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types:**
- `feat`: New feature or capability
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring without behavior change
- `test`: Adding or updating tests
- `chore`: Maintenance tasks (dependencies, tooling, etc.)
- `perf`: Performance improvements
- `build`: Build system or dependency changes
- `ci`: CI/CD configuration changes

**Examples:**

```
feat: implement lexer for Conduit source files

Add tokenization for all Conduit language constructs including
resources, fields, relationships, hooks, and expressions.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

```
docs: add CLAUDE.md for AI assistant guidance

Provides comprehensive guidance for Claude Code when working in this
repository, including project overview, architecture summary, and
implementation guidelines.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

```
refactor: simplify type checker error handling

Replace verbose error construction with builder pattern for cleaner
type checking code.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Commit body guidelines:**
- Explain **why** the change was made, not just what changed
- Keep subject line under 72 characters
- Wrap body at 72 characters
- Use imperative mood ("add" not "added" or "adds")
- Include Claude Code attribution footer on all commits

## Design Philosophy for AI Collaboration

**LLM-Optimized:**
- Explicit syntax eliminates ambiguity
- Namespaced functions prevent hallucination
- Structured error messages enable self-correction
- Pattern-based learning from introspection API

**Human-Readable:**
- Verbose but clear beats terse and ambiguous
- Intention is crystal clear from syntax
- Compile-time safety prevents bug classes

**Production-Ready:**
- Compiles to Go for native performance (10K+ req/s)
- Single binary deployment
- Sub-second compilation enables rapid iteration

## References

- **Compilation Target Research:** Go chosen for fast compilation, simple deployment, mature ecosystem
- **Type System:** Explicit nullability inspired by TypeScript, Swift
- **Query Language:** Fluent API inspired by Active Record, LINQ
- **Lifecycle Hooks:** Rails callbacks with explicit transaction control
- **Introspection:** Enables LLM pattern discovery and documentation generation
