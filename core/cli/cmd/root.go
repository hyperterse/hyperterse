package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	configFile  string
	source      string
	port        string
	logLevel    int
	verbose     bool
	logTags     string
	logFile     bool
	showVersion bool
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
	// Add hidden completion command for install.sh
	rootCmd.AddCommand(completionCmd)
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print the installed version and exit")

	// Root command should only print help.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		}
		return cmd.Help()
	}
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
