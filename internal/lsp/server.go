// Package lsp implements a Language Server Protocol server for Conduit.
// It provides IDE integration features including code completion, diagnostics,
// go-to-definition, hover information, and more.
package lsp

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/conduit-lang/conduit/internal/tooling"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

// Server implements the LSP server for Conduit
type Server struct {
	// api is the tooling API that provides compiler functionality
	api *tooling.API

	// conn is the JSON-RPC connection
	conn jsonrpc2.Conn

	// client is the LSP client interface
	client protocol.Client

	// logger for debugging
	logger *log.Logger

	// workspaceRoot is the root directory of the workspace
	workspaceRoot string

	// Server capabilities
	capabilities protocol.ServerCapabilities

	// cancel is used to signal server shutdown
	cancel context.CancelFunc
}

// NewServer creates a new LSP server instance
func NewServer() *Server {
	logger := log.New(os.Stderr, "[LSP] ", log.LstdFlags)

	return &Server{
		api:    tooling.NewAPI(),
		logger: logger,
		capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
				Save: &protocol.SaveOptions{
					IncludeText: false,
				},
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{":", "@", "."},
				ResolveProvider:   false,
			},
			HoverProvider: true,
			DefinitionProvider: &protocol.DefinitionOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			ReferencesProvider:      true,
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			DocumentFormattingProvider: &protocol.DocumentFormattingOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			DocumentRangeFormattingProvider: &protocol.DocumentRangeFormattingOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
		},
	}
}

// Run starts the LSP server
func (s *Server) Run(ctx context.Context) error {
	s.logger.Println("Starting Conduit Language Server")

	// Create context with cancellation for shutdown
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Create JSON-RPC stream handler
	stream := jsonrpc2.NewStream(stdrwc{})

	// Create connection
	conn := jsonrpc2.NewConn(stream)
	s.conn = conn

	// Create zap logger
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		s.logger.Printf("Warning: Failed to create zap logger: %v", err)
		// Fall back to nop logger
		zapLogger = zap.NewNop()
	}
	s.client = protocol.ClientDispatcher(conn, zapLogger)

	// Register handlers
	conn.Go(ctx, s.handler())

	// Wait for context cancellation
	<-ctx.Done()

	s.logger.Println("Shutting down Conduit Language Server")
	return conn.Close()
}

// handler returns the JSON-RPC handler function
func (s *Server) handler() jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		s.logger.Printf("Received: %s", req.Method())

		switch req.Method() {
		case protocol.MethodInitialize:
			return s.handleInitialize(ctx, reply, req)
		case protocol.MethodInitialized:
			return s.handleInitialized(ctx, reply, req)
		case protocol.MethodShutdown:
			return s.handleShutdown(ctx, reply, req)
		case protocol.MethodExit:
			return s.handleExit(ctx, reply, req)
		case protocol.MethodTextDocumentDidOpen:
			return s.handleTextDocumentDidOpen(ctx, reply, req)
		case protocol.MethodTextDocumentDidChange:
			return s.handleTextDocumentDidChange(ctx, reply, req)
		case protocol.MethodTextDocumentDidClose:
			return s.handleTextDocumentDidClose(ctx, reply, req)
		case protocol.MethodTextDocumentDidSave:
			return s.handleTextDocumentDidSave(ctx, reply, req)
		case protocol.MethodTextDocumentCompletion:
			return s.handleTextDocumentCompletion(ctx, reply, req)
		case protocol.MethodTextDocumentHover:
			return s.handleTextDocumentHover(ctx, reply, req)
		case protocol.MethodTextDocumentDefinition:
			return s.handleTextDocumentDefinition(ctx, reply, req)
		case protocol.MethodTextDocumentReferences:
			return s.handleTextDocumentReferences(ctx, reply, req)
		case protocol.MethodTextDocumentDocumentSymbol:
			return s.handleTextDocumentDocumentSymbol(ctx, reply, req)
		case protocol.MethodWorkspaceSymbol:
			return s.handleWorkspaceSymbol(ctx, reply, req)
		case protocol.MethodTextDocumentFormatting:
			return s.handleTextDocumentFormatting(ctx, reply, req)
		case protocol.MethodTextDocumentRangeFormatting:
			return s.handleTextDocumentRangeFormatting(ctx, reply, req)
		default:
			return reply(ctx, nil, jsonrpc2.ErrMethodNotFound)
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.InitializeParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse initialize params")
	}

	s.logger.Printf("Initialize from client: %v", params.ClientInfo)

	// Extract workspace root from params
	if len(params.WorkspaceFolders) > 0 {
		// Use workspace folders if available (LSP 3.6+)
		s.workspaceRoot = uri.URI(params.WorkspaceFolders[0].URI).Filename()
		s.logger.Printf("Workspace root set to: %s", s.workspaceRoot)
	} else if params.RootURI != "" {
		// Fall back to rootUri (deprecated but still used)
		s.workspaceRoot = params.RootURI.Filename()
		s.logger.Printf("Workspace root set to: %s (from rootUri)", s.workspaceRoot)
	} else if params.RootPath != "" {
		// Fall back to rootPath (deprecated)
		s.workspaceRoot = params.RootPath
		s.logger.Printf("Workspace root set to: %s (from rootPath)", s.workspaceRoot)
	}

	result := protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo: &protocol.ServerInfo{
			Name:    "conduit-lsp",
			Version: "0.1.0",
		},
	}

	return reply(ctx, result, nil)
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	s.logger.Println("Client initialized")
	return reply(ctx, nil, nil)
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	s.logger.Println("Shutdown requested")
	return reply(ctx, nil, nil)
}

