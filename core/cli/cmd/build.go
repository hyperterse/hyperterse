package cmd

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var (
	buildOutputDir string
	buildCleanDir  bool
)

const modelManifestFileName = "model.bin"
const legacyModelManifestFileName = "hyperterse.model.bin"

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:           "build [path]",
	Short:         "Build bundles and a model manifest",
	RunE:          buildBundle,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true, // Errors are already logged, suppress Cobra's error output
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&configFile, "file", "f", "", "Path to the configuration file (.hyperterse or .terse)")
	buildCmd.Flags().StringVarP(&buildOutputDir, "out", "o", "", "Output directory for build artifacts (default: dist)")
	buildCmd.Flags().BoolVar(&buildCleanDir, "clean-dir", false, "Clean output directory before building")
}

func buildBundle(cmd *cobra.Command, args []string) error {
	log := logger.New("build")
	if err := resolveBuildPathArg(args); err != nil {
		return err
	}

	configPath, err := filepath.Abs(configFile)
	if err != nil {
		return log.Errorf("invalid config path %s: %w", configFile, err)
	}

	model, project, err := internal.LoadConfigWithProject(configPath)
	if err != nil {
		return log.Errorf("error loading config: %w", err)
	}

	var bundledBuildDir string
	var tempBuildRoot string
	if project != nil {
		if err := framework.ValidateModel(model, project); err != nil {
			return log.Errorf("error validating project: %w", err)
		}

		tempBuildRoot, err = os.MkdirTemp("", "hyperterse-build-*")
		if err != nil {
			return log.Errorf("error creating temp build directory: %w", err)
		}
		defer os.RemoveAll(tempBuildRoot)

		project.BuildDir = filepath.Join(tempBuildRoot, "build")
		if err := framework.BundleRoutes(project); err != nil {
			return log.Errorf("error bundling route scripts: %w", err)
		}
		bundledBuildDir = project.BuildDir
	}

	outputDir := resolveBuildOutputDir(configPath, model)

	cleanDir := buildCleanDir
	if !cleanDir && model.Export != nil {
		cleanDir = model.Export.CleanDir
	}

	if cleanDir {
		if err := cleanDirectory(log, outputDir); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return log.Errorf("error creating output directory: %w", err)
	}

	binaryPath, err := findBinary(log)
	if err != nil {
		return err
	}
	binaryName := filepath.Base(binaryPath)
	binaryOutPath := filepath.Join(outputDir, binaryName)
	if err := copyFile(binaryPath, binaryOutPath); err != nil {
		return log.Errorf("error copying binary: %w", err)
	}

	// Copy bundled JS artifacts into the output.
	var buildOutDir string
	if bundledBuildDir != "" {
		buildOutDir = resolveBuildArtifactsOutputDir(outputDir)
		if !samePath(bundledBuildDir, buildOutDir) {
			if err := copyDir(bundledBuildDir, buildOutDir); err != nil {
				return log.Errorf("error copying bundled build artifacts: %w", err)
			}
		}
		rebaseProjectBundlePaths(project, bundledBuildDir, buildOutDir)
	}

	manifestModel, err := framework.BuildManifestModel(model, project, outputDir)
	if err != nil {
		return log.Errorf("error building model manifest: %w", err)
	}
	manifestBytes, err := proto.Marshal(manifestModel)
	if err != nil {
		return log.Errorf("error serializing model manifest: %w", err)
	}
	manifestPath := filepath.Join(outputDir, modelManifestFileName)
	legacyManifestPath := filepath.Join(outputDir, legacyModelManifestFileName)
	if !samePath(manifestPath, legacyManifestPath) {
		_ = os.Remove(legacyManifestPath)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		return log.Errorf("error writing model manifest: %w", err)
	}

	log.Successf("Build artifacts written to %s", outputDir)
	if runtime.GOOS == "windows" {
		log.Infof("Run: cd %s && .\\%s serve", outputDir, binaryName)
	} else {
		log.Infof("Run: cd %s && ./%s serve", outputDir, binaryName)
	}
	log.Infof("Manifest: %s", manifestPath)
	if buildOutDir != "" {
		log.Infof("Bundles: %s", buildOutDir)
	}

	return nil
}

