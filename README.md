# Conduit

**An LLM-First Programming Language for Web Applications**

Conduit is a programming language designed from the ground up for AI-assisted development. It provides explicit, unambiguous syntax that makes it easy for both LLMs and humans to build production-ready web applications.

## ⚠️ Project Status

**Alpha - Actively Under Development**

Conduit is in active development. The compiler is functional and can generate working Go applications, but many features are incomplete or missing. **Not recommended for production use.**

## What Works Today (v0.1.0)

### Core Features
- ✅ Basic compiler (lexer, parser, type checker, code generator)
- ✅ Resource definitions with fields and explicit nullability (`!` vs `?`)
- ✅ Primitive types (string, int, float, bool, uuid, timestamp, email, url, etc.)
- ✅ Structural types (array, hash, enum, inline structs)
- ✅ Field constraints (@min, @max, @unique, @primary, @auto, @default)
- ✅ Relationships (belongs_to with foreign_key metadata)
- ✅ REST API generation with CRUD endpoints
- ✅ Lifecycle hooks (`@before create/update/delete`, `@after create/update/delete`)
- ✅ Custom constraints (`@constraint` blocks - parsed, not yet executed)
- ✅ Database migrations (PostgreSQL)

### Standard Library (15 MVP Functions)
- ✅ String: length, slugify, upcase, downcase, trim, contains, replace
- ✅ Time: now, format, parse
- ✅ Array: length, contains
- ✅ UUID: generate, validate
- ✅ Random: int

### What's NOT Yet Implemented

**See [ROADMAP.md](ROADMAP.md) for complete details.**

Key missing features:
- ❌ `@has_many` relationships (one-to-many)
- ❌ `@scope` query scopes
- ❌ `@validate` procedural validation (execution)
- ❌ `@invariant` runtime invariants
- ❌ `@computed` fields
- ❌ `@function` custom functions
- ❌ Query builder (find, where, joins, etc.)
- ❌ Expression language (if/match/rescue)
- ❌ Most stdlib functions (Logger, Cache, Crypto, Context, etc.)
- ❌ LSP/IDE integration
- ❌ Hot reload / watch mode
- ❌ Testing framework
- ❌ GraphQL support
- ❌ Background jobs

**Important:** The LANGUAGE-SPEC.md is aspirational. Many documented features don't work yet. Always check ROADMAP.md for current status.

## Key Features

- 🤖 **LLM-Optimized Syntax** - Explicit nullability, namespaced stdlib, zero ambiguity
- ⚡ **Compile to Go** - Fast compilation, single binary deployment, native performance
- 🗄️ **Built-in ORM** - Type-safe queries, relationship management, automatic migrations
- 🌐 **REST API Generation** - Automatic CRUD endpoints from resource definitions
- 🔍 **Runtime Introspection** - Query schema, discover patterns, generate documentation
- 🛠️ **Developer Tooling** - LSP, debugger, hot reload, code formatter *(planned)*

## Quick Start

### Installation

**Prerequisites:**
- Go 1.23 or later
- PostgreSQL 15+ (for database features)

**Install from source:**

```bash
git clone https://github.com/conduit-lang/conduit.git
cd conduit
go build -o conduit ./cmd/conduit
sudo mv conduit /usr/local/bin/
```

**Set up environment:**

```bash
# Required for development builds
export CONDUIT_ROOT=/path/to/conduit
```

### Create Your First Project

```bash
# Create a new directory
mkdir my-blog
cd my-blog

# Create app directory and a resource file
mkdir -p app/resources
cat > app/resources/post.cdt << 'EOF'
/// Blog post with title and content
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  published: bool! @default(false)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.slug = String.slugify(self.title)
  }
}
EOF

# Build the application
conduit build

# Run it!
./build/app
```

This generates:
- REST API endpoints (`GET /posts`, `POST /posts`, etc.)
- Database schema and migrations
- Type-safe validation
- Lifecycle hooks

## Example

```conduit
/// Blog post with title and content
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  content: text! @min(100)
  status: enum ["draft", "published"]! @default("draft")

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.slug = String.slugify(self.title)
  }

  @constraint published_requires_content {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts must have at least 500 characters"
  }
}
```

This automatically generates:
- REST API endpoints (GET/POST/PUT/DELETE)
- Database schema and migrations
- Type-safe query methods
- Validation and constraints
- Lifecycle hook execution

## Documentation

- 📚 **[GETTING-STARTED.md](GETTING-STARTED.md)** - Detailed getting started guide
- 📖 **[LANGUAGE-SPEC.md](LANGUAGE-SPEC.md)** - Complete language specification

## Design Philosophy

### Explicitness over Brevity
LLMs don't experience tedium, they experience ambiguity. Verbose in service of clarity is good.

### Zero Ambiguity
- Every type specifies nullability: `string!` (required) vs `string?` (optional)
- All built-in functions are namespaced: `String.slugify()` not `slugify()`
- Transaction boundaries are explicit: `@transaction`, `@async`

### Progressive Disclosure
- Simple things stay simple (3 lines can be a complete resource)
- Complexity is available when needed
- No forced sophistication

## Why Conduit?

### For LLMs
- **No Hallucination**: Namespaced stdlib means LLMs can't invent functions
- **Type Safety**: Explicit nullability prevents null reference errors
- **Clear Patterns**: Structured conventions enable pattern replication
- **Error Recovery**: Structured errors enable self-correction

### For Humans
- **Readable**: Intention is crystal clear
- **Safe**: Compile-time checks prevent entire bug classes
- **Fast**: Sub-second compilation enables rapid iteration
- **Simple**: Single binary deployment, no runtime dependencies

### For Teams
- **Consistent**: Conventions enforced at compile-time
- **Discoverable**: Introspection API reveals patterns *(planned)*
- **Maintainable**: Explicit code is easier to understand
- **Scalable**: Go's performance handles production workloads

## Compilation Target

Conduit compiles to Go source code, which is then compiled to native binaries using the Go toolchain. This approach provides:

- **Fast compilation** - Go compiles in seconds
- **Simple deployment** - Single binary, no dependencies
- **Native performance** - 10,000+ req/s per server
- **Mature ecosystem** - Leverage existing Go libraries

## Contributing

We welcome contributions! Areas where help is needed:

- **Compiler**: Bug fixes, error messages, type checking improvements
- **Runtime**: Stdlib functions, validation, lifecycle management
- **ORM**: Relationship support, query builder, migrations
- **Documentation**: Examples, tutorials, API docs
- **Testing**: Unit tests, integration tests, edge cases
- **Tooling**: LSP, formatter, debugger integration

Please open an issue to discuss before starting major work.

## Technology Stack

- **Source Language**: Conduit (`.cdt` files)
- **Target Language**: Go 1.23+
- **Database**: PostgreSQL 15+ (primary), MySQL/SQLite (planned)
- **Compiler**: Go
- **Runtime**: Go standard library + custom runtime
- **Tooling**: Cobra (CLI), fsnotify (file watching), Delve (debugging)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Community

- **GitHub**: https://github.com/conduit-lang/conduit
- **Issues**: https://github.com/conduit-lang/conduit/issues
- **Discussions**: https://github.com/conduit-lang/conduit/discussions

## Acknowledgments

Conduit is inspired by:
- **Rails** - Convention over configuration
- **Prisma** - Type-safe database access
- **TypeScript** - Explicit type system
- **Elixir** - Explicit patterns and functional approach
- **Go** - Simplicity and fast compilation

Built with the belief that programming languages should serve both humans and AI.

---

**Questions?** Open an issue or start a discussion!
