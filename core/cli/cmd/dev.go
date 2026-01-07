package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/runtime"
	"github.com/spf13/cobra"
)

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the Hyperterse server in development mode with hot reload",
	Long: `Run the Hyperterse server in development mode with hot reload.
The server will automatically reload when the configuration file changes.`,
	RunE: runDevServer,
}

func init() {
	rootCmd.AddCommand(devCmd)

	// Use the same flags as root command (they're defined in root.go)
	devCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	devCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	devCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (sets log level to DEBUG)")
}

func runDevServer(cmd *cobra.Command, args []string) error {
	// Load initial config
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

	log.Println("Starting runtime in dev mode")
	log.Println("Runtime initialization")
	rt, err := runtime.NewRuntime(model, resolvedPort)
	if err != nil {
		log.PrintError("Failed to create runtime", err)
		os.Exit(1)
	}
	log.PrintSuccess("Runtime initialized")

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Println("Starting runtime server")
		if err := rt.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.PrintError("Failed to create file watcher", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Watch the config file
	err = watcher.Add(configFile)
	if err != nil {
		log.PrintError("Failed to watch config file", err)
		os.Exit(1)
	}

	log.Printf("Watching %s for changes...", configFile)

	// Debounce timer
	var reloadTimer *time.Timer

	// Watch for file changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Debounce: wait 500ms after last write before reloading
				if reloadTimer != nil {
					reloadTimer.Stop()
				}
				reloadTimer = time.AfterFunc(500*time.Millisecond, func() {
					log.Println("Config file changed, reloading...")

					// Reload config
					newModel, err := internal.LoadConfig(configFile)
					if err != nil {
						log.PrintError("Error reloading config", err)
						return
					}

					// Validate new model
					if err := parser.Validate(newModel); err != nil {
						if validationErr, ok := err.(*parser.ValidationErrors); ok {
							log.PrintValidationErrors(validationErr.Errors)
						} else {
							log.PrintError("Validation Error", err)
						}
						return
					}

					// Hot reload
					if err := rt.ReloadModel(newModel); err != nil {
						log.PrintError("Failed to reload model", err)
						return
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.PrintError("File watcher error", err)
		case err := <-serverErr:
			log.PrintError("Runtime error", err)
			os.Exit(1)
		}
	}
}
