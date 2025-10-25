# Conduit LSP Server

This package implements a Language Server Protocol (LSP) server for the Conduit programming language.

## Features

- **Code Completion**: Type completions, annotation completions, namespace method completions
- **Diagnostics**: Real-time syntax and type error reporting
- **Go-to-Definition**: Navigate to resource and field definitions
- **Hover Information**: View type information and documentation
- **Find References**: Find all references to a symbol
- **Document Symbols**: Navigate symbols within a document
- **Signature Help**: Function signature information

## Architecture

The LSP server is built in layers:

1. **`/internal/tooling`**: Core compiler integration API (completion, hover, symbols, diagnostics)
2. **`/internal/lsp`**: LSP protocol handlers that wrap the tooling API
3. **`/cmd/conduit/lsp.go`**: CLI command to start the LSP server

## Usage

### Starting the LSP Server

```bash
conduit lsp
```

The server communicates via JSON-RPC over stdin/stdout and is typically started automatically by your editor/IDE.

### VS Code Extension

See `/editors/vscode/` for the VS Code extension that integrates with this LSP server.

## Testing

Due to limitations with mocking the jsonrpc2.Request interface (which has unexported methods), the LSP layer is best tested through:

1. **Integration tests**: Use a real LSP client to test end-to-end functionality
2. **Unit tests on tooling API**: The `/internal/tooling` package has comprehensive unit tests
3. **Manual testing**: Start the server and connect it to a real editor

To test the tooling API directly:

```bash
go test ./internal/tooling/... -v
```

## Performance Targets

- Completion: <100ms
- Hover: <50ms
- Diagnostics: <200ms after document change
- Memory: <100MB for typical project

## Implementation Details

### Document Management

Documents are cached in memory for fast access. The server handles these notifications:

- `textDocument/didOpen`: Cache and parse document
- `textDocument/didChange`: Update and re-parse document
- `textDocument/didClose`: Remove from cache
- `textDocument/didSave`: Re-publish diagnostics

### Completion

Context-aware completion based on cursor position:

- After `:` - Type completions
- After `@` - Annotation completions
- After `.` - Namespace method completions

### Diagnostics

Published automatically on:
- Document open
- Document change
- Document save

Includes both syntax errors (from parser) and type errors (from type checker).

## Known Limitations

1. Signature help is basic - doesn't parse function call context
2. Find references only works within the current document (no cross-file references yet)
3. No workspace symbols support (only document symbols)

## Future Enhancements

- Cross-file go-to-definition and find references
- Workspace symbols
- Code actions (quick fixes, refactorings)
- Rename symbol
- Format document
- Incremental parsing for better performance on large files
