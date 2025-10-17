package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	runPort int
)

func init() {
	runCmd.Flags().IntVar(&runPort, "port", 3000, "Port to run the server on")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Build and run the application",
	Long:  "Build the Conduit application and start the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// First, build the application
		fmt.Println("Building application...")

		// Build the application by calling buildCmd.RunE directly
		if err := buildCmd.RunE(cmd, []string{}); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		// Check if binary exists
		if _, err := os.Stat("build/app"); os.IsNotExist(err) {
			return fmt.Errorf("build/app not found - build may have failed")
		}

		// Run the application
		fmt.Printf("\nStarting server on port %d...\n", runPort)

		app := exec.Command("./build/app")
		app.Stdout = os.Stdout
		app.Stderr = os.Stderr
		app.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", runPort))

		// Start the application
		if err := app.Start(); err != nil {
			return fmt.Errorf("failed to start application: %w", err)
		}

		// Handle Ctrl+C gracefully
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nShutting down server...")
			if app.Process != nil {
				app.Process.Kill()
			}
			os.Exit(0)
		}()

		// Wait for the application to finish
		if err := app.Wait(); err != nil {
			// Check if it was killed by signal
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == -1 {
					// Process was killed by signal, this is expected
					return nil
				}
			}
			return fmt.Errorf("application exited with error: %w", err)
		}

		return nil
	},
}
