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

type discoveryConfigFile struct {
	Root     string                   `yaml:"root"`
	Tools    discoveryDirectoryConfig `yaml:"tools"`
	Adapters discoveryDirectoryConfig `yaml:"adapters"`
}

type discoveryDirectoryConfig struct {
	Directory string `yaml:"directory"`
}

// CompileProjectIfPresent discovers tools/adapters directories and merges tools into model definitions.
// If the configured discovery root does not exist, it returns nil project with no error.
func CompileProjectIfPresent(configFilePath string, model *hyperterse.Model) (*Project, error) {
	baseDir := filepath.Dir(configFilePath)
	appDir, adaptersDir, toolsDir, err := resolveProjectDirectories(configFilePath)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("failed to stat project root directory: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("project root exists but is not a directory: %s", appDir)
	}

	log := logger.New("framework")
	log.Infof("Compiling v2 tools from %s", appDir)

	project := &Project{
		BaseDir:     baseDir,
		AppDir:      appDir,
		AdaptersDir: adaptersDir,
		ToolsDir:    toolsDir,
		BuildDir:    buildDir,
		Tools:       map[string]*Tool{},
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

	toolTerseFiles, err := discoverToolTerseFiles(toolsDir)
	if err != nil {
		return nil, err
	}
	sort.Strings(toolTerseFiles)

	for _, terseFile := range toolTerseFiles {
		tool, err := compileToolFile(project, terseFile)
		if err != nil {
			return nil, err
		}
		if _, exists := project.Tools[tool.ToolName]; exists {
			return nil, fmt.Errorf("duplicate tool name generated from tools: %s", tool.ToolName)
		}
		project.Tools[tool.ToolName] = tool
		model.Tools = append(model.Tools, tool.Definition)
	}

	log.Infof("Compiled %d tool(s) into model tools", len(project.Tools))
	return project, nil
}

func resolveProjectDirectories(configFilePath string) (string, string, string, error) {
	baseDir := filepath.Dir(configFilePath)
	rootDir := "app"
	toolsDirName := "tools"
	adaptersDirName := "adapters"

	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read config for discovery settings: %w", err)
	}

	var cfg discoveryConfigFile
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return "", "", "", fmt.Errorf("failed to decode discovery settings: %w", err)
	}
	if cfg.Root != "" {
		rootDir = cfg.Root
	}
	if cfg.Tools.Directory != "" {
		toolsDirName = cfg.Tools.Directory
	}
	if cfg.Adapters.Directory != "" {
		adaptersDirName = cfg.Adapters.Directory
	}

	appDir := resolveDiscoveryPath(baseDir, rootDir)
	adaptersDir := resolveDiscoveryPath(appDir, adaptersDirName)
	toolsDir := resolveDiscoveryPath(appDir, toolsDirName)
	return appDir, adaptersDir, toolsDir, nil
}

func resolveDiscoveryPath(baseDir, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return filepath.Clean(baseDir)
	}
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}
	return filepath.Clean(filepath.Join(baseDir, configured))
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

