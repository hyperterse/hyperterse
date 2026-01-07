package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/hyperterse/hyperterse/core/pb/runtime"
)

// QueryServiceHandler implements the QueryService
type QueryServiceHandler struct {
	executor *executor.Executor
}

// NewQueryServiceHandler creates a new QueryService handler
func NewQueryServiceHandler(exec *executor.Executor) *QueryServiceHandler {
	return &QueryServiceHandler{
		executor: exec,
	}
}

// ExecuteQuery executes a query
func (h *QueryServiceHandler) ExecuteQuery(ctx context.Context, req *runtime.ExecuteQueryRequest) (*runtime.ExecuteQueryResponse, error) {
	// Parse inputs from JSON strings
	inputs := make(map[string]interface{})
	for key, valueJSON := range req.Inputs {
		var value interface{}
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, treat as string
			value = valueJSON
		}
		inputs[key] = value
	}

	// Execute the query
	results, err := h.executor.ExecuteQuery(req.QueryName, inputs)
	if err != nil {
		logger.New("runtime").PrintError("Query execution failed", err)
		return &runtime.ExecuteQueryResponse{
			Success: false,
			Error:   err.Error(),
			Results: nil,
		}, nil
	}

	// Convert results to proto format
	protoResults := make([]*runtime.ResultRow, len(results))
	for i, row := range results {
		fields := make(map[string]string)
		for key, value := range row {
			// Convert value to JSON string
			valueJSON, err := json.Marshal(value)
			if err != nil {
				valueJSON = []byte(fmt.Sprintf("%v", value))
			}
			fields[key] = string(valueJSON)
		}
		protoResults[i] = &runtime.ResultRow{
			Fields: fields,
		}
	}

	return &runtime.ExecuteQueryResponse{
		Success: true,
		Error:   "",
		Results: protoResults,
	}, nil
}

// MCPServiceHandler implements the MCPService
type MCPServiceHandler struct {
	executor *executor.Executor
	model    *pb.Model
}

// NewMCPServiceHandler creates a new MCPService handler
func NewMCPServiceHandler(exec *executor.Executor, model *pb.Model) *MCPServiceHandler {
	return &MCPServiceHandler{
		executor: exec,
		model:    model,
	}
}

// ListTools returns all available queries as MCP tools
func (h *MCPServiceHandler) ListTools(ctx context.Context, req *runtime.ListToolsRequest) (*runtime.ListToolsResponse, error) {
	tools := make([]*runtime.Tool, 0, len(h.model.Queries))

	for _, query := range h.model.Queries {
		// Build tool inputs map
		toolInputs := make(map[string]*runtime.ToolInput)
		for _, input := range query.Inputs {
			toolInputs[input.Name] = &runtime.ToolInput{
				Type:         input.Type,
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
	}

	return &runtime.ListToolsResponse{
		Tools: tools,
	}, nil
}

// CallTool executes a tool (query) by name
func (h *MCPServiceHandler) CallTool(ctx context.Context, req *runtime.CallToolRequest) (*runtime.CallToolResponse, error) {
	// Parse arguments from JSON strings
	inputs := make(map[string]interface{})
	for key, valueJSON := range req.Arguments {
		var value interface{}
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, treat as string
			value = valueJSON
		}
		inputs[key] = value
	}

	// Execute the query
	results, err := h.executor.ExecuteQuery(req.Name, inputs)
	if err != nil {
		logger.New("runtime").PrintError("Tool execution failed", err)
		errorJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return &runtime.CallToolResponse{
			Content: string(errorJSON),
			IsError: true,
		}, nil
	}

	// Convert results to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		logger.New("runtime").PrintError("Failed to marshal results", err)
		errorJSON, _ := json.Marshal(map[string]string{"error": "failed to serialize results"})
		return &runtime.CallToolResponse{
			Content: string(errorJSON),
			IsError: true,
		}, nil
	}

	return &runtime.CallToolResponse{
		Content: string(resultsJSON),
		IsError: false,
	}, nil
}
