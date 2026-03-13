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

var (
	tsConventionPattern = regexp.MustCompile(`(?i)\.ts$`)
	agentNamePattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

type discoveryConfigFile struct {
	Root     string                   `yaml:"root"`
	Tools    discoveryDirectoryConfig `yaml:"tools"`
	Adapters discoveryDirectoryConfig `yaml:"adapters"`
	Agents   discoveryAgentsConfig    `yaml:"agents"`
}

type discoveryDirectoryConfig struct {
	Directory string `yaml:"directory"`
}

type discoveryAgentsConfig struct {
	Directory  string               `yaml:"directory"`
	ToolAccess *agentToolAccessSpec `yaml:"tool_access"`
}

// CompileProjectIfPresent discovers tools/adapters directories and merges tools into model definitions.
// If the configured discovery root does not exist, it returns nil project with no error.
func CompileProjectIfPresent(configFilePath string, model *hyperterse.Model) (*Project, error) {
	baseDir := filepath.Dir(configFilePath)
	appDir, adaptersDir, toolsDir, agentsDir, agentToolAccessDefaults, err := resolveProjectDirectories(configFilePath)
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
		BaseDir:                 baseDir,
		AppDir:                  appDir,
		AdaptersDir:             adaptersDir,
		ToolsDir:                toolsDir,
		AgentsDir:               agentsDir,
		BuildDir:                buildDir,
		Tools:                   map[string]*Tool{},
		Agents:                  map[string]*Agent{},
		AgentToolAccessDefaults: agentToolAccessDefaults,
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

	sortedToolNames := make([]string, 0, len(project.Tools))
	for toolName := range project.Tools {
		sortedToolNames = append(sortedToolNames, toolName)
	}
	sort.Strings(sortedToolNames)

	agentTerseFiles, err := discoverAgentTerseFiles(agentsDir)
	if err != nil {
		return nil, err
	}
	sort.Strings(agentTerseFiles)

	for _, terseFile := range agentTerseFiles {
		agent, err := compileAgentFile(project, terseFile, sortedToolNames)
		if err != nil {
			return nil, err
		}
		if _, exists := project.Agents[agent.AgentName]; exists {
			return nil, fmt.Errorf("duplicate agent name generated from agents: %s", agent.AgentName)
		}
		project.Agents[agent.AgentName] = agent
		model.Agents = append(model.Agents, agent.Definition)
	}

	log.Infof("Compiled %d tool(s) and %d agent(s) into model", len(project.Tools), len(project.Agents))
	return project, nil
}

func resolveProjectDirectories(configFilePath string) (string, string, string, string, AgentToolAccessPolicy, error) {
	baseDir := filepath.Dir(configFilePath)
	rootDir := "app"
	toolsDirName := "tools"
	adaptersDirName := "adapters"
	agentsDirName := "agents"
	agentToolAccessDefaults := AgentToolAccessPolicy{Mode: AgentToolAccessModeAllowAll}

	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", "", "", "", AgentToolAccessPolicy{}, fmt.Errorf("failed to read config for discovery settings: %w", err)
	}

	var cfg discoveryConfigFile
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return "", "", "", "", AgentToolAccessPolicy{}, fmt.Errorf("failed to decode discovery settings: %w", err)
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
	if cfg.Agents.Directory != "" {
		agentsDirName = cfg.Agents.Directory
	}
	if cfg.Agents.ToolAccess != nil {
		parsedDefaults, err := parseRootAgentToolAccessPolicy(cfg.Agents.ToolAccess)
		if err != nil {
			return "", "", "", "", AgentToolAccessPolicy{}, err
		}
		agentToolAccessDefaults = parsedDefaults
	}

	appDir := resolveDiscoveryPath(baseDir, rootDir)
	adaptersDir := resolveDiscoveryPath(appDir, adaptersDirName)
	toolsDir := resolveDiscoveryPath(appDir, toolsDirName)
	agentsDir := resolveDiscoveryPath(appDir, agentsDirName)
	return appDir, adaptersDir, toolsDir, agentsDir, agentToolAccessDefaults, nil
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

func discoverAgentTerseFiles(agentsDir string) ([]string, error) {
	var files []string
	if _, err := os.Stat(agentsDir); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat agents dir: %w", err)
	}
	err := filepath.WalkDir(agentsDir, func(path string, d fs.DirEntry, walkErr error) error {
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
		return nil, fmt.Errorf("failed to discover agent .terse files: %w", err)
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

func compileAgentFile(project *Project, terseFile string, allToolNames []string) (*Agent, error) {
	content, err := os.ReadFile(terseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config %s: %w", terseFile, err)
	}

	var cfg AgentFileConfig
	if err := strictYAMLUnmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse agent config %s: %w", terseFile, err)
	}

	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid agent config %s: field 'name' is required", terseFile)
	}
	if !agentNamePattern.MatchString(name) {
		return nil, fmt.Errorf("invalid agent config %s: field 'name' must match %s", terseFile, agentNamePattern.String())
	}
	instruction := strings.TrimSpace(cfg.Instruction)
	if instruction == "" {
		return nil, fmt.Errorf("invalid agent config %s: field 'instruction' is required", terseFile)
	}
	if cfg.Model == nil {
		return nil, fmt.Errorf("invalid agent config %s: field 'model' is required", terseFile)
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Model.Provider))
	if provider == "" {
		return nil, fmt.Errorf("invalid agent config %s: field 'model.provider' is required", terseFile)
	}
	modelName := strings.TrimSpace(cfg.Model.Model)
	if modelName == "" {
		return nil, fmt.Errorf("invalid agent config %s: field 'model.model' is required", terseFile)
	}

	declaredToolPolicy, err := parseAgentToolAccessPolicy(cfg.ToolAccess)
	if err != nil {
		return nil, fmt.Errorf("invalid agent config %s: %w", terseFile, err)
	}

	knownTools := make(map[string]struct{}, len(allToolNames))
	for _, toolName := range allToolNames {
		knownTools[toolName] = struct{}{}
	}
	effectiveMode, effectiveTools, inherited, err := resolveEffectiveAgentToolAccess(
		declaredToolPolicy,
		project.AgentToolAccessDefaults,
		knownTools,
		allToolNames,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid agent config %s: %w", terseFile, err)
	}

	toolAccessProto := &hyperterse.AgentToolAccessConfig{
		Mode:  string(effectiveMode),
		Tools: append([]string(nil), effectiveTools...),
	}
	modelConfigProto := &hyperterse.AgentModelConfig{
		Provider: provider,
		Model:    modelName,
		Options:  stringifyAgentModelOptions(cfg.Model.Options),
	}
	definition := &hyperterse.Agent{
		Name:        name,
		Description: strings.TrimSpace(cfg.Description),
		Instruction: instruction,
		Model:       modelConfigProto,
		ToolAccess:  toolAccessProto,
	}

	agent := &Agent{
		AgentName:  name,
		Directory:  filepath.Dir(terseFile),
		TerseFile:  terseFile,
		Definition: definition,
		ToolAccess: AgentToolAccessMetadata{
			DeclaredMode:   declaredToolPolicy.Mode,
			DeclaredTools:  append([]string(nil), declaredToolPolicy.Tools...),
			EffectiveMode:  effectiveMode,
			EffectiveTools: append([]string(nil), effectiveTools...),
			Inherited:      inherited,
		},
	}
	return agent, nil
}

