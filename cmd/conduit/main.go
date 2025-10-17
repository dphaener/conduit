package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information - will be set at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "conduit",
		Short: "Conduit programming language compiler and tooling",
		Long: `Conduit is an LLM-first programming language for building web applications.
It compiles to Go and provides explicit syntax optimized for AI-assisted development.`,
	}

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(migrateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
