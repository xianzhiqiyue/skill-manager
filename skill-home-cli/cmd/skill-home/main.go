package main

import (
	"fmt"
	"os"

	"github.com/skill-home/cli/internal/cmd"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	rootCmd := cmd.NewRootCmd(version, commit, buildDate)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
