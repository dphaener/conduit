package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/conduit-lang/conduit/internal/tooling"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// handleTextDocumentCompletion handles completion requests
func (s *Server) handleTextDocumentCompletion(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.CompletionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse completion params")
	}

	uri := string(params.TextDocument.URI)
	pos := tooling.Position{
		Line:      int(params.Position.Line),
		Character: int(params.Position.Character),
	}

	completions, err := s.api.GetCompletions(uri, pos)
	if err != nil {
		s.logger.Printf("Error getting completions: %v", err)
		return s.replyWithError(ctx, reply, jsonrpc2.InternalError, "Failed to get completions")
	}

	// Convert to LSP completion items
	items := make([]protocol.CompletionItem, 0, len(completions))
	for _, c := range completions {
		item := protocol.CompletionItem{
			Label:  c.Label,
			Kind:   convertCompletionKind(c.Kind),
			Detail: c.Detail,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: c.Documentation,
			},
			InsertText: c.InsertText,
		}

		// Set snippet format if InsertText contains snippet placeholders
		if strings.Contains(c.InsertText, "$0") || strings.Contains(c.InsertText, "${") {
			item.InsertTextFormat = protocol.InsertTextFormatSnippet
		} else {
			item.InsertTextFormat = protocol.InsertTextFormatPlainText
		}

		if c.SortText != "" {
			item.SortText = c.SortText
		}
		items = append(items, item)
	}

	result := protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}

	return reply(ctx, result, nil)
}

// handleTextDocumentHover handles hover requests
func (s *Server) handleTextDocumentHover(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.HoverParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse hover params")
	}

	uri := string(params.TextDocument.URI)
	pos := tooling.Position{
		Line:      int(params.Position.Line),
		Character: int(params.Position.Character),
	}

	hover, err := s.api.GetHover(uri, pos)
	if err != nil {
		s.logger.Printf("Error getting hover: %v", err)
		return s.replyWithError(ctx, reply, jsonrpc2.InternalError, "Failed to get hover information")
	}

	if hover == nil {
		return reply(ctx, nil, nil)
	}

	result := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: hover.Contents,
		},
		Range: &protocol.Range{
			Start: protocol.Position{
				Line:      uint32(hover.Range.Start.Line),
				Character: uint32(hover.Range.Start.Character),
			},
			End: protocol.Position{
				Line:      uint32(hover.Range.End.Line),
				Character: uint32(hover.Range.End.Character),
			},
		},
	}

	return reply(ctx, result, nil)
}

// handleTextDocumentDefinition handles go-to-definition requests
func (s *Server) handleTextDocumentDefinition(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DefinitionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse definition params")
	}

	uri := string(params.TextDocument.URI)
	pos := tooling.Position{
		Line:      int(params.Position.Line),
		Character: int(params.Position.Character),
	}

	location, err := s.api.GetDefinition(uri, pos)
	if err != nil {
		s.logger.Printf("Error getting definition: %v", err)
		return s.replyWithError(ctx, reply, jsonrpc2.InternalError, "Failed to get definition")
	}

	if location == nil {
		return reply(ctx, nil, nil)
	}

	result := protocol.Location{
		URI: protocol.DocumentURI(location.URI),
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(location.Range.Start.Line),
				Character: uint32(location.Range.Start.Character),
			},
			End: protocol.Position{
				Line:      uint32(location.Range.End.Line),
				Character: uint32(location.Range.End.Character),
			},
		},
	}

	return reply(ctx, result, nil)
}

// handleTextDocumentReferences handles find references requests
func (s *Server) handleTextDocumentReferences(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.ReferenceParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse references params")
	}

	uri := string(params.TextDocument.URI)
	pos := tooling.Position{
		Line:      int(params.Position.Line),
		Character: int(params.Position.Character),
	}

	references, err := s.api.GetReferences(uri, pos)
	if err != nil {
		s.logger.Printf("Error getting references: %v", err)
		return s.replyWithError(ctx, reply, jsonrpc2.InternalError, "Failed to get references")
	}

	// Convert to LSP locations
	locations := make([]protocol.Location, 0, len(references))
	for _, ref := range references {
		locations = append(locations, protocol.Location{
			URI: protocol.DocumentURI(ref.URI),
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(ref.Range.Start.Line),
					Character: uint32(ref.Range.Start.Character),
				},
				End: protocol.Position{
					Line:      uint32(ref.Range.End.Line),
					Character: uint32(ref.Range.End.Character),
				},
			},
		})
	}

	return reply(ctx, locations, nil)
}

