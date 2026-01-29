package cmd

import (
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/spf13/cobra"
)

var (
	exportOutputDir string
	exportCleanDir  bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:           "export",
	Short:         "Export a portable runtime bundle",
	RunE:          exportBundle,
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged, suppress Cobra's error output
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.terse)")
	exportCmd.Flags().StringVarP(&exportOutputDir, "out", "o", "", "Output directory for the script file (default: dist)")
	exportCmd.Flags().BoolVar(&exportCleanDir, "clean-dir", false, "Clean output directory before exporting")
	exportCmd.MarkFlagRequired("file")
}

func exportBundle(cmd *cobra.Command, args []string) error {
	log := logger.New("export")
	if configFile == "" {
		log.Errorf("Please provide a file path using -f or --file")
		os.Exit(1)
	}

	// Read config file
	configContent, err := os.ReadFile(configFile)
	if err != nil {
		log.Errorf("Error reading config file: %v", err)
		os.Exit(1)
	}

	// Load and validate config to get name and export settings
	model, err := internal.LoadConfig(configFile)
	if err != nil {
		log.Errorf("Error loading config: %v", err)
		os.Exit(1)
	}

	// Validate name is present
	if model.Name == "" {
		log.Errorf("Config name is required")
		os.Exit(1)
	}

	// Determine output directory (CLI flag takes precedence over config, then default)
	var outputDir string
	if exportOutputDir != "" {
		// Using --out/-o flag
		outputDir = exportOutputDir
	} else if model.Export != nil && model.Export.Out != "" {
		// Use config export.out setting
		outputDir = model.Export.Out
	} else {
		// Default: dist directory
		outputDir = "dist"
	}

	// Determine cleanDir setting (CLI flag takes precedence over config)
	cleanDir := exportCleanDir
	if !cleanDir && model.Export != nil {
		cleanDir = model.Export.CleanDir
	}

	// Clean directory if requested
	if cleanDir {
		if err := cleanDirectory(log, outputDir); err != nil {
			os.Exit(1)
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Errorf("Error creating output directory: %v", err)
		os.Exit(1)
	}

	// Script filename always uses config name
	scriptPath := filepath.Join(outputDir, model.Name)

	// Find the hyperterse binary
	binaryPath, err := findBinary(log)
	if err != nil {
		os.Exit(1)
	}

	binaryContent, err := os.ReadFile(binaryPath)
	if err != nil {
		log.Errorf("Error reading binary: %v", err)
		os.Exit(1)
	}

	// Generate bash script content
	scriptContent := generateBashScript(configContent, binaryContent)

	// Write script to file
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		log.Errorf("Error writing script file: %v", err)
		os.Exit(1)
	}

	log.Successf("Exported script to ./%s", scriptPath)
	log.Successf("Run: ./%s", scriptPath)

	return nil
}

func findBinary(log *logger.Logger) (string, error) {
	// First, try to find the binary in dist/hyperterse
	distPath := "dist/hyperterse"
	if _, err := os.Stat(distPath); err == nil {
		return distPath, nil
	}

	// Try to get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		log.Errorf("Could not determine executable path: %v", err)
		os.Exit(1)
	}

	// Resolve symlinks to get the actual path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err == nil {
		execPath = realPath
	}

	// Check if the executable exists and is readable
	if _, err := os.Stat(execPath); err != nil {
		log.Errorf("Executable not found at %s: %v", execPath, err)
		os.Exit(1)
	}

	return execPath, nil
}

// cleanDirectory removes all contents of a directory but keeps the directory itself
func cleanDirectory(log *logger.Logger, dirPath string) error {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean
		return nil
	}
	if err != nil {
		log.Errorf("Error checking directory %s: %v", dirPath, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		log.Errorf("Path %s is not a directory", dirPath)
		os.Exit(1)
	}

	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Errorf("Error reading directory %s: %v", dirPath, err)
		os.Exit(1)
	}

	// Remove all entries
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			log.Errorf("Error removing %s: %v", entryPath, err)
			os.Exit(1)
		}
	}

	return nil
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
