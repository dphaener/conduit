package commands

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/conduit-lang/conduit/internal/lsp"
	"github.com/spf13/cobra"
)

// NewLSPCommand creates the LSP command
func NewLSPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lsp",
		Short: "Start the Language Server Protocol server",
		Long: `Start the Conduit Language Server Protocol (LSP) server.

This command starts an LSP server that provides IDE integration features including:
  • Code completion
  • Diagnostics (syntax and type errors)
  • Go-to-definition
  • Hover information
  • Find references
  • Document symbols
  • Signature help

The LSP server communicates via JSON-RPC over stdin/stdout.
It is typically started automatically by your editor/IDE.`,
		RunE: runLSP,
	}
}

func runLSP(cmd *cobra.Command, args []string) error {
	// Create server
	server := lsp.NewServer()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run server
	return server.Run(ctx)
}
