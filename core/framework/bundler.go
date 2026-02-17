package framework

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/hyperterse/hyperterse/core/logger"
)

var tsImportPattern = regexp.MustCompile(`(?m)(?:import\s+(?:[^'"]+from\s+)?|export\s+[^'"]*from\s+)['"]([^'"]+)['"]`)
var defaultImportPattern = regexp.MustCompile(`^\s*import\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s+['"]([^'"]+)['"]`)
var namespaceImportPattern = regexp.MustCompile(`^\s*import\s+\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s+['"]([^'"]+)['"]`)
var sideEffectImportPattern = regexp.MustCompile(`^\s*import\s+['"]([^'"]+)['"]`)
var namedBindingsPattern = regexp.MustCompile(`^\s*import\s+\{([^}]+)\}\s+from\s+['"]([^'"]+)['"]`)

// BundleRoutes bundles TS scripts for each route and builds a shared vendor.js.
// Default backend is rolldown; fallback backend uses esbuild's Go API.
func BundleRoutes(project *Project) error {
	if project == nil {
		return nil
	}
	log := logger.New("bundler")

	buildDir := filepath.Join(project.BaseDir, ".hyperterse", "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("failed to create build dir: %w", err)
	}

	routeEntries := collectRouteTSEntries(project)
	if len(routeEntries) == 0 {
		log.Debugf("No TypeScript route entries found; skipping bundling")
		return nil
	}

	deps, err := collectExternalDeps(routeEntries)
	if err != nil {
		return err
	}
	sort.Strings(deps)

	project.VendorBundle = filepath.Join(buildDir, "vendor.js")
	if err := buildVendorBundle(project.VendorBundle, project.BaseDir, deps); err != nil {
		return err
	}

	if err := bundleRouteEntries(project, routeEntries, buildDir); err != nil {
		return err
	}

	log.Infof("Bundled %d route script(s)", len(routeEntries))
	return nil
}

func collectRouteTSEntries(project *Project) map[string]string {
	entries := map[string]string{}
	for _, route := range project.Routes {
		addScriptEntry(route, entries, "handler", route.Scripts.Handler)
		addScriptEntry(route, entries, "input_transform", route.Scripts.InputTransform)
		addScriptEntry(route, entries, "output_transform", route.Scripts.OutputTransform)
	}
	return entries
}

func addScriptEntry(route *Route, entries map[string]string, kind string, scriptPath string) {
	if scriptPath == "" {
		return
	}
	if strings.HasSuffix(strings.ToLower(scriptPath), ".ts") {
		entries[route.ToolName+"::"+kind] = scriptPath
	}
}

func collectExternalDeps(entries map[string]string) ([]string, error) {
	depSet := map[string]struct{}{}
	for _, filePath := range entries {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed reading TS entry %s: %w", filePath, err)
		}
		matches := tsImportPattern.FindAllSubmatch(content, -1)
		for _, match := range matches {
			spec := string(match[1])
			if strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") {
				continue
			}
			depSet[spec] = struct{}{}
		}
	}
	deps := make([]string, 0, len(depSet))
	for dep := range depSet {
		deps = append(deps, dep)
	}
	return deps, nil
}

