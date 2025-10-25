package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/watch"
)

var (
	watchPort    int
	watchAppPort int
	watchVerbose bool
)

func init() {
	watchCmd.Flags().IntVar(&watchPort, "port", 3000, "Development server port")
	watchCmd.Flags().IntVar(&watchAppPort, "app-port", 3001, "Application server port")
	watchCmd.Flags().BoolVar(&watchVerbose, "verbose", false, "Show verbose output")
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start development server with hot reload",
	Long: `Start the development server with automatic file watching and hot reload.

The watch command monitors your .cdt files for changes and automatically:
- Recompiles changed files incrementally
- Rebuilds the application binary
- Restarts the server
- Reloads connected browsers

Performance targets:
- File change detection: <10ms
- Incremental compile: <200ms
- Browser reload: <100ms
- Total (save to visible): <500ms

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
			Port:    watchPort,
			AppPort: watchAppPort,
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

		fmt.Println("\n📦 Conduit Development Server")
		fmt.Printf("   Dev server: http://localhost:%d\n", watchPort)
		fmt.Printf("   App server: http://localhost:%d\n", watchAppPort)
		fmt.Println("\n⌨️  Press Ctrl+C to stop\n")

		// Block until signal
		<-sigChan

		fmt.Println("\n\nShutting down...")

		// Stop dev server
		if err := devServer.Stop(); err != nil {
			return fmt.Errorf("error stopping dev server: %w", err)
		}

		fmt.Println("Goodbye!")
		return nil
	},
}
