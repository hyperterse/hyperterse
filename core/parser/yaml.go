package parser

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"gopkg.in/yaml.v3"
)

// ParseYAML parses YAML content into a protobuf Model
func ParseYAML(data []byte) (*hyperterse.Model, error) {
	model, err := ParseYAMLWithConfig(data)
	return model, err
}

// ParseYAMLWithConfig parses YAML content into a protobuf Model with ServerConfig
func ParseYAMLWithConfig(data []byte) (*hyperterse.Model, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	model := &hyperterse.Model{}

	// Parse name (required)
	if nameRaw, ok := raw["name"].(string); ok {
		model.Name = nameRaw
	}
	// Parse optional version
	if versionRaw, ok := raw["version"].(string); ok {
		model.Version = versionRaw
	}

	// Parse build configuration
	if buildRaw, ok := raw["build"].(map[string]any); ok {
		buildConfig := &hyperterse.ExportConfig{}

		// Check for out (directory)
		if outRaw, ok := buildRaw["out"].(string); ok && outRaw != "" {
			buildConfig.Out = outRaw
		}
		// Alias for out (directory)
		if buildConfig.Out == "" {
			if outDirRaw, ok := buildRaw["out_dir"].(string); ok && outDirRaw != "" {
				buildConfig.Out = outDirRaw
			}
		}

		// Check for clean_dir
		if cleanDirRaw, ok := buildRaw["clean_dir"].(bool); ok {
			buildConfig.CleanDir = cleanDirRaw
		}

		if buildConfig.Out != "" || buildConfig.CleanDir {
			model.Export = buildConfig
		}
	}

	// Parse server configuration
	if serverRaw, ok := raw["server"].(map[string]any); ok {
		serverConfig := &hyperterse.ServerConfig{}

		// Parse port
		if portRaw, ok := serverRaw["port"]; ok {
			switch v := portRaw.(type) {
			case int:
				serverConfig.Port = fmt.Sprintf("%d", v)
			case string:
				serverConfig.Port = v
			}
		}

		// Parse log_level
		if logLevelRaw, ok := serverRaw["log_level"]; ok {
			switch v := logLevelRaw.(type) {
			case int:
				serverConfig.LogLevel = int32(v)
			case float64:
				serverConfig.LogLevel = int32(v)
			}
		}

		model.Server = serverConfig
	}

	// Parse tools root config (v2 discovery + global cache defaults).
	// This maps tools.cache into model.tool_defaults.
	if toolsRaw, ok := raw["tools"].(map[string]any); ok {
		if cacheRaw, ok := toolsRaw["cache"].(map[string]any); ok {
			cacheConfig := parseCacheConfig(cacheRaw)
			if cacheConfig != nil {
				if model.ToolDefaults == nil {
					model.ToolDefaults = &hyperterse.ToolDefaultsConfig{}
				}
				model.ToolDefaults.Cache = cacheConfig
			}
		}
	}

	return model, nil
}

func parseCacheConfig(cacheRaw map[string]any) *hyperterse.CacheConfig {
	cacheConfig := &hyperterse.CacheConfig{}
	hasAnyField := false

	if enabledRaw, ok := cacheRaw["enabled"]; ok {
		if enabled, ok := enabledRaw.(bool); ok {
			cacheConfig.Enabled = enabled
			cacheConfig.HasEnabled = true
			hasAnyField = true
		}
	}

	if ttlRaw, ok := cacheRaw["ttl"]; ok {
		switch v := ttlRaw.(type) {
		case int:
			cacheConfig.Ttl = int32(v)
			cacheConfig.HasTtl = true
			hasAnyField = true
		case float64:
			cacheConfig.Ttl = int32(v)
			cacheConfig.HasTtl = true
			hasAnyField = true
		}
	}

	if !hasAnyField {
		return nil
	}

	return cacheConfig
}
