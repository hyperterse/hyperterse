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

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize a new Hyperterse configuration file",
	RunE:         runInit,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initOutputFile, "output", "o", "config.terse", "Output file path for the configuration")
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

	fmt.Printf("✓ Created configuration file: %s\n", initOutputFile)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to configure your adapters and queries\n", initOutputFile)
	fmt.Printf("  2. Run: hyperterse -f %s\n", initOutputFile)

	return nil
}

func generateConfigTemplate() string {
	return `server:
  port: 8080
  log_level: 3

adapters:
  my_database:
    connector: postgres
    connection_string: "postgresql://user:password@localhost:5432/dbname?sslmode=disable"
    options:
      max_connections: "10"

queries:
  get_user_by_id:
    use: my_database
    description: "Get a user by their ID"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = {{ inputs.user_id }}
    inputs:
      user_id:
        type: int
        description: "The ID of the user to retrieve"
`
}