func resolveBuildPathArg(args []string) error {
	log := logger.New("build")
	if source != "" {
		return log.Errorf("build does not support --source; use a file-based project")
	}

	if len(args) == 0 {
		if configFile == "" {
			configFile = ".hyperterse"
		}
		return nil
	}

	if configFile != "" {
		return log.Errorf("cannot combine path argument with --file")
	}

	target := args[0]
	info, err := os.Stat(target)
	if err != nil {
		return log.Errorf("invalid build path %q: %w", target, err)
	}
	if info.IsDir() {
		configFile = filepath.Join(target, ".hyperterse")
		return nil
	}
	// Allow directly passing a config file path.
	configFile = target
	return nil
}

func resolveBuildOutputDir(configPath string, model *hyperterse.Model) string {
	configDir := filepath.Dir(configPath)
	if buildOutputDir != "" {
		if filepath.IsAbs(buildOutputDir) {
			return filepath.Clean(buildOutputDir)
		}
		// CLI-provided relative paths are resolved from the current working directory.
		return filepath.Clean(buildOutputDir)
	}

	out := "dist"
	if model != nil && model.Export != nil && model.Export.Out != "" {
		out = model.Export.Out
	}

	// Config-provided relative paths are resolved from the config directory.
	if filepath.IsAbs(out) {
		return filepath.Clean(out)
	}
	return filepath.Clean(filepath.Join(configDir, out))
}

func resolveBuildArtifactsOutputDir(outputDir string) string {
	return filepath.Clean(filepath.Join(outputDir, "build"))
}

func rebaseProjectBundlePaths(project *framework.Project, fromBuildDir, toBuildDir string) {
	if project == nil {
		return
	}
	project.BuildDir = toBuildDir
	project.VendorBundle = rebasePathInBuild(project.VendorBundle, fromBuildDir, toBuildDir)
	for _, route := range project.Routes {
		if route == nil {
			continue
		}
		route.Scripts.Handler = rebasePathInBuild(route.Scripts.Handler, fromBuildDir, toBuildDir)
		route.Scripts.InputTransform = rebasePathInBuild(route.Scripts.InputTransform, fromBuildDir, toBuildDir)
		route.Scripts.OutputTransform = rebasePathInBuild(route.Scripts.OutputTransform, fromBuildDir, toBuildDir)
		for kind, bundlePath := range route.BundleOutputs {
			route.BundleOutputs[kind] = rebasePathInBuild(bundlePath, fromBuildDir, toBuildDir)
		}
	}
}

func rebasePathInBuild(path, fromBuildDir, toBuildDir string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(fromBuildDir, path)
	if err != nil {
		return path
	}
	if rel == ".." || rel == "." || rel == "" || relHasParentTraversal(rel) {
		return path
	}
	return filepath.Clean(filepath.Join(toBuildDir, rel))
}

func relHasParentTraversal(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func findBinary(log *logger.Logger) (string, error) {
	// Prefer the currently running executable so build output always matches
	// the exact command binary the user invoked.
	execPath, err := os.Executable()
	if err == nil {
		// Resolve symlinks to get the actual path
		realPath, err := filepath.EvalSymlinks(execPath)
		if err == nil {
			execPath = realPath
		}

		// Check if the executable exists and is readable
		if _, err := os.Stat(execPath); err == nil {
			return execPath, nil
		}
	}

	// Fallback to a workspace binary if available.
	distPath := "dist/hyperterse"
	if runtime.GOOS == "windows" {
		distPath = "dist/hyperterse.exe"
	}
	if _, err := os.Stat(distPath); err == nil {
		return distPath, nil
	}

	return "", log.Errorf("could not determine executable path")
}

// cleanDirectory removes all contents of a directory but keeps the directory itself
func cleanDirectory(log *logger.Logger, dirPath string) error {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean
		return nil
	}
	if err != nil {
		return log.Errorf("error checking directory %s: %w", dirPath, err)
	}
	if !info.IsDir() {
		return log.Errorf("path %s is not a directory", dirPath)
	}

	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return log.Errorf("error reading directory %s: %w", dirPath, err)
	}

	// Remove all entries
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return log.Errorf("error removing %s: %w", entryPath, err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		// Skip symlinks in build output to keep packaging deterministic.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		return copyFile(path, target)
	})
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}
