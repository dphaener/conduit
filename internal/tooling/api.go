// Package tooling provides a programmatic API for IDE integration via LSP.
// It exposes compiler functionality in a thread-safe, performance-optimized manner
// suitable for Language Server Protocol implementations.
package tooling

import (
	"fmt"
	"sync"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
)

// API provides thread-safe access to compiler functionality for IDE integration.
// It maintains document state and provides fast query operations for LSP features.
type API struct {
	// Document cache stores parsed ASTs and type information per URI
	documents map[string]*Document
	docsMutex sync.RWMutex

	// Symbol index for fast lookups
	symbolIndex *SymbolIndex

	// Configuration
	config *Config
}

// Config holds configuration for the tooling API
type Config struct {
	// CacheSize limits the number of documents cached in memory
	CacheSize int

	// EnableIncrementalParsing enables incremental parsing for better performance
	EnableIncrementalParsing bool
}

// Document represents a cached document with its parsed AST and type information
type Document struct {
	// URI is the document identifier (typically a file path)
	URI string

	// Content is the raw source code
	Content string

	// Version tracks document changes (incremented on each update)
	Version int

	// AST is the parsed abstract syntax tree
	AST *ast.Program

	// ParseErrors contains any syntax errors from parsing
	ParseErrors []parser.ParseError

	// TypeErrors contains any type errors from type checking
	TypeErrors typechecker.ErrorList

	// Symbols is a flattened list of all symbols in the document
	Symbols []*Symbol
}

// Position represents a position in a document (zero-based for LSP compatibility)
type Position struct {
	Line      int // Zero-based line number
	Character int // Zero-based character offset
}

// Range represents a range in a document
type Range struct {
	Start Position
	End   Position
}

// Location represents a source location with URI and range
type Location struct {
	URI   string
	Range Range
}

// Symbol represents a named entity in the source code
type Symbol struct {
	Name  string
	Kind  SymbolKind
	Range Range

	// Type information (if available)
	Type string

	// For fields: parent resource name
	// For resources: empty
	ContainerName string

	// Documentation comment
	Documentation string

	// For functions/methods: signature
	Signature string

	// Detail provides additional information
	Detail string
}

// SymbolKind categorizes symbols for IDE display
type SymbolKind int

const (
	// SymbolKindResource represents a resource definition
	SymbolKindResource SymbolKind = iota
	// SymbolKindField represents a field in a resource
	SymbolKindField
	// SymbolKindRelationship represents a relationship between resources
	SymbolKindRelationship
	// SymbolKindHook represents a lifecycle hook
	SymbolKindHook
	// SymbolKindValidation represents a validation rule
	SymbolKindValidation
	// SymbolKindConstraint represents a constraint
	SymbolKindConstraint
	// SymbolKindComputed represents a computed field
	SymbolKindComputed
	// SymbolKindScope represents a named scope
	SymbolKindScope
	// SymbolKindFunction represents a function
	SymbolKindFunction
	// SymbolKindVariable represents a variable
	SymbolKindVariable
)

// Hover represents hover information for a symbol
type Hover struct {
	// Contents is the hover text (markdown formatted)
	Contents string

	// Range is the range of the symbol
	Range Range
}

// CompletionItem represents a completion suggestion
type CompletionItem struct {
	// Label is the text to display
	Label string

	// Kind categorizes the completion
	Kind CompletionKind

	// Detail provides additional information
	Detail string

	// Documentation provides help text
	Documentation string

	// InsertText is the text to insert (if different from label)
	InsertText string

	// SortText controls ordering (if different from label)
	SortText string
}

// CompletionKind categorizes completion items
type CompletionKind int

const (
	// CompletionKindKeyword represents a keyword completion
	CompletionKindKeyword CompletionKind = iota
	// CompletionKindType represents a type completion
	CompletionKindType
	// CompletionKindField represents a field completion
	CompletionKindField
	// CompletionKindFunction represents a function completion
	CompletionKindFunction
	// CompletionKindResource represents a resource completion
	CompletionKindResource
	// CompletionKindSnippet represents a code snippet completion
	CompletionKindSnippet
)

