package observability

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

type Config struct {
	Enabled           bool
	TracesEnabled     bool
	MetricsEnabled    bool
	LogsEnabled       bool
	ServiceName       string
	ServiceVersion    string
	Environment       string
	OTLPEndpoint      string
	OTLPProtocol      string
	TraceSamplingRate float64
}

var envVarPattern = regexp.MustCompile(`\{\{\s*env\.(\w+)\s*\}\}`)

func ResolveConfig(_ *hyperterse.Model) (Config, error) {
	cfg := Config{
		Enabled:           false,
		TracesEnabled:     true,
		MetricsEnabled:    true,
		LogsEnabled:       true,
		ServiceName:       "hyperterse",
		ServiceVersion:    "dev",
		Environment:       "development",
		OTLPEndpoint:      "localhost:4317",
		OTLPProtocol:      "grpc",
		TraceSamplingRate: 1.0,
	}

	// Environment variable controls for paid observability features.
	overrideBool("HYPERTERSE_OTEL_ENABLED", &cfg.Enabled)
	overrideBool("HYPERTERSE_OTEL_TRACES_ENABLED", &cfg.TracesEnabled)
	overrideBool("HYPERTERSE_OTEL_METRICS_ENABLED", &cfg.MetricsEnabled)
	overrideBool("HYPERTERSE_OTEL_LOGS_ENABLED", &cfg.LogsEnabled)
	overrideString("HYPERTERSE_OTEL_SERVICE_NAME", &cfg.ServiceName)
	overrideString("HYPERTERSE_OTEL_SERVICE_VERSION", &cfg.ServiceVersion)
	overrideString("HYPERTERSE_OTEL_ENVIRONMENT", &cfg.Environment)
	overrideString("HYPERTERSE_OTEL_ENDPOINT", &cfg.OTLPEndpoint)
	overrideString("HYPERTERSE_OTEL_PROTOCOL", &cfg.OTLPProtocol)
	overrideFloat("HYPERTERSE_OTEL_TRACE_SAMPLING_RATIO", &cfg.TraceSamplingRate)

	if cfg.TraceSamplingRate < 0 {
		cfg.TraceSamplingRate = 0
	}
	if cfg.TraceSamplingRate > 1 {
		cfg.TraceSamplingRate = 1
	}

	var err error
	cfg.ServiceName, err = substituteEnvVars(cfg.ServiceName)
	if err != nil {
		return Config{}, fmt.Errorf("resolve observability service name: %w", err)
	}
	cfg.ServiceVersion, err = substituteEnvVars(cfg.ServiceVersion)
	if err != nil {
		return Config{}, fmt.Errorf("resolve observability service version: %w", err)
	}
	cfg.Environment, err = substituteEnvVars(cfg.Environment)
	if err != nil {
		return Config{}, fmt.Errorf("resolve observability environment: %w", err)
	}
	cfg.OTLPEndpoint, err = substituteEnvVars(cfg.OTLPEndpoint)
	if err != nil {
		return Config{}, fmt.Errorf("resolve observability otlp endpoint: %w", err)
	}
	cfg.OTLPProtocol, err = substituteEnvVars(strings.ToLower(cfg.OTLPProtocol))
	if err != nil {
		return Config{}, fmt.Errorf("resolve observability otlp protocol: %w", err)
	}

	return cfg, nil
}

func overrideString(name string, target *string) {
	if value := os.Getenv(name); value != "" {
		*target = value
	}
}

func overrideBool(name string, target *bool) {
	value := os.Getenv(name)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseBool(value)
	if err == nil {
		*target = parsed
	}
}

func overrideFloat(name string, target *float64) {
	value := os.Getenv(name)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err == nil {
		*target = parsed
	}
}

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
		if seen[placeholder] {
			continue
		}
		seen[placeholder] = true

		envValue, exists := os.LookupEnv(envVarName)
		if !exists {
			return "", fmt.Errorf("environment variable '%s' not found", envVarName)
		}
		result = strings.ReplaceAll(result, placeholder, envValue)
	}

	return result, nil
}
