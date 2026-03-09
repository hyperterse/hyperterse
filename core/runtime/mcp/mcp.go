package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	serverName          = "hyperterse"
	serverVersion       = "1.0.0"
	searchToolName      = "search"
	executeToolName     = "execute"
	executeToolParam    = "tool"
	executeInputsParam  = "inputs"
	searchQueryParam    = "query"
	relevanceScoreField = "relevance_score"
)

// Adapter configures and exposes an MCP SDK server backed by the existing
// Hyperterse execution stack (framework engine + tool executor).
type Adapter struct {
	model       *hyperterse.Model
	executor    *executor.Executor
	engine      *framework.Engine
	server      *mcpsdk.Server
	searchIndex *toolSearchIndex
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
		model:       model,
		executor:    exec,
		engine:      eng,
		server:      server,
		searchIndex: newToolSearchIndex(model.Tools),
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

	a.server.AddTool(&mcpsdk.Tool{
		Name:        searchToolName,
		Description: "Search available tools by natural-language intent and tool metadata.",
		InputSchema: buildSearchInputSchema(),
	}, a.callSearchTool)
	log.Debugf("Registered MCP tool: %s", searchToolName)

	a.server.AddTool(&mcpsdk.Tool{
		Name:        executeToolName,
		Description: "Execute a tool by name using a structured input object.",
		InputSchema: buildExecuteInputSchema(),
	}, a.callExecuteTool)
	log.Debugf("Registered MCP tool: %s", executeToolName)

	return nil
}

func (a *Adapter) callSearchTool(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	log := logger.New("mcp")
	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: searchToolName,
	}, "Calling MCP tool: %s", searchToolName)

	inputs, invalidParamsErr := parseArguments(req)
	if invalidParamsErr != nil {
		return invalidParamsErr, nil
	}

	queryRaw, ok := inputs[searchQueryParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", searchQueryParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	query, ok := queryRaw.(string)
	if !ok || strings.TrimSpace(query) == "" {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be a non-empty string", searchQueryParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	limit := a.searchLimit()
	hits := a.searchIndex.Search(query, limit)
	results := make([]map[string]any, 0, len(hits))
	for _, hit := range hits {
		results = append(results, map[string]any{
			"name":              hit.Tool.Name,
			relevanceScoreField: hit.RelevanceScore,
			"description":       hit.Tool.Description,
			"inputs":            buildInputMetadata(hit.Tool),
		})
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: searchToolName,
	}, "MCP tool call completed successfully")

	return resultPayload(results), nil
}

func (a *Adapter) callExecuteTool(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	log := logger.New("mcp")

	// Preserve tool-level auth behavior by forwarding incoming HTTP headers from
	// transport metadata into the framework auth context.
	if extra := req.GetExtra(); extra != nil && extra.Header != nil {
		ctx = framework.WithRequestHeaders(ctx, extra.Header)
	}

	inputs, invalidParamsErr := parseArguments(req)
	if invalidParamsErr != nil {
		return invalidParamsErr, nil
	}

	toolNameRaw, ok := inputs[executeToolParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", executeToolParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}
	toolName, ok := toolNameRaw.(string)
	if !ok || strings.TrimSpace(toolName) == "" {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be a non-empty string", executeToolParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	toolInputsRaw, ok := inputs[executeInputsParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", executeInputsParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	toolInputs, ok := toolInputsRaw.(map[string]any)
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be an object", executeInputsParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: toolName,
	}, "Calling MCP tool: %s", toolName)

	results, err := a.executeTool(ctx, toolName, toolInputs)
	if err != nil {
		return toolError(err.Error(), jsonrpcsdk.CodeInternalError), nil
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: toolName,
	}, "MCP tool call completed successfully")

	return resultPayload(results), nil
}

func (a *Adapter) executeTool(ctx context.Context, toolName string, inputs map[string]any) ([]map[string]any, error) {
	if a.engine != nil {
		return a.engine.Execute(ctx, toolName, inputs)
	}
	return a.executor.ExecuteTool(ctx, toolName, inputs)
}

func parseArguments(req *mcpsdk.CallToolRequest) (map[string]any, *mcpsdk.CallToolResult) {
	if len(req.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}

	var inputs map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &inputs); err != nil {
		return nil, toolError(
			fmt.Sprintf("invalid params: %v", err),
			jsonrpcsdk.CodeInvalidParams,
		)
	}
	if inputs == nil {
		return map[string]any{}, nil
	}
	return inputs, nil
}

func resultPayload(results []map[string]any) *mcpsdk.CallToolResult {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return toolError("failed to serialize results", jsonrpcsdk.CodeInternalError)
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(resultsJSON)},
		},
	}
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

func (a *Adapter) searchLimit() int {
	if a.model == nil || a.model.ToolDefaults == nil || a.model.ToolDefaults.Search == nil {
		return defaultSearchLimit
	}
	searchDefaults := a.model.ToolDefaults.Search
	if searchDefaults.HasLimit && searchDefaults.Limit > 0 {
		return int(searchDefaults.Limit)
	}
	return defaultSearchLimit
}

func buildSearchInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			searchQueryParam: map[string]any{
				"type":        "string",
				"description": "Natural-language query used to find relevant tools.",
			},
		},
		"required":             []string{searchQueryParam},
		"additionalProperties": false,
	}
}

func buildExecuteInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			executeToolParam: map[string]any{
				"type":        "string",
				"description": "Name of the target tool to execute.",
			},
			executeInputsParam: map[string]any{
				"type":        "object",
				"description": "Structured inputs for the target tool.",
			},
		},
		"required":             []string{executeToolParam, executeInputsParam},
		"additionalProperties": false,
	}
}

func buildInputMetadata(tool *hyperterse.Tool) []map[string]any {
	if tool == nil || len(tool.Inputs) == 0 {
		return []map[string]any{}
	}

	metadata := make([]map[string]any, 0, len(tool.Inputs))
	for _, input := range tool.Inputs {
		if input == nil {
			continue
		}
		entry := map[string]any{
			"name":     input.Name,
			"type":     types.PrimitiveEnumToString(input.Type),
			"optional": input.Optional,
		}
		if input.Description != "" {
			entry["description"] = input.Description
		}
		if input.DefaultValue != "" {
			entry["default_value"] = input.DefaultValue
		}
		metadata = append(metadata, entry)
	}
	return metadata
}