// Diagnostic represents a compilation error or warning
type Diagnostic struct {
	Range    Range
	Severity DiagnosticSeverity
	Code     string
	Message  string
	Source   string
}

// DiagnosticSeverity indicates the severity of a diagnostic
type DiagnosticSeverity int

const (
	// DiagnosticSeverityError represents an error diagnostic
	DiagnosticSeverityError DiagnosticSeverity = iota
	// DiagnosticSeverityWarning represents a warning diagnostic
	DiagnosticSeverityWarning
	// DiagnosticSeverityInfo represents an informational diagnostic
	DiagnosticSeverityInfo
	// DiagnosticSeverityHint represents a hint diagnostic
	DiagnosticSeverityHint
)

// NewAPI creates a new tooling API instance
func NewAPI() *API {
	return NewAPIWithConfig(&Config{
		CacheSize:                100,
		EnableIncrementalParsing: true,
	})
}

// NewAPIWithConfig creates a new tooling API with custom configuration
func NewAPIWithConfig(config *Config) *API {
	return &API{
		documents:   make(map[string]*Document),
		symbolIndex: NewSymbolIndex(),
		config:      config,
	}
}

// ParseFile parses a source file and returns the AST and any errors
func (a *API) ParseFile(uri, content string) (*Document, error) {
	// Tokenize
	l := lexer.New(content)
	tokens, _ := l.ScanTokens()

	// Parse
	p := parser.New(tokens)
	program, parseErrors := p.Parse()

	// Create document
	doc := &Document{
		URI:         uri,
		Content:     content,
		Version:     1,
		AST:         program,
		ParseErrors: parseErrors,
		Symbols:     make([]*Symbol, 0),
	}

	// If parsing succeeded, run type checker
	if len(parseErrors) == 0 {
		tc := typechecker.NewTypeChecker()
		typeErrors := tc.CheckProgram(program)
		doc.TypeErrors = typeErrors
	}

	// Extract symbols
	doc.Symbols = a.extractSymbols(doc)

	// Cache document
	a.docsMutex.Lock()
	a.documents[uri] = doc
	a.docsMutex.Unlock()

	// Update symbol index
	a.symbolIndex.Index(uri, doc.Symbols)

	return doc, nil
}

// UpdateDocument updates an existing document with new content
func (a *API) UpdateDocument(uri, content string, version int) (*Document, error) {
	a.docsMutex.Lock()
	defer a.docsMutex.Unlock()

	oldDoc, exists := a.documents[uri]
	if exists && oldDoc.Content == content {
		// Content unchanged, update version and return cached document
		oldDoc.Version = version
		a.documents[uri] = oldDoc
		return oldDoc, nil
	}

	// Temporarily release lock during expensive parsing
	a.docsMutex.Unlock()
	doc, err := a.parseFileInternal(uri, content)
	a.docsMutex.Lock()

	if err != nil {
		return nil, err
	}

	doc.Version = version
	a.documents[uri] = doc
	a.symbolIndex.Index(uri, doc.Symbols)

	return doc, nil
}

// parseFileInternal performs parsing without acquiring locks
func (a *API) parseFileInternal(uri, content string) (*Document, error) {
	// Tokenize
	l := lexer.New(content)
	tokens, _ := l.ScanTokens()

	// Parse
	p := parser.New(tokens)
	program, parseErrors := p.Parse()

	// Create document
	doc := &Document{
		URI:         uri,
		Content:     content,
		Version:     1,
		AST:         program,
		ParseErrors: parseErrors,
		Symbols:     make([]*Symbol, 0),
	}

	// If parsing succeeded, run type checker
	if len(parseErrors) == 0 {
		tc := typechecker.NewTypeChecker()
		typeErrors := tc.CheckProgram(program)
		doc.TypeErrors = typeErrors
	}

	// Extract symbols
	doc.Symbols = a.extractSymbols(doc)

	return doc, nil
}

