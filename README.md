# Conduit

**An LLM-First Programming Language for Web Applications**

Conduit is a programming language designed from the ground up for AI-assisted development. It provides explicit, unambiguous syntax that makes it easy for both LLMs and humans to build production-ready web applications.

## Key Features

- 🤖 **LLM-Optimized Syntax** - Explicit nullability, namespaced stdlib, zero ambiguity
- ⚡ **Compile to Go** - Fast compilation, single binary deployment, native performance
- 🗄️ **Built-in ORM** - Type-safe queries, relationship management, automatic migrations
- 🌐 **REST API Generation** - Automatic CRUD endpoints from resource definitions
- 🔍 **Runtime Introspection** - Query schema, discover patterns, generate documentation
- 🛠️ **Developer Tooling** - LSP, debugger, hot reload, code formatter

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
}
```

This automatically generates:
- REST API endpoints (GET/POST/PUT/DELETE)
- Database schema and migrations
- Type-safe query methods
- Validation and constraints
- Lifecycle hook execution

## Project Status

**Current Stage:** Design & Planning

This repository contains the complete language specification and implementation guides. Active development of the compiler and runtime will begin soon.

## Documentation

### Getting Started
- 📚 **[GETTING-STARTED.md](GETTING-STARTED.md)** - Quick start guide for developers

### Language Reference
- 📖 **[LANGUAGE-SPEC.md](LANGUAGE-SPEC.md)** - Complete language specification
- 🏗️ **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture overview

### Implementation Guides
- 🔧 **[IMPLEMENTATION-COMPILER.md](IMPLEMENTATION-COMPILER.md)** - Compiler (lexer, parser, type checker, code generator)
- ⚙️ **[IMPLEMENTATION-RUNTIME.md](IMPLEMENTATION-RUNTIME.md)** - Runtime system (introspection, lifecycle, validation)
- 🗃️ **[IMPLEMENTATION-ORM.md](IMPLEMENTATION-ORM.md)** - ORM (query builder, relationships, migrations)
- 🌐 **[IMPLEMENTATION-WEB.md](IMPLEMENTATION-WEB.md)** - Web framework (routing, handlers, middleware)
- 🛠️ **[IMPLEMENTATION-TOOLING.md](IMPLEMENTATION-TOOLING.md)** - Developer tools (CLI, LSP, debugger, formatter)

### Pattern Validation
- 🧪 **[docs/PATTERN-VALIDATION-GUIDE.md](docs/PATTERN-VALIDATION-GUIDE.md)** - LLM pattern validation system
- ✅ **[docs/PATTERN-QUALITY-CHECKLIST.md](docs/PATTERN-QUALITY-CHECKLIST.md)** - Pattern quality review checklist

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
- **Discoverable**: Introspection API reveals patterns
- **Maintainable**: Explicit code is easier to understand
- **Scalable**: Go's performance handles production workloads
- **Validated**: Built-in LLM pattern validation ensures AI-friendliness

## Compilation Target

Conduit compiles to Go source code, which is then compiled to native binaries using the Go toolchain. This approach provides:

- **Fast compilation** - Go compiles in seconds
- **Simple deployment** - Single binary, no dependencies
- **Native performance** - 10,000+ req/s per server
- **Mature ecosystem** - Leverage existing Go libraries

## Timeline

### Phase 1: Foundation (Months 1-3)
- Compiler implementation (lexer, parser, type checker)
- Go code generation
- Basic ORM with PostgreSQL
- CLI tool

### Phase 2: Web Framework (Months 4-6)
- REST API generation
- Middleware system
- Request/response handling
- Validation framework

### Phase 3: Tooling (Months 7-9)
- Language Server Protocol (LSP)
- Watch mode and hot reload
- Debug Adapter Protocol (DAP)
- Code formatter

### Phase 4: Introspection (Months 10-12)
- Runtime introspection API
- Pattern extraction
- Documentation generation
- LLM integration examples

## Contributing

We welcome contributions! This project is in the planning phase. Once implementation begins, we'll need help with:

- Compiler development (Go)
- Runtime implementation (Go)
- Standard library functions
- Documentation and examples
- Testing and validation
- IDE integrations

## Technology Stack

- **Source Language**: Conduit (`.cdt` files)
- **Target Language**: Go 1.23+
- **Database**: PostgreSQL 15+ (primary), MySQL/SQLite (planned)
- **Compiler**: Go (self-hosted)
- **Runtime**: Go standard library + custom runtime
- **Tooling**: Cobra (CLI), fsnotify (file watching), Delve (debugging)

## License

[To be determined - likely MIT or Apache 2.0]

## Community

- **GitHub**: https://github.com/conduit-lang/conduit
- **Discord**: [Coming soon]
- **Forum**: [Coming soon]
- **Documentation**: [Coming soon]

## Acknowledgments

Conduit is inspired by:
- **Rails** - Convention over configuration
- **Prisma** - Type-safe database access
- **TypeScript** - Explicit type system
- **Elixir** - Explicit pipe operator and pattern matching
- **Go** - Simplicity and fast compilation

Built with the belief that programming languages should serve both humans and AI.

---

**Status**: Design Complete, Implementation Starting Soon

**Questions?** Open an issue or join our Discord!
