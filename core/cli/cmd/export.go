package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	exportOutputDir string
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:          "export",
	Short:        "Export a portable runtime bundle",
	RunE:         exportBundle,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.terse)")
	exportCmd.Flags().StringVarP(&exportOutputDir, "output", "o", "dist", "Output directory for the script file")
	exportCmd.MarkFlagRequired("file")
}

func exportBundle(cmd *cobra.Command, args []string) error {
	if configFile == "" {
		return fmt.Errorf("please provide a file path using -f or --file")
	}

	// Read config file
	configContent, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Find the hyperterse binary
	binaryPath, err := findBinary()
	if err != nil {
		return fmt.Errorf("error finding binary: %w", err)
	}

	binaryContent, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("error reading binary: %w", err)
	}

	// Get config file name without extension for script name
	configBaseName := strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile))
	scriptPath := filepath.Join(exportOutputDir, configBaseName)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(exportOutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Generate bash script content
	scriptContent := generateBashScript(configContent, binaryContent)

	// Write script to file
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("error writing script file: %w", err)
	}

	fmt.Printf("✓ Exported script to %s\n", scriptPath)
	fmt.Printf("  Run: %s\n", scriptPath)

	return nil
}

func findBinary() (string, error) {
	// First, try to find the binary in dist/hyperterse
	distPath := "dist/hyperterse"
	if _, err := os.Stat(distPath); err == nil {
		return distPath, nil
	}

	// Try to get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}

	// Resolve symlinks to get the actual path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err == nil {
		execPath = realPath
	}

	// Check if the executable exists and is readable
	if _, err := os.Stat(execPath); err != nil {
		return "", fmt.Errorf("executable not found at %s: %w", execPath, err)
	}

	return execPath, nil
}

func generateBashScript(configContent []byte, binaryContent []byte) string {
	// Base64 encode the config and binary
	configB64 := base64.StdEncoding.EncodeToString(configContent)
	binaryB64 := base64.StdEncoding.EncodeToString(binaryContent)

	// Generate bash script that extracts and runs
	// Base64 strings are safe to embed in double quotes (only contain A-Z, a-z, 0-9, +, /, =)
	script := `#!/bin/bash
set -e

# Extract embedded config and binary
CONFIG_B64="` + configB64 + `"
BINARY_B64="` + binaryB64 + `"

# Create temporary directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Decode config (try -d first, fallback to -D for older macOS)
if echo "dGVzdA==" | base64 -d >/dev/null 2>&1; then
	echo "$CONFIG_B64" | base64 -d > "$TMPDIR/config.terse"
	echo "$BINARY_B64" | base64 -d > "$TMPDIR/hyperterse"
else
	echo "$CONFIG_B64" | base64 -D > "$TMPDIR/config.terse"
	echo "$BINARY_B64" | base64 -D > "$TMPDIR/hyperterse"
fi

chmod +x "$TMPDIR/hyperterse"

# Run hyperterse with the embedded config
"$TMPDIR/hyperterse" run --file "$TMPDIR/config.terse" "$@"
`

	return script
}
