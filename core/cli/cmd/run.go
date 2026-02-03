package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime"
	"github.com/hyperterse/hyperterse/core/types"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Run the Hyperterse server",
	RunE:          runServer,
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged, suppress Cobra's error output
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Use the same flags as root command (they're defined in root.go)
	runCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	runCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "", false, "Enable verbose logging (sets log level to DEBUG)")
	runCmd.Flags().StringVarP(&source, "source", "s", "", "YAML configuration as a string (alternative to --file)")
	runCmd.Flags().StringVar(&logTags, "log-tags", "", "Filter logs by tags (comma-separated, use -tag to exclude). Overrides HYPERTERSE_LOG_TAGS env var")
	runCmd.Flags().BoolVar(&logFile, "log-file", false, "Stream logs to file in /tmp/.hyperterse/logs/")
}

func runServer(cmd *cobra.Command, args []string) error {
	rt, err := PrepareRuntime()
	if err != nil {
		return err
	}
	return rt.Start()
}

// PrepareRuntime loads config, validates, and creates a runtime ready to start
func PrepareRuntime() (*runtime.Runtime, error) {
	// Set log level early based on CLI flags (before loading config)
	// This ensures logs during config loading respect the log level
	// We'll update it after loading config if config file specifies a different level
	if verbose {
		logger.SetLogLevel(logger.LogLevelDebug)
	} else if logLevel > 0 {
		logger.SetLogLevel(logLevel)
	} else {
		// Default to INFO if no CLI flag specified (will be updated from config if needed)
		logger.SetLogLevel(logger.LogLevelInfo)
	}

	// Initialize tag filtering early (CLI flag takes precedence over env var)
	tagFilterStr := logTags
	if tagFilterStr == "" {
		tagFilterStr = os.Getenv("HYPERTERSE_LOG_TAGS")
	}
	if tagFilterStr != "" {
		logger.SetTagFilter(tagFilterStr)
	}

	// Initialize log file streaming if enabled
	var filePath string
	if logFile {
		var err error
		filePath, err = logger.SetLogFile()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize log file: %w", err)
		}
	}

	// Load .env files from config file directory if a config file is provided
	// This allows .env files to be placed next to the config file
	if configFile != "" {
		if configDir := filepath.Dir(configFile); configDir != "" && configDir != "." {
			LoadEnvFiles(configDir)
		}
	}

	var model *hyperterse.Model
	var err error

	// Load from source string if provided, otherwise from file
	if source != "" {
		if configFile != "" {
			return nil, fmt.Errorf("cannot specify both --file and --source flags")
		}
		model, err = internal.LoadConfigFromString(source)
	} else {
		if configFile == "" {
			return nil, fmt.Errorf("please provide a file path using -f or --file, or a source string using -s or --source")
		}
		model, err = internal.LoadConfig(configFile)
	}
	if err != nil {
		return nil, err
	}

	resolvedPort := internal.ResolvePort(port, model)
	resolvedLogLevel := internal.ResolveLogLevel(verbose, logLevel, model)
	// Update log level if config file specifies a different level and no CLI flag was provided
	if logLevel == 0 && !verbose {
		logger.SetLogLevel(resolvedLogLevel)
	}

	log := logger.New("main")

	// Log file path if streaming is enabled
	if logFile {
		log.Infof("Log file: %s", filePath)
	}

	// Log parsed configuration
	log.Infof("Configuration loaded")
	log.Debugf("Adapters: %d", len(model.Adapters))
	if len(model.Adapters) > 0 {
		for _, adapter := range model.Adapters {
			log.Debugf("  Adapter: %s (%s)", adapter.Name, adapter.Connector.String())
		}
	}

	log.Debugf("Queries: %d", len(model.Queries))
	if len(model.Queries) > 0 {
		for _, query := range model.Queries {
			log.Debugf("  Query: %s", query.Name)
			if len(query.Use) > 0 {
				log.Debugf("    Uses: %s", strings.Join(query.Use, ", "))
			}
			if len(query.Inputs) > 0 {
				inputNames := make([]string, 0, len(query.Inputs))
				for _, input := range query.Inputs {
					optional := ""
					if input.Optional {
						optional = " (optional)"
					}
					typeStr := types.PrimitiveEnumToString(input.Type)
					inputNames = append(inputNames, fmt.Sprintf("%s:%s%s", input.Name, typeStr, optional))
				}
				log.Debugf("    Inputs: %s", strings.Join(inputNames, ", "))
			}
		}
	}

	if err := parser.Validate(model); err != nil {
		// Validation errors are already logged by the validator
		// Just return the error to exit the program
		return nil, err
	}
	log.Infof("Validation successful")

	rt, err := runtime.NewRuntime(model, resolvedPort)
	if err != nil {
		return nil, err
	}
	log.Infof("Runtime initialized")

	return rt, nil
}
