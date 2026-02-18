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
// (route bundle references, auth policy hooks, and vendor bundle path) so the
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

	toolNames := make([]string, 0, len(project.Routes))
	for toolName := range project.Routes {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		route := project.Routes[toolName]
		if route == nil {
			continue
		}

		handlerPath := firstNonEmpty(route.BundleOutputs["handler"], route.Scripts.Handler)
		inputPath := firstNonEmpty(route.BundleOutputs["input_transform"], route.Scripts.InputTransform)
		outputPath := firstNonEmpty(route.BundleOutputs["output_transform"], route.Scripts.OutputTransform)

		manifest.Routes = append(manifest.Routes, &hyperterse.RouteBundleManifest{
			ToolName:              route.ToolName,
			RoutePath:             route.RoutePath,
			HandlerBundle:         toManifestPath(manifestDir, handlerPath),
			InputTransformBundle:  toManifestPath(manifestDir, inputPath),
			OutputTransformBundle: toManifestPath(manifestDir, outputPath),
			AuthPlugin:            route.Auth.Plugin,
			AuthPolicy:            copyStringMap(route.Auth.Policy),
		})
	}

	cloned.Framework = manifest
	return cloned, nil
}

// ProjectFromManifestModel reconstructs minimal framework project metadata from a
// compiled model manifest. The returned project contains routes with prebuilt JS
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
		Routes:       map[string]*Route{},
	}

	queryByName := make(map[string]*hyperterse.Query, len(model.Queries))
	for _, query := range model.Queries {
		if query != nil {
			queryByName[query.Name] = query
		}
	}

	for _, routeManifest := range frameworkManifest.GetRoutes() {
		if routeManifest == nil {
			continue
		}
		query, ok := queryByName[routeManifest.GetToolName()]
		if !ok {
			return nil, fmt.Errorf("manifest route %q does not match any compiled query", routeManifest.GetToolName())
		}

		handlerPath := resolveManifestPath(manifestDir, routeManifest.GetHandlerBundle())
		inputPath := resolveManifestPath(manifestDir, routeManifest.GetInputTransformBundle())
		outputPath := resolveManifestPath(manifestDir, routeManifest.GetOutputTransformBundle())

		route := &Route{
			ToolName:  routeManifest.GetToolName(),
			RoutePath: firstNonEmpty(routeManifest.GetRoutePath(), routeManifest.GetToolName()),
			Query:     query,
			Scripts: RouteScripts{
				Handler:         handlerPath,
				InputTransform:  inputPath,
				OutputTransform: outputPath,
			},
			Auth: RouteAuth{
				Plugin: routeManifest.GetAuthPlugin(),
				Policy: copyStringMap(routeManifest.GetAuthPolicy()),
			},
			BundleOutputs: map[string]string{},
		}
		if handlerPath != "" {
			route.BundleOutputs["handler"] = handlerPath
		}
		if inputPath != "" {
			route.BundleOutputs["input_transform"] = inputPath
		}
		if outputPath != "" {
			route.BundleOutputs["output_transform"] = outputPath
		}

		project.Routes[route.ToolName] = route
	}

	if len(project.Routes) == 0 {
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