// handleExit handles the exit notification
func (s *Server) handleExit(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	s.logger.Println("Exit requested")
	// Reply first, then trigger shutdown
	if err := reply(ctx, nil, nil); err != nil {
		s.logger.Printf("Error replying to exit: %v", err)
	}
	// Cancel the context to trigger graceful shutdown
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// handleTextDocumentDidOpen handles document open notifications
func (s *Server) handleTextDocumentDidOpen(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse didOpen params")
	}

	uri := string(params.TextDocument.URI)
	content := params.TextDocument.Text
	version := int(params.TextDocument.Version)

	s.logger.Printf("Document opened: %s (version %d)", uri, version)

	// Parse and cache the document
	_, err := s.api.ParseFile(uri, content)
	if err != nil {
		s.logger.Printf("Error parsing document: %v", err)
	}

	// Publish diagnostics
	s.publishDiagnostics(ctx, uri)

	return reply(ctx, nil, nil)
}

// handleTextDocumentDidChange handles document change notifications
func (s *Server) handleTextDocumentDidChange(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse didChange params")
	}

	uri := string(params.TextDocument.URI)
	version := int(params.TextDocument.Version)

	if len(params.ContentChanges) == 0 {
		return reply(ctx, nil, nil)
	}

	// We use full document sync, so take the last change
	content := params.ContentChanges[len(params.ContentChanges)-1].Text

	s.logger.Printf("Document changed: %s (version %d)", uri, version)

	// Update document
	_, err := s.api.UpdateDocument(uri, content, version)
	if err != nil {
		s.logger.Printf("Error updating document: %v", err)
	}

	// Publish diagnostics
	s.publishDiagnostics(ctx, uri)

	return reply(ctx, nil, nil)
}

// handleTextDocumentDidClose handles document close notifications
func (s *Server) handleTextDocumentDidClose(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse didClose params")
	}

	uri := string(params.TextDocument.URI)
	s.logger.Printf("Document closed: %s", uri)

	// Remove from cache
	s.api.CloseDocument(uri)

	return reply(ctx, nil, nil)
}

// handleTextDocumentDidSave handles document save notifications
func (s *Server) handleTextDocumentDidSave(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return s.replyWithError(ctx, reply, jsonrpc2.InvalidParams, "Failed to parse didSave params")
	}

	uri := string(params.TextDocument.URI)
	s.logger.Printf("Document saved: %s", uri)

	// Re-publish diagnostics on save
	s.publishDiagnostics(ctx, uri)

	return reply(ctx, nil, nil)
}

// publishDiagnostics publishes diagnostics for a document
func (s *Server) publishDiagnostics(ctx context.Context, uri string) {
	diagnostics := s.api.GetDiagnostics(uri)

	lspDiagnostics := make([]protocol.Diagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		lspDiagnostics = append(lspDiagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(d.Range.Start.Line),
					Character: uint32(d.Range.Start.Character),
				},
				End: protocol.Position{
					Line:      uint32(d.Range.End.Line),
					Character: uint32(d.Range.End.Character),
				},
			},
			Severity: convertSeverity(d.Severity),
			Code:     d.Code,
			Source:   d.Source,
			Message:  d.Message,
		})
	}

	params := protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentURI(uri),
		Diagnostics: lspDiagnostics,
	}

	err := s.client.PublishDiagnostics(ctx, &params)
	if err != nil {
		s.logger.Printf("Error publishing diagnostics: %v", err)
	}
}

// replyWithError sends an LSP-compliant error response
func (s *Server) replyWithError(ctx context.Context, reply jsonrpc2.Replier, code jsonrpc2.Code, message string) error {
	return reply(ctx, nil, &jsonrpc2.Error{
		Code:    code,
		Message: message,
	})
}

// convertSeverity converts tooling diagnostic severity to LSP severity
func convertSeverity(severity tooling.DiagnosticSeverity) protocol.DiagnosticSeverity {
	switch severity {
	case tooling.DiagnosticSeverityError:
		return protocol.DiagnosticSeverityError
	case tooling.DiagnosticSeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case tooling.DiagnosticSeverityInfo:
		return protocol.DiagnosticSeverityInformation
	case tooling.DiagnosticSeverityHint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

// stdrwc implements io.ReadWriteCloser for stdin/stdout
type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
