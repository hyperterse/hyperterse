package agents

import (
	"context"
	"fmt"
	"os"
	"strings"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func resolveAgentModel(ctx context.Context, cfg *hyperterse.AgentModelConfig) (adkmodel.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("agent model config is required")
	}
	provider := normalizeProviderName(cfg.Provider)
	switch provider {
	case "gemini", "google_ai_studio":
		return resolveGeminiModel(ctx, cfg)
	case "vertex", "vertex_ai":
		return resolveVertexModel(ctx, cfg)
	case "openai_compatible", "openai":
		return resolveOpenAICompatibleModel(cfg)
	default:
		return nil, fmt.Errorf("unsupported agent model provider %q", cfg.Provider)
	}
}

func resolveGeminiModel(ctx context.Context, cfg *hyperterse.AgentModelConfig) (adkmodel.LLM, error) {
	clientConfig := &genai.ClientConfig{
		APIKey: resolveSecretOption(cfg.Options, "api_key", "api_key_env", "GOOGLE_API_KEY"),
	}
	return gemini.NewModel(ctx, cfg.Model, clientConfig)
}

func resolveVertexModel(ctx context.Context, cfg *hyperterse.AgentModelConfig) (adkmodel.LLM, error) {
	clientConfig := &genai.ClientConfig{
		Backend: genai.BackendVertexAI,
	}
	if project := resolveStringOption(cfg.Options, "project", "GOOGLE_CLOUD_PROJECT"); project != "" {
		clientConfig.Project = project
	}
	if location := resolveStringOption(cfg.Options, "location", "GOOGLE_CLOUD_LOCATION"); location != "" {
		clientConfig.Location = location
	}
	if clientConfig.Location == "" {
		clientConfig.Location = os.Getenv("GOOGLE_CLOUD_REGION")
	}
	if apiKey := resolveSecretOption(cfg.Options, "api_key", "api_key_env", "GOOGLE_API_KEY"); apiKey != "" {
		clientConfig.APIKey = apiKey
	}
	return gemini.NewModel(ctx, cfg.Model, clientConfig)
}

func resolveOpenAICompatibleModel(cfg *hyperterse.AgentModelConfig) (adkmodel.LLM, error) {
	baseURL := strings.TrimSpace(resolveMapOption(cfg.Options, "base_url"))
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiKey := resolveSecretOption(cfg.Options, "api_key", "api_key_env", "OPENAI_API_KEY")
	return newOpenAICompatibleModel(cfg.Model, baseURL, apiKey, nil)
}

func normalizeProviderName(provider string) string {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	return strings.ReplaceAll(normalized, "-", "_")
}

func resolveSecretOption(options map[string]string, valueKey, envKey, defaultEnv string) string {
	if options == nil {
		return strings.TrimSpace(os.Getenv(defaultEnv))
	}
	if value := strings.TrimSpace(options[valueKey]); value != "" {
		return value
	}
	if envVarName := strings.TrimSpace(options[envKey]); envVarName != "" {
		return strings.TrimSpace(os.Getenv(envVarName))
	}
	return strings.TrimSpace(os.Getenv(defaultEnv))
}

func resolveStringOption(options map[string]string, key, fallbackEnv string) string {
	if value := resolveMapOption(options, key); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(fallbackEnv))
}

func resolveMapOption(options map[string]string, key string) string {
	if options == nil {
		return ""
	}
	return strings.TrimSpace(options[key])
}
