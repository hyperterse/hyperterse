package cmd

import (
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:          "generate",
	Short:        "Generate artifacts from configuration",
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
