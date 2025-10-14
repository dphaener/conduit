# IMPLEMENTATION-TOOLING.md

**Component:** Tooling & Development Experience
**Status:** Implementation Ready
**Last Updated:** 2025-10-13
**Estimated Effort:** 28-32 weeks (195-230 person-days)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Component 1: CLI Tool Architecture](#component-1-cli-tool-architecture)
4. [Component 2: Language Server Protocol (LSP)](#component-2-language-server-protocol-lsp)
5. [Component 3: Watch Mode & Hot Reload](#component-3-watch-mode--hot-reload)
6. [Component 4: Debug Adapter Protocol (DAP)](#component-4-debug-adapter-protocol-dap)
7. [Component 5: Code Formatting Engine](#component-5-code-formatting-engine)
8. [Component 6: Project Templates](#component-6-project-templates)
9. [Component 7: Build System](#component-7-build-system)
10. [Component 8: Documentation Generation](#component-8-documentation-generation)
11. [Development Phases](#development-phases)
12. [Testing Strategy](#testing-strategy)
13. [Integration Points](#integration-points)
14. [Performance Targets](#performance-targets)
15. [Risk Mitigation](#risk-mitigation)
16. [Success Criteria](#success-criteria)

---

## Overview

### Purpose

The Tooling & Development Experience component provides the essential developer tooling that bridges traditional developer workflows and LLM-assisted development. It makes the language accessible to both humans and AI systems through:

1. **CLI Tool** - Fast, intuitive command-line interface for all operations
2. **LSP Server** - Professional IDE integration with rich features
3. **Hot Reload** - Sub-second feedback loop for development
4. **Debugger** - Source-level debugging via Delve integration
5. **Formatter** - Deterministic, opinionated code formatting
6. **Templates** - Smart project scaffolding and code generation
7. **Build System** - Fast incremental compilation with caching
8. **Documentation** - Auto-generated API docs from introspection

### Design Philosophy

**Three User Types:**
1. **Human Developers** - Traditional IDE features (syntax highlighting, autocomplete, errors)
2. **LLMs** - Introspection queries, pattern discovery, structured output
3. **Hybrid Workflows** - Humans use IDE, LLMs use CLI, both benefit from hot reload

**Core Principles:**
- **Fast is a feature** - CLI < 100ms, LSP < 50ms, hot reload < 500ms
- **Zero-config philosophy** - Sensible defaults, no decision fatigue
- **Explicitness over magic** - Clear, predictable behavior
- **Unified toolchain** - One CLI for everything
- **LLM-first design** - Structured output, pattern awareness

### Key Innovations

1. **Pattern-Aware Tooling** - Tools understand conventions, not just syntax
2. **LLM-Optimized CLI** - Structured JSON output for AI consumption
3. **Introspection Integration** - Query codebase structure at runtime
4. **Zero-Config Hot Reload** - Automatic change detection and reloading
5. **Source Map Debugging** - Debug generated Go from source files

### Compilation Target

**Primary:** Go
**Dependencies:**
- github.com/spf13/cobra (CLI framework)
- github.com/fsnotify/fsnotify (file watching)
- github.com/go-delve/delve (debugging)
- Standard LSP protocol (JSON-RPC over stdio)

**Generated Output:**
- CLI binary (llmlang)
- LSP server binary (llmlang-lsp)
- Debug adapter binary (llmlang-dap)

---

## Architecture

### High-Level Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                        CLI Tool (Cobra)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │    Build     │  │     Run      │  │   Format     │        │
│  │   Commands   │  │   Commands   │  │   Commands   │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ Introspect   │  │   Template   │  │     Docs     │        │
│  │   Commands   │  │   Commands   │  │   Commands   │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└────────────────────────┬───────────────────────────────────────┘
                         │
           ┌─────────────┴──────────────┐
           │                            │
           ▼                            ▼
┌─────────────────────┐      ┌─────────────────────┐
│   Compiler API      │      │  Introspection API  │
│  (Parse, TypeCheck  │      │  (Resources, Routes │
│   CodeGen, Build)   │      │   Patterns, Deps)   │
└─────────────────────┘      └─────────────────────┘
           │                            │
           └─────────────┬──────────────┘
                         ▼
┌────────────────────────────────────────────────────────────────┐
│                    LSP Server (JSON-RPC)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │    Hover     │  │  Completion  │  │  Definition  │        │
│  │   Provider   │  │   Provider   │  │   Provider   │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ Diagnostics  │  │   Symbols    │  │   Rename     │        │
│  │   Provider   │  │   Provider   │  │   Provider   │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└────────────────────────┬───────────────────────────────────────┘
                         │
           ┌─────────────┴──────────────┐
           │                            │
           ▼                            ▼
┌─────────────────────┐      ┌─────────────────────┐
│   Document Cache    │      │   Symbol Index      │
│  (ASTs per file)    │      │  (Trie structure)   │
└─────────────────────┘      └─────────────────────┘
                                        │
                                        ▼
┌────────────────────────────────────────────────────────────────┐
│                   Watch Mode & Hot Reload                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ File Watcher │  │   Debouncer  │  │    Change    │        │
│  │  (fsnotify)  │  │  (300ms)     │  │   Analyzer   │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ Incremental  │  │   Process    │  │   Browser    │        │
│  │    Build     │  │   Manager    │  │   Refresh    │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌────────────────────────────────────────────────────────────────┐
│                  Debug Adapter (Delve Wrapper)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  Breakpoint  │  │   Variable   │  │  Call Stack  │        │
│  │ Translation  │  │  Inspection  │  │   Mapping    │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│  ┌──────────────┐  ┌──────────────┐                          │
│  │  Source Map  │  │  DAP Proto   │                          │
│  │   Handler    │  │  (JSON-RPC)  │                          │
│  └──────────────┘  └──────────────┘                          │
└────────────────────────────────────────────────────────────────┘
```

### Layered Architecture

**Layer 1: CLI Interface**
- Command routing (Cobra)
- Flag parsing
- Output formatting (human/JSON)
- Error handling

**Layer 2: Core Services**
- Compiler integration
- Introspection queries
- Build system
- Template engine

**Layer 3: IDE Integration**
- LSP server
- DAP adapter
- File watching
- Hot reload

**Layer 4: Code Quality**
- Formatter
- Linter
- Pattern validator
- Documentation generator

---

## Component 1: CLI Tool Architecture

### Responsibility

Fast, intuitive command-line interface for all development operations.

### Command Structure

```go
type CLI struct {
    cobra     *cobra.Command
    compiler  *Compiler
    intro     *IntrospectionAPI
    config    *Config
    formatter *OutputFormatter
}

// Root command
var rootCmd = &cobra.Command{
    Use:   "llmlang",
    Short: "LLM-First Programming Language CLI",
    Long:  "Unified toolchain for building, running, and managing LLMLang projects",
}
```

### Core Commands Implementation

#### New Command (Project Scaffolding)

```go
var newCmd = &cobra.Command{
    Use:   "new [project-name]",
    Short: "Create a new project from template",
    Args:  cobra.ExactArgs(1),
    RunE:  runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
    projectName := args[0]
    template := cmd.Flag("template").Value.String()

    // Create project directory
    if err := os.MkdirAll(projectName, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Load template
    tmpl, err := templates.Load(template)
    if err != nil {
        return fmt.Errorf("failed to load template: %w", err)
    }

    // Generate project
    generator := &ProjectGenerator{
        Name:     projectName,
        Template: tmpl,
    }

    if err := generator.Generate(); err != nil {
        return fmt.Errorf("failed to generate project: %w", err)
    }

    fmt.Printf("✓ Created project '%s' from template '%s'\n", projectName, template)
    fmt.Printf("✓ Ready to run!\n\n")
    fmt.Printf("Next steps:\n")
    fmt.Printf("  cd %s\n", projectName)
    fmt.Printf("  llmlang dev\n")

    return nil
}

func init() {
    newCmd.Flags().StringP("template", "t", "minimal", "Template to use")
    rootCmd.AddCommand(newCmd)
}
```

#### Build Command

```go
var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Compile the project",
    RunE:  runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
    start := time.Now()

    // Load configuration
    config, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Create compiler
    compiler := NewCompiler(config)

    // Find source files
    files, err := findSourceFiles(".")
    if err != nil {
        return fmt.Errorf("failed to find source files: %w", err)
    }

    // Compile
    buildOpts := BuildOptions{
        Mode:   cmd.Flag("mode").Value.String(),
        Output: cmd.Flag("output").Value.String(),
        Watch:  cmd.Flag("watch").Value.String() == "true",
    }

    result, err := compiler.Build(files, buildOpts)
    if err != nil {
        // Print compilation errors
        printCompilationErrors(err)
        return fmt.Errorf("compilation failed")
    }

    duration := time.Since(start)
    fmt.Printf("✓ Compiled %d files in %.2fs\n", len(files), duration.Seconds())
    fmt.Printf("✓ Binary: %s\n", result.OutputPath)

    return nil
}

func printCompilationErrors(err error) {
    if compErr, ok := err.(*CompilationError); ok {
        for _, diag := range compErr.Diagnostics {
            fmt.Printf("\n%s\n", formatDiagnostic(diag))
        }
    } else {
        fmt.Printf("Error: %v\n", err)
    }
}

func formatDiagnostic(d *Diagnostic) string {
    var buf strings.Builder

    // Error header
    buf.WriteString(fmt.Sprintf("ERROR[%s]: %s\n", d.Code, d.Message))
    buf.WriteString("\n")

    // Source context
    lines := d.SourceLines()
    for i, line := range lines {
        lineNum := d.StartLine + i
        marker := "  "
        if lineNum == d.StartLine {
            marker = "> "
        }
        buf.WriteString(fmt.Sprintf("%s%4d │ %s\n", marker, lineNum, line))

        // Add squiggle under error
        if lineNum == d.StartLine {
            spaces := strings.Repeat(" ", d.StartColumn-1)
            squiggles := strings.Repeat("^", d.EndColumn-d.StartColumn)
            buf.WriteString(fmt.Sprintf("       │ %s%s %s\n", spaces, squiggles, d.Hint))
        }
    }

    buf.WriteString("\n")

    // Suggestion
    if d.Suggestion != "" {
        buf.WriteString(fmt.Sprintf("Suggested fix:\n"))
        buf.WriteString(fmt.Sprintf("  %s\n", d.Suggestion))
    }

    return buf.String()
}

func init() {
    buildCmd.Flags().StringP("mode", "m", "dev", "Build mode (dev|release)")
    buildCmd.Flags().StringP("output", "o", "", "Output file path")
    buildCmd.Flags().BoolP("watch", "w", false, "Watch mode")
    rootCmd.AddCommand(buildCmd)
}
```

#### Introspection Commands

```go
var introspectCmd = &cobra.Command{
    Use:   "introspect",
    Short: "Query codebase structure",
}

var introspectResourcesCmd = &cobra.Command{
    Use:   "resources",
    Short: "List all resources",
    RunE:  runIntrospectResources,
}

func runIntrospectResources(cmd *cobra.Command, args []string) error {
    // Load introspection data
    intro, err := LoadIntrospectionAPI()
    if err != nil {
        return fmt.Errorf("failed to load introspection: %w", err)
    }

    resources := intro.Resources()

    // Check JSON output flag
    if cmd.Flag("json").Value.String() == "true" {
        return outputJSON(resources)
    }

    // Human-friendly output
    fmt.Printf("\nResources:\n\n")
    for _, r := range resources {
        fmt.Printf("  %s (%d fields, %d relationships)\n",
            r.Name, len(r.Fields), len(r.Relationships))

        // Show fields
        fmt.Printf("    Fields: ")
        fieldNames := make([]string, len(r.Fields))
        for i, f := range r.Fields {
            fieldNames[i] = f.Name
        }
        fmt.Printf("%s\n", strings.Join(fieldNames, ", "))

        // Show relationships
        if len(r.Relationships) > 0 {
            fmt.Printf("    Relationships:\n")
            for _, rel := range r.Relationships {
                fmt.Printf("      - %s (%s, %s)\n", rel.Name, rel.Type, rel.Kind)
            }
        }

        fmt.Println()
    }

    return nil
}

func outputJSON(data interface{}) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}

func init() {
    introspectResourcesCmd.Flags().Bool("json", false, "JSON output")
    introspectCmd.AddCommand(introspectResourcesCmd)
    rootCmd.AddCommand(introspectCmd)
}
```

### Output Formatting

```go
type OutputFormatter struct {
    jsonMode bool
    noColor  bool
    quiet    bool
}

func (f *OutputFormatter) Success(message string) {
    if f.quiet {
        return
    }

    if f.jsonMode {
        f.json(map[string]interface{}{
            "status":  "success",
            "message": message,
        })
        return
    }

    checkmark := "✓"
    if f.noColor {
        fmt.Printf("[SUCCESS] %s\n", message)
    } else {
        fmt.Printf("\033[32m%s\033[0m %s\n", checkmark, message)
    }
}

func (f *OutputFormatter) Error(err error) {
    if f.jsonMode {
        f.json(map[string]interface{}{
            "status": "error",
            "error":  err.Error(),
        })
        return
    }

    cross := "✗"
    if f.noColor {
        fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
    } else {
        fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m %v\n", cross, err)
    }
}

func (f *OutputFormatter) Progress(current, total int) {
    if f.quiet || f.jsonMode {
        return
    }

    percent := float64(current) / float64(total) * 100
    fmt.Printf("\rProgress: [%d/%d] %.1f%%", current, total, percent)
    if current == total {
        fmt.Println()
    }
}

func (f *OutputFormatter) json(data interface{}) {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encoder.Encode(data)
}
```

### Testing Strategy

**Unit Tests:**
- Test command routing
- Test flag parsing
- Test output formatting
- Test error handling

**Integration Tests:**
- Test full command workflows
- Test CLI → Compiler integration
- Test JSON output format

**Coverage Target:** >85%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Component 2: Language Server Protocol (LSP)

### Responsibility

Professional IDE integration with hover, completion, diagnostics, and more.

### LSP Server Structure

```go
type LanguageServer struct {
    compiler   *Compiler
    intro      *IntrospectionAPI
    formatter  *Formatter

    // Document cache
    documents  map[string]*Document
    docsMutex  sync.RWMutex

    // Symbol index
    symbolIndex *SymbolIndex

    // Capabilities
    capabilities *ServerCapabilities
}

type Document struct {
    URI     string
    Content string
    Version int
    AST     *AST
    Errors  []*Diagnostic
}

type ServerCapabilities struct {
    TextDocumentSync           int
    HoverProvider              bool
    CompletionProvider         *CompletionOptions
    DefinitionProvider         bool
    ReferencesProvider         bool
    DocumentSymbolProvider     bool
    DocumentFormattingProvider bool
    RenameProvider             bool
    CodeActionProvider         bool
}
```

### Initialize Handler

```go
func (ls *LanguageServer) Initialize(params *InitializeParams) (*InitializeResult, error) {
    // Set up capabilities
    ls.capabilities = &ServerCapabilities{
        TextDocumentSync: 2, // Incremental
        HoverProvider:    true,
        CompletionProvider: &CompletionOptions{
            TriggerCharacters: []string{".", "@", ":", "{"},
        },
        DefinitionProvider:         true,
        ReferencesProvider:         true,
        DocumentSymbolProvider:     true,
        DocumentFormattingProvider: true,
        RenameProvider:             true,
        CodeActionProvider:         true,
    }

    // Initialize symbol index
    ls.symbolIndex = NewSymbolIndex()

    // Index workspace
    go ls.indexWorkspace(params.RootURI)

    return &InitializeResult{
        Capabilities: ls.capabilities,
        ServerInfo: &ServerInfo{
            Name:    "llmlang-lsp",
            Version: "1.0.0",
        },
    }, nil
}
```

### Document Synchronization

```go
func (ls *LanguageServer) DidOpen(params *DidOpenTextDocumentParams) error {
    ls.docsMutex.Lock()
    defer ls.docsMutex.Unlock()

    doc := &Document{
        URI:     params.TextDocument.URI,
        Content: params.TextDocument.Text,
        Version: params.TextDocument.Version,
    }

    // Parse document
    ast, errors := ls.compiler.Parse(doc.Content)
    doc.AST = ast
    doc.Errors = errors

    // Store document
    ls.documents[doc.URI] = doc

    // Update symbol index
    ls.symbolIndex.Index(doc.URI, ast)

    // Publish diagnostics
    ls.publishDiagnostics(doc)

    return nil
}

func (ls *LanguageServer) DidChange(params *DidChangeTextDocumentParams) error {
    ls.docsMutex.Lock()
    defer ls.docsMutex.Unlock()

    doc, ok := ls.documents[params.TextDocument.URI]
    if !ok {
        return fmt.Errorf("document not found: %s", params.TextDocument.URI)
    }

    // Apply changes (incremental)
    for _, change := range params.ContentChanges {
        if change.Range != nil {
            doc.Content = applyChange(doc.Content, change)
        } else {
            doc.Content = change.Text
        }
    }

    doc.Version = params.TextDocument.Version

    // Re-parse document
    ast, errors := ls.compiler.Parse(doc.Content)
    doc.AST = ast
    doc.Errors = errors

    // Update symbol index
    ls.symbolIndex.Index(doc.URI, ast)

    // Publish diagnostics
    ls.publishDiagnostics(doc)

    return nil
}

func (ls *LanguageServer) publishDiagnostics(doc *Document) {
    diagnostics := make([]*LSPDiagnostic, len(doc.Errors))
    for i, err := range doc.Errors {
        diagnostics[i] = &LSPDiagnostic{
            Range: &LSPRange{
                Start: &LSPPosition{Line: err.StartLine - 1, Character: err.StartColumn - 1},
                End:   &LSPPosition{Line: err.EndLine - 1, Character: err.EndColumn - 1},
            },
            Severity: 1, // Error
            Code:     err.Code,
            Source:   "llmlang",
            Message:  err.Message,
        }
    }

    // Send notification
    ls.conn.Notify("textDocument/publishDiagnostics", &PublishDiagnosticsParams{
        URI:         doc.URI,
        Diagnostics: diagnostics,
    })
}
```

### Hover Provider

```go
func (ls *LanguageServer) Hover(params *HoverParams) (*Hover, error) {
    ls.docsMutex.RLock()
    doc, ok := ls.documents[params.TextDocument.URI]
    ls.docsMutex.RUnlock()

    if !ok {
        return nil, fmt.Errorf("document not found")
    }

    // Find symbol at position
    position := Position{
        Line:   params.Position.Line + 1,
        Column: params.Position.Character + 1,
    }

    symbol := ls.findSymbolAt(doc.AST, position)
    if symbol == nil {
        return nil, nil
    }

    // Get type information
    typeInfo := ls.getTypeInfo(symbol)

    // Get documentation
    documentation := ls.getDocumentation(symbol)

    // Format hover content
    var content strings.Builder
    content.WriteString("```llmlang\n")
    content.WriteString(fmt.Sprintf("%s: %s\n", symbol.Name, typeInfo))
    content.WriteString("```\n\n")

    if documentation != "" {
        content.WriteString(documentation)
        content.WriteString("\n\n")
    }

    // Add constraints
    if len(symbol.Constraints) > 0 {
        content.WriteString("**Constraints:**\n")
        for _, c := range symbol.Constraints {
            content.WriteString(fmt.Sprintf("- %s\n", c))
        }
    }

    return &Hover{
        Contents: &MarkupContent{
            Kind:  "markdown",
            Value: content.String(),
        },
        Range: &LSPRange{
            Start: &LSPPosition{Line: symbol.StartLine - 1, Character: symbol.StartColumn - 1},
            End:   &LSPPosition{Line: symbol.EndLine - 1, Character: symbol.EndColumn - 1},
        },
    }, nil
}
```

### Completion Provider

```go
func (ls *LanguageServer) Completion(params *CompletionParams) (*CompletionList, error) {
    ls.docsMutex.RLock()
    doc, ok := ls.documents[params.TextDocument.URI]
    ls.docsMutex.RUnlock()

    if !ok {
        return nil, fmt.Errorf("document not found")
    }

    position := Position{
        Line:   params.Position.Line + 1,
        Column: params.Position.Character + 1,
    }

    // Get completion context
    context := ls.getCompletionContext(doc, position)

    var items []*CompletionItem

    switch context.Type {
    case CompletionTypeField:
        // Type completions
        items = append(items, ls.getTypeCompletions()...)

    case CompletionTypeAnnotation:
        // Annotation completions (@before, @after, @min, etc.)
        items = append(items, ls.getAnnotationCompletions()...)

    case CompletionTypeNamespace:
        // Namespace method completions (String., Time., etc.)
        items = append(items, ls.getNamespaceCompletions(context.Namespace)...)

    case CompletionTypePattern:
        // Pattern-based completions (from introspection)
        patterns := ls.intro.Patterns(context.Category)
        for _, p := range patterns {
            items = append(items, &CompletionItem{
                Label:         p.Name,
                Kind:          CompletionItemKindSnippet,
                Detail:        p.Description,
                InsertText:    p.Template,
                InsertTextFormat: 2, // Snippet
            })
        }
    }

    return &CompletionList{
        IsIncomplete: false,
        Items:        items,
    }, nil
}

func (ls *LanguageServer) getTypeCompletions() []*CompletionItem {
    types := []string{"string", "int", "float", "bool", "uuid", "timestamp", "text", "json"}
    items := make([]*CompletionItem, len(types))

    for i, t := range types {
        items[i] = &CompletionItem{
            Label:      t,
            Kind:       CompletionItemKindKeyword,
            Detail:     fmt.Sprintf("Type: %s", t),
            InsertText: t,
        }
    }

    return items
}

func (ls *LanguageServer) getAnnotationCompletions() []*CompletionItem {
    annotations := []struct {
        name     string
        template string
        detail   string
    }{
        {"@before create", "@before create {\n  $0\n}", "Lifecycle hook before creating"},
        {"@after update", "@after update {\n  $0\n}", "Lifecycle hook after updating"},
        {"@min", "@min($1)", "Minimum value constraint"},
        {"@max", "@max($1)", "Maximum value constraint"},
        {"@unique", "@unique", "Unique constraint"},
        {"@primary", "@primary", "Primary key"},
        {"@auto", "@auto", "Auto-generated value"},
    }

    items := make([]*CompletionItem, len(annotations))
    for i, a := range annotations {
        items[i] = &CompletionItem{
            Label:            a.name,
            Kind:             CompletionItemKindSnippet,
            Detail:           a.detail,
            InsertText:       a.template,
            InsertTextFormat: 2, // Snippet
        }
    }

    return items
}
```

### Definition Provider

```go
func (ls *LanguageServer) Definition(params *DefinitionParams) ([]*Location, error) {
    ls.docsMutex.RLock()
    doc, ok := ls.documents[params.TextDocument.URI]
    ls.docsMutex.RUnlock()

    if !ok {
        return nil, fmt.Errorf("document not found")
    }

    position := Position{
        Line:   params.Position.Line + 1,
        Column: params.Position.Character + 1,
    }

    // Find symbol at position
    symbol := ls.findSymbolAt(doc.AST, position)
    if symbol == nil {
        return nil, nil
    }

    // If it's a reference, find the definition
    if symbol.Kind == SymbolKindReference {
        defSymbol := ls.symbolIndex.FindDefinition(symbol.Name)
        if defSymbol == nil {
            return nil, nil
        }

        return []*Location{{
            URI: defSymbol.URI,
            Range: &LSPRange{
                Start: &LSPPosition{Line: defSymbol.StartLine - 1, Character: defSymbol.StartColumn - 1},
                End:   &LSPPosition{Line: defSymbol.EndLine - 1, Character: defSymbol.EndColumn - 1},
            },
        }}, nil
    }

    // Already at definition
    return []*Location{{
        URI: doc.URI,
        Range: &LSPRange{
            Start: &LSPPosition{Line: symbol.StartLine - 1, Character: symbol.StartColumn - 1},
            End:   &LSPPosition{Line: symbol.EndLine - 1, Character: symbol.EndColumn - 1},
        },
    }}, nil
}
```

### Symbol Index

```go
type SymbolIndex struct {
    symbols map[string]*Symbol // name → symbol
    mutex   sync.RWMutex
}

type Symbol struct {
    Name        string
    Kind        SymbolKind
    URI         string
    StartLine   int
    StartColumn int
    EndLine     int
    EndColumn   int
    Type        string
    Constraints []string
}

func (si *SymbolIndex) Index(uri string, ast *AST) {
    si.mutex.Lock()
    defer si.mutex.Unlock()

    // Walk AST and collect symbols
    visitor := &SymbolVisitor{
        URI:     uri,
        Symbols: make([]*Symbol, 0),
    }

    ast.Walk(visitor)

    // Add symbols to index
    for _, sym := range visitor.Symbols {
        si.symbols[sym.Name] = sym
    }
}

func (si *SymbolIndex) FindDefinition(name string) *Symbol {
    si.mutex.RLock()
    defer si.mutex.RUnlock()

    return si.symbols[name]
}
```

### Testing Strategy

**Unit Tests:**
- Test each LSP handler
- Test document synchronization
- Test symbol indexing

**Integration Tests:**
- Test with mock LSP client
- Test full LSP protocol flow
- Test performance with large files

**Performance Tests:**
- Hover response < 50ms
- Completion response < 100ms
- Diagnostics < 200ms

**Coverage Target:** >80%

### Estimated Effort

**Time:** 4-8 weeks
**Team:** 2 engineers
**Complexity:** High
**Risk:** High (LSP complexity, performance)

---

## Component 3: Watch Mode & Hot Reload

### Responsibility

File system watching, incremental compilation, automatic reloading.

### File Watcher Implementation

```go
import "github.com/fsnotify/fsnotify"

type FileWatcher struct {
    watcher   *fsnotify.Watcher
    debouncer *Debouncer
    patterns  []string
    ignored   []string
    onChange  func([]string) error
}

func NewFileWatcher(patterns, ignored []string, onChange func([]string) error) (*FileWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    return &FileWatcher{
        watcher:   watcher,
        debouncer: NewDebouncer(300 * time.Millisecond),
        patterns:  patterns,
        ignored:   ignored,
        onChange:  onChange,
    }, nil
}

func (fw *FileWatcher) Start() error {
    // Add directories to watch
    dirs, err := fw.findDirectories()
    if err != nil {
        return err
    }

    for _, dir := range dirs {
        if err := fw.watcher.Add(dir); err != nil {
            return err
        }
    }

    // Start watching
    go fw.watch()

    return nil
}

func (fw *FileWatcher) watch() {
    for {
        select {
        case event, ok := <-fw.watcher.Events:
            if !ok {
                return
            }

            // Filter ignored files
            if fw.shouldIgnore(event.Name) {
                continue
            }

            // Handle event
            switch event.Op {
            case fsnotify.Write, fsnotify.Create:
                fw.debouncer.Add(event.Name)
            }

        case err, ok := <-fw.watcher.Errors:
            if !ok {
                return
            }
            log.Printf("Watcher error: %v", err)
        }
    }
}

func (fw *FileWatcher) shouldIgnore(path string) bool {
    for _, pattern := range fw.ignored {
        if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
            return true
        }
    }
    return false
}
```

### Debouncer

```go
type Debouncer struct {
    duration time.Duration
    timer    *time.Timer
    files    map[string]struct{}
    mutex    sync.Mutex
    callback func([]string)
}

func NewDebouncer(duration time.Duration) *Debouncer {
    return &Debouncer{
        duration: duration,
        files:    make(map[string]struct{}),
    }
}

func (d *Debouncer) Add(file string) {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    d.files[file] = struct{}{}

    if d.timer != nil {
        d.timer.Stop()
    }

    d.timer = time.AfterFunc(d.duration, func() {
        d.flush()
    })
}

func (d *Debouncer) flush() {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    if len(d.files) == 0 {
        return
    }

    files := make([]string, 0, len(d.files))
    for file := range d.files {
        files = append(files, file)
    }

    d.files = make(map[string]struct{})

    if d.callback != nil {
        d.callback(files)
    }
}

func (d *Debouncer) SetCallback(callback func([]string)) {
    d.callback = callback
}
```

### Change Impact Analyzer

```go
type ChangeImpact struct {
    Scope             ImpactScope
    RequiresRestart   bool
    AffectedResources []string
    AffectedRoutes    []string
}

type ImpactScope int

const (
    ScopeUI      ImpactScope = iota // Browser refresh only
    ScopeBackend                     // Server restart
    ScopeConfig                      // Full restart
)

func AnalyzeImpact(files []string) *ChangeImpact {
    impact := &ChangeImpact{
        AffectedResources: make([]string, 0),
        AffectedRoutes:    make([]string, 0),
    }

    for _, file := range files {
        switch {
        case strings.HasPrefix(file, "ui/"):
            // UI change - browser refresh only
            impact.Scope = ScopeUI

        case strings.HasPrefix(file, "resources/"):
            // Resource change - server restart
            if impact.Scope < ScopeBackend {
                impact.Scope = ScopeBackend
            }
            impact.RequiresRestart = true
            impact.AffectedResources = append(impact.AffectedResources, file)

        case strings.HasPrefix(file, "config/"):
            // Config change - full restart
            impact.Scope = ScopeConfig
            impact.RequiresRestart = true
        }
    }

    return impact
}
```

### Hot Reload Manager

```go
type HotReloadManager struct {
    compiler      *Compiler
    serverProcess *os.Process
    wsConnections map[string]*websocket.Conn
    mutex         sync.RWMutex
}

func (hrm *HotReloadManager) HandleChange(files []string) error {
    start := time.Now()

    log.Printf("[%s] Changed: %s\n", time.Now().Format("15:04:05"), strings.Join(files, ", "))
    log.Printf("[%s] Recompiling...\n", time.Now().Format("15:04:05"))

    // Analyze impact
    impact := AnalyzeImpact(files)

    // Incremental build
    result, err := hrm.compiler.IncrementalBuild(files)
    if err != nil {
        log.Printf("[%s] ✗ Build failed: %v\n", time.Now().Format("15:04:05"), err)
        hrm.notifyError(err)
        return err
    }

    duration := time.Since(start)
    log.Printf("[%s] ✓ Build successful (%.0fms)\n", time.Now().Format("15:04:05"), duration.Seconds()*1000)

    // Handle reload based on impact
    switch impact.Scope {
    case ScopeUI:
        log.Printf("[%s] 🔄 Browser refreshed\n", time.Now().Format("15:04:05"))
        hrm.notifyBrowserRefresh()

    case ScopeBackend:
        log.Printf("[%s] 🔄 Hot reloaded: %s\n", time.Now().Format("15:04:05"), strings.Join(impact.AffectedResources, ", "))
        if err := hrm.restartServer(); err != nil {
            return err
        }
        log.Printf("[%s] Server restarted\n", time.Now().Format("15:04:05"))

    case ScopeConfig:
        log.Printf("[%s] 🔄 Full restart\n", time.Now().Format("15:04:05"))
        if err := hrm.fullRestart(); err != nil {
            return err
        }
    }

    return nil
}

func (hrm *HotReloadManager) restartServer() error {
    // Gracefully stop old server
    if hrm.serverProcess != nil {
        hrm.serverProcess.Signal(syscall.SIGTERM)
        hrm.serverProcess.Wait()
    }

    // Start new server
    cmd := exec.Command("./build/app")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }

    hrm.serverProcess = cmd.Process

    return nil
}

func (hrm *HotReloadManager) notifyBrowserRefresh() {
    hrm.mutex.RLock()
    defer hrm.mutex.RUnlock()

    message, _ := json.Marshal(map[string]interface{}{
        "type":      "reload",
        "scope":     "ui",
        "timestamp": time.Now().Unix(),
    })

    for _, conn := range hrm.wsConnections {
        conn.WriteMessage(websocket.TextMessage, message)
    }
}

func (hrm *HotReloadManager) notifyError(err error) {
    hrm.mutex.RLock()
    defer hrm.mutex.RUnlock()

    message, _ := json.Marshal(map[string]interface{}{
        "type":  "error",
        "error": err.Error(),
    })

    for _, conn := range hrm.wsConnections {
        conn.WriteMessage(websocket.TextMessage, message)
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test file watching
- Test debouncing
- Test change impact analysis

**Integration Tests:**
- Test full hot reload flow
- Test concurrent changes
- Test error recovery

**Coverage Target:** >85%

### Estimated Effort

**Time:** 3-4 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Medium (state consistency)

---

## Component 4: Debug Adapter Protocol (DAP)

### Responsibility

Source-level debugging via Delve integration with source maps.

### DAP Adapter

```go
type DebugAdapter struct {
    delve      *DelveClient
    sourceMaps *SourceMapRegistry
    session    *DebugSession
}

type DebugSession struct {
    SessionID   string
    Breakpoints map[string][]*Breakpoint
    Variables   map[string]*Variable
}

type Breakpoint struct {
    ID            int
    SourceFile    string
    SourceLine    int
    GeneratedFile string
    GeneratedLine int
    Verified      bool
}
```

### Source Map Handler

```go
type SourceMapRegistry struct {
    maps  map[string]*SourceMap
    mutex sync.RWMutex
}

type SourceMap struct {
    SourceFile    string
    GeneratedFile string
    Mappings      []*LineMapping
}

type LineMapping struct {
    SourceLine      int
    SourceColumn    int
    GeneratedLine   int
    GeneratedColumn int
}

func (smr *SourceMapRegistry) TranslateBreakpoint(sourceFile string, sourceLine int) (*Breakpoint, error) {
    smr.mutex.RLock()
    defer smr.mutex.RUnlock()

    sourceMap, ok := smr.maps[sourceFile]
    if !ok {
        return nil, fmt.Errorf("no source map for %s", sourceFile)
    }

    // Find mapping for source line
    for _, mapping := range sourceMap.Mappings {
        if mapping.SourceLine == sourceLine {
            return &Breakpoint{
                SourceFile:    sourceFile,
                SourceLine:    sourceLine,
                GeneratedFile: sourceMap.GeneratedFile,
                GeneratedLine: mapping.GeneratedLine,
            }, nil
        }
    }

    return nil, fmt.Errorf("no mapping for line %d", sourceLine)
}
```

### SetBreakpoints Handler

```go
func (da *DebugAdapter) SetBreakpoints(params *SetBreakpointsParams) (*SetBreakpointsResponse, error) {
    sourceFile := params.Source.Path
    breakpoints := make([]*Breakpoint, 0)

    for _, bp := range params.Breakpoints {
        // Translate source breakpoint to generated code
        translated, err := da.sourceMaps.TranslateBreakpoint(sourceFile, bp.Line)
        if err != nil {
            log.Printf("Failed to translate breakpoint: %v", err)
            breakpoints = append(breakpoints, &Breakpoint{
                SourceFile: sourceFile,
                SourceLine: bp.Line,
                Verified:   false,
            })
            continue
        }

        // Set breakpoint in Delve
        delveID, err := da.delve.CreateBreakpoint(translated.GeneratedFile, translated.GeneratedLine)
        if err != nil {
            log.Printf("Failed to set breakpoint in Delve: %v", err)
            breakpoints = append(breakpoints, &Breakpoint{
                SourceFile: sourceFile,
                SourceLine: bp.Line,
                Verified:   false,
            })
            continue
        }

        translated.ID = delveID
        translated.Verified = true
        breakpoints = append(breakpoints, translated)
    }

    // Store breakpoints
    da.session.Breakpoints[sourceFile] = breakpoints

    // Convert to DAP response
    dapBreakpoints := make([]*DAPBreakpoint, len(breakpoints))
    for i, bp := range breakpoints {
        dapBreakpoints[i] = &DAPBreakpoint{
            ID:       bp.ID,
            Verified: bp.Verified,
            Line:     bp.SourceLine,
        }
    }

    return &SetBreakpointsResponse{
        Breakpoints: dapBreakpoints,
    }, nil
}
```

### Variable Inspection

```go
func (da *DebugAdapter) Variables(params *VariablesParams) (*VariablesResponse, error) {
    // Get variables from Delve
    delveVars, err := da.delve.ListVariables(params.VariablesReference)
    if err != nil {
        return nil, err
    }

    // Convert to DAP variables
    variables := make([]*DAPVariable, len(delveVars))
    for i, dv := range delveVars {
        variables[i] = &DAPVariable{
            Name:  dv.Name,
            Value: dv.Value,
            Type:  dv.Type,
            VariablesReference: dv.VariablesReference,
        }
    }

    return &VariablesResponse{
        Variables: variables,
    }, nil
}
```

### Stack Trace Mapping

```go
func (da *DebugAdapter) StackTrace(params *StackTraceParams) (*StackTraceResponse, error) {
    // Get stack trace from Delve
    delveStack, err := da.delve.StackTrace()
    if err != nil {
        return nil, err
    }

    // Map stack frames back to source
    frames := make([]*StackFrame, len(delveStack))
    for i, df := range delveStack {
        // Translate generated location to source
        source, line, err := da.sourceMaps.TranslateLocation(df.File, df.Line)
        if err != nil {
            // Can't map - show generated code
            frames[i] = &StackFrame{
                ID:   df.ID,
                Name: df.Function,
                Source: &Source{
                    Path: df.File,
                },
                Line:   df.Line,
                Column: df.Column,
            }
            continue
        }

        frames[i] = &StackFrame{
            ID:   df.ID,
            Name: df.Function,
            Source: &Source{
                Path: source,
            },
            Line:   line,
            Column: 0,
        }
    }

    return &StackTraceResponse{
        StackFrames: frames,
        TotalFrames: len(frames),
    }, nil
}
```

### Testing Strategy

**Unit Tests:**
- Test source map translation
- Test breakpoint handling
- Test variable inspection

**Integration Tests:**
- Test with Delve
- Test breakpoint flow
- Test stack trace mapping

**Coverage Target:** >75%

### Estimated Effort

**Time:** 4-5 weeks
**Team:** 2 engineers
**Complexity:** High
**Risk:** High (source mapping accuracy)

---

## Component 5: Code Formatting Engine

### Responsibility

Deterministic, idempotent code formatting.

### Formatter Implementation

```go
type Formatter struct {
    config FormatConfig
}

type FormatConfig struct {
    IndentSize      int  // 2
    MaxLineLength   int  // 100
    AlignFields     bool // true
    AlignConstraints bool // true
    BlankLines      int  // 1 between resources
}

func (f *Formatter) Format(source string) (string, error) {
    // Parse to AST
    ast, err := Parse(source)
    if err != nil {
        return "", err
    }

    // Normalize AST (deterministic order)
    NormalizeAST(ast)

    // Pretty print
    output := f.PrettyPrint(ast)

    // Verify idempotence
    reFormatted, err := f.Format(output)
    if err != nil {
        return "", err
    }

    if reFormatted != output {
        return "", errors.New("formatter not idempotent")
    }

    return output, nil
}

func (f *Formatter) PrettyPrint(ast *AST) string {
    printer := &Printer{
        config: f.config,
        buf:    new(strings.Builder),
        indent: 0,
    }

    ast.Walk(printer)

    return printer.buf.String()
}
```

### Printer

```go
type Printer struct {
    config FormatConfig
    buf    *strings.Builder
    indent int
}

func (p *Printer) VisitResource(node *ResourceNode) {
    // Documentation comment
    if node.Documentation != "" {
        p.writeDocComment(node.Documentation)
    }

    // Resource declaration
    p.writeLine("resource %s {", node.Name)
    p.indent++

    // Fields
    if len(node.Fields) > 0 {
        p.writeFields(node.Fields)
        p.writeLine("")
    }

    // Relationships
    if len(node.Relationships) > 0 {
        p.writeRelationships(node.Relationships)
        p.writeLine("")
    }

    // Hooks
    if len(node.Hooks) > 0 {
        p.writeHooks(node.Hooks)
    }

    p.indent--
    p.writeLine("}")
    p.writeLine("") // Blank line between resources
}

func (p *Printer) writeFields(fields []*FieldNode) {
    // Calculate max name length for alignment
    maxNameLen := 0
    maxTypeLen := 0

    for _, f := range fields {
        if len(f.Name) > maxNameLen {
            maxNameLen = len(f.Name)
        }
        typeStr := f.Type.String()
        if len(typeStr) > maxTypeLen {
            maxTypeLen = len(typeStr)
        }
    }

    // Write fields with alignment
    for _, f := range fields {
        // Inline comment
        comment := ""
        if f.InlineComment != "" {
            comment = " // " + f.InlineComment
        }

        // Name and type alignment
        namePad := strings.Repeat(" ", maxNameLen-len(f.Name)+1)
        typePad := strings.Repeat(" ", maxTypeLen-len(f.Type.String())+1)

        p.write("%s%s:%s%s%s",
            p.indentStr(),
            f.Name,
            namePad,
            f.Type.String(),
            typePad,
        )

        // Constraints
        for _, constraint := range f.Constraints {
            p.write("@%s ", constraint.String())
        }

        p.write("%s\n", comment)
    }
}

func (p *Printer) writeHooks(hooks []*HookNode) {
    for _, h := range hooks {
        p.writeLine("@%s %s {", h.Timing, h.Operation)
        p.indent++

        // Hook body
        for _, stmt := range h.Statements {
            p.writeStatement(stmt)
        }

        p.indent--
        p.writeLine("}")
    }
}

func (p *Printer) writeLine(format string, args ...interface{}) {
    p.buf.WriteString(p.indentStr())
    fmt.Fprintf(p.buf, format, args...)
    p.buf.WriteString("\n")
}

func (p *Printer) indentStr() string {
    return strings.Repeat(" ", p.indent*p.config.IndentSize)
}
```

### Format Command Integration

```go
var formatCmd = &cobra.Command{
    Use:   "format [files...]",
    Short: "Format source files",
    RunE:  runFormat,
}

func runFormat(cmd *cobra.Command, args []string) error {
    formatter := NewFormatter(FormatConfig{
        IndentSize:       2,
        MaxLineLength:    100,
        AlignFields:      true,
        AlignConstraints: true,
        BlankLines:       1,
    })

    // Find files to format
    files := args
    if len(files) == 0 {
        var err error
        files, err = findSourceFiles(".")
        if err != nil {
            return err
        }
    }

    checkOnly := cmd.Flag("check").Value.String() == "true"
    showDiff := cmd.Flag("diff").Value.String() == "true"
    writeChanges := cmd.Flag("write").Value.String() == "true"

    hasChanges := false

    for _, file := range files {
        // Read file
        content, err := os.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read %s: %w", file, err)
        }

        // Format
        formatted, err := formatter.Format(string(content))
        if err != nil {
            return fmt.Errorf("failed to format %s: %w", file, err)
        }

        // Check if changed
        if formatted == string(content) {
            continue
        }

        hasChanges = true

        if checkOnly {
            fmt.Printf("✗ %s needs formatting\n", file)
            continue
        }

        if showDiff {
            diff := generateDiff(string(content), formatted)
            fmt.Printf("diff %s\n%s\n", file, diff)
        }

        if writeChanges {
            if err := os.WriteFile(file, []byte(formatted), 0644); err != nil {
                return fmt.Errorf("failed to write %s: %w", file, err)
            }
            fmt.Printf("✓ Formatted %s\n", file)
        }
    }

    if checkOnly && hasChanges {
        return fmt.Errorf("some files need formatting")
    }

    return nil
}

func init() {
    formatCmd.Flags().Bool("check", false, "Check if files are formatted")
    formatCmd.Flags().Bool("diff", false, "Show diff")
    formatCmd.Flags().Bool("write", false, "Write changes to files")
    rootCmd.AddCommand(formatCmd)
}
```

### Testing Strategy

**Unit Tests:**
- Test formatting rules
- Test idempotence
- Test determinism

**Fuzz Tests:**
- Fuzz with random valid inputs
- Check for consistency

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Component 6: Project Templates

### Responsibility

Project scaffolding, code generation, template management.

### Template Structure

```go
type Template struct {
    Name        string
    Description string
    Version     string
    Variables   map[string]*TemplateVariable
    Files       []*TemplateFile
    Hooks       *TemplateHooks
}

type TemplateVariable struct {
    Name        string
    Description string
    Type        VariableType
    Default     interface{}
    Required    bool
    Options     []string
}

type TemplateFile struct {
    SourcePath string
    TargetPath string
    Content    string
    Template   bool // Use template engine
}

type TemplateHooks struct {
    AfterCreate []string
}
```

### Template Engine

```go
type TemplateEngine struct {
    templates map[string]*Template
}

func (te *TemplateEngine) Execute(tmpl *Template, vars map[string]interface{}) error {
    // Create target directory
    projectName := vars["project_name"].(string)
    if err := os.MkdirAll(projectName, 0755); err != nil {
        return err
    }

    // Process files
    for _, file := range tmpl.Files {
        targetPath := filepath.Join(projectName, file.TargetPath)

        // Create parent directory
        if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
            return err
        }

        var content string
        if file.Template {
            // Apply template variables
            var err error
            content, err = te.applyVariables(file.Content, vars)
            if err != nil {
                return err
            }
        } else {
            content = file.Content
        }

        // Write file
        if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
            return err
        }
    }

    // Run after_create hooks
    for _, hook := range tmpl.Hooks.AfterCreate {
        cmd := exec.Command("sh", "-c", hook)
        cmd.Dir = projectName
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        if err := cmd.Run(); err != nil {
            log.Printf("Warning: hook failed: %v", err)
        }
    }

    return nil
}

func (te *TemplateEngine) applyVariables(content string, vars map[string]interface{}) (string, error) {
    tmpl, err := template.New("file").Parse(content)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

### Built-in Templates

**Minimal Template:**
```yaml
name: minimal
description: Minimal project with basic structure
version: 1.0.0

variables:
  project_name:
    description: Project name
    type: string
    required: true

files:
  - source: resources/.gitkeep
    target: resources/.gitkeep
  - source: config/database.yaml
    target: config/database.yaml
  - source: config/server.yaml
    target: config/server.yaml

hooks:
  after_create:
    - llmlang build
```

**Blog Template:**
```yaml
name: blog
description: Blog with posts, comments, and users
version: 1.0.0

files:
  - source: resources/post.llm
    target: resources/post.llm
    template: true
  - source: resources/user.llm
    target: resources/user.llm
    template: true
  - source: resources/comment.llm
    target: resources/comment.llm
    template: true
```

### Code Generator

```go
type CodeGenerator struct {
    formatter *Formatter
}

func (cg *CodeGenerator) GenerateResource(spec *ResourceSpec) (string, error) {
    var buf strings.Builder

    buf.WriteString(fmt.Sprintf("resource %s {\n", spec.Name))
    buf.WriteString("  id: uuid! @primary @auto\n")
    buf.WriteString("\n")

    // Fields
    for _, field := range spec.Fields {
        buf.WriteString(fmt.Sprintf("  %s: %s!\n", field.Name, field.Type))
    }

    buf.WriteString("\n")
    buf.WriteString("  created_at: timestamp! @auto\n")
    buf.WriteString("  updated_at: timestamp! @auto_update\n")
    buf.WriteString("}\n")

    // Format
    return cg.formatter.Format(buf.String())
}
```

### Testing Strategy

**Unit Tests:**
- Test template loading
- Test variable substitution
- Test file generation

**Integration Tests:**
- Test full project creation
- Test all built-in templates

**Coverage Target:** >85%

### Estimated Effort

**Time:** 1-2 weeks
**Team:** 1 engineer
**Complexity:** Low
**Risk:** Low

---

## Component 7: Build System

### Responsibility

Fast incremental compilation with caching and dependency tracking.

### Build Cache

```go
type BuildCache struct {
    cacheDir string
    entries  map[string]*CacheEntry
    mutex    sync.RWMutex
}

type CacheEntry struct {
    SourcePath   string
    SourceHash   string
    Dependencies []string
    LastModified time.Time
    CachePath    string
}

func (bc *BuildCache) IsValid(sourcePath string) bool {
    bc.mutex.RLock()
    defer bc.mutex.RUnlock()

    entry, ok := bc.entries[sourcePath]
    if !ok {
        return false
    }

    // Check if source file changed
    currentHash := hashFile(sourcePath)
    if currentHash != entry.SourceHash {
        return false
    }

    // Check if any dependencies changed
    for _, dep := range entry.Dependencies {
        if !bc.IsValid(dep) {
            return false
        }
    }

    return true
}

func (bc *BuildCache) Store(sourcePath string, ast *AST, deps []string) error {
    bc.mutex.Lock()
    defer bc.mutex.Unlock()

    hash := hashFile(sourcePath)
    cachePath := filepath.Join(bc.cacheDir, hash+".ast")

    // Serialize AST
    data, err := serializeAST(ast)
    if err != nil {
        return err
    }

    // Write to cache
    if err := os.WriteFile(cachePath, data, 0644); err != nil {
        return err
    }

    // Store entry
    bc.entries[sourcePath] = &CacheEntry{
        SourcePath:   sourcePath,
        SourceHash:   hash,
        Dependencies: deps,
        LastModified: time.Now(),
        CachePath:    cachePath,
    }

    return nil
}

func (bc *BuildCache) Load(sourcePath string) (*AST, error) {
    bc.mutex.RLock()
    entry, ok := bc.entries[sourcePath]
    bc.mutex.RUnlock()

    if !ok {
        return nil, fmt.Errorf("not in cache")
    }

    // Read from cache
    data, err := os.ReadFile(entry.CachePath)
    if err != nil {
        return nil, err
    }

    // Deserialize AST
    return deserializeAST(data)
}
```

### Dependency Graph

```go
type DependencyGraph struct {
    nodes map[string]*Node
    edges map[string][]string
}

type Node struct {
    Path         string
    Type         NodeType
    Hash         string
    LastModified time.Time
}

func (dg *DependencyGraph) FindAffected(changedFiles []string) []string {
    affected := make(map[string]struct{})

    var visit func(string)
    visit = func(file string) {
        if _, ok := affected[file]; ok {
            return
        }

        affected[file] = struct{}{}

        // Visit dependents
        for _, edge := range dg.edges {
            for _, dep := range edge {
                if dep == file {
                    visit(edge[0])
                }
            }
        }
    }

    for _, file := range changedFiles {
        visit(file)
    }

    result := make([]string, 0, len(affected))
    for file := range affected {
        result = append(result, file)
    }

    return result
}

func (dg *DependencyGraph) TopologicalSort(files []string) []string {
    // Kahn's algorithm
    inDegree := make(map[string]int)
    for _, file := range files {
        inDegree[file] = 0
    }

    for _, file := range files {
        for _, dep := range dg.edges[file] {
            if _, ok := inDegree[dep]; ok {
                inDegree[dep]++
            }
        }
    }

    queue := make([]string, 0)
    for file, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, file)
        }
    }

    result := make([]string, 0, len(files))
    for len(queue) > 0 {
        file := queue[0]
        queue = queue[1:]
        result = append(result, file)

        for _, dep := range dg.edges[file] {
            if _, ok := inDegree[dep]; !ok {
                continue
            }

            inDegree[dep]--
            if inDegree[dep] == 0 {
                queue = append(queue, dep)
            }
        }
    }

    return result
}
```

### Incremental Builder

```go
func (b *Builder) IncrementalBuild(changedFiles []string) (*BuildResult, error) {
    // Find affected files
    affected := b.depGraph.FindAffected(changedFiles)

    // Topological sort (build dependencies first)
    buildOrder := b.depGraph.TopologicalSort(affected)

    // Build each file
    for _, file := range buildOrder {
        if b.cache.IsValid(file) {
            continue // Use cached result
        }

        if err := b.BuildFile(file); err != nil {
            return nil, err
        }

        b.cache.Update(file)
    }

    // Link final binary
    return b.Link()
}
```

### Testing Strategy

**Unit Tests:**
- Test cache invalidation
- Test dependency tracking
- Test topological sort

**Integration Tests:**
- Test full build flow
- Test incremental builds
- Test parallel builds

**Performance Tests:**
- Incremental build < 2s
- Full build < 30s (medium project)
- Cache hit rate >90%

**Coverage Target:** >80%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Medium (correctness)

---

## Component 8: Documentation Generation

### Responsibility

Auto-generate API documentation from introspection data.

### Documentation Generator

```go
type DocGenerator struct {
    intro   *IntrospectionAPI
    tmpl    *template.Template
    output  string
}

func (dg *DocGenerator) Generate() error {
    // Collect documentation data
    data := &DocData{
        Resources:   dg.intro.Resources(),
        Routes:      dg.intro.Routes(),
        Patterns:    dg.intro.Patterns(""),
        Types:       dg.getTypeReference(),
    }

    // Generate documentation pages
    if err := dg.generateIndex(data); err != nil {
        return err
    }

    if err := dg.generateResourceDocs(data.Resources); err != nil {
        return err
    }

    if err := dg.generateRouteDocs(data.Routes); err != nil {
        return err
    }

    if err := dg.generatePatternCatalog(data.Patterns); err != nil {
        return err
    }

    return nil
}

func (dg *DocGenerator) generateResourceDocs(resources []ResourceMetadata) error {
    for _, r := range resources {
        content := dg.formatResourceDoc(r)

        outputPath := filepath.Join(dg.output, "resources", r.Name+".html")
        if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
            return err
        }
    }

    return nil
}

func (dg *DocGenerator) formatResourceDoc(r ResourceMetadata) string {
    var buf strings.Builder

    buf.WriteString(fmt.Sprintf("# %s Resource\n\n", r.Name))

    if r.Documentation != "" {
        buf.WriteString(fmt.Sprintf("> %s\n\n", r.Documentation))
    }

    // Fields table
    buf.WriteString("## Fields\n\n")
    buf.WriteString("| Name | Type | Constraints | Description |\n")
    buf.WriteString("|------|------|-------------|-------------|\n")

    for _, f := range r.Fields {
        constraints := strings.Join(f.Constraints, ", ")
        buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
            f.Name, f.Type, constraints, f.Description))
    }

    buf.WriteString("\n")

    // Relationships
    if len(r.Relationships) > 0 {
        buf.WriteString("## Relationships\n\n")
        for _, rel := range r.Relationships {
            buf.WriteString(fmt.Sprintf("### %s: %s (%s)\n\n",
                rel.Name, rel.Type, rel.Kind))
            buf.WriteString(fmt.Sprintf("%s\n\n", rel.Documentation))
        }
    }

    // Endpoints
    if len(r.Endpoints) > 0 {
        buf.WriteString("## Endpoints\n\n")
        for _, ep := range r.Endpoints {
            buf.WriteString(fmt.Sprintf("### %s\n\n", ep.Summary))
            buf.WriteString(fmt.Sprintf("```http\n%s %s\n```\n\n",
                ep.Method, ep.Path))

            if len(ep.Middleware) > 0 {
                buf.WriteString(fmt.Sprintf("**Middleware:** %s\n\n",
                    strings.Join(ep.Middleware, ", ")))
            }
        }
    }

    return buf.String()
}
```

### Testing Strategy

**Unit Tests:**
- Test documentation formatting
- Test markdown generation

**Integration Tests:**
- Test full documentation generation
- Test with real introspection data

**Coverage Target:** >80%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1 engineer
**Complexity:** Low-Medium
**Risk:** Low

---

## Development Phases

### Phase Summary

| Phase | Component | Duration | Team | Risk |
|-------|-----------|----------|------|------|
| 0 | Foundation | 2 weeks | 1 | Low |
| 1 | CLI Core | 2 weeks | 1 | Low |
| 2 | LSP Foundation | 4 weeks | 2 | High |
| 3 | LSP Advanced | 2 weeks | 2 | Medium |
| 4 | Watch Mode | 3 weeks | 1 | Medium |
| 5 | Hot Reload | 3 weeks | 1-2 | High |
| 6 | Code Formatter | 2 weeks | 1 | Low |
| 7 | Debugging (DAP) | 4 weeks | 2 | High |
| 8 | Project Templates | 2 weeks | 1 | Low |
| 9 | Docs Generation | 2 weeks | 1 | Low |
| 10 | Polish & Performance | 2 weeks | 2 | Medium |

**Total Duration:** 28 weeks
**Total Effort:** 195-230 person-days
**Recommended Team Size:** 2 engineers

### Critical Path

```
Foundation (Phase 0)
    ↓
CLI Core (Phase 1) → LSP Foundation (Phase 2) → LSP Advanced (Phase 3)
    ↓                                                  ↓
Watch Mode (Phase 4) → Hot Reload (Phase 5)          │
    ↓                                                  ↓
Code Formatter (Phase 6), Templates (Phase 8)        │
    ↓                                                  ↓
Debugging (Phase 7), Docs (Phase 9) ← ← ← ← ← ← ← ← ┘
    ↓
Polish & Performance (Phase 10)
```

---

## Testing Strategy

### Testing Pyramid

```
        ┌─────────────────┐
        │   E2E Tests     │   10% - Full workflows
        │    (10%)        │
        ├─────────────────┤
        │ Integration     │   30% - Component interactions
        │   Tests (30%)   │
        ├─────────────────┤
        │   Unit Tests    │   60% - Individual functions
        │    (60%)        │
        └─────────────────┘
```

### Unit Tests (60%)

**Focus:** Individual functions, components

**Examples:**
- CLI command parsing
- LSP protocol handling
- File watching logic
- Formatter rules
- Template variable substitution

**Tools:**
- Go testing package
- Table-driven tests
- Mock file systems

**Target:** >85% code coverage

---

### Integration Tests (30%)

**Focus:** Component interactions

**Examples:**
- CLI → Compiler integration
- LSP → Parser integration
- Watch mode → Build system
- Debugger → Delve integration
- Formatter → Editor integration

**Tools:**
- httptest for LSP
- Temporary directories
- Subprocess management

**Target:** All major workflows covered

---

### End-to-End Tests (10%)

**Focus:** Full user workflows

**Examples:**
- Create project → build → run
- Edit file → watch mode rebuild
- Set breakpoint → debug session
- Format file → verify output

**Tools:**
- Bash scripts
- Real file systems
- Full server instances

**Target:** Common use cases covered

---

### Performance Tests

**Focus:** Performance benchmarks

**Examples:**
- CLI response times
- LSP response times (hover, completion)
- Build times (incremental, full)
- Hot reload speed
- Format speed

**Tools:**
- Go benchmarks
- Custom timing scripts
- Load testing

**Targets:**
- CLI introspection: < 100ms
- LSP hover: < 50ms
- LSP completion: < 100ms
- Incremental build: < 2s
- Hot reload: < 500ms

---

## Integration Points

### 1. Compiler Integration

**Interface:**
```go
type Compiler interface {
    Parse(files []string) (*AST, error)
    TypeCheck(ast *AST) error
    GenerateCode(ast *AST, opts CodeGenOptions) ([]byte, error)
    CollectMetadata(ast *AST) (*Metadata, error)
}
```

**Integration:**
- CLI calls compiler for build/run commands
- LSP uses compiler for parsing and type checking
- Watch mode triggers incremental compilation
- Debug adapter generates source maps during compilation

---

### 2. Introspection API Integration

**Interface:**
```go
type IntrospectionAPI interface {
    Resources() []ResourceMetadata
    Routes() []RouteMetadata
    Patterns(category string) []PatternMetadata
    Dependencies(resource string, opts DependencyOptions) (*DependencyGraph, error)
}
```

**Integration:**
- CLI introspection commands query API
- LSP provides pattern-aware completions
- Documentation generator reads API data
- Template engine uses patterns for scaffolding

---

### 3. Web Framework Integration

**Watch Mode → Server Restart:**
- File watcher detects backend changes
- Hot reload manager restarts server process
- WebSocket notifies browser clients
- State preserved where possible

---

## Performance Targets

### CLI Performance

**Targets:**
- `llmlang version`: < 10ms
- `llmlang introspect resources`: < 100ms
- `llmlang build` (incremental): < 2s
- `llmlang format`: < 100ms per 1000 LOC

**Optimization Strategies:**
- Lazy loading
- Caching introspection queries
- Avoiding unnecessary allocations
- Efficient JSON serialization

---

### LSP Performance

**Targets:**
- Hover: < 50ms
- Completion: < 100ms
- Diagnostics: < 200ms (after change)
- Full document parse: < 1s (1000 LOC)

**Optimization Strategies:**
- Incremental parsing
- Symbol indexing
- Background processing
- LRU caching

---

### Watch Mode Performance

**Targets:**
- Change detection: < 100ms
- Incremental build: < 2s
- Hot reload: < 500ms (UI changes)
- Hot reload: < 3s (backend changes)

**Optimization Strategies:**
- OS event-based watching (fsnotify)
- Debouncing (300ms)
- Build caching
- Parallel compilation

---

## Risk Mitigation

### Risk 1: LSP Performance on Large Projects

**Impact:** CRITICAL
**Probability:** HIGH

**Mitigation:**
1. Incremental parsing and type checking
2. Background processing
3. Symbol indexing with efficient data structures
4. Memory management with LRU caching
5. Regular profiling and optimization

---

### Risk 2: Hot Reload State Inconsistency

**Impact:** HIGH
**Probability:** MEDIUM

**Mitigation:**
1. Conservative reload strategy (default to full restart)
2. Change impact analysis
3. State validation after reload
4. Clear user communication about reload type
5. Extensive testing of reload scenarios

---

### Risk 3: Build System Performance Below Target

**Impact:** HIGH
**Probability:** MEDIUM

**Mitigation:**
1. Aggressive caching at all phases
2. Incremental compilation with dependency tracking
3. Parallel processing where possible
4. Regular profiling and optimization
5. Clear progress reporting to user

---

### Risk 4: Debug Source Mapping Inaccuracy

**Impact:** MEDIUM
**Probability:** HIGH

**Mitigation:**
1. Comprehensive source map generation
2. Preserve code structure in generation
3. Debug comments in generated code
4. Extensive testing of source mapping
5. Fallback to debugging generated Go directly

---

### Risk 5: Format Engine Inconsistency

**Impact:** MEDIUM
**Probability:** LOW

**Mitigation:**
1. Deterministic algorithm (no randomness)
2. Comprehensive testing of idempotence
3. Fuzzing with random valid inputs
4. Version locking of format rules
5. Zero tolerance for inconsistencies

---

### Risk 6: CLI Response Time Exceeds 100ms

**Impact:** HIGH
**Probability:** MEDIUM

**Mitigation:**
1. Optimize cold start time
2. Profile and optimize hot paths
3. Background processing for slow queries
4. Caching introspection results
5. Share cache with LSP server

---

## Success Criteria

### Functional Requirements

- [ ] All CLI commands work correctly
- [ ] LSP provides full IDE features
- [ ] Hot reload works reliably
- [ ] Debugging with source maps works
- [ ] Formatter is idempotent and deterministic
- [ ] Templates create working projects
- [ ] Build system supports incremental compilation
- [ ] Documentation generation produces complete docs

### Performance Requirements

- [ ] CLI introspection < 100ms
- [ ] LSP hover < 50ms
- [ ] LSP completion < 100ms
- [ ] Incremental build < 2s
- [ ] Hot reload < 500ms (UI), < 3s (backend)
- [ ] Format < 100ms per 1000 LOC
- [ ] Zero memory leaks

### Quality Requirements

- [ ] >85% test coverage overall
- [ ] All integration tests pass
- [ ] All performance tests meet targets
- [ ] No critical bugs
- [ ] Clear error messages
- [ ] Complete documentation

### Developer Experience

- [ ] Clear CLI output (human and JSON)
- [ ] Responsive IDE integration
- [ ] Fast feedback loop
- [ ] Easy project setup
- [ ] Helpful error messages
- [ ] Intuitive commands

---

## Appendix: Configuration Example

```yaml
# .llmlang.yaml
tooling:
  # Format settings
  format:
    indent_size: 2
    max_line_length: 100
    align_fields: true
    align_constraints: true

  # Watch mode
  watch:
    patterns:
      - "**/*.llm"
      - "config/**/*.yaml"
    ignored:
      - ".git/**"
      - "build/**"
      - "**/*.tmp"

  # LSP settings
  lsp:
    max_completion_items: 50
    hover_detail_level: full
    symbol_index_on_startup: true

  # Build settings
  build:
    cache_dir: .llmlang-cache
    parallel_jobs: 4
    incremental: true

  # Debug settings
  debug:
    source_maps: true
    delve_port: 2345
```

---

**Document Status:** Complete
**Last Updated:** 2025-10-13
**Next Steps:** Begin Phase 0 - Foundation