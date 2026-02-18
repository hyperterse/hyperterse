package framework

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/types"
	"gopkg.in/yaml.v3"
)

var tsConventionPattern = regexp.MustCompile(`(?i)\.ts$`)

// CompileProjectIfPresent discovers app routes and merges them into model queries.
// If the app directory does not exist, it returns nil project with no error.
func CompileProjectIfPresent(configFilePath string, model *hyperterse.Model) (*Project, error) {
	baseDir := filepath.Dir(configFilePath)
	appDir := filepath.Join(baseDir, "app")
	adaptersDir := filepath.Join(appDir, "adapters")
	routesDir := filepath.Join(appDir, "routes")
	buildOutDir := "dist"
	if model != nil && model.Export != nil && model.Export.Out != "" {
		buildOutDir = model.Export.Out
	}
	buildDir := filepath.Join(baseDir, buildOutDir, "build")
	if filepath.IsAbs(buildOutDir) {
		buildDir = filepath.Join(buildOutDir, "build")
	}

	stat, err := os.Stat(appDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat app directory: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("app exists but is not a directory: %s", appDir)
	}

	log := logger.New("framework")
	log.Infof("Compiling v2 app routes from %s", appDir)

	project := &Project{
		BaseDir:     baseDir,
		AppDir:      appDir,
		AdaptersDir: adaptersDir,
		RoutesDir:   routesDir,
		BuildDir:    buildDir,
		Routes:      map[string]*Route{},
	}

	adapterFiles, err := discoverAdapterFiles(adaptersDir)
	if err != nil {
		return nil, err
	}
	sort.Strings(adapterFiles)
	for _, adapterFile := range adapterFiles {
		adapter, err := compileAdapterFile(adapterFile)
		if err != nil {
			return nil, err
		}
		model.Adapters = append(model.Adapters, adapter)
	}

	routeTerseFiles, err := discoverRouteTerseFiles(routesDir)
	if err != nil {
		return nil, err
	}
	sort.Strings(routeTerseFiles)

	for _, terseFile := range routeTerseFiles {
		route, err := compileRouteFile(project, terseFile)
		if err != nil {
			return nil, err
		}
		if _, exists := project.Routes[route.ToolName]; exists {
			return nil, fmt.Errorf("duplicate tool name generated from routes: %s", route.ToolName)
		}
		project.Routes[route.ToolName] = route
		model.Queries = append(model.Queries, route.Query)
	}

	log.Infof("Compiled %d route(s) into model queries", len(project.Routes))
	return project, nil
}

func discoverAdapterFiles(adaptersDir string) ([]string, error) {
	var files []string
	if _, err := os.Stat(adaptersDir); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat adapters dir: %w", err)
	}
	err := filepath.WalkDir(adaptersDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".terse") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover adapter .terse files: %w", err)
	}
	return files, nil
}

func discoverRouteTerseFiles(routesDir string) ([]string, error) {
	var files []string
	if _, err := os.Stat(routesDir); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat routes dir: %w", err)
	}
	err := filepath.WalkDir(routesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "config.terse") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover route .terse files: %w", err)
	}
	return files, nil
}

func compileAdapterFile(adapterFile string) (*hyperterse.Adapter, error) {
	content, err := os.ReadFile(adapterFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read adapter config %s: %w", adapterFile, err)
	}
	var cfg AdapterFileConfig
	if err := strictYAMLUnmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse adapter config %s: %w", adapterFile, err)
	}
	name := cfg.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(adapterFile), filepath.Ext(adapterFile))
	}
	connectorEnum, err := types.StringToConnectorEnum(cfg.Connector)
	if err != nil {
		return nil, fmt.Errorf("invalid connector in %s: %w", adapterFile, err)
	}
	adapter := &hyperterse.Adapter{
		Name:             name,
		Connector:        connectorEnum,
		ConnectionString: cfg.ConnectionString,
	}
	if adapter.Options == nil {
		adapter.Options = &hyperterse.AdapterOptions{Options: map[string]string{}}
	}
	for k, v := range cfg.Options {
		adapter.Options.Options[k] = fmt.Sprintf("%v", v)
	}
	return adapter, nil
}

