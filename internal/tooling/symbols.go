package tooling

import (
	"fmt"
	"strings"
	"sync"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// SymbolIndex maintains a searchable index of all symbols across documents
type SymbolIndex struct {
	// symbols maps symbol name to all definitions
	symbols map[string][]*IndexedSymbol
	mutex   sync.RWMutex
}

// IndexedSymbol represents a symbol with its location
type IndexedSymbol struct {
	URI   string
	Range Range
	*Symbol
}

// NewSymbolIndex creates a new symbol index
func NewSymbolIndex() *SymbolIndex {
	return &SymbolIndex{
		symbols: make(map[string][]*IndexedSymbol),
	}
}

// Index adds symbols from a document to the index
func (si *SymbolIndex) Index(uri string, symbols []*Symbol) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// Remove old symbols from this document
	si.removeDocumentLocked(uri)

	// Add new symbols
	for _, sym := range symbols {
		indexed := &IndexedSymbol{
			URI:    uri,
			Range:  sym.Range,
			Symbol: sym,
		}

		si.symbols[sym.Name] = append(si.symbols[sym.Name], indexed)
	}
}

// RemoveDocument removes all symbols from a document
func (si *SymbolIndex) RemoveDocument(uri string) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	si.removeDocumentLocked(uri)
}

func (si *SymbolIndex) removeDocumentLocked(uri string) {
	// Remove symbols from this document
	for name, syms := range si.symbols {
		filtered := make([]*IndexedSymbol, 0, len(syms))
		for _, sym := range syms {
			if sym.URI != uri {
				filtered = append(filtered, sym)
			}
		}
		if len(filtered) > 0 {
			si.symbols[name] = filtered
		} else {
			delete(si.symbols, name)
		}
	}
}

// FindDefinition finds the definition of a symbol by name
func (si *SymbolIndex) FindDefinition(name string) *IndexedSymbol {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	syms, ok := si.symbols[name]
	if !ok || len(syms) == 0 {
		return nil
	}

	// Return the first definition (typically there's only one)
	// Prefer resource definitions over field definitions
	for _, sym := range syms {
		if sym.Kind == SymbolKindResource {
			return sym
		}
	}

	return syms[0]
}

// FindReferences finds all references to a symbol
func (si *SymbolIndex) FindReferences(name string) []Location {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	syms, ok := si.symbols[name]
	if !ok {
		return nil
	}

	locations := make([]Location, len(syms))
	for i, sym := range syms {
		locations[i] = Location{
			URI:   sym.URI,
			Range: sym.Range,
		}
	}

	return locations
}

// SearchSymbols searches for symbols matching a query across all documents
func (si *SymbolIndex) SearchSymbols(query string) []*IndexedSymbol {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	if query == "" {
		// Return all symbols if query is empty
		result := make([]*IndexedSymbol, 0)
		for _, syms := range si.symbols {
			result = append(result, syms...)
		}
		return result
	}

	query = strings.ToLower(query)
	result := make([]*IndexedSymbol, 0)

	for name, syms := range si.symbols {
		// Case-insensitive substring match
		if strings.Contains(strings.ToLower(name), query) {
			result = append(result, syms...)
		}
	}

	return result
}

// extractSymbols extracts all symbols from a document's AST
func (a *API) extractSymbols(doc *Document) []*Symbol {
	if doc.AST == nil {
		return nil
	}

	symbols := make([]*Symbol, 0)

	for _, resource := range doc.AST.Resources {
		symbols = append(symbols, a.extractResourceSymbols(resource)...)
	}

	return symbols
}

// extractResourceSymbols extracts symbols from a single resource
func (a *API) extractResourceSymbols(resource *ast.ResourceNode) []*Symbol {
	if resource == nil {
		return nil
	}

	symbols := make([]*Symbol, 0)

	// Add resource symbol
	// resource.Loc points to the "resource" keyword, not the name
	// The name appears after "resource " (8 characters + 1 space = 9)
	nameStartColumn := resource.Loc.Column + len("resource ")
	symbols = append(symbols, &Symbol{
		Name: resource.Name,
		Kind: SymbolKindResource,
		Range: Range{
			Start: Position{
				Line:      resource.Loc.Line - 1,
				Character: nameStartColumn - 1, // Convert to 0-based
			},
			End: Position{
				Line:      resource.Loc.Line - 1,
				Character: nameStartColumn + len(resource.Name) - 1, // Convert to 0-based
			},
		},
		Type:          "resource",
		Documentation: resource.Documentation,
		Detail:        fmt.Sprintf("resource %s", resource.Name),
	})

	// Add field symbols
	if resource.Fields != nil {
		for _, field := range resource.Fields {
			if field == nil || field.Type == nil {
				continue
			}
			typeStr := formatType(field.Type)
			symbols = append(symbols, &Symbol{
				Name: field.Name,
				Kind: SymbolKindField,
				Range: Range{
					Start: Position{
						Line:      field.Loc.Line - 1,
						Character: field.Loc.Column - 1,
					},
					End: Position{
						Line:      field.Loc.Line - 1,
						Character: field.Loc.Column + len(field.Name),
					},
				},
				Type:          typeStr,
				ContainerName: resource.Name,
				Detail:        fmt.Sprintf("%s: %s", field.Name, typeStr),
			})
		}
	}

	// Add relationship symbols
	symbols = append(symbols, extractRelationshipSymbols(resource)...)

	// Add hook symbols
	symbols = append(symbols, extractHookSymbols(resource)...)

	// Add computed field symbols
	symbols = append(symbols, extractComputedSymbols(resource)...)

	// Add scope symbols
	symbols = append(symbols, extractScopeSymbols(resource)...)

	return symbols
}

