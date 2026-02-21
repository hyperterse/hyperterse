package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/types"
)

var (
	// log is the logger instance for the validator package
	log = logger.New("parser")
)

// ValidationErrors represents a collection of validation errors
type ValidationErrors struct {
	Errors []string
}

// Error implements the error interface
// Returns a simple message since detailed errors are already logged
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("validation failed with %d error(s):", len(ve.Errors)))
	for i, errMsg := range ve.Errors {
		builder.WriteString(fmt.Sprintf("\n  %d. %s", i+1, errMsg))
	}
	return builder.String()
}

// Format returns a formatted string representation of the errors
func (ve *ValidationErrors) Format() string {
	return ve.Error()
}

// Validate performs comprehensive validation on the Model
func Validate(model *hyperterse.Model) error {
	log.Infof("Starting validation")
	var errors []string

	// Tool name pattern: must start with a letter, lowercase only (lower-snake-case or lower-kebab-case)
	toolNamePattern := regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

	// 0. Validate name is required
	if model.Name == "" {
		errors = append(errors, "name is required")
	} else {
		// 0a. Name must start with a letter and be in lower-snake-case or lower-kebab-case
		if !toolNamePattern.MatchString(model.Name) {
			errors = append(errors, fmt.Sprintf("name '%s' is invalid. Must start with a letter and be in lower-snake-case or lower-kebab-case (lowercase letters, numbers, hyphens, and underscores only)", model.Name))
		}
	}

	// 0b. Validate optional tools.cache configuration
	if model.ToolDefaults != nil && model.ToolDefaults.Cache != nil {
		cache := model.ToolDefaults.Cache
		if !cache.HasEnabled {
			errors = append(errors, "tools.cache.enabled is required when tools.cache is specified")
		}
		if cache.HasTtl && cache.Ttl <= 0 {
			errors = append(errors, "tools.cache.ttl must be greater than 0 when specified")
		}
	}

	// 1. Validate adapters is required and has at least one entry
	if len(model.Adapters) == 0 {
		errors = append(errors, "adapters is required and should have at least one entry")
	}

	// Track adapter names for uniqueness and cross-reference validation
	adapterNames := make(map[string]bool)
	// Name pattern: must start with a letter, followed by letters, numbers, hyphens, and underscores
	namePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	for i, adapter := range model.Adapters {
		prefix := fmt.Sprintf("adapters[%d]", i)

		// 2. Adapter name is required
		if adapter.Name == "" {
			errors = append(errors, fmt.Sprintf("Adapter [%d] - requires a name", i))
			continue
		}

		prefix = adapter.Name

		// 2a. Adapter name must start with a letter
		if !namePattern.MatchString(adapter.Name) {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - name is invalid. Must start with a letter and can contain letters, numbers, hyphens, and underscores", adapter.Name))
		}

		// 2. Adapter name must be unique
		if adapterNames[adapter.Name] {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - already defined. Adapters must be unique", adapter.Name))
		}
		adapterNames[adapter.Name] = true

		// 3. Connector is required and must be one of: postgres, redis, mysql, mongodb
		if adapter.Connector == connectors.Connector_CONNECTOR_UNSPECIFIED {
			errors = append(errors, fmt.Sprintf("Adapter '%s' requires a connector", prefix))
		} else if adapter.Connector != connectors.Connector_CONNECTOR_POSTGRES &&
			adapter.Connector != connectors.Connector_CONNECTOR_REDIS &&
			adapter.Connector != connectors.Connector_CONNECTOR_MYSQL &&
			adapter.Connector != connectors.Connector_CONNECTOR_MONGODB {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - connector is invalid. Must be one of: %s", prefix, strings.Join(types.GetValidConnectors(), ", ")))
		}

		// 4. Connection string is required
		if adapter.ConnectionString == "" {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - connection_string is required", prefix))
		}
	}

	// 5. Validate tools is required and has at least one entry
	if len(model.Tools) == 0 {
		errors = append(errors, "tools is required and should have at least one entry")
	}

	// Build list of adapter names for error messages
	var adapterNameList []string
	for name := range adapterNames {
		adapterNameList = append(adapterNameList, name)
	}

	// Track tool names for uniqueness
	toolNames := make(map[string]bool)

	for i, tool := range model.Tools {
		prefix := fmt.Sprintf("tools[%d]", i)

		// 6. Tool name is required
		if tool.Name == "" {
			errors = append(errors, fmt.Sprintf("Tool [%d] - name is required", i))
			continue
		}

		prefix = tool.Name

		// 6. Tool name must be unique
		if toolNames[tool.Name] {
			errors = append(errors, fmt.Sprintf("Tool '%s' - name must be unique", tool.Name))
		}
		toolNames[tool.Name] = true

		// 6a. Tool name must start with a letter and be in lower-snake-case or lower-kebab-case
		if !toolNamePattern.MatchString(tool.Name) {
			errors = append(errors, fmt.Sprintf("Tool '%s' - name is invalid. Must start with a letter and be in lower-snake-case or lower-kebab-case (lowercase letters, numbers, hyphens, and underscores only)", tool.Name))
		}
		// 7. Tool use is required and must reference a valid adapter
		if len(tool.Use) == 0 {
			errors = append(errors, fmt.Sprintf("Tool '%s' - use is required", prefix))
		} else {
			for _, useAdapter := range tool.Use {
				if !adapterNames[useAdapter] {
					errors = append(errors, fmt.Sprintf("Tool '%s' - use '%s' is invalid. Must reference one of the defined adapter names: %s", prefix, useAdapter, strings.Join(adapterNameList, ", ")))
				}
			}
		}

		// 8. Tool description is required
		if tool.Description == "" {
			errors = append(errors, fmt.Sprintf("%s.description is required", prefix))
		}

		// 9. Tool statement is required
		if tool.Statement == "" {
			errors = append(errors, fmt.Sprintf("%s.statement is required", prefix))
		}

		// 10. Validate inputs if specified
		inputNames := make(map[string]bool)
		for j, input := range tool.Inputs {
			inputPrefix := fmt.Sprintf("%s.inputs[%d]", prefix, j)

			// Input name is required
			if input.Name == "" {
				errors = append(errors, fmt.Sprintf("%s.name is required", inputPrefix))
				continue
			}

			// Input name must start with a letter
			if !namePattern.MatchString(input.Name) {
				errors = append(errors, fmt.Sprintf("%s.name '%s' is invalid. Must start with a letter and can contain letters, numbers, hyphens, and underscores", inputPrefix, input.Name))
			}

			// Input name must be unique within the tool
			if inputNames[input.Name] {
				errors = append(errors, fmt.Sprintf("%s.name '%s' must be unique within the tool", inputPrefix, input.Name))
			}
			inputNames[input.Name] = true

			// Input type is required and must be valid
			typeStr := types.PrimitiveEnumToString(input.Type)
			if typeStr == "" {
				errors = append(errors, fmt.Sprintf("%s.type is required", inputPrefix))
			} else if !types.IsValidPrimitiveType(typeStr) {
				errors = append(errors, fmt.Sprintf("%s.type '%s' must be one of: %s", inputPrefix, typeStr, strings.Join(types.GetValidPrimitives(), ", ")))
			}

			// If input is optional, it must have a default value
			// We check if DefaultValue is empty, but since we now convert all default values to strings,
			// an empty string default is only possible if explicitly set to "" or if the key was missing.
			// The parser ensures that if the key exists, DefaultValue is populated.
			if input.Optional && input.DefaultValue == "" {
				errors = append(errors, fmt.Sprintf("%s is marked as optional but does not have a default value", inputPrefix))
			}
		}

		// 10a. Validate that all {{ inputs.x }} references in statement are defined
		if tool.Statement != "" {
			referencedInputs := extractInputReferences(tool.Statement)
			if len(referencedInputs) > 0 {
				// If statement references inputs, inputs must be defined
				if len(tool.Inputs) == 0 {
					errors = append(errors, fmt.Sprintf("%s.statement references inputs but %s.inputs is not specified", prefix, prefix))
				} else {
					// Check that all referenced inputs exist
					for _, refInput := range referencedInputs {
						if !inputNames[refInput] {
							errors = append(errors, fmt.Sprintf("%s.statement references '{{ inputs.%s }}' but %s.inputs does not contain '%s'", prefix, refInput, prefix, refInput))
						}
					}
				}
			}
		}

		// 11. Validate optional tool.cache override
		if tool.Cache != nil {
			if !tool.Cache.HasEnabled {
				errors = append(errors, fmt.Sprintf("%s.cache.enabled is required when %s.cache is specified", prefix, prefix))
			}
			if tool.Cache.HasTtl && tool.Cache.Ttl <= 0 {
				errors = append(errors, fmt.Sprintf("%s.cache.ttl must be greater than 0 when specified", prefix))
			}
		}
	}

	if len(errors) > 0 {
		return log.Errorf("%w", &ValidationErrors{Errors: errors})
	}

	log.Infof("Validation completed successfully")
	return nil
}

// extractInputReferences extracts all input names referenced in the statement
// using the pattern {{ inputs.x }} and returns them as a unique set
func extractInputReferences(statement string) []string {
	// Regex to match {{ inputs.x }} pattern
	// Matches: {{ inputs. followed by identifier, then }}
	inputRefRegex := regexp.MustCompile(`\{\{\s*inputs\.([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

	matches := inputRefRegex.FindAllStringSubmatch(statement, -1)
	if len(matches) == 0 {
		return nil
	}

	// Use a map to track unique input names
	seen := make(map[string]bool)
	var result []string

	for _, match := range matches {
		if len(match) >= 2 {
			inputName := match[1]
			if !seen[inputName] {
				seen[inputName] = true
				result = append(result, inputName)
			}
		}
	}

	return result
}
