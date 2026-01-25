package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Environment variable pattern: {{ env.VARIABLE_NAME }}
	envVarPattern = regexp.MustCompile(`\{\{\s*env\.(\w+)\s*\}\}`)
)

// SubstituteEnvVars replaces {{ env.VARIABLE_NAME }} placeholders with environment variable values
// This is called at runtime (server startup/connection time) to prevent sensitive data from being
// baked into the final bundle. Only allowed in connection_string and statement fields.
func SubstituteEnvVars(value string) (string, error) {
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
