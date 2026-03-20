package framework

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// ValidateModel performs v2-aware model validation.
// Each tool must define exactly one execution mode:
// - adapter-backed (`use`)
// - script-backed (`handler`)
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

	if len(project.Tools) == 0 && len(project.Prompts) == 0 && len(project.Resources) == 0 &&
		len(project.Templates) == 0 && len(project.Agents) == 0 {
		return fmt.Errorf("project root exists but no tool, prompt, resource, template, or agent .terse files were discovered")
	}
	for toolName, tool := range project.Tools {
		if tool.Definition == nil {
			return fmt.Errorf("tool '%s' did not compile a tool definition", toolName)
		}
		hasHandler := tool.Scripts.Handler != ""
		hasUse := len(tool.Definition.Use) > 0

		if hasHandler && hasUse {
			return fmt.Errorf("tool '%s' cannot define both handler and use adapter binding", toolName)
		}
		if !hasHandler && !hasUse {
			return fmt.Errorf("tool '%s' requires exactly one of handler or use adapter binding", toolName)
		}
	}

	for agentName, compiledAgent := range project.Agents {
		if compiledAgent.Definition == nil {
			return fmt.Errorf("agent '%s' did not compile an agent definition", agentName)
		}
		definition := compiledAgent.Definition
		if definition.Name == "" {
			return fmt.Errorf("agent '%s' is missing name", agentName)
		}
		if definition.Instruction == "" {
			return fmt.Errorf("agent '%s' is missing instruction", agentName)
		}
		if definition.Model == nil {
			return fmt.Errorf("agent '%s' is missing model configuration", agentName)
		}
		if definition.Model.Provider == "" {
			return fmt.Errorf("agent '%s' is missing model.provider", agentName)
		}
		if definition.Model.Model == "" {
			return fmt.Errorf("agent '%s' is missing model.model", agentName)
		}
		if definition.ToolAccess != nil {
			switch AgentToolAccessMode(definition.ToolAccess.Mode) {
			case AgentToolAccessModeAllowAll, AgentToolAccessModeAllowNone, AgentToolAccessModeInherit:
				// no-op
			case AgentToolAccessModeAllowList:
				if len(definition.ToolAccess.Tools) == 0 {
					return fmt.Errorf("agent '%s' has allow_list tool access mode but empty tool list", agentName)
				}
			default:
				return fmt.Errorf("agent '%s' has unsupported tool access mode %q", agentName, definition.ToolAccess.Mode)
			}
			for _, toolName := range definition.ToolAccess.Tools {
				if _, exists := project.Tools[toolName]; !exists {
					return fmt.Errorf("agent '%s' references unknown tool %q", agentName, toolName)
				}
			}
		}
	}
	return nil
}
