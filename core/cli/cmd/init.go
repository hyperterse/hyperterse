package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Use:          "init [folder-name]",
	Short:        "Initialize a new Hyperterse configuration file",
	RunE:         runInit,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initOutputFile, "output", "o", "", "Output file path for the configuration (default: <folder>/.hyperterse)")
}

func runInit(cmd *cobra.Command, args []string) error {
	var folderName string
	if len(args) > 0 {
		folderName = strings.TrimSpace(args[0])
	}
	if folderName == "" {
		fmt.Fprint(os.Stderr, "Project folder name: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		folderName = strings.TrimSpace(line)
		if folderName == "" {
			return fmt.Errorf("folder name is required. Use: hyperterse init <folder-name>")
		}
	}

	baseDir := folderName
	if baseDir == "." {
		baseDir = ""
	}
	if baseDir != "" {
		baseDir = filepath.Clean(baseDir)
	}

	outputPath := initOutputFile
	if outputPath == "" {
		if baseDir != "" {
			outputPath = filepath.Join(baseDir, ".hyperterse")
		} else {
			outputPath = ".hyperterse"
		}
	} else if baseDir != "" && !filepath.IsAbs(outputPath) && filepath.Dir(outputPath) == "." {
		outputPath = filepath.Join(baseDir, outputPath)
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file '%s' already exists. Use a different filename or remove the existing file", outputPath)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Generate the config content
	configContent := generateConfigTemplate()

	// Write the file
	if err := os.WriteFile(outputPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	scaffoldBase := filepath.Dir(outputPath)
	appAdaptersDir := filepath.Join(scaffoldBase, scaffoldRootDir, scaffoldAdaptersDir)
	appToolDir := filepath.Join(scaffoldBase, scaffoldRootDir, scaffoldToolsDir, "hello-world")
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

	displayBase := scaffoldBase
	if displayBase == "." {
		displayBase = ""
	}
	relAdapters := filepath.Join(scaffoldRootDir, scaffoldAdaptersDir, "my-database.terse")
	relToolConfig := filepath.Join(scaffoldRootDir, scaffoldToolsDir, "hello-world", "config.terse")
	if displayBase != "" {
		relAdapters = filepath.Join(displayBase, relAdapters)
		relToolConfig = filepath.Join(displayBase, relToolConfig)
	}

	fmt.Printf("✓ Created configuration file: %s\n", outputPath)
	fmt.Printf("✓ Created adapter config: %s\n", relAdapters)
	fmt.Printf("✓ Created tool config: %s\n", relToolConfig)
	editPath := filepath.Join(scaffoldBase, scaffoldRootDir)
	if editPath == filepath.Join(".", scaffoldRootDir) {
		editPath = scaffoldRootDir
	}
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s and files under %s/%s + %s/%s\n", outputPath, editPath, scaffoldAdaptersDir, editPath, scaffoldToolsDir)
	fmt.Printf("  2. Run: hyperterse start -f %s\n", outputPath)

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
