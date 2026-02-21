package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime"
	"github.com/hyperterse/hyperterse/core/types"
	"github.com/spf13/cobra"
)

var watch bool

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:           "start [path]",
	Short:         "Run the Hyperterse server",
	RunE:          startServer,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged, suppress Cobra's error output
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Start command runtime flags
	startCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.hyperterse or .terse)")
	startCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	startCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	startCmd.Flags().BoolVarP(&verbose, "verbose", "", false, "Enable verbose logging (sets log level to DEBUG)")
	startCmd.Flags().StringVarP(&source, "source", "s", "", "Configuration as a string (alternative to --file)")
	startCmd.Flags().StringVar(&logTags, "log-tags", "", "Filter logs by tags (comma-separated, use -tag to exclude). Overrides HYPERTERSE_LOG_TAGS env var")
	startCmd.Flags().BoolVar(&logFile, "log-file", false, "Stream logs to file in /tmp/.hyperterse/logs/")
	startCmd.Flags().BoolVar(&watch, "watch", false, "Watch .terse/.ts files and hot-reload on changes")
}

func startServer(cmd *cobra.Command, args []string) error {
	if err := resolveStartPathArg(args); err != nil {
		return err
	}
	if watch {
		return startServerWithWatch()
	}
	rt, err := PrepareRuntime()
	if err != nil {
		return err
	}
	return rt.Start()
}

func resolveStartPathArg(args []string) error {
	log := logger.New("start")
	if len(args) == 0 {
		return nil
	}
	if source != "" {
		return log.Errorf("cannot combine path argument with --source")
	}
	if configFile != "" {
		return log.Errorf("cannot combine path argument with --file")
	}

	target := args[0]
	info, err := os.Stat(target)
	if err != nil {
		return log.Errorf("invalid start path %q: %w", target, err)
	}

	if info.IsDir() {
		configFile = filepath.Join(target, ".hyperterse")
		return nil
	}

	// Allow directly passing a config file path.
	configFile = target
	return nil
}

func startServerWithWatch() error {
	log := logger.New("watch")

	if source != "" {
		return log.Errorf("--watch does not support --source; use a file-based project")
	}

	configPath := configFile
	if configPath == "" {
		configPath = ".hyperterse"
		configFile = configPath
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(configPath); err != nil {
		return err
	}
	if err := addAppWatches(watcher, filepath.Join(filepath.Dir(configPath), "app")); err != nil {
		return err
	}

	restart := make(chan struct{}, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = watcher.Add(event.Name)
						continue
					}
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 && shouldTriggerReload(event.Name) {
					if debounce != nil {
						debounce.Stop()
					}
					debounce = time.AfterFunc(500*time.Millisecond, func() {
						select {
						case restart <- struct{}{}:
						default:
						}
					})
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	log.Infof("Watching %s and app/** for changes", configPath)

	rt, err := PrepareRuntime()
	if err != nil {
		return err
	}
	if err := rt.StartAsync(); err != nil {
		return err
	}

	for {
		select {
		case <-sigChan:
			return rt.Stop()
		case <-restart:
			log.Infof("Changes detected, reloading")
			newRt, err := PrepareRuntime()
			if err != nil {
				log.Warnf("Reload failed, keeping current server running: %v", err)
				continue
			}
			if err := rt.Stop(); err != nil {
				return log.Errorf("failed to stop server for reload: %w", err)
			}
			if err := newRt.StartAsync(); err != nil {
				return log.Errorf("failed to start new server: %w", err)
			}
			rt = newRt
			log.Infof("Server reloaded successfully")
		}
	}
}

func addAppWatches(watcher *fsnotify.Watcher, appDir string) error {
	info, err := os.Stat(appDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(appDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func shouldTriggerReload(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".terse") || strings.HasSuffix(lower, ".ts")
}

// PrepareRuntime loads config, validates, and creates a runtime ready to start
func PrepareRuntime() (*runtime.Runtime, error) {
	log := logger.New("main")
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
			return nil, log.Errorf("failed to initialize log file: %w", err)
		}
	}

	// Load .env files from config file directory if a config file is provided
	// This allows .env files to be placed next to the config file
	if configFile != "" {
		if configDir := filepath.Dir(configFile); configDir != "" && configDir != "." {
			LoadEnvFiles(configDir)
		}
	} else {
		// Default start behavior: run from current directory using ./.hyperterse
		configFile = ".hyperterse"
		LoadEnvFiles(".")
	}

	var model *hyperterse.Model
	var project *framework.Project
	var err error

	// Load from source string if provided, otherwise from file
	if source != "" {
		if configFile != "" {
			return nil, log.Errorf("cannot specify both --file and --source flags")
		}
		model, err = internal.LoadConfigFromString(source)
	} else {
		model, project, err = internal.LoadConfigWithProject(configFile)
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

	log.Debugf("Tools: %d", len(model.Tools))
	if len(model.Tools) > 0 {
		for _, tool := range model.Tools {
			log.Debugf("  Tool: %s", tool.Name)
			if len(tool.Use) > 0 {
				log.Debugf("    Uses: %s", strings.Join(tool.Use, ", "))
			}
			if len(tool.Inputs) > 0 {
				inputNames := make([]string, 0, len(tool.Inputs))
				for _, input := range tool.Inputs {
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

	if project != nil {
		if err := framework.ValidateModel(model, project); err != nil {
			return nil, err
		}
		if err := framework.BundleTools(project); err != nil {
			return nil, err
		}
	} else {
		if err := parser.Validate(model); err != nil {
			return nil, err
		}
	}
	log.Infof("Validation successful")

	rt, err := runtime.NewRuntime(model, resolvedPort, GetVersion(), runtime.WithProject(project))
	if err != nil {
		return nil, err
	}
	log.Infof("Runtime initialized")

	return rt, nil
}
