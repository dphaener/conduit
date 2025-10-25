package lsp

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/tooling"
	"go.lsp.dev/protocol"
)

func TestServerInitialization(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.api == nil {
		t.Error("Server API is nil")
	}

	if server.logger == nil {
		t.Error("Server logger is nil")
	}

	// Check capabilities
	if server.capabilities.CompletionProvider == nil {
		t.Error("CompletionProvider is nil")
	}

	if server.capabilities.DefinitionProvider == nil {
		t.Error("DefinitionProvider is nil")
	}

	// Check boolean capabilities are set correctly
	caps := server.capabilities
	if caps.HoverProvider != true {
		t.Error("HoverProvider should be true")
	}

	if caps.ReferencesProvider != true {
		t.Error("ReferencesProvider should be true")
	}

	if caps.DocumentSymbolProvider != true {
		t.Error("DocumentSymbolProvider should be true")
	}

	if caps.WorkspaceSymbolProvider != true {
		t.Error("WorkspaceSymbolProvider should be true")
	}
}

func TestConvertSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    tooling.DiagnosticSeverity
		expected protocol.DiagnosticSeverity
	}{
		{
			name:     "Error severity",
			input:    tooling.DiagnosticSeverityError,
			expected: protocol.DiagnosticSeverityError,
		},
		{
			name:     "Warning severity",
			input:    tooling.DiagnosticSeverityWarning,
			expected: protocol.DiagnosticSeverityWarning,
		},
		{
			name:     "Info severity",
			input:    tooling.DiagnosticSeverityInfo,
			expected: protocol.DiagnosticSeverityInformation,
		},
		{
			name:     "Hint severity",
			input:    tooling.DiagnosticSeverityHint,
			expected: protocol.DiagnosticSeverityHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("convertSeverity(%v): expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestStdRWC(t *testing.T) {
	// Test that stdrwc struct exists and implements expected methods
	rwc := stdrwc{}

	// Test Read method exists (we won't actually read from stdin)
	_ = rwc.Read

	// Test Write method exists
	_ = rwc.Write

	// Test Close method exists
	_ = rwc.Close
}