func buildVendorBundle(vendorOut, projectDir string, deps []string) error {
	if projectDir != "" {
		absDir, err := filepath.Abs(projectDir)
		if err != nil {
			return fmt.Errorf("failed to resolve project dir %s: %w", projectDir, err)
		}
		projectDir = absDir
	}

	if len(deps) == 0 {
		emptyRegistry := "globalThis.__hyperterse_vendor = globalThis.__hyperterse_vendor || {};\n"
		if err := os.WriteFile(vendorOut, []byte(emptyRegistry), 0o644); err != nil {
			return fmt.Errorf("failed to write empty vendor bundle: %w", err)
		}
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "hyperterse-vendor-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for vendor bundle: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	type depBuild struct {
		dep    string
		global string
		code   []byte
	}
	builds := make([]depBuild, 0, len(deps))
	for i, dep := range deps {
		globalName := fmt.Sprintf("__hyperterse_vendor_mod_%d", i)
		outPath := filepath.Join(tmpDir, fmt.Sprintf("dep_%d.js", i))
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{dep},
			Outfile:     outPath,
			Bundle:      true,
			Format:      api.FormatIIFE,
			Platform:    api.PlatformBrowser,
			Target:      api.ES2020,
			GlobalName:  globalName,
			AbsWorkingDir: projectDir,
			Write:       true,
			LogLevel:    api.LogLevelSilent,
		})
		if len(result.Errors) > 0 {
			return fmt.Errorf("failed to build vendor dependency %s: %s", dep, result.Errors[0].Text)
		}
		code, err := os.ReadFile(outPath)
		if err != nil {
			return fmt.Errorf("failed reading vendor dep bundle %s: %w", dep, err)
		}
		builds = append(builds, depBuild{dep: dep, global: globalName, code: code})
	}

	var vendor bytes.Buffer
	vendor.WriteString("globalThis.__hyperterse_vendor = globalThis.__hyperterse_vendor || {};\n")
	for _, depBuild := range builds {
		vendor.Write(depBuild.code)
		vendor.WriteString("\n")
		vendor.WriteString(fmt.Sprintf("globalThis.__hyperterse_vendor[%q] = typeof %s !== \"undefined\" ? %s : {};\n", depBuild.dep, depBuild.global, depBuild.global))
	}
	if err := os.WriteFile(vendorOut, vendor.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write vendor bundle: %w", err)
	}
	return nil
}

func bundleRouteEntries(project *Project, entries map[string]string, buildDir string) error {
	for entryKey, entryPath := range entries {
		parts := strings.Split(entryKey, "::")
		if len(parts) != 2 {
			continue
		}
		toolName, kind := parts[0], parts[1]
		route, ok := project.Routes[toolName]
		if !ok {
			continue
		}
		routeBuildDir := filepath.Join(buildDir, "routes", toolName)
		if err := os.MkdirAll(routeBuildDir, 0o755); err != nil {
			return fmt.Errorf("failed to create route build dir: %w", err)
		}
		outFile := filepath.Join(routeBuildDir, kind+".js")

		source, err := os.ReadFile(entryPath)
		if err != nil {
			return fmt.Errorf("failed to read route entry %s: %w", entryPath, err)
		}
		rewritten, usesVendor, err := rewriteRouteSourceForVendor(string(source))
		if err != nil {
			return fmt.Errorf("failed to rewrite imports for %s: %w", entryPath, err)
		}

		if !usesVendor {
			if err := runBundler("route", entryPath, outFile, nil); err != nil {
				return err
			}
		} else if err := runBundlerFromSource("route", rewritten, entryPath, outFile); err != nil {
			return err
		}
		route.BundleOutputs[kind] = outFile
	}
	return nil
}

