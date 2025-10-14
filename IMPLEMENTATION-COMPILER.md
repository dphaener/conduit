# Conduit Compiler Implementation Guide

**Component:** Language Core & Compiler
**Last Updated:** 2025-10-13
**Status:** Implementation Ready

---

## Overview

The Conduit compiler transforms `.cdt` source files into executable Go programs. Unlike traditional compilers, it's designed with dual audiences in mind: **LLMs generating code** and **humans maintaining it**. The compiler enforces zero-ambiguity syntax, generates introspection metadata, and provides structured error messages optimized for AI consumption.

### Key Responsibilities

1. **Lexical Analysis** - Tokenize `.cdt` source files
2. **Parsing** - Build Abstract Syntax Tree (AST) from tokens
3. **Type Checking** - Enforce explicit nullability and type safety
4. **Expression Evaluation** - Handle complex expression language in hooks/validations
5. **Code Generation** - Emit idiomatic Go code with CRUD operations
6. **Introspection** - Generate runtime metadata for pattern discovery
7. **Error Reporting** - Provide LLM-optimized structured error messages

### Design Principles

- **Zero Ambiguity** - Every syntactic construct has exactly one interpretation
- **LLM-First Errors** - Structured JSON errors with fix suggestions
- **Fast Compilation** - Target < 1 second for typical projects
- **Progressive Disclosure** - Simple code stays simple (3 lines = valid resource)
- **Pattern Enforcement** - Compile-time convention checking

---

## Architecture

### Compilation Pipeline

```
┌─────────────┐
│ Source Code │
│  (.cdt)     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Lexer     │ ◄── Tokenization
│ (Tokenizer) │     • Keywords, identifiers, literals
└──────┬──────┘     • Position tracking
       │
       ▼
┌─────────────┐
│   Parser    │ ◄── Syntax Analysis
│  (AST Gen)  │     • Recursive descent
└──────┬──────┘     • Error recovery
       │
       ▼
┌─────────────┐
│ Type System │ ◄── Semantic Analysis
│  (Checker)  │     • Nullability enforcement
└──────┬──────┘     • Type inference
       │
       ▼
┌─────────────┐
│ Expression  │ ◄── Expression Language
│  Evaluator  │     • Hooks, validations
└──────┬──────┘     • Standard library calls
       │
       ▼
┌─────────────┐
│    Code     │ ◄── Go Code Generation
│  Generator  │     • CRUD operations
└──────┬──────┘     • Database schema
       │
       ▼
┌─────────────┐
│ Introspection│◄── Metadata Generation
│  Metadata   │     • Patterns, types, docs
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Go Binary  │
│  (Output)   │
└─────────────┘
```

### Component Dependencies

```
Lexer → Parser → Type Checker → Code Generator
                      ↓              ↓
                  AST Query      Introspection
                                   Metadata
```

---

## Component 1: Lexer (Tokenizer)

### Responsibility

Convert raw source text into a stream of tokens with position tracking.

### Token Types

```go
type TokenType int

const (
    // Keywords
    TOKEN_RESOURCE TokenType = iota
    TOKEN_ON
    TOKEN_AFTER
    TOKEN_BEFORE
    TOKEN_TRANSACTION
    TOKEN_ASYNC
    TOKEN_RESCUE
    TOKEN_WHERE
    TOKEN_HAS

    // Types
    TOKEN_STRING
    TOKEN_TEXT
    TOKEN_INT
    TOKEN_FLOAT
    TOKEN_BOOL
    TOKEN_TIMESTAMP
    TOKEN_UUID
    TOKEN_JSON
    TOKEN_ENUM
    TOKEN_ARRAY
    TOKEN_HASH

    // Operators
    TOKEN_BANG       // !
    TOKEN_QUESTION   // ?
    TOKEN_AT         // @
    TOKEN_PIPE       // |
    TOKEN_ARROW      // ->
    TOKEN_COLON      // :
    TOKEN_DOT        // .
    TOKEN_COMMA      // ,
    TOKEN_EQUALS     // =

    // Delimiters
    TOKEN_LBRACE     // {
    TOKEN_RBRACE     // }
    TOKEN_LPAREN     // (
    TOKEN_RPAREN     // )
    TOKEN_LBRACKET   // [
    TOKEN_RBRACKET   // ]

    // Literals
    TOKEN_IDENTIFIER
    TOKEN_INT_LITERAL
    TOKEN_FLOAT_LITERAL
    TOKEN_STRING_LITERAL
    TOKEN_TRUE
    TOKEN_FALSE
    TOKEN_NULL

    // Special
    TOKEN_COMMENT
    TOKEN_NEWLINE
    TOKEN_EOF
    TOKEN_ERROR
)

type Token struct {
    Type    TokenType
    Lexeme  string
    Literal interface{}
    Line    int
    Column  int
}
```

