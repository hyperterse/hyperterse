package cmd

import (
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate artifacts from configuration",
	Long: `Generate various artifacts from your Hyperterse configuration file.
Available subcommands include 'skills' and 'llms'.`,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
