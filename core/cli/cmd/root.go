package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// version stores the version string, set via SetVersion()
var version = "dev"

// SetVersion sets the version string (called from main.init())
func SetVersion(v string) {
	version = v
}

// GetVersion returns the current version string
func GetVersion() string {
	return version
}

var (
	configFile string
	source     string
	port       string
	logLevel   int
	verbose    bool
	logTags    string
	logFile    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "hyperterse",
	Short:         "Hyperterse\nConnect your data to your AI agents",
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged, suppress Cobra's error output
}

// completionCmd is a hidden command used by install.sh to generate shell completions
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for hyperterse.
This command is used internally by install.sh and is hidden from help.`,
	Hidden:       true,
	ValidArgs:    []string{"bash", "zsh", "fish", "powershell"},
	Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags are optional - not required for help, version, upgrade, or init commands
	rootCmd.PersistentFlags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.terse)")
	rootCmd.PersistentFlags().StringVarP(&source, "source", "s", "", "Configuration as a string (alternative to --file)")

	// Add flags that run command uses (for backward compatibility when using root command)
	rootCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	rootCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging (sets log level to DEBUG)")
	rootCmd.Flags().StringVar(&logTags, "log-tags", "", "Filter logs by tags (comma-separated, use -tag to exclude). Overrides HYPERTERSE_LOG_TAGS env var")
	rootCmd.Flags().BoolVar(&logFile, "log-file", false, "Stream logs to file in /tmp/.hyperterse/logs/")

	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Add hidden completion command for install.sh
	rootCmd.AddCommand(completionCmd)

	// Make root command run the server (backward compatibility)
	// Only require config flags when actually running the server, not for help/version
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Check for --version flag first (before checking for config)
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			printVersion()
			return nil
		}

		// Only require config when actually running the server
		if configFile == "" && source == "" {
			// If no subcommand and no flags, show help instead of error
			return cmd.Help()
		}
		return runServer(cmd, args)
	}
}

func printVersion() {
	v := GetVersion()
	if v == "dev" {
		// Try to get version from build info
		if info, ok := debug.ReadBuildInfo(); ok {
			// Check for vcs.revision or other version info
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
					v = setting.Value[:7]
				}
			}
			if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
				v = info.Main.Version
			}
		}
	}
	fmt.Println(v)
}

// LoadEnvFiles attempts to load .env files from multiple locations.
// It tries each location in order and stops at the first successful load.
// This ensures .env files work in development, when built, and when deployed.
// Priority order:
// 1. From the provided directory (if not empty)
// 2. From the current working directory
// 3. From the directory containing the executable binary
// System environment variables always take precedence over .env file values.
func LoadEnvFiles(fromDir string) {
	envFiles := []string{".env.local", ".env.development", ".env"}

	// Try loading from the provided directory first (e.g., config file directory)
	if fromDir != "" {
		for _, envFile := range envFiles {
			envPath := filepath.Join(fromDir, envFile)
			if err := godotenv.Load(envPath); err == nil {
				return // Successfully loaded, stop trying
			}
		}
	}

	// Try loading from current working directory
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			return // Successfully loaded, stop trying
		}
	}

	// Try loading from the directory containing the executable binary
	if execPath, err := os.Executable(); err == nil {
		if realPath, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = realPath
		}
		execDir := filepath.Dir(execPath)
		for _, envFile := range envFiles {
			envPath := filepath.Join(execDir, envFile)
			if err := godotenv.Load(envPath); err == nil {
				return // Successfully loaded, stop trying
			}
		}
	}
}
