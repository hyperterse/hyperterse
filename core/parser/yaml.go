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

	// Parse tools root config (v2 discovery + global defaults).
	// This maps tools.cache/tools.search into model.tool_defaults.
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
		if searchRaw, ok := toolsRaw["search"].(map[string]any); ok {
			searchConfig := parseSearchConfig(searchRaw)
			if searchConfig != nil {
				if model.ToolDefaults == nil {
					model.ToolDefaults = &hyperterse.ToolDefaultsConfig{}
				}
				model.ToolDefaults.Search = searchConfig
			}
		}
	}

	// Parse optional inline prompts list. Discovery settings use prompts.directory
	// and are handled by the framework compiler.
	if promptsRaw, ok := raw["prompts"].([]any); ok {
		for _, entry := range promptsRaw {
			promptMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			prompt := &hyperterse.PromptDefinition{}
			if name, ok := promptMap["name"].(string); ok {
				prompt.Name = name
			}
			if title, ok := promptMap["title"].(string); ok {
				prompt.Title = title
			}
			if description, ok := promptMap["description"].(string); ok {
				prompt.Description = description
			}
			if argumentsRaw, ok := promptMap["arguments"].([]any); ok {
				for _, argumentEntry := range argumentsRaw {
					argumentMap, ok := argumentEntry.(map[string]any)
					if !ok {
						continue
					}
					argument := &hyperterse.PromptArgument{}
					if name, ok := argumentMap["name"].(string); ok {
						argument.Name = name
					}
					if title, ok := argumentMap["title"].(string); ok {
						argument.Title = title
					}
					if description, ok := argumentMap["description"].(string); ok {
						argument.Description = description
					}
					if required, ok := argumentMap["required"].(bool); ok {
						argument.Required = required
					}
					if completionRaw, ok := argumentMap["completion"].([]any); ok {
						for _, completionEntry := range completionRaw {
							if completionValue, ok := completionEntry.(string); ok {
								argument.Completion = append(argument.Completion, completionValue)
							}
						}
					}
					prompt.Arguments = append(prompt.Arguments, argument)
				}
			}
			if messagesRaw, ok := promptMap["messages"].([]any); ok {
				for _, messageEntry := range messagesRaw {
					messageMap, ok := messageEntry.(map[string]any)
					if !ok {
						continue
					}
					message := &hyperterse.PromptMessage{}
					if role, ok := messageMap["role"].(string); ok {
						message.Role = role
					}
					if text, ok := messageMap["text"].(string); ok {
						message.Text = text
					}
					prompt.Messages = append(prompt.Messages, message)
				}
			}
			model.Prompts = append(model.Prompts, prompt)
		}
	}

	// Parse optional inline resources list. Discovery settings use resources.directory
	// and are handled by the framework compiler.
	if resourcesRaw, ok := raw["resources"].([]any); ok {
		for _, entry := range resourcesRaw {
			resourceMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			resource := &hyperterse.ResourceDefinition{}
			if uri, ok := resourceMap["uri"].(string); ok {
				resource.Uri = uri
			}
			if name, ok := resourceMap["name"].(string); ok {
				resource.Name = name
			}
			if title, ok := resourceMap["title"].(string); ok {
				resource.Title = title
			}
			if description, ok := resourceMap["description"].(string); ok {
				resource.Description = description
			}
			if mimeType, ok := resourceMap["mime_type"].(string); ok {
				resource.MimeType = mimeType
			}
			if text, ok := resourceMap["text"].(string); ok {
				resource.Text = text
			}
			if filePath, ok := resourceMap["file"].(string); ok {
				resource.File = filePath
			}
			model.Resources = append(model.Resources, resource)
		}
	}

	if resourceTemplatesRaw, ok := raw["resource_templates"].([]any); ok {
		for _, entry := range resourceTemplatesRaw {
			templateMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			template := &hyperterse.ResourceTemplateDefinition{}
			if uriTemplate, ok := templateMap["uri_template"].(string); ok {
				template.UriTemplate = uriTemplate
			}
			if name, ok := templateMap["name"].(string); ok {
				template.Name = name
			}
			if title, ok := templateMap["title"].(string); ok {
				template.Title = title
			}
			if description, ok := templateMap["description"].(string); ok {
				template.Description = description
			}
			if mimeType, ok := templateMap["mime_type"].(string); ok {
				template.MimeType = mimeType
			}
			if textTemplate, ok := templateMap["text_template"].(string); ok {
				template.TextTemplate = textTemplate
			}
			if fileTemplate, ok := templateMap["file_template"].(string); ok {
				template.FileTemplate = fileTemplate
			}
			if argumentsRaw, ok := templateMap["arguments"].([]any); ok {
				for _, argumentEntry := range argumentsRaw {
					argumentMap, ok := argumentEntry.(map[string]any)
					if !ok {
						continue
					}
					argument := &hyperterse.ResourceTemplateArgument{}
					if name, ok := argumentMap["name"].(string); ok {
						argument.Name = name
					}
					if title, ok := argumentMap["title"].(string); ok {
						argument.Title = title
					}
					if description, ok := argumentMap["description"].(string); ok {
						argument.Description = description
					}
					if required, ok := argumentMap["required"].(bool); ok {
						argument.Required = required
					}
					if completionRaw, ok := argumentMap["completion"].([]any); ok {
						for _, completionEntry := range completionRaw {
							if completionValue, ok := completionEntry.(string); ok {
								argument.Completion = append(argument.Completion, completionValue)
							}
						}
					}
					template.Arguments = append(template.Arguments, argument)
				}
			}
			model.ResourceTemplates = append(model.ResourceTemplates, template)
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

func parseSearchConfig(searchRaw map[string]any) *hyperterse.SearchConfig {
	searchConfig := &hyperterse.SearchConfig{}
	hasAnyField := false

	if limitRaw, ok := searchRaw["limit"]; ok {
		switch v := limitRaw.(type) {
		case int:
			searchConfig.Limit = int32(v)
			searchConfig.HasLimit = true
			hasAnyField = true
		case float64:
			searchConfig.Limit = int32(v)
			searchConfig.HasLimit = true
			hasAnyField = true
		}
	}

	if !hasAnyField {
		return nil
	}

	return searchConfig
}