func stringifyAgentModelOptions(raw map[string]any) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func parseRootAgentToolAccessPolicy(spec *agentToolAccessSpec) (AgentToolAccessPolicy, error) {
	if spec == nil {
		return AgentToolAccessPolicy{Mode: AgentToolAccessModeAllowAll}, nil
	}
	mode, err := parseAgentToolAccessMode(spec.Mode, false)
	if err != nil {
		return AgentToolAccessPolicy{}, fmt.Errorf("invalid agents.tool_access.mode: %w", err)
	}
	tools := normalizeAgentToolNames(spec.Tools)
	if mode == AgentToolAccessModeAllowList && len(tools) == 0 {
		return AgentToolAccessPolicy{}, fmt.Errorf("agents.tool_access.tools is required when mode is %q", AgentToolAccessModeAllowList)
	}
	if mode != AgentToolAccessModeAllowList && len(tools) > 0 {
		return AgentToolAccessPolicy{}, fmt.Errorf("agents.tool_access.tools is only supported when mode is %q", AgentToolAccessModeAllowList)
	}
	return AgentToolAccessPolicy{Mode: mode, Tools: tools}, nil
}

func parseAgentToolAccessPolicy(spec *agentToolAccessSpec) (AgentToolAccessPolicy, error) {
	if spec == nil {
		return AgentToolAccessPolicy{Mode: AgentToolAccessModeInherit}, nil
	}
	mode, err := parseAgentToolAccessMode(spec.Mode, true)
	if err != nil {
		return AgentToolAccessPolicy{}, fmt.Errorf("field 'tool_access.mode': %w", err)
	}
	tools := normalizeAgentToolNames(spec.Tools)
	if mode == AgentToolAccessModeInherit && len(tools) > 0 {
		return AgentToolAccessPolicy{}, fmt.Errorf("field 'tool_access.tools' cannot be set when mode is %q", AgentToolAccessModeInherit)
	}
	if mode == AgentToolAccessModeAllowList && len(tools) == 0 {
		return AgentToolAccessPolicy{}, fmt.Errorf("field 'tool_access.tools' is required when mode is %q", AgentToolAccessModeAllowList)
	}
	if mode != AgentToolAccessModeAllowList && mode != AgentToolAccessModeInherit && len(tools) > 0 {
		return AgentToolAccessPolicy{}, fmt.Errorf("field 'tool_access.tools' is only supported when mode is %q", AgentToolAccessModeAllowList)
	}
	return AgentToolAccessPolicy{Mode: mode, Tools: tools}, nil
}

