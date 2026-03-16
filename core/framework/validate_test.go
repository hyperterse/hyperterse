package framework

import (
	"strings"
	"testing"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func TestValidateModel_AllowsPromptOnlyProject(t *testing.T) {
	model := &hyperterse.Model{Name: "project_model"}
	project := &Project{
		Prompts: map[string]*Prompt{
			"greeting": {
				Name:       "greeting",
				Definition: &hyperterse.PromptDefinition{Name: "greeting"},
			},
		},
	}

	if err := ValidateModel(model, project); err != nil {
		t.Fatalf("expected prompt-only project to validate, got: %v", err)
	}
}

func TestValidateModel_RejectsProjectWithoutMcpEntities(t *testing.T) {
	model := &hyperterse.Model{Name: "project_model"}
	project := &Project{}

	err := ValidateModel(model, project)
	if err == nil {
		t.Fatal("expected validation error for empty project")
	}
	if !strings.Contains(err.Error(), "no tool/prompt/resource .terse files were discovered") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