### Implementation Notes

```go
type Lexer struct {
    source   string
    start    int  // Start of current token
    current  int  // Current position
    line     int  // Current line number
    column   int  // Current column
    tokens   []Token
}

func (l *Lexer) ScanTokens() ([]Token, []Error) {
    // Scan all tokens
    // Track position for error reporting
    // Handle multi-line strings
    // Recognize keywords vs identifiers
}

func (l *Lexer) scanToken() Token {
    // Advance character by character
    // Match token patterns
    // Handle special cases (nullability markers, namespaces)
}
```

### Special Cases

1. **Nullability Markers** - `!` and `?` must be recognized after type names
2. **Namespace Separator** - `.` in `String.slugify()` vs field access
3. **Multi-line Strings** - Handle quotes, escapes, newlines
4. **Comments** - Single-line `#` comments, potentially multi-line `###`

### Performance Target

- **Speed:** < 10ms per 1000 lines
- **Memory:** < 5MB for 10k LOC

---

## Component 2: Parser

### Responsibility

Build Abstract Syntax Tree (AST) from token stream using recursive descent.

### AST Node Types

```go
// Root
type Program struct {
    Resources []ResourceNode
}

// Resource Definition
type ResourceNode struct {
    Name          string
    Documentation string
    Fields        []FieldNode
    Hooks         []HookNode
    Validations   []ValidationNode
    Relationships []RelationshipNode
    Location      SourceLocation
}

// Field Definition
type FieldNode struct {
    Name        string
    Type        TypeNode
    Nullable    bool      // ! vs ?
    Default     ExprNode
    Constraints []ConstraintNode
    Location    SourceLocation
}

// Type Representation
type TypeNode struct {
    Kind         TypeKind  // primitive, array, hash, enum, resource
    Name         string
    ElementType  *TypeNode // for array<T>
    KeyType      *TypeNode // for hash<K,V>
    ValueType    *TypeNode
    EnumValues   []string
}

// Lifecycle Hook
type HookNode struct {
    Timing      string   // "before", "after"
    Event       string   // "create", "update", "delete"
    Middleware  []string
    IsAsync     bool
    IsTransaction bool
    Body        []StatementNode
    Location    SourceLocation
}

// Expression (used in hooks, validations, defaults)
type ExprNode interface {
    exprNode()
}

// Implementations: LiteralExpr, BinaryExpr, UnaryExpr, CallExpr,
// FieldAccessExpr, SafeNavigationExpr, etc.
```

### Parsing Strategy

**Recursive Descent** with error recovery:

```go
type Parser struct {
    tokens  []Token
    current int
    errors  []ParseError
}

// Top-level
func (p *Parser) parseProgram() *Program {
    resources := []ResourceNode{}
    for !p.isAtEnd() {
        if res := p.parseResource(); res != nil {
            resources = append(resources, *res)
        }
    }
    return &Program{Resources: resources}
}

// Resource parsing
func (p *Parser) parseResource() *ResourceNode {
    p.consume(TOKEN_RESOURCE, "Expected 'resource' keyword")
    name := p.consume(TOKEN_IDENTIFIER, "Expected resource name")

    p.consume(TOKEN_LBRACE, "Expected '{'")

    fields := []FieldNode{}
    hooks := []HookNode{}

    for !p.check(TOKEN_RBRACE) && !p.isAtEnd() {
        if p.match(TOKEN_AT) {
            hooks = append(hooks, p.parseHook())
        } else {
            fields = append(fields, p.parseField())
        }
    }

    p.consume(TOKEN_RBRACE, "Expected '}'")

    return &ResourceNode{
        Name: name.Lexeme,
        Fields: fields,
        Hooks: hooks,
    }
}
```

### Error Recovery

When encountering syntax errors:

1. **Panic Mode Recovery** - Skip tokens until next synchronization point
2. **Synchronization Points** - Start of new resource, field, or hook
3. **Partial AST** - Generate nodes with error markers for IDE support
4. **Error Collection** - Collect all errors, don't stop on first

```go
func (p *Parser) synchronize() {
    p.advance()

    for !p.isAtEnd() {
        if p.previous().Type == TOKEN_NEWLINE {
            return
        }

        switch p.peek().Type {
        case TOKEN_RESOURCE, TOKEN_ON, TOKEN_AFTER, TOKEN_BEFORE:
            return
        }

        p.advance()
    }
}
```

### Performance Target

- **Speed:** < 50ms for 1000 lines
- **Memory:** < 20MB for typical project

---

## Component 3: Type System

### Type Representation

```go
type Type interface {
    String() string
    IsNullable() bool
    IsAssignableFrom(other Type) bool
}

// Primitive Types
type PrimitiveType struct {
    Kind     string  // "string", "int", "bool", etc.
    Nullable bool
}

// Array Type
type ArrayType struct {
    ElementType Type
    Nullable    bool
}

// Hash Type
type HashType struct {
    KeyType   Type
    ValueType Type
    Nullable  bool
}

// Enum Type
type EnumType struct {
    Name     string
    Values   []string
    Nullable bool
}

// Resource Type (for relationships)
type ResourceType struct {
    Name     string
    Nullable bool
}
```

### Type Inference Rules

1. **Field Types** - Must be explicit (no inference)
2. **Local Variables** - Inferred from assignment
3. **Function Returns** - Explicit in stdlib, inferred in custom functions
4. **Expressions** - Inferred from operands

```go
// Example type inference
x := 42              // inferred as int!
y := self.name       // inferred from field type
z := String.upcase(y) // inferred as string! (from stdlib signature)
```

### Nullability Rules

```
Required (!):
- Cannot be assigned null
- Cannot be assigned nullable value without check
- Field access always safe

Optional (?):
- Can be assigned null
- Requires null check before use
- Safe navigation (?.) auto-checks
```

### Type Checking

```go
type TypeChecker struct {
    resources map[string]*ResourceNode
    errors    []TypeError
}

func (tc *TypeChecker) CheckProgram(prog *Program) []TypeError {
    // Build symbol table
    tc.collectResources(prog)

    // Check each resource
    for _, res := range prog.Resources {
        tc.checkResource(&res)
    }

    return tc.errors
}

func (tc *TypeChecker) checkField(field *FieldNode) {
    // Validate type exists
    // Check default value matches type
    // Validate constraints
}

func (tc *TypeChecker) checkExpression(expr ExprNode, expected Type) Type {
    // Infer expression type
    // Ensure compatibility with expected
    // Check nullability flows
}
```

### Type Errors

```json
{
  "code": "TYP001",
  "type": "type_mismatch",
  "message": "Type mismatch: expected string!, got int!",
  "location": {"file": "post.cdt", "line": 42, "column": 15},
  "expected": "string!",
  "actual": "int!",
  "suggestion": "Use String.from_int() to convert",
  "example": "self.slug = String.from_int(self.id)"
}
```

---

## Component 4: Expression Language

### Expression Grammar

