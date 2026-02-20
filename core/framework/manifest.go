package framework

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"google.golang.org/protobuf/proto"
)

// BuildManifestModel clones the compiled model and embeds framework runtime metadata
// (tool bundle references, auth policy hooks, and vendor bundle path) so the
// runtime can start directly from the manifest without reparsing app sources.
func BuildManifestModel(model *hyperterse.Model, project *Project, manifestDir string) (*hyperterse.Model, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	cloned, ok := proto.Clone(model).(*hyperterse.Model)
	if !ok {
		return nil, fmt.Errorf("failed to clone model")
	}
	if project == nil {
		return cloned, nil
	}

	manifest := &hyperterse.FrameworkManifest{
		VendorBundle: toManifestPath(manifestDir, project.VendorBundle),
	}

	toolNames := make([]string, 0, len(project.Tools))
	for toolName := range project.Tools {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		tool := project.Tools[toolName]
		if tool == nil {
			continue
		}

		handlerPath := firstNonEmpty(tool.BundleOutputs["handler"], tool.Scripts.Handler)
		inputPath := firstNonEmpty(tool.BundleOutputs["input_transform"], tool.Scripts.InputTransform)
		outputPath := firstNonEmpty(tool.BundleOutputs["output_transform"], tool.Scripts.OutputTransform)

		manifest.Tools = append(manifest.Tools, &hyperterse.ToolBundleManifest{
			ToolName:              tool.ToolName,
			ToolPath:              tool.ToolPath,
			HandlerBundle:         toManifestPath(manifestDir, handlerPath),
			InputTransformBundle:  toManifestPath(manifestDir, inputPath),
			OutputTransformBundle: toManifestPath(manifestDir, outputPath),
			AuthPlugin:            tool.Auth.Plugin,
			AuthPolicy:            copyStringMap(tool.Auth.Policy),
		})
	}

	cloned.Framework = manifest
	return cloned, nil
}

// ProjectFromManifestModel reconstructs minimal framework project metadata from a
// compiled model manifest. The returned project contains tools with prebuilt JS
// bundle references and is ready for runtime execution.
func ProjectFromManifestModel(model *hyperterse.Model, manifestPath string) (*Project, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	frameworkManifest := model.GetFramework()
	if frameworkManifest == nil {
		return nil, nil
	}

	manifestDir := filepath.Dir(manifestPath)
	vendorPath := resolveManifestPath(manifestDir, frameworkManifest.GetVendorBundle())
	buildDir := ""
	if vendorPath != "" {
		buildDir = filepath.Dir(vendorPath)
	}
	project := &Project{
		BaseDir:      manifestDir,
		BuildDir:     buildDir,
		VendorBundle: vendorPath,
		Tools:        map[string]*Tool{},
	}

	queryByName := make(map[string]*hyperterse.Query, len(model.Queries))
	for _, query := range model.Queries {
		if query != nil {
			queryByName[query.Name] = query
		}
	}

	for _, toolManifest := range frameworkManifest.GetTools() {
		if toolManifest == nil {
			continue
		}
		query, ok := queryByName[toolManifest.GetToolName()]
		if !ok {
			return nil, fmt.Errorf("manifest tool %q does not match any compiled query", toolManifest.GetToolName())
		}

		handlerPath := resolveManifestPath(manifestDir, toolManifest.GetHandlerBundle())
		inputPath := resolveManifestPath(manifestDir, toolManifest.GetInputTransformBundle())
		outputPath := resolveManifestPath(manifestDir, toolManifest.GetOutputTransformBundle())

		tool := &Tool{
			ToolName: toolManifest.GetToolName(),
			ToolPath: firstNonEmpty(toolManifest.GetToolPath(), toolManifest.GetToolName()),
			Query:    query,
			Scripts: ToolScripts{
				Handler:         handlerPath,
				InputTransform:  inputPath,
				OutputTransform: outputPath,
			},
			Auth: ToolAuth{
				Plugin: toolManifest.GetAuthPlugin(),
				Policy: copyStringMap(toolManifest.GetAuthPolicy()),
			},
			BundleOutputs: map[string]string{},
		}
		if handlerPath != "" {
			tool.BundleOutputs["handler"] = handlerPath
		}
		if inputPath != "" {
			tool.BundleOutputs["input_transform"] = inputPath
		}
		if outputPath != "" {
			tool.BundleOutputs["output_transform"] = outputPath
		}

		project.Tools[tool.ToolName] = tool
	}

	if len(project.Tools) == 0 {
		return nil, nil
	}
	return project, nil
}

func toManifestPath(manifestDir, targetPath string) string {
	if targetPath == "" {
		return ""
	}
	cleanTarget := filepath.Clean(targetPath)
	absBase, errBase := filepath.Abs(manifestDir)
	absTarget, errTarget := filepath.Abs(cleanTarget)
	if errBase == nil && errTarget == nil {
		if rel, err := filepath.Rel(absBase, absTarget); err == nil && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return filepath.ToSlash(filepath.Clean(rel))
		}
	}
	return filepath.ToSlash(cleanTarget)
}

func resolveManifestPath(manifestDir, targetPath string) string {
	if targetPath == "" {
		return ""
	}
	path := filepath.FromSlash(targetPath)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(manifestDir, path))
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