func rewriteRouteSourceForVendor(source string) (string, bool, error) {
	lines := strings.Split(source, "\n")
	rewritten := make([]string, 0, len(lines))
	usesVendor := false
	modCounter := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import ") {
			rewritten = append(rewritten, line)
			continue
		}

		if match := namedBindingsPattern.FindStringSubmatch(trimmed); len(match) == 3 {
			bindings, spec := match[1], match[2]
			if !isExternalSpecifier(spec) {
				rewritten = append(rewritten, line)
				continue
			}
			modCounter++
			modSetup, modVar := vendorModuleConst(spec, modCounter)
			rewritten = append(rewritten, modSetup)
			for _, part := range strings.Split(bindings, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if strings.Contains(part, " as ") {
					aliasParts := strings.SplitN(part, " as ", 2)
					orig := strings.TrimSpace(aliasParts[0])
					alias := strings.TrimSpace(aliasParts[1])
					rewritten = append(rewritten, fmt.Sprintf("const %s = %s[%q];", alias, modVar, orig))
				} else {
					rewritten = append(rewritten, fmt.Sprintf("const %s = %s[%q];", part, modVar, part))
				}
			}
			usesVendor = true
			continue
		}

		if match := defaultImportPattern.FindStringSubmatch(trimmed); len(match) == 3 {
			ident, spec := match[1], match[2]
			if !isExternalSpecifier(spec) {
				rewritten = append(rewritten, line)
				continue
			}
			modCounter++
			modSetup, modVar := vendorModuleConst(spec, modCounter)
			rewritten = append(rewritten, modSetup)
			rewritten = append(rewritten, fmt.Sprintf("const %s = (%s.default ?? %s);", ident, modVar, modVar))
			usesVendor = true
			continue
		}

		if match := namespaceImportPattern.FindStringSubmatch(trimmed); len(match) == 3 {
			ident, spec := match[1], match[2]
			if !isExternalSpecifier(spec) {
				rewritten = append(rewritten, line)
				continue
			}
			modCounter++
			modSetup, modVar := vendorModuleConst(spec, modCounter)
			rewritten = append(rewritten, modSetup)
			rewritten = append(rewritten, fmt.Sprintf("const %s = %s;", ident, modVar))
			usesVendor = true
			continue
		}

		if match := sideEffectImportPattern.FindStringSubmatch(trimmed); len(match) == 2 {
			spec := match[1]
			if !isExternalSpecifier(spec) {
				rewritten = append(rewritten, line)
				continue
			}
			modCounter++
			modSetup, modVar := vendorModuleConst(spec, modCounter)
			rewritten = append(rewritten, modSetup)
			rewritten = append(rewritten, fmt.Sprintf("void %s;", modVar))
			usesVendor = true
			continue
		}

		// Keep unknown import forms as-is for compatibility.
		rewritten = append(rewritten, line)
	}
	return strings.Join(rewritten, "\n"), usesVendor, nil
}

func runBundlerFromSource(bundleType, source, sourcePath, outFile string) error {
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   source,
			ResolveDir: filepath.Dir(sourcePath),
			Sourcefile: sourcePath,
			Loader:     api.LoaderTS,
		},
		Outfile:    outFile,
		Bundle:     true,
		Format:     api.FormatIIFE,
		Platform:   api.PlatformNeutral,
		Sourcemap:  api.SourceMapInline,
		Target:     api.ES2020,
		GlobalName: "HyperterseBundle",
		Write:      true,
		LogLevel:   api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("esbuild failed for %s bundle: %s", bundleType, result.Errors[0].Text)
	}
	return nil
}

func runBundler(bundleType, entryPath, outFile string, externalDeps []string) error {
	// Default: rolldown native binary if available in PATH.
	if _, err := exec.LookPath("rolldown"); err == nil {
		// Keep invocation minimal to avoid requiring a user-maintained config.
		// If rolldown invocation fails, we fallback to esbuild Go API.
		args := []string{"build", entryPath, "--format", "iife", "--platform", "neutral", "--name", "HyperterseBundle", "--file", outFile}
		for _, dep := range externalDeps {
			args = append(args, "--external", dep)
		}
		cmd := exec.Command("rolldown", args...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		logger.New("bundler").Warnf("rolldown failed for %s bundle, falling back to esbuild: %v (%s)", bundleType, err, strings.TrimSpace(string(output)))
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryPath},
		Outfile:     outFile,
		Bundle:      true,
		Format:      api.FormatIIFE,
		Platform:    api.PlatformNeutral,
		Sourcemap:   api.SourceMapInline,
		Target:      api.ES2020,
		External:    externalDeps,
		GlobalName:  "HyperterseBundle",
		Write:       true,
		LogLevel:    api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("esbuild failed for %s bundle: %s", bundleType, result.Errors[0].Text)
	}
	return nil
}

func isExternalSpecifier(spec string) bool {
	return !strings.HasPrefix(spec, ".") && !strings.HasPrefix(spec, "/")
}

func vendorModuleConst(spec string, index int) (string, string) {
	modVar := fmt.Sprintf("__ht_vendor_mod_%d", index)
	line := fmt.Sprintf("const %s = (globalThis.__hyperterse_vendor || {})[%q]; if (!%s) { throw new Error(\"Missing vendor module: %s\"); }", modVar, spec, modVar, spec)
	return line, modVar
}
