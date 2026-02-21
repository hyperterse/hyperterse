package framework

import (
	"context"
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
)

// Engine orchestrates auth, transforms, custom handlers, and DB execution.
type Engine struct {
	model        *hyperterse.Model
	executor     *executor.Executor
	project      *Project
	authRegistry *AuthRegistry
	scriptRT     *ScriptRuntime
}

func NewEngine(model *hyperterse.Model, exec *executor.Executor, project *Project) *Engine {
	return &Engine{
		model:        model,
		executor:     exec,
		project:      project,
		authRegistry: NewAuthRegistry(),
		scriptRT:     NewScriptRuntime(),
	}
}

func (e *Engine) Project() *Project {
	return e.project
}

func (e *Engine) GetTool(toolName string) *Tool {
	if e.project == nil {
		return nil
	}
	return e.project.Tools[toolName]
}

func (e *Engine) Execute(ctx context.Context, toolName string, userInputs map[string]any) ([]map[string]any, error) {
	tool := e.GetTool(toolName)
	if tool == nil {
		return e.executor.ExecuteTool(ctx, toolName, userInputs)
	}

	if err := e.authRegistry.Authorize(ctx, tool); err != nil {
		return nil, err
	}

	currentInputs := userInputs
	if bundlePath := tool.BundleOutputs["input_transform"]; bundlePath != "" {
		exportName := tool.Scripts.InputTransformExport
		if exportName == "" {
			exportName = "default"
		}
		transformed, err := e.scriptRT.Invoke(ctx, e.projectVendorPath(), bundlePath, exportName, map[string]any{
			"inputs": userInputs,
			"tool":   tool.ToolPath,
		})
		if err != nil {
			return nil, fmt.Errorf("input transform failed for tool %s: %w", toolName, err)
		}
		if m, ok := transformed.(map[string]any); ok {
			currentInputs = m
		}
	}

	var results []map[string]any
	if bundlePath := tool.BundleOutputs["handler"]; bundlePath != "" {
		exportName := tool.Scripts.HandlerExport
		if exportName == "" {
			exportName = "default"
		}
		customResult, err := e.scriptRT.Invoke(ctx, e.projectVendorPath(), bundlePath, exportName, map[string]any{
			"inputs": currentInputs,
			"tool":   tool.ToolPath,
		})
		if err != nil {
			return nil, fmt.Errorf("handler script failed for tool %s: %w", toolName, err)
		}
		results = coerceResults(customResult)
	} else {
		dbResults, err := e.executor.ExecuteTool(ctx, toolName, currentInputs)
		if err != nil {
			return nil, err
		}
		results = dbResults
	}

	if bundlePath := tool.BundleOutputs["output_transform"]; bundlePath != "" {
		exportName := tool.Scripts.OutputTransformExport
		if exportName == "" {
			exportName = "default"
		}
		transformed, err := e.scriptRT.Invoke(ctx, e.projectVendorPath(), bundlePath, exportName, map[string]any{
			"results": results,
			"tool":    tool.ToolPath,
		})
		if err != nil {
			return nil, fmt.Errorf("output transform failed for tool %s: %w", toolName, err)
		}
		results = coerceResults(transformed)
	}

	return results, nil
}

func coerceResults(value any) []map[string]any {
	if value == nil {
		return []map[string]any{}
	}
	if typed, ok := value.([]map[string]any); ok {
		return typed
	}
	if rows, ok := value.([]any); ok {
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				out = append(out, m)
			} else {
				out = append(out, map[string]any{"value": row})
			}
		}
		return out
	}
	if m, ok := value.(map[string]any); ok {
		return []map[string]any{m}
	}
	return []map[string]any{{"value": value}}
}

func (e *Engine) projectVendorPath() string {
	if e.project == nil {
		return ""
	}
	return e.project.VendorBundle
}
