package main

import (
	"os"

	"github.com/hyperterse/hyperterse/core/cli"
	"github.com/hyperterse/hyperterse/core/cli/cmd"
	"github.com/joho/godotenv"
)

// Version can be set at build time using -ldflags
var Version = "dev"

func init() {
	// Set the version in cmd package so it can be accessed by commands
	cmd.SetVersion(Version)

	// Load .env file if it exists (ignore errors if file doesn't exist)
	// This allows users to use .env files without needing to manually source them
	_ = godotenv.Load(".env.local", ".env.development", ".env")
}

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
