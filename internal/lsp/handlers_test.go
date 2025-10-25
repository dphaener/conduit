package lsp

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/tooling"
	"go.lsp.dev/protocol"
)

func TestConvertCompletionKind(t *testing.T) {
	tests := []struct {
		name     string
		input    tooling.CompletionKind
		expected protocol.CompletionItemKind
	}{
		{"Keyword", tooling.CompletionKindKeyword, protocol.CompletionItemKindKeyword},
		{"Type", tooling.CompletionKindType, protocol.CompletionItemKindClass},
		{"Field", tooling.CompletionKindField, protocol.CompletionItemKindField},
		{"Function", tooling.CompletionKindFunction, protocol.CompletionItemKindFunction},
		{"Resource", tooling.CompletionKindResource, protocol.CompletionItemKindClass},
		{"Snippet", tooling.CompletionKindSnippet, protocol.CompletionItemKindSnippet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertCompletionKind(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConvertSymbolKind(t *testing.T) {
	tests := []struct {
		name     string
		input    tooling.SymbolKind
		expected protocol.SymbolKind
	}{
		{"Resource", tooling.SymbolKindResource, protocol.SymbolKindClass},
		{"Field", tooling.SymbolKindField, protocol.SymbolKindField},
		{"Relationship", tooling.SymbolKindRelationship, protocol.SymbolKindProperty},
		{"Hook", tooling.SymbolKindHook, protocol.SymbolKindMethod},
		{"Validation", tooling.SymbolKindValidation, protocol.SymbolKindMethod},
		{"Constraint", tooling.SymbolKindConstraint, protocol.SymbolKindMethod},
		{"Computed", tooling.SymbolKindComputed, protocol.SymbolKindProperty},
		{"Scope", tooling.SymbolKindScope, protocol.SymbolKindNamespace},
		{"Function", tooling.SymbolKindFunction, protocol.SymbolKindFunction},
		{"Variable", tooling.SymbolKindVariable, protocol.SymbolKindVariable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSymbolKind(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHandleHover(t *testing.T) {
	// Test is covered by server_test.go integration tests
	// Direct testing of private handlers requires embedding jsonrpc2 infrastructure
	t.Skip("Covered by integration tests in server_test.go")
}

func TestHandleDefinition(t *testing.T) {
	// Test is covered by server_test.go integration tests
	t.Skip("Covered by integration tests in server_test.go")
}

func TestHandleReferences(t *testing.T) {
	// Test is covered by server_test.go integration tests
	t.Skip("Covered by integration tests in server_test.go")
}

func TestHandleDocumentSymbol(t *testing.T) {
	// Test is covered by server_test.go integration tests
	t.Skip("Covered by integration tests in server_test.go")
}

func TestHandleWorkspaceSymbol(t *testing.T) {
	// Test is covered by server_test.go integration tests
	t.Skip("Covered by integration tests in server_test.go")
}

func TestCompletionSnippetFormat(t *testing.T) {
	// Test is covered by server_test.go integration tests
	t.Skip("Covered by integration tests in server_test.go")
}