func discoverToolTerseFiles(toolsDir string) ([]string, error) {
	var files []string
	if _, err := os.Stat(toolsDir); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat tools dir: %w", err)
	}
	err := filepath.WalkDir(toolsDir, func(path string, d fs.DirEntry, walkErr error) error {
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
		return nil, fmt.Errorf("failed to discover tool .terse files: %w", err)
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

func compileToolFile(project *Project, terseFile string) (*Tool, error) {
	content, err := os.ReadFile(terseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool config %s: %w", terseFile, err)
	}

	var cfg ToolFileConfig
	if err := strictYAMLUnmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse tool config %s: %w", terseFile, err)
	}

	toolDir := filepath.Dir(terseFile)
	toolPath, err := toolPathFromDirectory(project.ToolsDir, toolDir)
	if err != nil {
		return nil, err
	}
	toolName := cfg.Name
	if toolName == "" {
		toolName = toolNameFromToolPath(toolPath)
	}

	compiledTool, err := toolConfigToProto(toolName, cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid tool config %s: %w", terseFile, err)
	}

	tool := &Tool{
		ToolName:   toolName,
		ToolPath:   toolPath,
		Directory:  toolDir,
		TerseFile:  terseFile,
		Definition: compiledTool,
		Auth: ToolAuth{
			Plugin: cfg.Auth.Plugin,
			Policy: cfg.Auth.Policy,
		},
		BundleOutputs: map[string]string{},
	}
	handlerPath, handlerExport := resolveScriptRef(project.BaseDir, toolDir, cfg.Handler, "default")
	inputPath, inputExport := resolveScriptRef(project.BaseDir, toolDir, cfg.Mappers.Input, "default")
	outputPath, outputExport := resolveScriptRef(project.BaseDir, toolDir, cfg.Mappers.Output, "default")
	tool.Scripts = ToolScripts{
		Handler:               handlerPath,
		HandlerExport:         handlerExport,
		InputTransform:        inputPath,
		InputTransformExport:  inputExport,
		OutputTransform:       outputPath,
		OutputTransformExport: outputExport,
	}
	applyToolScriptConventions(tool)

	return tool, nil
}

func resolveScriptPath(baseDir, toolDir, scriptPath string) string {
	if scriptPath == "" {
		return ""
	}
	if filepath.IsAbs(scriptPath) {
		return scriptPath
	}
	if strings.HasPrefix(scriptPath, "./") || strings.HasPrefix(scriptPath, "../") {
		return filepath.Join(toolDir, scriptPath)
	}
	toolLocal := filepath.Join(toolDir, scriptPath)
	if _, err := os.Stat(toolLocal); err == nil {
		return toolLocal
	}
	return filepath.Join(baseDir, scriptPath)
}

func resolveScriptRef(baseDir, toolDir, scriptRef, defaultExport string) (string, string) {
	scriptPath, exportName := parseScriptReference(scriptRef)
	if scriptPath == "" {
		return "", ""
	}
	if exportName == "" {
		exportName = defaultExport
	}
	return resolveScriptPath(baseDir, toolDir, scriptPath), exportName
}

func parseScriptReference(scriptRef string) (string, string) {
	ref := strings.TrimSpace(scriptRef)
	if ref == "" {
		return "", ""
	}

	parts := strings.SplitN(ref, "#", 2)
	path := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return path, ""
	}
	return path, strings.TrimSpace(parts[1])
}

func applyToolScriptConventions(tool *Tool) {
	entries, err := os.ReadDir(tool.Directory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !tsConventionPattern.MatchString(entry.Name()) {
			continue
		}
		fileName := strings.ToLower(entry.Name())
		baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		fullPath := filepath.Join(tool.Directory, entry.Name())
		if tool.Scripts.Handler == "" && (baseName == "handler" || strings.Contains(fileName, "handler")) {
			tool.Scripts.Handler = fullPath
			if tool.Scripts.HandlerExport == "" {
				tool.Scripts.HandlerExport = "default"
			}
			continue
		}
		if tool.Scripts.InputTransform == "" && (baseName == "input" || (strings.Contains(fileName, "input") && strings.Contains(fileName, "validator"))) {
			tool.Scripts.InputTransform = fullPath
			if tool.Scripts.InputTransformExport == "" {
				tool.Scripts.InputTransformExport = "default"
			}
			continue
		}
		if tool.Scripts.OutputTransform == "" && (baseName == "output" || (strings.Contains(fileName, "data") && strings.Contains(fileName, "mapper"))) {
			tool.Scripts.OutputTransform = fullPath
			if tool.Scripts.OutputTransformExport == "" {
				tool.Scripts.OutputTransformExport = "default"
			}
			continue
		}
	}
}

func strictYAMLUnmarshal(content []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	return decoder.Decode(out)
}

func toolConfigToProto(toolName string, cfg ToolFileConfig) (*hyperterse.Tool, error) {
	tool := &hyperterse.Tool{
		Name:        toolName,
		Description: cfg.Description,
		Statement:   cfg.Statement,
	}
	if tool.Description == "" {
		tool.Description = fmt.Sprintf("Tool generated from app tool: %s", toolName)
	}

	// Custom handler tools are allowed without use/statement. They bypass DB execution.
	// We still add a harmless placeholder to remain compatible with existing validators/executors.
	if tool.Statement == "" {
		tool.Statement = "SELECT 1"
	}

	switch v := cfg.Use.(type) {
	case string:
		if v != "" {
			tool.Use = []string{v}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				tool.Use = append(tool.Use, s)
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
		tool.Inputs = append(tool.Inputs, &hyperterse.Input{
			Name:         name,
			Optional:     inputSpec.Optional,
			Type:         primitive,
			Description:  inputSpec.Description,
			DefaultValue: defaultValue,
		})
	}

	return tool, nil
}
