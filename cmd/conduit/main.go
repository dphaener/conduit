package main

import (
	"os"

	"github.com/conduit-lang/conduit/internal/cli/commands"
)

var (
	// Version information - will be set at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = "unknown"
)

func main() {
	// Set version info in commands package
	commands.Version = Version
	commands.GitCommit = GitCommit
	commands.BuildDate = BuildDate
	commands.GoVersion = GoVersion

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
