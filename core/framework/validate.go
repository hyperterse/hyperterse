package framework

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// ValidateModel performs v2-aware model validation.
// It intentionally relaxes legacy assumptions for handler-only tools.
func ValidateModel(model *hyperterse.Model, project *Project) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	if model.Name == "" {
		return fmt.Errorf("name is required")
	}
	if project == nil {
		// legacy validation path is handled by parser.Validate
		return nil
	}

	// v2 path: ensure at least one tool exists and each tool has either statement/use
	// or a custom handler script.
	if len(project.Tools) == 0 {
		return fmt.Errorf("app directory exists but no tool .terse files were discovered")
	}
	for toolName, tool := range project.Tools {
		if tool.Query == nil {
			return fmt.Errorf("tool '%s' did not compile a query", toolName)
		}
		if tool.Scripts.Handler == "" && len(tool.Query.Use) == 0 {
			return fmt.Errorf("tool '%s' requires either scripts.handler or use adapter binding", toolName)
		}
	}
	return nil
}
