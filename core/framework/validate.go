package framework

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// ValidateModel performs v2-aware model validation.
// It intentionally relaxes legacy assumptions for handler-only routes.
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

	// v2 path: ensure at least one route exists and each route has either statement/use
	// or a custom handler script.
	if len(project.Routes) == 0 {
		return fmt.Errorf("app directory exists but no route .terse files were discovered")
	}
	for toolName, route := range project.Routes {
		if route.Query == nil {
			return fmt.Errorf("route '%s' did not compile a query", toolName)
		}
		if route.Scripts.Handler == "" && len(route.Query.Use) == 0 {
			return fmt.Errorf("route '%s' requires either scripts.handler or use adapter binding", toolName)
		}
	}
	return nil
}
