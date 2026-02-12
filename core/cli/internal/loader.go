package internal

import (
	"os"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// LoadConfig loads and parses a configuration file, returning the model (which includes server config)
func LoadConfig(filePath string) (*hyperterse.Model, error) {
	log := logger.New("parser")

	log.Debugf("Loading configuration file")
	log.Debugf("File path: %s", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, log.Errorf("error reading file: %w", err)
	}

	log.Debugf("File size: %d bytes", len(content))

	var model *hyperterse.Model
	parserType := ""

	// Determine parser based on file extension
	if strings.HasSuffix(filePath, ".terse") {
		parserType = "YAML"
		log.Debugf("Parsing configuration with YAML parser")
		model, err = parser.ParseYAMLWithConfig(content)
		if err != nil {
			return nil, log.Errorf("config error: %w", err)
		}
	} else {
		parserType = "DSL"
		log.Debugf("Parsing configuration with DSL parser")
		p := parser.NewParser(string(content))
		model, err = p.Parse()
		if err != nil {
			return nil, log.Errorf("parsing error: %w", err)
		}
	}

	log.Debugf("Parser type: %s", parserType)
	log.Debugf("Configuration parsed successfully")

	return model, nil
}

// LoadConfigFromString loads and parses a configuration from a YAML string, returning the model
func LoadConfigFromString(yamlContent string) (*hyperterse.Model, error) {
	log := logger.New("parser")

	log.Debugf("Loading configuration from string")
	log.Debugf("Content length: %d bytes", len(yamlContent))
	log.Debugf("Parsing configuration with YAML parser")

	model, err := parser.ParseYAMLWithConfig([]byte(yamlContent))
	if err != nil {
		return nil, log.Errorf("config error: %w", err)
	}

	log.Debugf("Configuration parsed successfully")
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

// ResolveOTLPEndpoint resolves the OTLP endpoint from CLI/env/default.
func ResolveOTLPEndpoint() string {
	if envEndpoint := os.Getenv("HYPERTERSE_OTEL_ENDPOINT"); envEndpoint != "" {
		return envEndpoint
	}
	return "localhost:4317"
}