// handleTextDocumentDocumentSymbol handles document symbol requests
func (s *Server) handleTextDocumentDocumentSymbol(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DocumentSymbolParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse document symbol params")
	}

	uri := string(params.TextDocument.URI)

	symbols, err := s.api.GetDocumentSymbols(uri)
	if err != nil {
		s.logger.Printf("Error getting document symbols: %v", err)
		return s.replyWithError(ctx, reply, jsonrpc2.InternalError, "Failed to get document symbols")
	}

	// Convert to LSP document symbols
	lspSymbols := make([]protocol.DocumentSymbol, 0, len(symbols))
	for _, sym := range symbols {
		detail := sym.Detail
		lspSym := protocol.DocumentSymbol{
			Name:   sym.Name,
			Kind:   convertSymbolKind(sym.Kind),
			Detail: detail,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(sym.Range.Start.Line),
					Character: uint32(sym.Range.Start.Character),
				},
				End: protocol.Position{
					Line:      uint32(sym.Range.End.Line),
					Character: uint32(sym.Range.End.Character),
				},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(sym.Range.Start.Line),
					Character: uint32(sym.Range.Start.Character),
				},
				End: protocol.Position{
					Line:      uint32(sym.Range.End.Line),
					Character: uint32(sym.Range.End.Character),
				},
			},
		}
		lspSymbols = append(lspSymbols, lspSym)
	}

	return reply(ctx, lspSymbols, nil)
}

// handleWorkspaceSymbol handles workspace symbol search requests
func (s *Server) handleWorkspaceSymbol(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.WorkspaceSymbolParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse workspace symbol params")
	}

	query := params.Query
	indexedSymbols := s.api.GetWorkspaceSymbols(query)

	// Convert to LSP symbol information directly from indexed symbols
	symbols := make([]protocol.SymbolInformation, 0, len(indexedSymbols))
	for _, indexed := range indexedSymbols {
		symbols = append(symbols, protocol.SymbolInformation{
			Name: indexed.Symbol.Name,
			Kind: convertSymbolKind(indexed.Symbol.Kind),
			Location: protocol.Location{
				URI: protocol.DocumentURI(indexed.URI),
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(indexed.Range.Start.Line),
						Character: uint32(indexed.Range.Start.Character),
					},
					End: protocol.Position{
						Line:      uint32(indexed.Range.End.Line),
						Character: uint32(indexed.Range.End.Character),
					},
				},
			},
			ContainerName: indexed.Symbol.ContainerName,
		})
	}

	return reply(ctx, symbols, nil)
}

// Helper functions to convert between tooling and LSP types

func convertCompletionKind(kind tooling.CompletionKind) protocol.CompletionItemKind {
	switch kind {
	case tooling.CompletionKindKeyword:
		return protocol.CompletionItemKindKeyword
	case tooling.CompletionKindType:
		return protocol.CompletionItemKindClass
	case tooling.CompletionKindField:
		return protocol.CompletionItemKindField
	case tooling.CompletionKindFunction:
		return protocol.CompletionItemKindFunction
	case tooling.CompletionKindResource:
		return protocol.CompletionItemKindClass
	case tooling.CompletionKindSnippet:
		return protocol.CompletionItemKindSnippet
	default:
		return protocol.CompletionItemKindText
	}
}

func convertSymbolKind(kind tooling.SymbolKind) protocol.SymbolKind {
	switch kind {
	case tooling.SymbolKindResource:
		return protocol.SymbolKindClass
	case tooling.SymbolKindField:
		return protocol.SymbolKindField
	case tooling.SymbolKindRelationship:
		return protocol.SymbolKindProperty
	case tooling.SymbolKindHook:
		return protocol.SymbolKindMethod
	case tooling.SymbolKindValidation:
		return protocol.SymbolKindMethod
	case tooling.SymbolKindConstraint:
		return protocol.SymbolKindMethod
	case tooling.SymbolKindComputed:
		return protocol.SymbolKindProperty
	case tooling.SymbolKindScope:
		return protocol.SymbolKindNamespace
	case tooling.SymbolKindFunction:
		return protocol.SymbolKindFunction
	case tooling.SymbolKindVariable:
		return protocol.SymbolKindVariable
	default:
		return protocol.SymbolKindObject
	}
}
