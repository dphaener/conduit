package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the Conduit compiler version, Git commit, build date, and Go version",
	Run: func(cmd *cobra.Command, args []string) {
		// Set GoVersion to actual runtime if not set at build time
		goVer := GoVersion
		if goVer == "unknown" {
			goVer = runtime.Version()
		}

		fmt.Printf("Conduit version: %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Go version: %s\n", goVer)
	},
}
