package parser

import (
	"strings"
	"testing"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func TestValidate_AllowsPromptOnlyModel(t *testing.T) {
	model := &hyperterse.Model{
		Name: "prompt_only_model",
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name: "greeting_prompt",
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "hello"},
				},
			},
		},
	}

	if err := Validate(model); err != nil {
		t.Fatalf("expected prompt-only model to be valid, got error: %v", err)
	}
}

func TestValidate_RejectsDuplicatePromptResourceAndTemplateIdentifiers(t *testing.T) {
	model := &hyperterse.Model{
		Name: "dup_identifier_model",
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name: "dup_prompt",
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "first"},
				},
			},
			{
				Name: "dup_prompt",
				Messages: []*hyperterse.PromptMessage{
					{Role: "assistant", Text: "second"},
				},
			},
		},
		Resources: []*hyperterse.ResourceDefinition{
			{Uri: "memory://dup", Text: "one"},
			{Uri: "memory://dup", Text: "two"},
		},
		ResourceTemplates: []*hyperterse.ResourceTemplateDefinition{
			{UriTemplate: "memory://tmpl/{id}", TextTemplate: "one"},
			{UriTemplate: "memory://tmpl/{id}", TextTemplate: "two"},
		},
	}

	err := Validate(model)
	if err == nil {
		t.Fatal("expected duplicate identifiers to fail validation")
	}

	errorText := err.Error()
	expectedSubstrings := []string{
		"Prompt 'dup_prompt' - name must be unique",
		"Resource 'memory://dup' - uri must be unique",
		"Resource template 'memory://tmpl/{id}' - uri_template must be unique",
	}
	for _, expected := range expectedSubstrings {
		if !strings.Contains(errorText, expected) {
			t.Fatalf("expected error to contain %q, got: %s", expected, errorText)
		}
	}
}
