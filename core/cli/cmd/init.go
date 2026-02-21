package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initOutputFile string
)

const (
	scaffoldRootDir     = "app"
	scaffoldToolsDir    = "tools"
	scaffoldAdaptersDir = "adapters"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize a new Hyperterse configuration file",
	RunE:         runInit,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initOutputFile, "output", "o", ".hyperterse", "Output file path for the configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if file already exists
	if _, err := os.Stat(initOutputFile); err == nil {
		return fmt.Errorf("file '%s' already exists. Use a different filename or remove the existing file", initOutputFile)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(initOutputFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Generate the config content
	configContent := generateConfigTemplate()

	// Write the file
	if err := os.WriteFile(initOutputFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	baseDir := filepath.Dir(initOutputFile)
	appAdaptersDir := filepath.Join(baseDir, scaffoldRootDir, scaffoldAdaptersDir)
	appToolDir := filepath.Join(baseDir, scaffoldRootDir, scaffoldToolsDir, "hello-world")
	if err := os.MkdirAll(appAdaptersDir, 0755); err != nil {
		return fmt.Errorf("failed to create app adapters directory: %w", err)
	}
	if err := os.MkdirAll(appToolDir, 0755); err != nil {
		return fmt.Errorf("failed to create app tool directory: %w", err)
	}

	adapterConfig := `connector: postgres
connection_string: "postgresql://user:password@localhost:5432/dbname?sslmode=disable"
options:
  max_connections: "10"
`
	if err := os.WriteFile(filepath.Join(appAdaptersDir, "my-database.terse"), []byte(adapterConfig), 0644); err != nil {
		return fmt.Errorf("failed to write adapter .terse: %w", err)
	}

	toolConfig := `description: "Hello world tool"
use: my-database
statement: |
  SELECT first_name FROM users WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "User ID provided by the agent."
mappers:
  output: "user-data-mapper.ts"
auth:
  plugin: allow_all
`
	if err := os.WriteFile(filepath.Join(appToolDir, "config.terse"), []byte(toolConfig), 0644); err != nil {
		return fmt.Errorf("failed to write tool config.terse: %w", err)
	}

	handlerTS := `type Row = Record<string, unknown>;

export default async function outputTransform(payload: { results?: Row[] }) {
  const row = payload?.results?.[0] ?? {};
  const name = String(row.first_name ?? "there");
  return ` + "`Hello ${name}!`" + `;
}
`
	if err := os.WriteFile(filepath.Join(appToolDir, "user-data-mapper.ts"), []byte(handlerTS), 0644); err != nil {
		return fmt.Errorf("failed to write tool user-data-mapper.ts: %w", err)
	}

	fmt.Printf("✓ Created configuration file: %s\n", initOutputFile)
	fmt.Printf("✓ Created adapter config: %s\n", filepath.Join(scaffoldRootDir, scaffoldAdaptersDir, "my-database.terse"))
	fmt.Printf("✓ Created tool config: %s\n", filepath.Join(scaffoldRootDir, scaffoldToolsDir, "hello-world", "config.terse"))
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s and files under %s/%s + %s/%s\n", initOutputFile, scaffoldRootDir, scaffoldAdaptersDir, scaffoldRootDir, scaffoldToolsDir)
	fmt.Printf("  2. Run: hyperterse start -f %s\n", initOutputFile)

	return nil
}

func generateConfigTemplate() string {
	return `name: my-service
version: 1.0.0

root: app

tools:
  directory: tools

adapters:
  directory: adapters

server:
  port: 8080
  log_level: 3
`
}
