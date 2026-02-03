package main

import (
	"os"

	"github.com/hyperterse/hyperterse/core/cli"
	"github.com/hyperterse/hyperterse/core/cli/cmd"
)

// Version can be set at build time using -ldflags
var Version = "dev"

func init() {
	// Set the version in cmd package so it can be accessed by commands
	cmd.SetVersion(Version)

	// Load .env files from multiple locations:
	// 1. Current working directory (for development)
	// 2. Directory containing the executable binary (for built binaries)
	// System environment variables always take precedence.
	cmd.LoadEnvFiles("")
}

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