func extractRelationshipSymbols(resource *ast.ResourceNode) []*Symbol {
	symbols := make([]*Symbol, 0, len(resource.Relationships))

	for _, rel := range resource.Relationships {
		relType := fmt.Sprintf("%s%s", rel.Type, formatNullability(rel.Nullable))
		symbols = append(symbols, &Symbol{
			Name: rel.Name,
			Kind: SymbolKindRelationship,
			Range: Range{
				Start: Position{
					Line:      rel.Loc.Line - 1,
					Character: rel.Loc.Column - 1,
				},
				End: Position{
					Line:      rel.Loc.Line - 1,
					Character: rel.Loc.Column + len(rel.Name),
				},
			},
			Type:          relType,
			ContainerName: resource.Name,
			Detail:        fmt.Sprintf("%s: %s (%s)", rel.Name, relType, relationshipKindString(rel.Kind)),
		})
	}

	return symbols
}

func extractHookSymbols(resource *ast.ResourceNode) []*Symbol {
	symbols := make([]*Symbol, 0, len(resource.Hooks))

	for _, hook := range resource.Hooks {
		hookName := fmt.Sprintf("%s_%s", hook.Timing, hook.Event)
		symbols = append(symbols, &Symbol{
			Name: hookName,
			Kind: SymbolKindHook,
			Range: Range{
				Start: Position{
					Line:      hook.Loc.Line - 1,
					Character: hook.Loc.Column - 1,
				},
				End: Position{
					Line:      hook.Loc.Line - 1,
					Character: hook.Loc.Column + len(hookName),
				},
			},
			ContainerName: resource.Name,
			Detail:        fmt.Sprintf("@%s %s", hook.Timing, hook.Event),
		})
	}

	return symbols
}

func extractComputedSymbols(resource *ast.ResourceNode) []*Symbol {
	symbols := make([]*Symbol, 0, len(resource.Computed))

	for _, computed := range resource.Computed {
		typeStr := formatType(computed.Type)
		symbols = append(symbols, &Symbol{
			Name: computed.Name,
			Kind: SymbolKindComputed,
			Range: Range{
				Start: Position{
					Line:      computed.Loc.Line - 1,
					Character: computed.Loc.Column - 1,
				},
				End: Position{
					Line:      computed.Loc.Line - 1,
					Character: computed.Loc.Column + len(computed.Name),
				},
			},
			Type:          typeStr,
			ContainerName: resource.Name,
			Detail:        fmt.Sprintf("@computed %s: %s", computed.Name, typeStr),
		})
	}

	return symbols
}

func extractScopeSymbols(resource *ast.ResourceNode) []*Symbol {
	symbols := make([]*Symbol, 0, len(resource.Scopes))

	for _, scope := range resource.Scopes {
		symbols = append(symbols, &Symbol{
			Name: scope.Name,
			Kind: SymbolKindScope,
			Range: Range{
				Start: Position{
					Line:      scope.Loc.Line - 1,
					Character: scope.Loc.Column - 1,
				},
				End: Position{
					Line:      scope.Loc.Line - 1,
					Character: scope.Loc.Column + len(scope.Name),
				},
			},
			ContainerName: resource.Name,
			Detail:        fmt.Sprintf("@scope %s", scope.Name),
		})
	}

	return symbols
}

// findSymbolAtPosition finds the symbol at a given position in a document
func (a *API) findSymbolAtPosition(doc *Document, pos Position) *Symbol {
	for _, sym := range doc.Symbols {
		if positionInRange(pos, sym.Range) {
			return sym
		}
	}
	return nil
}

// positionInRange checks if a position is within a range
func positionInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}

	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}

	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}

	return true
}

// formatType formats an AST type node as a string
func formatType(t *ast.TypeNode) string {
	if t == nil {
		return ""
	}

	var sb strings.Builder

	switch t.Kind {
	case ast.TypePrimitive:
		sb.WriteString(t.Name)
	case ast.TypeArray:
		sb.WriteString("array<")
		sb.WriteString(formatType(t.ElementType))
		sb.WriteString(">")
	case ast.TypeHash:
		sb.WriteString("hash<")
		sb.WriteString(formatType(t.KeyType))
		sb.WriteString(", ")
		sb.WriteString(formatType(t.ValueType))
		sb.WriteString(">")
	case ast.TypeEnum:
		sb.WriteString("enum[")
		for i, v := range t.EnumValues {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", v))
		}
		sb.WriteString("]")
	case ast.TypeResource:
		sb.WriteString(t.Name)
	}

	sb.WriteString(formatNullability(t.Nullable))

	return sb.String()
}

// formatNullability formats the nullability marker
func formatNullability(nullable bool) string {
	if nullable {
		return "?"
	}
	return "!"
}

// relationshipKindString returns a string representation of a relationship kind
func relationshipKindString(kind ast.RelationshipKind) string {
	switch kind {
	case ast.RelationshipBelongsTo:
		return "belongs_to"
	case ast.RelationshipHasMany:
		return "has_many"
	case ast.RelationshipHasManyThrough:
		return "has_many_through"
	case ast.RelationshipHasOne:
		return "has_one"
	default:
		return "unknown"
	}
}