```
expression     → assignment
assignment     → IDENTIFIER "=" expression | logical_or
logical_or     → logical_and ( "||" logical_and )*
logical_and    → equality ( "&&" equality )*
equality       → comparison ( ( "==" | "!=" ) comparison )*
comparison     → term ( ( ">" | ">=" | "<" | "<=" ) term )*
term           → factor ( ( "+" | "-" ) factor )*
factor         → unary ( ( "*" | "/" | "%" ) unary )*
unary          → ( "!" | "-" ) unary | call
call           → primary ( "(" arguments? ")" | "." IDENTIFIER )*
primary        → literal | IDENTIFIER | "(" expression ")"

literal        → NUMBER | STRING | "true" | "false" | "null"
arguments      → expression ( "," expression )*
```

### Standard Library Functions

All stdlib functions are **namespaced** to eliminate ambiguity:

```go
// String namespace
String.slugify(text: string) -> string
String.upcase(text: string) -> string
String.downcase(text: string) -> string
String.replace(text: string, pattern: string, replacement: string) -> string
String.from_int(n: int) -> string

// Time namespace
Time.now() -> timestamp
Time.format(t: timestamp, format: string) -> string
Time.parse(s: string) -> timestamp?

// Array namespace
Array.first<T>(arr: array<T>) -> T?
Array.last<T>(arr: array<T>) -> T?
Array.count<T>(arr: array<T>) -> int

// Context namespace
Context.current_user() -> User?
Context.request_id() -> uuid

// Logger namespace
Logger.info(message: string, **metadata)
Logger.error(message: string, **metadata)
```

### Function Resolution

```go
type FunctionResolver struct {
    stdlib map[string]FunctionSignature
}

func (fr *FunctionResolver) ResolveCall(namespace, name string, args []ExprNode) (FunctionSignature, error) {
    fullName := namespace + "." + name
    sig, exists := fr.stdlib[fullName]

    if !exists {
        return nil, fmt.Errorf("undefined function: %s", fullName)
    }

    // Validate argument types
    // Check arity

    return sig, nil
}
```

---

## Component 5: Code Generator

### Responsibility

Transform AST into idiomatic Go code with:
- Struct definitions for resources
- CRUD operations
- Database schema generation
- HTTP handlers
- Validation logic
- Lifecycle hooks

### Generated Code Structure

```
output/
├── models/
│   ├── user.go           # Generated from User resource
│   ├── post.go           # Generated from Post resource
│   └── base.go           # Common model functionality
├── handlers/
│   ├── user_handlers.go  # Auto-generated REST handlers
│   └── post_handlers.go
├── migrations/
│   ├── 001_create_users.sql
│   └── 002_create_posts.sql
├── introspection/
│   └── metadata.json     # Runtime introspection data
└── main.go               # Application entry point
```

### Code Generation Examples

#### Resource → Go Struct

**Input (.cdt):**
```
resource User {
  username: string!
  email: string!
  bio: text?
  created_at: timestamp!
}
```

**Output (Go):**
```go
package models

import (
    "time"
    "database/sql"
)

type User struct {
    ID        int64      `db:"id" json:"id"`
    Username  string     `db:"username" json:"username"`
    Email     string     `db:"email" json:"email"`
    Bio       sql.NullString `db:"bio" json:"bio,omitempty"`
    CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// CRUD Operations
func (u *User) Create(db *sql.DB) error { /* ... */ }
func (u *User) Update(db *sql.DB) error { /* ... */ }
func (u *User) Delete(db *sql.DB) error { /* ... */ }
func FindUserByID(db *sql.DB, id int64) (*User, error) { /* ... */ }
func FindAllUsers(db *sql.DB) ([]*User, error) { /* ... */ }
```

#### Lifecycle Hooks → Go Methods

**Input (.cdt):**
```
@after create @transaction {
  self.slug = String.slugify(self.title)!

  @async {
    Email.send(self.author, "post_published") rescue |err| {
      Logger.error("Email failed", error: err)
    }
  }
}
```

