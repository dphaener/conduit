# Conduit Tooling API

The Tooling API provides programmatic access to the Conduit compiler for IDE integration via Language Server Protocol (LSP). It exposes parsing, type checking, symbol resolution, and code intelligence features in a thread-safe, performance-optimized manner.

## Features

- **Fast Parsing**: Parse `.cdt` files with sub-50ms response times
- **Type Checking**: Full type checking with nullability analysis
- **Symbol Indexing**: Fast symbol lookup for go-to-definition and references
- **Code Intelligence**: Hover information, completions, and diagnostics
- **Document Management**: Thread-safe document caching and versioning
- **Incremental Updates**: Efficient document re-parsing on changes

## Architecture

The tooling API is built on top of the existing compiler infrastructure:

```
┌─────────────────────────────────────────┐
│         Tooling API (api.go)            │
│  ┌────────────┐    ┌────────────┐      │
│  │  Document  │    │   Symbol   │      │
│  │   Cache    │    │   Index    │      │
│  └────────────┘    └────────────┘      │
└─────────────────────────────────────────┘
              │            │
              ▼            ▼
┌─────────────────────────────────────────┐
│         Compiler Infrastructure          │
│  ┌────────┐  ┌────────┐  ┌───────────┐ │
│  │ Lexer  │→ │ Parser │→ │   Type    │ │
│  │        │  │        │  │  Checker  │ │
│  └────────┘  └────────┘  └───────────┘ │
└─────────────────────────────────────────┘
```

## Usage

### Basic Parsing

```go
import "github.com/conduit-lang/conduit/internal/tooling"

// Create API instance
api := tooling.NewAPI()

// Parse a source file
source := `
resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
}
`

doc, err := api.ParseFile("user.cdt", source)
if err != nil {
    log.Fatal(err)
}

// Check for errors
diagnostics := api.GetDiagnostics("user.cdt")
for _, diag := range diagnostics {
    fmt.Printf("%s:%d:%d: %s\n",
        "user.cdt", diag.Range.Start.Line, diag.Range.Start.Character, diag.Message)
}
```

### Hover Information

```go
// Get hover information at a position
hover, err := api.GetHover("user.cdt", tooling.Position{
    Line:      2,  // Zero-based
    Character: 5,
})
if err != nil {
    log.Fatal(err)
}

if hover != nil {
    fmt.Println(hover.Contents)  // Markdown-formatted hover text
}
```

### Code Completions

```go
// Get completions at a position
completions, err := api.GetCompletions("user.cdt", tooling.Position{
    Line:      3,
    Character: 10,
})
if err != nil {
    log.Fatal(err)
}

for _, item := range completions {
    fmt.Printf("%s: %s\n", item.Label, item.Detail)
}
```

### Go-to-Definition

```go
// Get definition location
loc, err := api.GetDefinition("user.cdt", tooling.Position{
    Line:      5,
    Character: 10,
})
if err != nil {
    log.Fatal(err)
}

if loc != nil {
    fmt.Printf("Definition at %s:%d:%d\n",
        loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
}
```

### Find References

```go
// Find all references to a symbol
refs, err := api.GetReferences("user.cdt", tooling.Position{
    Line:      2,
    Character: 5,
})
if err != nil {
    log.Fatal(err)
}

for _, ref := range refs {
    fmt.Printf("Reference at %s:%d:%d\n",
        ref.URI, ref.Range.Start.Line, ref.Range.Start.Character)
}
```

### Document Symbols

```go
// Get all symbols in a document
symbols, err := api.GetDocumentSymbols("user.cdt")
if err != nil {
    log.Fatal(err)
}

for _, sym := range symbols {
    fmt.Printf("%s: %s (%s)\n", sym.Name, sym.Type, sym.Kind)
}
```

## API Reference

### Core Types

#### API
Main API instance for document management and queries.

**Methods:**
- `ParseFile(uri, content string) (*Document, error)` - Parse a source file
- `UpdateDocument(uri, content string, version int) (*Document, error)` - Update an existing document
- `GetDocument(uri string) (*Document, bool)` - Get a cached document
- `CloseDocument(uri string)` - Remove a document from cache
- `GetDiagnostics(uri string) []Diagnostic` - Get diagnostics for a document
- `GetHover(uri string, pos Position) (*Hover, error)` - Get hover information
- `GetCompletions(uri string, pos Position) ([]CompletionItem, error)` - Get completions
- `GetDefinition(uri string, pos Position) (*Location, error)` - Go-to-definition
- `GetReferences(uri string, pos Position) ([]Location, error)` - Find references
- `GetDocumentSymbols(uri string) ([]*Symbol, error)` - Get document symbols