// GetDocument retrieves a cached document
func (a *API) GetDocument(uri string) (*Document, bool) {
	a.docsMutex.RLock()
	defer a.docsMutex.RUnlock()

	doc, exists := a.documents[uri]
	return doc, exists
}

// CloseDocument removes a document from the cache
func (a *API) CloseDocument(uri string) {
	a.docsMutex.Lock()
	delete(a.documents, uri)
	a.docsMutex.Unlock()

	a.symbolIndex.RemoveDocument(uri)
}

// GetDiagnostics returns diagnostics for a document
func (a *API) GetDiagnostics(uri string) []Diagnostic {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil
	}

	diagnostics := make([]Diagnostic, 0)

	// Add parse errors
	for _, err := range doc.ParseErrors {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{
					Line:      err.Token.Line - 1,
					Character: err.Token.Column - 1,
				},
				End: Position{
					Line:      err.Token.Line - 1,
					Character: err.Token.Column + len(err.Token.Lexeme),
				},
			},
			Severity: DiagnosticSeverityError,
			Code:     "parse_error",
			Message:  err.Message,
			Source:   "conduit",
		})
	}

	// Add type errors
	for _, err := range doc.TypeErrors {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{
					Line:      err.Location.Line - 1,
					Character: err.Location.Column - 1,
				},
				End: Position{
					Line:      err.Location.Line - 1,
					Character: err.Location.Column + 1,
				},
			},
			Severity: diagnosticSeverityFromTypeError(err),
			Code:     string(err.Code),
			Message:  err.Error(),
			Source:   "conduit",
		})
	}

	return diagnostics
}

// GetHover returns hover information for a position in a document.
// Returns (nil, nil) if no symbol is found at the position.
func (a *API) GetHover(uri string, pos Position) (*Hover, error) {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	symbol := a.findSymbolAtPosition(doc, pos)
	if symbol == nil {
		return nil, nil //nolint:nilnil // nil hover is valid when no symbol at position
	}

	return a.buildHover(symbol), nil
}

// GetCompletions returns completion items for a position in a document
func (a *API) GetCompletions(uri string, pos Position) ([]CompletionItem, error) {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	// Determine completion context
	context := a.getCompletionContext(doc, pos)

	return a.buildCompletions(context), nil
}

// GetDefinition returns the definition location of a symbol at a position.
// Returns (nil, nil) if no symbol is found at the position.
func (a *API) GetDefinition(uri string, pos Position) (*Location, error) {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	symbol := a.findSymbolAtPosition(doc, pos)
	if symbol == nil {
		return nil, nil //nolint:nilnil // nil location is valid when no symbol at position
	}

	// If it's a reference to another resource/type, find its definition
	if symbol.Kind == SymbolKindField && symbol.Type != "" {
		// Check if type is a resource
		defSymbol := a.symbolIndex.FindDefinition(symbol.Type)
		if defSymbol != nil {
			return &Location{
				URI:   defSymbol.URI,
				Range: defSymbol.Range,
			}, nil
		}
	}

	// Return the symbol's own location
	return &Location{
		URI:   uri,
		Range: symbol.Range,
	}, nil
}

// GetReferences returns all references to a symbol
func (a *API) GetReferences(uri string, pos Position) ([]Location, error) {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	symbol := a.findSymbolAtPosition(doc, pos)
	if symbol == nil {
		return []Location{}, nil
	}

	refs := a.symbolIndex.FindReferences(symbol.Name)
	if refs == nil {
		return []Location{}, nil
	}

	return refs, nil
}

// GetDocumentSymbols returns all symbols in a document
func (a *API) GetDocumentSymbols(uri string) ([]*Symbol, error) {
	doc, exists := a.GetDocument(uri)
	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	return doc.Symbols, nil
}

// Helper functions

func diagnosticSeverityFromTypeError(err *typechecker.TypeError) DiagnosticSeverity {
	switch err.Severity {
	case typechecker.SeverityError:
		return DiagnosticSeverityError
	case typechecker.SeverityWarning:
		return DiagnosticSeverityWarning
	default:
		return DiagnosticSeverityError
	}
}