**Output (Go):**
```go
func (p *Post) AfterCreate(db *sql.DB) error {
    // Transaction boundary
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Synchronous hook code
    p.Slug = stdlib.String.Slugify(p.Title)

    if err := p.Update(tx); err != nil {
        return err
    }

    // Async block - enqueue background job
    jobs.Enqueue("email_notification", map[string]interface{}{
        "user_id": p.AuthorID,
        "template": "post_published",
    })

    return tx.Commit()
}
```

### Code Generation Strategy

```go
type CodeGenerator struct {
    output    io.Writer
    indent    int
    resources map[string]*ResourceNode
}

func (cg *CodeGenerator) GenerateProgram(prog *Program) error {
    // Generate package declaration
    cg.writePackage("main")

    // Generate imports
    cg.writeImports()

    // Generate models
    for _, res := range prog.Resources {
        cg.generateResource(&res)
    }

    // Generate router
    cg.generateRouter(prog)

    // Generate main()
    cg.generateMain()

    return nil
}

func (cg *CodeGenerator) generateResource(res *ResourceNode) {
    // Struct definition
    cg.generateStruct(res)

    // CRUD methods
    cg.generateCreate(res)
    cg.generateUpdate(res)
    cg.generateDelete(res)
    cg.generateFinders(res)

    // Hooks
    for _, hook := range res.Hooks {
        cg.generateHook(res, &hook)
    }

    // Validations
    cg.generateValidations(res)
}
```

### Performance Target

- **Speed:** < 100ms for 50 resources
- **Output Quality:** `go fmt` compatible, idiomatic Go

---

## Component 6: Error Handling

### Error Message Design

#### For LLMs (JSON Format)

```json
{
  "status": "error",
  "errors": [
    {
      "code": "SYN001",
      "type": "syntax",
      "severity": "error",
      "file": "models/user.cdt",
      "line": 15,
      "column": 12,
      "field_path": "User.username",
      "message": "Missing nullability indicator (! or ?)",
      "context": {
        "current": "username: string",
        "expected": ["username: string!", "username: string?"]
      },
      "suggestion": "Add ! for required or ? for optional",
      "fix": {
        "auto_fixable": true,
        "replacement": "username: string!"
      },
      "documentation": "https://docs.conduit-lang.org/errors/SYN001"
    }
  ],
  "warnings": [],
  "compilation_time_ms": 127
}
```

#### For Humans (Terminal Output)

```
❌ Syntax Error in models/user.cdt

Line 15, Column 12:
  14 | resource User {
  15 |   username: string @required
               ~~~~~~~~~~~~~~~~~~~~^ Missing nullability marker
  16 |   email: string!

The field 'username' needs to specify if it can be null.

Quick Fix:
  username: string!  ← Required field
  username: string?  ← Optional field

Old syntax detected: @required is no longer supported in v2.0
Migration: conduit migrate v1-to-v2 models/user.cdt

Learn more: https://docs.conduit-lang.org/nullability
```

### Error Categories

| Code Range | Category | Severity | Examples |
|------------|----------|----------|----------|
| SYN001-099 | Syntax | Error | Missing delimiters, invalid tokens |
| TYP100-199 | Type | Error | Type mismatches, undefined types |
| SEM200-299 | Semantic | Error | Undefined variables, circular refs |
| REL300-399 | Relationship | Error | Invalid foreign keys |
| PAT400-499 | Pattern | Warning | Convention violations |
| VAL500-599 | Validation | Error | Invalid constraints |
| GEN600-699 | Generation | Error | Codegen failures |
| OPT700-799 | Optimization | Info | Performance hints |

### Error Recovery

