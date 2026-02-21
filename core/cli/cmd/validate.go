package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:           "validate [path]",
	Short:         "Validate a Hyperterse project or config",
	RunE:          validateConfig,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.hyperterse or .terse)")
	validateCmd.Flags().StringVarP(&source, "source", "s", "", "Configuration as a string (alternative to --file)")
}

func validateConfig(cmd *cobra.Command, args []string) error {
	log := logger.New("validate")
	if err := resolveValidatePathArg(args); err != nil {
		return err
	}

	var (
		model    *hyperterse.Model
		project  *framework.Project
		err      error
		loadFrom string
	)

	if source != "" {
		if configFile != "" {
			return log.Errorf("cannot specify both --file and --source flags")
		}
		model, err = internal.LoadConfigFromString(source)
		loadFrom = "source"
	} else {
		model, project, err = internal.LoadConfigWithProject(configFile)
		loadFrom = configFile
	}
	if err != nil {
		return err
	}

	if project != nil {
		if err := framework.ValidateModel(model, project); err != nil {
			return log.Errorf("validation failed: %w", err)
		}
		// Validate tool scripts/dependencies by running bundling in a temp directory.
		tempBuildRoot, err := os.MkdirTemp("", "hyperterse-validate-*")
		if err != nil {
			return log.Errorf("validation failed: unable to create temp build directory: %w", err)
		}
		defer os.RemoveAll(tempBuildRoot)

		project.BuildDir = filepath.Join(tempBuildRoot, "build")
		if err := framework.BundleTools(project); err != nil {
			return log.Errorf("validation failed: %w", err)
		}
	} else {
		if err := parser.Validate(model); err != nil {
			return log.Errorf("validation failed: %w", err)
		}
	}

	printValidationSummary(log, loadFrom, model, project)
	log.Successf("Configuration is valid: %s", loadFrom)
	return nil
}

func resolveValidatePathArg(args []string) error {
	log := logger.New("validate")
	if len(args) == 0 {
		if configFile == "" && source == "" {
			configFile = ".hyperterse"
		}
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
		return log.Errorf("invalid validate path %q: %w", target, err)
	}
	if info.IsDir() {
		configFile = filepath.Join(target, ".hyperterse")
		return nil
	}

	// Allow directly passing a config file path.
	configFile = target
	return nil
}

func printValidationSummary(log *logger.Logger, loadFrom string, model *hyperterse.Model, project *framework.Project) {
	log.Info("Validation report:")
	log.Infof("  root config: %s", loadFrom)

	if project == nil {
		log.Infof("  adapters: %d", len(model.GetAdapters()))
		log.Infof("  tools: %d", len(model.GetTools()))
		return
	}

	log.Infof("  app directory: %s", project.AppDir)
	log.Infof("  adapter directory: %s", project.AdaptersDir)
	log.Infof("  tools directory: %s", project.ToolsDir)

	adapterFiles := listTerseFiles(project.AdaptersDir)
	log.Infof("  adapter files (%d):", len(adapterFiles))
	if len(adapterFiles) == 0 {
		log.Info("    - none")
	} else {
		for _, adapterFile := range adapterFiles {
			log.Infof("    - %s", displayPath(project.BaseDir, adapterFile))
		}
	}

	toolNames := make([]string, 0, len(project.Tools))
	for toolName := range project.Tools {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	log.Infof("  tools (%d):", len(toolNames))
	if len(toolNames) == 0 {
		log.Info("    - none")
	}

	vendorValidated := "no"
	if project.VendorBundle != "" {
		if _, err := os.Stat(project.VendorBundle); err == nil {
			vendorValidated = "yes"
		}
	}
	totalBundles := 0

	for _, toolName := range toolNames {
		tool := project.Tools[toolName]
		if tool == nil {
			continue
		}

		log.Infof("    - tool: %s", tool.ToolName)
		log.Infof("      config: %s", displayPath(project.BaseDir, tool.TerseFile))
		if len(tool.Definition.GetUse()) > 0 {
			log.Infof("      adapters: %s", strings.Join(tool.Definition.GetUse(), ", "))
		} else {
			log.Info("      adapters: none (script-only tool)")
		}

		for _, scriptKind := range []string{"handler", "input_transform", "output_transform"} {
			scriptPath := scriptPathForKind(tool, scriptKind)
			if scriptPath == "" {
				continue
			}
			scriptLabel := strings.ReplaceAll(scriptKind, "_", " ")
			bundlePath := tool.BundleOutputs[scriptKind]
			if bundlePath != "" {
				totalBundles++
			}
			log.Infof(
				"      %s: %s -> %s",
				scriptLabel,
				displayPath(project.BaseDir, scriptPath),
				describeBundleOutput(bundlePath),
			)
		}
	}

	log.Infof("  vendor bundle validated: %s", vendorValidated)
	log.Infof("  tool bundles validated: %d", totalBundles)
}

func listTerseFiles(dir string) []string {
	entries := make([]string, 0)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return entries
	}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".terse") {
			entries = append(entries, path)
		}
		return nil
	})
	sort.Strings(entries)
	return entries
}

func scriptPathForKind(tool *framework.Tool, kind string) string {
	switch kind {
	case "handler":
		return tool.Scripts.Handler
	case "input_transform":
		return tool.Scripts.InputTransform
	case "output_transform":
		return tool.Scripts.OutputTransform
	default:
		return ""
	}
}

func describeBundleOutput(bundlePath string) string {
	if bundlePath == "" {
		return "not bundled"
	}
	return fmt.Sprintf("bundled as %s", filepath.Base(bundlePath))
}

func displayPath(baseDir, target string) string {
	if target == "" {
		return ""
	}
	cleanTarget := filepath.Clean(target)
	rel, err := filepath.Rel(baseDir, cleanTarget)
	if err != nil {
		return cleanTarget
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return cleanTarget
	}
	return rel
}
