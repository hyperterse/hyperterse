package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/runtime"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Hyperterse server",
	Long: `Run the Hyperterse server with the specified configuration file.
The server will start and listen for incoming requests.`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Use the same flags as root command (they're defined in root.go)
	runCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	runCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (sets log level to DEBUG)")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load config
	model, err := internal.LoadConfig(configFile)
	if err != nil {
		log := logger.New("main")
		log.PrintError("Error loading config", err)
		os.Exit(1)
	}

	// Resolve port and log level
	resolvedPort := internal.ResolvePort(port, model)
	resolvedLogLevel := internal.ResolveLogLevel(verbose, logLevel, model)

	// Set log level
	logger.SetLogLevel(resolvedLogLevel)
	log := logger.New("main")

	// Log parsed adapters
	log.Println("Parsed Configuration:")
	if len(model.Adapters) > 0 {
		log.Println("\tAdapters:")
		for _, adapter := range model.Adapters {
			log.Printf("\t  - Name: %s, Connector: %s", adapter.Name, adapter.Connector.String())
		}
		log.Println("")
	} else {
		log.Println("\tAdapters: (none)")
		log.Println("")
	}

	// Log parsed queries
	if len(model.Queries) > 0 {
		log.Println("\tQueries:")
		for _, query := range model.Queries {
			log.Printf("\t  - Name: %s", query.Name)
			if query.Description != "" {
				log.Printf("\t    Description: %s", query.Description)
			}
			if len(query.Use) > 0 {
				log.Printf("\t    Uses: %s", strings.Join(query.Use, ", "))
			}
			if len(query.Inputs) > 0 {
				inputNames := make([]string, 0, len(query.Inputs))
				for _, input := range query.Inputs {
					optional := ""
					if input.Optional {
						optional = " (optional)"
					}
					inputNames = append(inputNames, fmt.Sprintf("%s:%s%s", input.Name, input.Type, optional))
				}
				log.Printf("\t    Inputs: %s", strings.Join(inputNames, ", "))
			} else {
				log.Printf("\t    Inputs: (none)")
			}
			if len(query.Data) > 0 {
				dataNames := make([]string, 0, len(query.Data))
				for _, data := range query.Data {
					dataNames = append(dataNames, fmt.Sprintf("%s:%s", data.Name, data.Type))
				}
				log.Printf("\t    Outputs: %s", strings.Join(dataNames, ", "))
			} else {
				log.Printf("\t    Outputs: (none)")
			}
		}
		log.Println("")
	} else {
		log.Println("\tQueries: (none)")
		log.Println("")
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

	log.PrintSuccess("Validation successful!")

	log.Println("Starting runtime")
	log.Println("Runtime initialization")
	rt, err := runtime.NewRuntime(model, resolvedPort)
	if err != nil {
		log.PrintError("Failed to create runtime", err)
		os.Exit(1)
	}
	log.PrintSuccess("Runtime initialized")

	log.Println("Starting runtime server")
	if err := rt.Start(); err != nil {
		log.PrintError("Runtime error", err)
		os.Exit(1)
	}
	log.PrintSuccess("Runtime server started")

	return nil
}