```go
type ErrorRecovery struct {
    strategy RecoveryStrategy
}

type RecoveryStrategy int

const (
    PANIC_MODE RecoveryStrategy = iota  // Skip to sync point
    PHRASE_LEVEL                         // Skip to next valid phrase
    ERROR_PRODUCTIONS                    // Insert error node in AST
)

func (er *ErrorRecovery) Recover(p *Parser, err ParseError) {
    switch er.strategy {
    case PANIC_MODE:
        p.synchronize()
    case PHRASE_LEVEL:
        p.skipToNextField()
    case ERROR_PRODUCTIONS:
        p.insertErrorNode()
    }
}
```

---

## Component 7: Introspection Metadata

### Metadata Generation

The compiler generates structured metadata for runtime introspection:

```json
{
  "version": "1.0.0",
  "resources": [
    {
      "name": "User",
      "fields": [
        {
          "name": "username",
          "type": "string!",
          "constraints": ["unique", "min_length:3"]
        }
      ],
      "hooks": [
        {
          "timing": "after",
          "event": "create",
          "has_transaction": true,
          "has_async": false
        }
      ],
      "routes": [
        {"method": "GET", "path": "/users", "handler": "index"},
        {"method": "POST", "path": "/users", "handler": "create"}
      ]
    }
  ],
  "patterns": [
    {
      "name": "authenticated_handler",
      "template": "@on create: [auth]",
      "occurrences": 5
    }
  ]
}
```

### Metadata Embedding

```go
// Embedded in generated Go code
var IntrospectionMetadata = `{...}`

func GetMetadata() map[string]interface{} {
    var meta map[string]interface{}
    json.Unmarshal([]byte(IntrospectionMetadata), &meta)
    return meta
}
```

---

## Development Phases

### Phase 1: Foundation (Weeks 1-4)

**Goal:** Basic compilation working end-to-end

**Week 1-2: Lexer & Parser**
- [ ] Implement tokenizer for core syntax
- [ ] Build recursive descent parser
- [ ] Define AST node structures
- [ ] Add position tracking
- [ ] **Success:** Parse simple resources without errors

**Week 3: Type System Core**
- [ ] Implement type representation
- [ ] Basic type checking logic
- [ ] Nullability validation
- [ ] **Success:** Detect type mismatches

**Week 4: Go Code Generation**
- [ ] AST to Go transformer
- [ ] Basic struct generation
- [ ] Import management
- [ ] **Success:** Generate compilable Go code

**Milestone:** Compile "Hello World" resource to running Go server

### Phase 2: Completeness (Weeks 5-8)

**Goal:** Full v2.0 syntax support

**Week 5-6: Advanced Syntax**
- [ ] Lifecycle hooks parsing
- [ ] Computed fields
- [ ] Query language
- [ ] Validation blocks

**Week 7: Standard Library**
- [ ] Namespace resolution
- [ ] Built-in function signatures
- [ ] Type checking for stdlib calls

**Week 8: Error Handling**
- [ ] Comprehensive error messages
- [ ] JSON error format
- [ ] Error recovery mechanisms

**Milestone:** Compile complex blog example successfully

### Phase 3: Developer Experience (Weeks 9-12)

**Goal:** Production-ready tooling

**Week 9-10: Diagnostics**
- [ ] Advanced error detection
- [ ] Warning system
- [ ] Performance hints

**Week 11: Tooling APIs**
- [ ] AST access API
- [ ] Type query API
- [ ] Incremental compilation

**Week 12: Documentation**
- [ ] API documentation
- [ ] Integration guides
- [ ] Example collection

**Milestone:** External developers can build tools

### Phase 4: Optimization (Weeks 13-16)

**Goal:** Performance and scale

**Week 13-14: Performance**
- [ ] Incremental compilation
- [ ] Parallel parsing
- [ ] Memory optimization

**Week 15-16: Advanced Features**
- [ ] Pattern extraction
- [ ] Introspection generation
- [ ] Query optimization

**Milestone:** Compile large applications in < 1 second

---

## Performance Targets

| Component | Target | Measurement |
|-----------|--------|-------------|
| Lexer | < 10ms/1000 LOC | Tokenization speed |
| Parser | < 50ms/1000 LOC | AST generation |
| Type Checker | < 100ms/resource | Type validation |
| Code Generator | < 100ms/50 resources | Go code emission |
| **Full Compilation** | **< 1 second** | **Typical project** |

