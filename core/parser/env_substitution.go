package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

var (
	// Environment variable pattern: {{ env.VARIABLE_NAME }}
	envVarPattern = regexp.MustCompile(`\{\{\s*env\.(\w+)\s*\}\}`)
)

// substituteEnvVars replaces {{ env.VARIABLE_NAME }} placeholders with environment variable values
func substituteEnvVars(value string) (string, error) {
	result := value
	matches := envVarPattern.FindAllStringSubmatch(value, -1)
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		envVarName := match[1]
		placeholder := match[0]

		// Avoid processing the same placeholder multiple times
		if seen[placeholder] {
			continue
		}
		seen[placeholder] = true

		// Get the environment variable value
		envValue, exists := os.LookupEnv(envVarName)
		if !exists {
			return "", fmt.Errorf("environment variable '%s' not found (required at server startup)", envVarName)
		}

		// Replace all occurrences of this placeholder
		result = strings.ReplaceAll(result, placeholder, envValue)
	}

	return result, nil
}

// SubstituteEnvVarAndConvertConnector substitutes environment variables in connector string
// This is called during parsing (before enum conversion) to handle connector field substitution
// Returns the substituted string or an error if environment variable substitution fails
func SubstituteEnvVarAndConvertConnector(connectorStr string) (string, error) {
	substituted, err := substituteEnvVars(connectorStr)
	if err != nil {
		return "", fmt.Errorf("configuration error at server startup: failed to substitute environment variables in connector: %w", err)
	}
	return substituted, nil
}

// SubstituteEnvVarsInModel performs environment variable substitution on all string fields in the model
// This is called once after parsing to ensure all environment variables are substituted at server startup
func SubstituteEnvVarsInModel(model *hyperterse.Model) error {
	// Substitute in server config
	if model.Server != nil && model.Server.Port != "" {
		substituted, err := substituteEnvVars(model.Server.Port)
		if err != nil {
			return fmt.Errorf("configuration error at server startup: failed to substitute environment variables in port: %w", err)
		}
		model.Server.Port = substituted
	}

	// Substitute in adapters
	for _, adapter := range model.Adapters {
		if adapter.ConnectionString != "" {
			substituted, err := substituteEnvVars(adapter.ConnectionString)
			if err != nil {
				return fmt.Errorf("configuration error at server startup: failed to substitute environment variables in connection_string for adapter '%s': %w", adapter.Name, err)
			}
			adapter.ConnectionString = substituted
		}

		// Substitute in adapter options
		if adapter.Options != nil {
			for key, value := range adapter.Options.Options {
				substituted, err := substituteEnvVars(value)
				if err != nil {
					return fmt.Errorf("configuration error at server startup: failed to substitute environment variables in option '%s' for adapter '%s': %w", key, adapter.Name, err)
				}
				adapter.Options.Options[key] = substituted
			}
		}
	}

	// Substitute in query input default values
	for _, query := range model.Queries {
		for _, input := range query.Inputs {
			if input.DefaultValue != "" {
				substituted, err := substituteEnvVars(input.DefaultValue)
				if err != nil {
					return fmt.Errorf("configuration error at server startup: failed to substitute environment variables in default value for input '%s' in query '%s': %w", input.Name, query.Name, err)
				}
				input.DefaultValue = substituted
			}
		}
	}

	return nil
}
