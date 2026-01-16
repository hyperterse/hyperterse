package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// LoadConfig loads and parses a configuration file, returning the model (which includes server config)
func LoadConfig(filePath string) (*hyperterse.Model, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var model *hyperterse.Model

	// Determine parser based on file extension
	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		model, err = parser.ParseYAMLWithConfig(content)
		if err != nil {
			return nil, fmt.Errorf("config error: %w", err)
		}
	} else {
		// Default to DSL parser for .hyperterse files
		p := parser.NewParser(string(content))
		model, err = p.Parse()
		if err != nil {
			return nil, fmt.Errorf("parsing error: %w", err)
		}
	}

	return model, nil
}

// LoadConfigFromString loads and parses a configuration from a YAML string, returning the model
func LoadConfigFromString(yamlContent string) (*hyperterse.Model, error) {
	model, err := parser.ParseYAMLWithConfig([]byte(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return model, nil
}

// ResolvePort resolves the port from CLI flag, config file, env var, or default
func ResolvePort(cliPort string, model *hyperterse.Model) string {
	if cliPort != "" {
		return cliPort
	}
	if model != nil && model.Server != nil && model.Server.Port != "" {
		return model.Server.Port
	}
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

// ResolveLogLevel resolves the log level from verbose flag, CLI flag, config file, or default
func ResolveLogLevel(verbose bool, cliLogLevel int, model *hyperterse.Model) int {
	if verbose {
		return logger.LogLevelDebug
	}
	if cliLogLevel > 0 {
		return cliLogLevel
	}
	if model != nil && model.Server != nil && model.Server.LogLevel > 0 {
		return int(model.Server.LogLevel)
	}
	return logger.LogLevelInfo
}