### Memory Targets

- Lexer: < 5MB for 10k LOC
- Parser: < 20MB for typical project
- Type Checker: < 50MB for large projects
- Code Generator: < 100MB peak

---

## Testing Strategy

### Unit Tests

```go
// Lexer tests
func TestLexer_TokenizeResource(t *testing.T)
func TestLexer_NullabilityMarkers(t *testing.T)
func TestLexer_NamespacedCalls(t *testing.T)

// Parser tests
func TestParser_SimpleResource(t *testing.T)
func TestParser_NestedTypes(t *testing.T)
func TestParser_ErrorRecovery(t *testing.T)

// Type checker tests
func TestTypeChecker_NullabilityViolation(t *testing.T)
func TestTypeChecker_TypeMismatch(t *testing.T)

// Code generator tests
func TestCodeGen_ResourceToStruct(t *testing.T)
func TestCodeGen_HooksToMethods(t *testing.T)
```

### Integration Tests

```go
func TestCompiler_EndToEnd(t *testing.T) {
    source := `
        resource User {
            username: string!
            email: string!
        }
    `

    result := compiler.Compile(source)
    assert.True(t, result.Success)
    assert.Contains(t, result.Output, "type User struct")
}
```

### LLM Validation Tests

```go
func TestCompiler_LLMErrorUnderstanding(t *testing.T) {
    // Generate invalid code
    // Compile and get error
    // Feed error to LLM
    // Verify LLM can fix the error
}
```

### Benchmark Tests

```go
func BenchmarkCompiler_LargeProject(b *testing.B) {
    // 50 resources, 500 fields
    // Measure compilation time
    // Target: < 1 second
}
```

---

## Integration Points

### With Runtime System

- Compiler generates metadata → Runtime loads metadata
- Compiler emits hooks → Runtime executes hooks
- Compiler enforces patterns → Runtime introspection queries patterns

### With Web Framework

- Compiler generates routes → Framework registers routes
- Compiler generates handlers → Framework calls handlers
- Compiler validates middleware → Framework applies middleware

### With ORM

- Compiler parses schema → ORM generates SQL
- Compiler validates relationships → ORM loads relationships
- Compiler generates CRUD → ORM executes queries

### With Tooling

- Compiler exposes AST → LSP provides completions
- Compiler tracks positions → LSP navigates code
- Compiler detects errors → LSP shows diagnostics

---

## Critical Success Factors

✅ **Compilation Speed** < 1 second for typical project
✅ **LLM Success Rate** 95%+ first-attempt compilation
✅ **Error Clarity** 90%+ errors self-correctable by LLM
✅ **Zero Ambiguity** Single interpretation for every construct
✅ **Pattern Enforcement** 100% convention compliance

---

## Major Risks & Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Expression language complexity | HIGH | Phased implementation, start simple |
| Type system soundness | HIGH | Formal specification, property testing |
| Compilation speed degradation | MEDIUM | Incremental compilation from day 1 |
| LLMs can't adapt to syntax | MEDIUM | Extensive prompt engineering examples |

---

## Next Steps

### Immediate Actions
1. Create parser generator grammar specification
2. Define AST node types in Go
3. Implement basic tokenizer
4. Write type system specification
5. Design error message format

### Research Needed
1. Parser error recovery strategies
2. Incremental compilation algorithms
3. LSP implementation requirements
4. Go code generation patterns

### Prototypes to Build
1. Simple tokenizer for subset of syntax
2. AST structure with visitor pattern
3. Type checker for basic types
4. Go code generator for resources

---

## References

- See `LANGUAGE-SPEC.md` for complete syntax specification
- See `docs/open-questions.md` for unresolved design decisions
- See `docs/design-decisions.md` for architectural rationale
- See `docs/research/` for historical analysis documents

---

**End of Compiler Implementation Guide**
