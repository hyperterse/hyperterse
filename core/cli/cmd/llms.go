package cmd

import (
	"os"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/runtime/handlers"
	"github.com/spf13/cobra"
)

var (
	llmsOutput  string
	llmsBaseURL string
)

// llmsCmd represents the llms command
var llmsCmd = &cobra.Command{
	Use:   "llms",
	Short: "Generate llms.txt documentation file",
	Long: `Generate a complete llms.txt documentation file from your configuration.
This file contains markdown documentation describing all endpoints and queries.`,
	RunE: generateLLMs,
}

func init() {
	generateCmd.AddCommand(llmsCmd)

	llmsCmd.Flags().StringVarP(&llmsOutput, "output", "o", "llms.txt", "Output path for the llms.txt file")
	llmsCmd.Flags().StringVar(&llmsBaseURL, "base-url", "http://localhost:8080", "Base URL for the API endpoints")
}

func generateLLMs(cmd *cobra.Command, args []string) error {
	log := logger.New("generate")

	// Load config
	model, err := internal.LoadConfig(configFile)
	if err != nil {
		log.PrintError("Error loading config", err)
		os.Exit(1)
	}

	// Validate model
	if err := parser.Validate(model); err != nil {
		if validationErr, ok := err.(*parser.ValidationErrors); ok {
			log.PrintValidationErrors(validationErr.Errors)
		} else {
			log.PrintError("Validation Error", err)
		}
		os.Exit(1)
	}

	// Generate documentation
	doc := handlers.GenerateLLMDocumentation(model, llmsBaseURL)

	// Write to file
	if err := os.WriteFile(llmsOutput, []byte(doc), 0644); err != nil {
		log.PrintError("Failed to write llms.txt", err)
		os.Exit(1)
	}

	log.PrintSuccess("llms.txt generated: " + llmsOutput)
	return nil
}
