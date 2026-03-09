package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func TestResolveAgentModel_SubstitutesModelOptionEnvVars(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://example.openai.compat/v1")
	t.Setenv("OPENAI_TOKEN", "token-from-env")

	cfg := &hyperterse.AgentModelConfig{
		Provider: "openai_compatible",
		Model:    "gpt-4o-mini",
		Options: map[string]string{
			"base_url": "{{ env.OPENAI_BASE_URL }}",
			"api_key":  "{{ env.OPENAI_TOKEN }}",
		},
	}

	llm, err := resolveAgentModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	model, ok := llm.(*openAICompatibleModel)
	if !ok {
		t.Fatalf("expected *openAICompatibleModel, got %T", llm)
	}
	if model.baseURL != "https://example.openai.compat/v1" {
		t.Fatalf("expected substituted baseURL, got %q", model.baseURL)
	}
	if model.apiKey != "token-from-env" {
		t.Fatalf("expected substituted api key, got %q", model.apiKey)
	}
}

func TestResolveAgentModel_FailsWhenModelOptionEnvMissing(t *testing.T) {
	cfg := &hyperterse.AgentModelConfig{
		Provider: "openai_compatible",
		Model:    "gpt-4o-mini",
		Options: map[string]string{
			"base_url": "{{ env.MISSING_OPENAI_BASE_URL }}",
		},
	}

	_, err := resolveAgentModel(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error when env substitution is missing")
	}
	if !strings.Contains(err.Error(), "MISSING_OPENAI_BASE_URL") {
		t.Fatalf("expected missing env var in error, got %q", err.Error())
	}
}
