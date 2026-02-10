package cmd

import (
	"os"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:           "validate",
	Short:         "Validate a Hyperterse configuration file",
	RunE:          validateConfig,
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged by parser/validator
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&source, "source", "s", "", "Configuration as a string (alternative to --file)")
}

func validateConfig(cmd *cobra.Command, args []string) error {
	log := logger.New("validate")

	var (
		model    *hyperterse.Model
		err      error
		loadFrom string
	)

	if source != "" {
		if configFile != "" {
			log.Errorf("cannot specify both --file and --source flags")
			os.Exit(1)
		}
		model, err = internal.LoadConfigFromString(source)
		loadFrom = "source"
	} else {
		if configFile == "" {
			log.Errorf("please provide a file path using -f or --file, or a source string using -s or --source")
			os.Exit(1)
		}
		model, err = internal.LoadConfig(configFile)
		loadFrom = configFile
	}
	if err != nil {
		return err
	}

	if err := parser.Validate(model); err != nil {
		log.Errorf("validation failed: %v", err)
		os.Exit(1)
	}

	log.Successf("Configuration is valid: %s", loadFrom)
	return nil
}
