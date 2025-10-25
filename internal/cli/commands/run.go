package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/cli/config"
)

var (
	runPort       int
	runHotReload  bool
	runBuildFirst bool
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build and run the application",
		Long: `Build the Conduit application and start the development server.

The run command will:
  1. Build your Conduit application
  2. Start the web server
  3. Watch for changes (if --hot-reload is enabled)

Examples:
  conduit run
  conduit run --port 8080
  conduit run --hot-reload
  conduit run --no-build`,
		RunE: runRun,
	}

	cmd.Flags().IntVarP(&runPort, "port", "p", 3000, "Port to run the server on")
	cmd.Flags().BoolVar(&runHotReload, "hot-reload", false, "Enable hot reload on file changes (stub)")
	cmd.Flags().BoolVar(&runBuildFirst, "build", true, "Build before running")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	warningColor := color.New(color.FgYellow)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		if buildVerbose {
			warningColor.Printf("Warning: %v\n", err)
		}
	}

	// Use config port if not overridden
	if cmd.Flags().Changed("port") == false && cfg != nil {
		runPort = cfg.Server.Port
	}

	// Determine output path
	outputPath := "build/app"
	if cfg != nil && cfg.Build.Output != "" {
		outputPath = cfg.Build.Output
	}

	// Build the application first (unless --no-build)
	if runBuildFirst {
		infoColor.Println("Building application...")

		buildCmd := NewBuildCommand()
		if err := buildCmd.RunE(buildCmd, []string{}); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	// Check if binary exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("%s not found - build may have failed", outputPath)
	}

	// Show hot reload status
	if runHotReload {
		warningColor.Println("\nNote: Hot reload is not yet implemented (coming in tooling milestone)")
		infoColor.Println("For now, restart 'conduit run' to see changes")
	}

	// Run the application
	successColor.Printf("Starting server on port %d...\n", runPort)
	infoColor.Printf("Server URL: http://localhost:%d\n\n", runPort)

	app := exec.Command("./"+outputPath)
	app.Stdout = os.Stdout
	app.Stderr = os.Stderr
	app.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", runPort))

	// Start the application
	if err := app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Handle Ctrl+C gracefully with timeout
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	shutdownComplete := make(chan struct{})
	go func() {
		<-sigChan
		infoColor.Println("\nShutting down server...")
		if app.Process != nil {
			// Send SIGTERM first for graceful shutdown
			if err := app.Process.Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, fall back to SIGKILL
				app.Process.Kill()
				close(shutdownComplete)
				return
			}

			// Wait for graceful shutdown with 10 second timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			done := make(chan struct{})
			go func() {
				app.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Graceful shutdown completed
				infoColor.Println("Server stopped gracefully")
			case <-ctx.Done():
				// Timeout - force kill
				warningColor.Println("Shutdown timeout - forcing termination")
				app.Process.Kill()
			}
		}
		close(shutdownComplete)
	}()

	// Wait for the application to finish or shutdown signal
	select {
	case <-shutdownComplete:
		// Shutdown was triggered and completed
		return nil
	default:
		// Wait for the application to finish (don't call os.Exit)
		if err := app.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Check for signal-based termination
				if exitErr.ExitCode() == -1 || strings.Contains(err.Error(), "signal") {
					// Process was killed by signal, this is expected
					return nil
				}
			}
			return fmt.Errorf("application exited with error: %w", err)
		}

		return nil
	}
}
