package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	"github.com/hyperterse/hyperterse/core/types"
	jsonrpcsdk "github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "hyperterse"
	serverVersion = "1.0.0"
)

// Adapter configures and exposes an MCP SDK server backed by the existing
// Hyperterse execution stack (framework engine + tool executor).
type Adapter struct {
	model    *hyperterse.Model
	executor *executor.Executor
	engine   *framework.Engine
	server   *mcpsdk.Server
}

// New creates an MCP SDK adapter and registers all tools.
func New(model *hyperterse.Model, exec *executor.Executor, eng *framework.Engine) (*Adapter, error) {
	if model == nil {
		return nil, fmt.Errorf("mcp adapter requires a model")
	}
	if exec == nil {
		return nil, fmt.Errorf("mcp adapter requires an executor")
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	adapter := &Adapter{
		model:    model,
		executor: exec,
		engine:   eng,
		server:   server,
	}

	if err := adapter.registerTools(); err != nil {
		return nil, err
	}

	return adapter, nil
}

func (a *Adapter) Server() *mcpsdk.Server {
	return a.server
}

func (a *Adapter) registerTools() error {
	log := logger.New("mcp")
	for _, tool := range a.model.Tools {
		if tool == nil {
			continue
		}

		tool := tool
		a.server.AddTool(&mcpsdk.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: buildInputSchema(tool),
		}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			return a.callTool(ctx, req, tool)
		})

		log.Debugf("Registered MCP tool: %s", tool.Name)
	}
	return nil
}

func (a *Adapter) callTool(ctx context.Context, req *mcpsdk.CallToolRequest, tool *hyperterse.Tool) (*mcpsdk.CallToolResult, error) {
	log := logger.New("mcp")
	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: tool.Name,
	}, "Calling MCP tool: %s", tool.Name)

	// Preserve tool-level auth behavior by forwarding incoming HTTP headers from
	// transport metadata into the framework auth context.
	if extra := req.GetExtra(); extra != nil && extra.Header != nil {
		ctx = framework.WithRequestHeaders(ctx, extra.Header)
	}

	var inputs map[string]any
	if len(req.Params.Arguments) > 0 {
		if err := json.Unmarshal(req.Params.Arguments, &inputs); err != nil {
			return toolError(
				fmt.Sprintf("invalid params: %v", err),
				jsonrpcsdk.CodeInvalidParams,
			), nil
		}
	} else {
		inputs = map[string]any{}
	}

	var (
		results []map[string]any
		err     error
	)
	if a.engine != nil {
		results, err = a.engine.Execute(ctx, tool.Name, inputs)
	} else {
		results, err = a.executor.ExecuteTool(ctx, tool.Name, inputs)
	}
	if err != nil {
		return toolError(err.Error(), jsonrpcsdk.CodeInternalError), nil
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return toolError("failed to serialize results", jsonrpcsdk.CodeInternalError), nil
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: tool.Name,
	}, "MCP tool call completed successfully")

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(resultsJSON)},
		},
	}, nil
}

func toolError(message string, code int) *mcpsdk.CallToolResult {
	payload := map[string]any{
		"error": message,
		"code":  code,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"internal error","code":-32603}`)
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(data)},
		},
		IsError: true,
	}
}

func buildInputSchema(tool *hyperterse.Tool) map[string]any {
	properties := make(map[string]any)
	required := make([]string, 0, len(tool.Inputs))

	for _, input := range tool.Inputs {
		if input == nil {
			continue
		}

		primitiveType := types.PrimitiveEnumToString(input.Type)
		schemaType, schemaFormat := primitiveToJSONSchema(primitiveType)
		prop := map[string]any{
			"type": schemaType,
		}
		if schemaFormat != "" {
			prop["format"] = schemaFormat
		}
		if input.Description != "" {
			prop["description"] = input.Description
		}
		if input.DefaultValue != "" {
			prop["default"] = parseDefaultValue(input.DefaultValue, primitiveType)
		}

		properties[input.Name] = prop
		if !input.Optional {
			required = append(required, input.Name)
		}
	}

	inputSchema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}
	return inputSchema
}

func primitiveToJSONSchema(primitiveType string) (schemaType string, schemaFormat string) {
	switch primitiveType {
	case "boolean":
		return "boolean", ""
	case "int":
		return "integer", ""
	case "float":
		return "number", ""
	case "datetime":
		return "string", "date-time"
	case "string":
		return "string", ""
	default:
		// Unknown primitives fall back to string to keep tools callable.
		return "string", ""
	}
}

func parseDefaultValue(valueStr, primitiveType string) any {
	switch primitiveType {
	case "int":
		var v int64
		if err := json.Unmarshal([]byte(valueStr), &v); err == nil {
			return v
		}
		return valueStr
	case "float":
		var v float64
		if err := json.Unmarshal([]byte(valueStr), &v); err == nil {
			return v
		}
		return valueStr
	case "boolean":
		var v bool
		if err := json.Unmarshal([]byte(valueStr), &v); err == nil {
			return v
		}
		return valueStr
	default:
		return valueStr
	}
}