#### Document
Represents a parsed and type-checked document.

**Fields:**
- `URI string` - Document identifier
- `Content string` - Source code
- `Version int` - Document version
- `AST *ast.Program` - Parsed AST
- `ParseErrors []parser.ParseError` - Syntax errors
- `TypeErrors typechecker.ErrorList` - Type errors
- `Symbols []*Symbol` - Extracted symbols

#### Position
Zero-based line and character position (LSP-compatible).

**Fields:**
- `Line int` - Zero-based line number
- `Character int` - Zero-based character offset

#### Range
Source range with start and end positions.

**Fields:**
- `Start Position` - Range start
- `End Position` - Range end

#### Symbol
Represents a named entity in the source code.

**Fields:**
- `Name string` - Symbol name
- `Kind SymbolKind` - Symbol kind (resource, field, etc.)
- `Range Range` - Source location
- `Type string` - Type information
- `ContainerName string` - Parent resource name
- `Documentation string` - Doc comment
- `Detail string` - Additional information

#### Diagnostic
Represents a compilation error or warning.

**Fields:**
- `Range Range` - Error location
- `Severity DiagnosticSeverity` - Error, warning, info, or hint
- `Code string` - Error code
- `Message string` - Error message
- `Source string` - Error source (always "conduit")

## Performance

The API is designed for LSP use cases with the following performance targets:

- **Hover**: < 50ms response time
- **Completions**: < 100ms response time
- **Diagnostics**: < 200ms after change
- **Parse**: < 1s for 1000 LOC files

### Thread Safety

The API is fully thread-safe and can be used concurrently:

```go
api := tooling.NewAPI()

// Safe for concurrent use
go func() {
    api.GetHover("file1.cdt", pos)
}()

go func() {
    api.GetCompletions("file2.cdt", pos)
}()
```

### Caching

The API maintains an internal document cache to avoid re-parsing unchanged files:

- Documents are cached by URI
- Cache is automatically invalidated on content changes
- Symbol index is incrementally updated
- Default cache size: 100 documents (configurable)

## Configuration

```go
config := &tooling.Config{
    CacheSize:                50,     // Max documents in cache
    EnableIncrementalParsing: true,   // Enable incremental parsing
}

api := tooling.NewAPIWithConfig(config)
```

## Integration with LSP

The API types are designed to map directly to LSP protocol types:

```go
// LSP textDocument/hover request
func handleHover(params lsp.HoverParams) (*lsp.Hover, error) {
    hover, err := api.GetHover(params.TextDocument.URI, tooling.Position{
        Line:      params.Position.Line,
        Character: params.Position.Character,
    })
    if err != nil {
        return nil, err
    }

    return &lsp.Hover{
        Contents: lsp.MarkupContent{
            Kind:  lsp.MarkdownKind,
            Value: hover.Contents,
        },
        Range: toLSPRange(hover.Range),
    }, nil
}
```

## Testing

The package includes comprehensive tests with >85% code coverage:

```bash
# Run tests
go test ./internal/tooling/...

# Run with coverage
go test -cover ./internal/tooling/...

# Run benchmarks
go test -bench=. ./internal/tooling/...
```

## Implementation Notes

### Symbol Extraction

Symbols are extracted from the AST after successful parsing:

- Resources become `SymbolKindResource`
- Fields become `SymbolKindField`
- Relationships become `SymbolKindRelationship`
- Hooks become `SymbolKindHook`
- Computed fields become `SymbolKindComputed`
- Scopes become `SymbolKindScope`

### Type Information

Type strings follow Conduit syntax with nullability markers:

- `string!` - Required string
- `int?` - Optional integer
- `array<string!>!` - Required array of required strings
- `hash<string!, int!>?` - Optional hash

### Completion Context Detection

The API detects completion context by analyzing the line content:

- After `@` → Annotation completions
- After `:` → Type completions
- After `Namespace.` → Namespace method completions
- Default → Keyword completions

## Future Enhancements

- Incremental parsing for faster updates
- Semantic tokens for syntax highlighting
- Code actions (quick fixes)
- Rename refactoring
- Document formatting
- Signature help
- Call hierarchy
