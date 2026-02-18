package cmd

import (
	"os"
	"path/filepath"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// serveCmd runs a precompiled model manifest created by `hyperterse build`.
var serveCmd = &cobra.Command{
	Use:           "serve [manifest-or-dir]",
	Short:         "Serve a built Hyperterse model manifest",
	RunE:          serveModelManifest,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the model manifest file (default: ./model.bin)")
	serveCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides manifest and PORT env var)")
	serveCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides manifest)")
	serveCmd.Flags().BoolVarP(&verbose, "verbose", "", false, "Enable verbose logging (sets log level to DEBUG)")
	serveCmd.Flags().StringVar(&logTags, "log-tags", "", "Filter logs by tags (comma-separated, use -tag to exclude). Overrides HYPERTERSE_LOG_TAGS env var")
	serveCmd.Flags().BoolVar(&logFile, "log-file", false, "Stream logs to file in /tmp/.hyperterse/logs/")
}

func serveModelManifest(cmd *cobra.Command, args []string) error {
	manifestPath, err := resolveServeManifestPath(args)
	if err != nil {
		return err
	}

	rt, err := prepareRuntimeFromManifest(manifestPath)
	if err != nil {
		return err
	}
	return rt.Start()
}

func resolveServeManifestPath(args []string) (string, error) {
	log := logger.New("serve")

	var manifestPath string
	if len(args) > 0 {
		if configFile != "" {
			return "", log.Errorf("cannot combine path argument with --file")
		}
		manifestPath = args[0]
	} else if configFile != "" {
		manifestPath = configFile
	} else {
		manifestPath = modelManifestFileName
	}

	if info, err := os.Stat(manifestPath); err == nil && info.IsDir() {
		manifestPath = filepath.Join(manifestPath, modelManifestFileName)
	}

	absManifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", log.Errorf("invalid manifest path %q: %w", manifestPath, err)
	}
	return absManifestPath, nil
}

func prepareRuntimeFromManifest(manifestPath string) (*runtime.Runtime, error) {
	log := logger.New("main")

	// Configure logger before loading manifest for consistent startup logs.
	if verbose {
		logger.SetLogLevel(logger.LogLevelDebug)
	} else if logLevel > 0 {
		logger.SetLogLevel(logLevel)
	} else {
		logger.SetLogLevel(logger.LogLevelInfo)
	}

	tagFilterStr := logTags
	if tagFilterStr == "" {
		tagFilterStr = os.Getenv("HYPERTERSE_LOG_TAGS")
	}
	if tagFilterStr != "" {
		logger.SetTagFilter(tagFilterStr)
	}

	if logFile {
		filePath, err := logger.SetLogFile()
		if err != nil {
			return nil, log.Errorf("failed to initialize log file: %w", err)
		}
		log.Infof("Log file: %s", filePath)
	}

	LoadEnvFiles(filepath.Dir(manifestPath))

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, log.Errorf("failed to read manifest %s: %w", manifestPath, err)
	}

	model := &hyperterse.Model{}
	if err := proto.Unmarshal(manifestBytes, model); err != nil {
		return nil, log.Errorf("failed to decode model manifest %s: %w", manifestPath, err)
	}
	project, err := framework.ProjectFromManifestModel(model, manifestPath)
	if err != nil {
		return nil, log.Errorf("failed to rebuild framework project from manifest: %w", err)
	}

	resolvedPort := internal.ResolvePort(port, model)
	resolvedLogLevel := internal.ResolveLogLevel(verbose, logLevel, model)
	if logLevel == 0 && !verbose {
		logger.SetLogLevel(resolvedLogLevel)
	}

	log.Infof("Manifest loaded")
	if project != nil {
		if err := framework.ValidateModel(model, project); err != nil {
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
