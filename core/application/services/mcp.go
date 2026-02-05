package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// MCPService implements the unified MCP service used by all transports
type MCPService struct {
	executor interfaces.Executor
	model    *hyperterse.Model
}

// NewMCPService creates a new MCPService
func NewMCPService(executor interfaces.Executor, model *hyperterse.Model) *MCPService {
	return &MCPService{
		executor: executor,
		model:    model,
	}
}

// ListTools returns all available queries as MCP tools
func (s *MCPService) ListTools(ctx context.Context, req *runtime.ListToolsRequest) (*runtime.ListToolsResponse, error) {
	tools := make([]*runtime.Tool, 0, len(s.model.Queries))

	for _, query := range s.model.Queries {
		// Build tool inputs map
		toolInputs := make(map[string]*runtime.ToolInput)
		for _, input := range query.Inputs {
			toolInputs[input.Name] = &runtime.ToolInput{
				Type:         primitiveEnumToString(input.Type),
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

// CallTool executes a tool (query) by name with context propagation
func (s *MCPService) CallTool(ctx context.Context, req *runtime.CallToolRequest) (*runtime.CallToolResponse, error) {
	// Parse arguments from JSON strings
	inputs := make(map[string]any)
	for key, valueJSON := range req.Arguments {
		var value any
		// Try to unmarshal the JSON-encoded value
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, the valueJSON might be a raw string
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
	}

	// Execute the query with context for cancellation support
	results, err := s.executor.ExecuteQuery(ctx, req.Name, inputs)
	if err != nil {
		errorJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return &runtime.CallToolResponse{
			Content: string(errorJSON),
			IsError: true,
		}, nil
	}

	// Convert results to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
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

// primitiveEnumToString converts a protobuf Primitive enum to its string representation
// This is a temporary helper until types package is generated
func primitiveEnumToString(p primitives.Primitive) string {
	switch p {
	case primitives.Primitive_PRIMITIVE_STRING:
		return "string"
	case primitives.Primitive_PRIMITIVE_INT:
		return "int"
	case primitives.Primitive_PRIMITIVE_FLOAT:
		return "float"
	case primitives.Primitive_PRIMITIVE_BOOLEAN:
		return "boolean"
	case primitives.Primitive_PRIMITIVE_DATETIME:
		return "datetime"
	default:
		return "string" // Default fallback
	}
}