func parseAgentToolAccessMode(modeRaw string, allowInherit bool) (AgentToolAccessMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(modeRaw))
	if normalized == "" {
		if allowInherit {
			return AgentToolAccessModeInherit, nil
		}
		return AgentToolAccessModeAllowAll, nil
	}
	switch AgentToolAccessMode(normalized) {
	case AgentToolAccessModeAllowAll, AgentToolAccessModeAllowNone, AgentToolAccessModeAllowList:
		return AgentToolAccessMode(normalized), nil
	case AgentToolAccessModeInherit:
		if !allowInherit {
			return "", fmt.Errorf("must be one of: %q, %q, %q", AgentToolAccessModeAllowAll, AgentToolAccessModeAllowNone, AgentToolAccessModeAllowList)
		}
		return AgentToolAccessModeInherit, nil
	default:
		if allowInherit {
			return "", fmt.Errorf("must be one of: %q, %q, %q, %q", AgentToolAccessModeInherit, AgentToolAccessModeAllowAll, AgentToolAccessModeAllowNone, AgentToolAccessModeAllowList)
		}
		return "", fmt.Errorf("must be one of: %q, %q, %q", AgentToolAccessModeAllowAll, AgentToolAccessModeAllowNone, AgentToolAccessModeAllowList)
	}
}

func normalizeAgentToolNames(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, toolName := range raw {
		name := strings.TrimSpace(toolName)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func resolveEffectiveAgentToolAccess(
	declared AgentToolAccessPolicy,
	defaults AgentToolAccessPolicy,
	knownTools map[string]struct{},
	allToolNames []string,
) (AgentToolAccessMode, []string, bool, error) {
	effective := declared
	inherited := false
	if declared.Mode == AgentToolAccessModeInherit {
		effective = defaults
		inherited = true
	}
	if effective.Mode == AgentToolAccessModeInherit || effective.Mode == "" {
		return "", nil, inherited, fmt.Errorf("failed to resolve effective tool access policy")
	}

	switch effective.Mode {
	case AgentToolAccessModeAllowAll:
		return effective.Mode, append([]string(nil), allToolNames...), inherited, nil
	case AgentToolAccessModeAllowNone:
		return effective.Mode, []string{}, inherited, nil
	case AgentToolAccessModeAllowList:
		if len(effective.Tools) == 0 {
			return "", nil, inherited, fmt.Errorf("tool allowlist cannot be empty when mode is %q", AgentToolAccessModeAllowList)
		}
		allowlist := make([]string, 0, len(effective.Tools))
		for _, toolName := range effective.Tools {
			if _, ok := knownTools[toolName]; !ok {
				return "", nil, inherited, fmt.Errorf("unknown tool in allowlist: %q", toolName)
			}
			allowlist = append(allowlist, toolName)
		}
		return effective.Mode, allowlist, inherited, nil
	default:
		return "", nil, inherited, fmt.Errorf("unsupported tool access mode: %q", effective.Mode)
	}
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
	// DB-backed tools should not auto-discover handler scripts. Handler convention
	// discovery is only for script-backed tools (no adapter binding).
	allowHandlerConvention := len(tool.Definition.Use) == 0
	for _, entry := range entries {
		if entry.IsDir() || !tsConventionPattern.MatchString(entry.Name()) {
			continue
		}
		fileName := strings.ToLower(entry.Name())
		baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		fullPath := filepath.Join(tool.Directory, entry.Name())
		if allowHandlerConvention && tool.Scripts.Handler == "" && (baseName == "handler" || strings.Contains(fileName, "handler")) {
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

func parseToolUseBinding(raw any) ([]string, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		binding := strings.TrimSpace(v)
		if binding == "" {
			return nil, nil
		}
		return []string{binding}, nil
	case []any:
		if len(v) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("field 'use' must be a single adapter name (string), arrays are not supported")
	default:
		return nil, fmt.Errorf("field 'use' must be a string")
	}
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

	useBindings, err := parseToolUseBinding(cfg.Use)
	if err != nil {
		return nil, err
	}
	tool.Use = useBindings

	hasUse := len(useBindings) > 0
	hasHandler := strings.TrimSpace(cfg.Handler) != ""
	if hasUse && hasHandler {
		return nil, fmt.Errorf("tool cannot define both 'use' and 'handler'; choose one")
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
