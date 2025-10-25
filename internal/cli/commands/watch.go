package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/watch"
)

// NewWatchCommand creates the watch command
func NewWatchCommand() *cobra.Command {
	var (
		port     int
		appPort  int
		verbose  bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Start development server with hot reload",
		Long: `Start the development server with automatic file watching and hot reload.

The watch command monitors your .cdt files for changes and automatically:
  • Recompiles changed files incrementally
  • Rebuilds the application binary
  • Restarts the server
  • Reloads connected browsers

Performance targets:
  • File change detection: <10ms
  • Incremental compile: <200ms
  • Browser reload: <100ms
  • Total (save to visible): <500ms

Examples:
  # Start with default ports (3000 for dev server, 3001 for app)
  conduit watch

  # Use custom ports
  conduit watch --port 8080 --app-port 8081

  # Enable verbose logging
  conduit watch --verbose
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if app directory exists
			if _, err := os.Stat("app"); os.IsNotExist(err) {
				return fmt.Errorf("app/ directory not found - are you in a Conduit project?")
			}

			// Create dev server configuration
			config := &watch.DevServerConfig{
				Port:    port,
				AppPort: appPort,
				WatchPatterns: []string{
					"*.cdt",
					"*.css",
					"*.js",
					"*.html",
				},
				IgnorePatterns: []string{
					"*.swp",
					"*.swo",
					"*~",
					".DS_Store",
				},
			}

			// Create dev server
			devServer, err := watch.NewDevServer(config)
			if err != nil {
				return fmt.Errorf("failed to create dev server: %w", err)
			}

			// Start dev server
			if err := devServer.Start(); err != nil {
				return fmt.Errorf("failed to start dev server: %w", err)
			}

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Display banner
			banner := color.New(color.FgCyan, color.Bold)
			info := color.New(color.FgWhite)

			fmt.Println()
			banner.Println("📦 Conduit Development Server")
			info.Printf("   Dev server: http://localhost:%d\n", port)
			info.Printf("   App server: http://localhost:%d\n", appPort)
			fmt.Println()
			color.New(color.FgYellow).Println("⌨️  Press Ctrl+C to stop")
			fmt.Println()

			// Block until signal
			<-sigChan

			fmt.Println("\n\nShutting down...")

			// Stop dev server
			if err := devServer.Stop(); err != nil {
				return fmt.Errorf("error stopping dev server: %w", err)
			}

			color.New(color.FgGreen).Println("Goodbye!")
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 3000, "Development server port")
	cmd.Flags().IntVar(&appPort, "app-port", 3001, "Application server port")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show verbose output")

	return cmd
}