func compileRouteFile(project *Project, terseFile string) (*Route, error) {
	content, err := os.ReadFile(terseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read route config %s: %w", terseFile, err)
	}

	var cfg RouteFileConfig
	if err := strictYAMLUnmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse route config %s: %w", terseFile, err)
	}

	routeDir := filepath.Dir(terseFile)
	routePath, err := routePathFromDirectory(project.RoutesDir, routeDir)
	if err != nil {
		return nil, err
	}
	toolName := cfg.Name
	if toolName == "" {
		toolName = toolNameFromRoutePath(routePath)
	}

	query, err := routeConfigToQuery(toolName, cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid route config %s: %w", terseFile, err)
	}

	route := &Route{
		ToolName:  toolName,
		RoutePath: routePath,
		Directory: routeDir,
		TerseFile: terseFile,
		Query:     query,
		Scripts: RouteScripts{
			Handler:         resolveScriptPath(project.BaseDir, routeDir, cfg.Scripts.Handler),
			InputTransform:  resolveScriptPath(project.BaseDir, routeDir, cfg.Scripts.InputTransform),
			OutputTransform: resolveScriptPath(project.BaseDir, routeDir, cfg.Scripts.OutputTransform),
		},
		Auth: RouteAuth{
			Plugin: cfg.Auth.Plugin,
			Policy: cfg.Auth.Policy,
		},
		BundleOutputs: map[string]string{},
	}
	applyRouteScriptConventions(route)

	return route, nil
}

func resolveScriptPath(baseDir, routeDir, scriptPath string) string {
	if scriptPath == "" {
		return ""
	}
	if filepath.IsAbs(scriptPath) {
		return scriptPath
	}
	if strings.HasPrefix(scriptPath, "./") || strings.HasPrefix(scriptPath, "../") {
		return filepath.Join(routeDir, scriptPath)
	}
	routeLocal := filepath.Join(routeDir, scriptPath)
	if _, err := os.Stat(routeLocal); err == nil {
		return routeLocal
	}
	return filepath.Join(baseDir, scriptPath)
}

func applyRouteScriptConventions(route *Route) {
	entries, err := os.ReadDir(route.Directory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !tsConventionPattern.MatchString(entry.Name()) {
			continue
		}
		fileName := strings.ToLower(entry.Name())
		fullPath := filepath.Join(route.Directory, entry.Name())
		if route.Scripts.Handler == "" && strings.Contains(fileName, "handler") {
			route.Scripts.Handler = fullPath
			continue
		}
		if route.Scripts.InputTransform == "" && strings.Contains(fileName, "input") && strings.Contains(fileName, "validator") {
			route.Scripts.InputTransform = fullPath
			continue
		}
		if route.Scripts.OutputTransform == "" && (strings.Contains(fileName, "data") && strings.Contains(fileName, "mapper")) {
			route.Scripts.OutputTransform = fullPath
			continue
		}
	}
}

func strictYAMLUnmarshal(content []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	return decoder.Decode(out)
}

func routeConfigToQuery(toolName string, cfg RouteFileConfig) (*hyperterse.Query, error) {
	query := &hyperterse.Query{
		Name:        toolName,
		Description: cfg.Description,
		Statement:   cfg.Statement,
	}
	if query.Description == "" {
		query.Description = fmt.Sprintf("Tool generated from app route: %s", toolName)
	}

	// Custom handler routes are allowed without use/statement. They bypass DB execution.
	// We still add a harmless placeholder to remain compatible with existing validators/executors.
	if query.Statement == "" {
		query.Statement = "SELECT 1"
	}

	switch v := cfg.Use.(type) {
	case string:
		if v != "" {
			query.Use = []string{v}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				query.Use = append(query.Use, s)
			}
		}
	}

	for name, inputSpec := range cfg.Inputs {
		primitive, err := types.StringToPrimitiveEnum(inputSpec.Type)
		if err != nil {
			return nil, fmt.Errorf("input '%s' has invalid type '%s': %w", name, inputSpec.Type, err)
		}
		defaultValue := ""
		if inputSpec.Default != nil {
			defaultValue = fmt.Sprintf("%v", inputSpec.Default)
		}
		query.Inputs = append(query.Inputs, &hyperterse.Input{
			Name:         name,
			Optional:     inputSpec.Optional,
			Type:         primitive,
			Description:  inputSpec.Description,
			DefaultValue: defaultValue,
		})
	}

	for name, dataSpec := range cfg.Data {
		primitive, err := types.StringToPrimitiveEnum(dataSpec.Type)
		if err != nil {
			return nil, fmt.Errorf("data '%s' has invalid type '%s': %w", name, dataSpec.Type, err)
		}
		query.Data = append(query.Data, &hyperterse.Data{
			Name:        name,
			Type:        primitive,
			Description: dataSpec.Description,
			MapTo:       dataSpec.MapTo,
		})
	}

	return query, nil
}
