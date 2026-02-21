package framework

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// ValidateModel performs v2-aware model validation.
// It allows handler-only tools without adapter bindings.
func ValidateModel(model *hyperterse.Model, project *Project) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	if model.Name == "" {
		return fmt.Errorf("name is required")
	}
	if project == nil {
		// non-project validation path is handled by parser.Validate
		return nil
	}

	// v2 path: ensure at least one tool exists and each tool has either statement/use
	// or a custom handler script.
	if len(project.Tools) == 0 {
		return fmt.Errorf("project root exists but no tool .terse files were discovered")
	}
	for toolName, tool := range project.Tools {
		if tool.Definition == nil {
			return fmt.Errorf("tool '%s' did not compile a tool definition", toolName)
		}
		if tool.Scripts.Handler == "" && len(tool.Definition.Use) == 0 {
			return fmt.Errorf("tool '%s' requires either handler or use adapter binding", toolName)
		}
	}
	return nil
}
