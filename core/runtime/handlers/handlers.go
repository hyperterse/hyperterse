package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	"github.com/hyperterse/hyperterse/core/types"
)

// QueryServiceHandler implements the QueryService
type QueryServiceHandler struct {
	executor *executor.Executor
	engine   *framework.Engine
}

// NewQueryServiceHandler creates a new QueryService handler
func NewQueryServiceHandler(exec *executor.Executor, eng *framework.Engine) *QueryServiceHandler {
	return &QueryServiceHandler{
		executor: exec,
		engine:   eng,
	}
}

// ExecuteQuery executes a query with context propagation for cancellation support
func (h *QueryServiceHandler) ExecuteQuery(ctx context.Context, req *runtime.ExecuteQueryRequest) (*runtime.ExecuteQueryResponse, error) {
	log := logger.New("handler")
	log.InfofCtx(ctx, map[string]any{
		observability.AttrQueryName: req.QueryName,
	}, "Executing query via handler: %s", req.QueryName)
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrQueryName: req.QueryName,
	}, "Input count: %d", len(req.Inputs))

	// Parse inputs from JSON strings
	inputs := make(map[string]any)
	for key, valueJSON := range req.Inputs {
		var value any
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, treat as string
			log.DebugfCtx(ctx, map[string]any{
				observability.AttrQueryName: req.QueryName,
			}, "Failed to unmarshal input '%s', treating as string", key)
			value = valueJSON
		}
		inputs[key] = value
	}

	// Execute via framework engine (supports route auth/transforms/custom handlers)
	var results []map[string]any
	var err error
	if h.engine != nil {
		results, err = h.engine.Execute(ctx, req.QueryName, inputs)
	} else {
		results, err = h.executor.ExecuteQuery(ctx, req.QueryName, inputs)
	}
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrQueryName: req.QueryName,
		}, "Query execution failed: %v", err)
		return &runtime.ExecuteQueryResponse{
			Success: false,
			Error:   err.Error(),
			Results: nil,
		}, nil
	}

	log.DebugfCtx(ctx, map[string]any{
		observability.AttrQueryName: req.QueryName,
	}, "Query executed successfully, converting %d result(s) to proto format", len(results))

	// Convert results to proto format
	protoResults := make([]*runtime.ResultRow, len(results))
	for i, row := range results {
		fields := make(map[string]string)
		for key, value := range row {
			// Convert value to JSON string
			valueJSON, err := json.Marshal(value)
			if err != nil {
				valueJSON = fmt.Appendf(nil, "%v", value)
			}
			fields[key] = string(valueJSON)
		}
		protoResults[i] = &runtime.ResultRow{
			Fields: fields,
		}
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrQueryName: req.QueryName,
	}, "Query execution completed successfully")
	return &runtime.ExecuteQueryResponse{
		Success: true,
		Error:   "",
		Results: protoResults,
	}, nil
}

// MCPServiceHandler implements the MCPService
type MCPServiceHandler struct {
	executor *executor.Executor
	model    *hyperterse.Model
	engine   *framework.Engine
}

// NewMCPServiceHandler creates a new MCPService handler
func NewMCPServiceHandler(exec *executor.Executor, model *hyperterse.Model, eng *framework.Engine) *MCPServiceHandler {
	return &MCPServiceHandler{
		executor: exec,
		model:    model,
		engine:   eng,
	}
}

// ListTools returns all available queries as MCP tools
func (h *MCPServiceHandler) ListTools(ctx context.Context, req *runtime.ListToolsRequest) (*runtime.ListToolsResponse, error) {
	log := logger.New("mcp")
	log.InfofCtx(ctx, nil, "Listing MCP tools")
	log.DebugfCtx(ctx, nil, "Query count: %d", len(h.model.Queries))

	tools := make([]*runtime.Tool, 0, len(h.model.Queries))

	for _, query := range h.model.Queries {
		// Build tool inputs map
		toolInputs := make(map[string]*runtime.ToolInput)
		for _, input := range query.Inputs {
			toolInputs[input.Name] = &runtime.ToolInput{
				Type:         types.PrimitiveEnumToString(input.Type),
				Description:  input.Description,
				Optional:     input.Optional,
				DefaultValue: input.DefaultValue,
			}
		}

		tools = append(tools, &runtime.Tool{
			Name:        query.Name,
			Description: query.Description,
			Inputs:      toolInputs,
		})
		log.DebugfCtx(ctx, map[string]any{
			observability.AttrQueryName: query.Name,
		}, "Added tool: %s", query.Name)
	}

	log.InfofCtx(ctx, nil, "Listed %d MCP tool(s)", len(tools))
	return &runtime.ListToolsResponse{
		Tools: tools,
	}, nil
}

// CallTool executes a tool (query) by name with context propagation
func (h *MCPServiceHandler) CallTool(ctx context.Context, req *runtime.CallToolRequest) (*runtime.CallToolResponse, error) {
	log := logger.New("mcp")
	log.InfofCtx(ctx, map[string]any{
		observability.AttrQueryName: req.Name,
	}, "Calling MCP tool: %s", req.Name)
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrQueryName: req.Name,
	}, "Argument count: %d", len(req.Arguments))

	// Parse arguments from JSON strings
	// Arguments are stored as JSON-encoded strings (e.g., "\"pending\"" for string "pending")
	inputs := make(map[string]any)
	for key, valueJSON := range req.Arguments {
		var value any
		// Try to unmarshal the JSON-encoded value
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, the valueJSON might be a raw string
			// Try to unquote it if it looks like a JSON string
			trimmed := strings.TrimSpace(valueJSON)
			if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
				// It's a JSON string, try to unmarshal it properly
				var strValue string
				if err := json.Unmarshal([]byte(trimmed), &strValue); err == nil {
					value = strValue
				} else {
					// Fallback: remove quotes manually
					value = trimmed[1 : len(trimmed)-1]
				}
			} else {
				// Not a JSON string, use as-is
				value = valueJSON
			}
		}
		inputs[key] = value
		log.DebugfCtx(ctx, map[string]any{
			observability.AttrQueryName: req.Name,
		}, "Parsed argument: %s", key)
	}

	// Execute via framework engine (supports route auth/transforms/custom handlers)
	var results []map[string]any
	var err error
	if h.engine != nil {
		results, err = h.engine.Execute(ctx, req.Name, inputs)
	} else {
		results, err = h.executor.ExecuteQuery(ctx, req.Name, inputs)
	}
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrQueryName: req.Name,
		}, "Tool execution failed: %v", err)
		errorJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return &runtime.CallToolResponse{
			Content: string(errorJSON),
			IsError: true,
		}, nil
	}

	log.DebugfCtx(ctx, map[string]any{
		observability.AttrQueryName: req.Name,
	}, "Tool executed successfully, marshaling %d result(s)", len(results))

	// Convert results to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrQueryName: req.Name,
		}, "Failed to marshal results: %v", err)
		errorJSON, _ := json.Marshal(map[string]string{"error": "failed to serialize results"})
		return &runtime.CallToolResponse{
			Content: string(errorJSON),
			IsError: true,
		}, nil
	}

	log.DebugfCtx(ctx, map[string]any{
		observability.AttrQueryName: req.Name,
	}, "Response size: %d bytes", len(resultsJSON))
	log.InfofCtx(ctx, map[string]any{
		observability.AttrQueryName: req.Name,
	}, "MCP tool call completed successfully")
	return &runtime.CallToolResponse{
		Content: string(resultsJSON),
		IsError: false,
	}, nil
}
