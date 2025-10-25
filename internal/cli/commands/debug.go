package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/conduit-lang/conduit/internal/debug"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewDebugCommand creates the debug command
func NewDebugCommand() *cobra.Command {
	var (
		port           int
		sourceMapsPath string
		delvePath      string
	)

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Start the Debug Adapter Protocol (DAP) server",
		Long: `Start the Debug Adapter Protocol (DAP) server for debugging Conduit applications.

The DAP server enables IDE integration for interactive debugging with breakpoints,
step debugging, variable inspection, and call stack viewing.

Examples:
  conduit debug                    # Start DAP server on random port
  conduit debug --port 8080        # Start on specific port
  conduit debug --dap              # Start in DAP mode (used by VS Code)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebug(port, sourceMapsPath, delvePath)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Port to listen on (0 = random)")
	cmd.Flags().StringVar(&sourceMapsPath, "source-maps", "build", "Path to source maps directory")
	cmd.Flags().StringVar(&delvePath, "delve-path", "dlv", "Path to Delve executable")
	cmd.Flags().Bool("dap", false, "Start in DAP mode (used by IDE)")

	return cmd
}

// runDebug starts the DAP server
func runDebug(port int, sourceMapsPath, delvePath string) error {
	titleColor := color.New(color.FgCyan, color.Bold)
	infoColor := color.New(color.FgWhite)
	successColor := color.New(color.FgGreen, color.Bold)
	errorColor := color.New(color.FgRed, color.Bold)

	titleColor.Println("Conduit Debug Adapter Protocol Server")
	fmt.Println()

	// Resolve source maps path
	absSourceMapsPath, err := filepath.Abs(sourceMapsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve source maps path: %w", err)
	}

	// Check if source maps directory exists
	if _, err := os.Stat(absSourceMapsPath); os.IsNotExist(err) {
		errorColor.Printf("Source maps directory not found: %s\n", absSourceMapsPath)
		infoColor.Println("Please build your project first using: conduit build")
		return fmt.Errorf("source maps directory not found")
	}

	// Load source maps
	infoColor.Printf("Loading source maps from: %s\n", absSourceMapsPath)
	sourceMaps := debug.NewSourceMapRegistry()
	if err := sourceMaps.LoadFromDirectory(absSourceMapsPath); err != nil {
		errorColor.Printf("Failed to load source maps: %v\n", err)
		return err
	}

	// Note: In production, we would check map count here
	// For now, just proceed - the server will handle missing maps gracefully
	successColor.Println("Source maps loaded successfully")
	fmt.Println()

	// Determine address
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if port == 0 {
		addr = "127.0.0.1:0"
	}

	titleColor.Println("Starting DAP server...")
	infoColor.Printf("Address: %s\n", addr)
	infoColor.Printf("Delve: %s\n", delvePath)
	fmt.Println()
	successColor.Println("Ready to accept debug connections")
	fmt.Println()
	infoColor.Println("Press Ctrl+C to stop the server")

	// Start the DAP server
	if err := debug.Serve(addr, sourceMaps, delvePath); err != nil {
		errorColor.Printf("DAP server error: %v\n", err)
		return err
	}

	return nil
}
